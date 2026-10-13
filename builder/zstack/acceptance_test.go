package zstack

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/stretchr/testify/require"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
)

// Opt in explicitly; this test creates and deletes resources in the configured cloud.
// Credentials use the normal ZSTACK_* environment variables, never test fixtures.
func TestAcceptanceCloud(t *testing.T) {
	if os.Getenv("ZSTACK_ACC") != "1" {
		t.Skip("set ZSTACK_ACC=1 and ZSTACK_ACC_* resource IDs to run cloud acceptance")
	}
	for _, key := range []string{"ZSTACK_ACC_NETWORK_UUID", "ZSTACK_ACC_BACKUP_STORAGE_UUID"} {
		require.NotEmpty(t, os.Getenv(key), key)
	}
	imageUUID, imageURL := os.Getenv("ZSTACK_ACC_IMAGE_UUID"), os.Getenv("ZSTACK_ACC_IMAGE_URL")
	require.True(t, (imageUUID != "") != (imageURL != ""), "set exactly one of ZSTACK_ACC_IMAGE_UUID and ZSTACK_ACC_IMAGE_URL")
	name := fmt.Sprintf("packer-quality-%d", time.Now().UnixNano())
	comm := os.Getenv("ZSTACK_ACC_COMMUNICATOR")
	if comm == "" {
		comm = "none"
	}
	raw := map[string]interface{}{
		"network_uuid":        os.Getenv("ZSTACK_ACC_NETWORK_UUID"),
		"backup_storage_uuid": os.Getenv("ZSTACK_ACC_BACKUP_STORAGE_UUID"),
		"instance_name":       name, "image_name": name,
		"cpu_num": 1, "memory_size": 1024,
		"communicator": comm, "vm_running_timeout": "10m", "image_ready_timeout": "10m",
		"api_timeout": "10m", "cleanup_timeout": "5m",
		"skip_export": os.Getenv("ZSTACK_ACC_SKIP_EXPORT") == "1",
	}
	if imageURL != "" {
		raw["source_image_url"] = imageURL
		raw["source_image"] = name + "-source"
		raw["architecture"] = "x86_64"
	} else {
		raw["image_uuid"] = imageUUID
	}
	if comm == "ssh" {
		raw["ssh_username"] = os.Getenv("ZSTACK_ACC_SSH_USERNAME")
		raw["ssh_password"] = os.Getenv("ZSTACK_ACC_SSH_PASSWORD")
		raw["ssh_timeout"] = "5m"
		raw["ssh_clear_authorized_keys"] = true
	}
	if disk := os.Getenv("ZSTACK_ACC_DATA_DISK_OFFERING_UUID"); disk != "" {
		raw["data_disk_offering_uuids"] = []string{disk}
	}
	b := &Builder{}
	_, _, err := b.Prepare(raw)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	ui := &packersdk.BasicUi{Reader: os.Stdin, Writer: os.Stdout, ErrorWriter: os.Stderr}
	hook := &acceptanceHook{ssh: comm == "ssh", expectedUser: os.Getenv("ZSTACK_ACC_SSH_USERNAME")}
	scenario := os.Getenv("ZSTACK_ACC_SCENARIO")
	var cancelledVM string
	if scenario == "cancel" {
		hook.after = func(ctx context.Context) error { cancelledVM = b.config.InstanceUuid; cancel(); return ctx.Err() }
	}
	if scenario == "snapshot" {
		hook.after = func(ctx context.Context) error {
			d, e := b.config.AccessConfig.driver(ctx, 10*time.Minute)
			if e != nil {
				return e
			}
			id := newResource().UUID
			t.Logf("acceptance snapshot UUID: %s", id)
			snapshot, e := d.CreateVolumeSnapshot(b.config.RootVolumeUuid, param.CreateVolumeSnapshotParam{Params: param.CreateVolumeSnapshotParamDetail{Name: name + "-snapshot", ResourceUuid: &id}})
			if e != nil {
				return e
			}
			defer func() {
				cleanup, stop := context.WithTimeout(context.Background(), 5*time.Minute)
				defer stop()
				require.NoError(t, d.client.Delete(cleanup, "v1/volume-snapshots", snapshot.UUID, "Permissive"))
			}()
			child := &Builder{}
			_, _, e = child.Prepare(map[string]interface{}{"source_volume_snapshot_uuid": snapshot.UUID, "image_name": name + "-snapshot-output", "backup_storage_uuid": b.config.BackupStorageUuid, "skip_export": true, "image_ready_timeout": "10m"})
			if e != nil {
				return e
			}
			result, e := child.Run(ctx, ui, &packersdk.MockHook{})
			if result != nil {
				require.NoError(t, result.Destroy())
			}
			return e
		}
	}
	artifact, err := b.Run(ctx, ui, hook)
	if scenario == "cancel" {
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, artifact)
		require.NotEmpty(t, cancelledVM)
		require.Empty(t, b.config.InstanceUuid)
		d, e := b.config.AccessConfig.driver(context.Background(), time.Minute)
		require.NoError(t, e)
		vm, e := d.GetVmInstance(cancelledVM)
		require.Error(t, e)
		require.Nil(t, vm)
		return
	}
	if artifact != nil {
		t.Cleanup(func() { require.NoError(t, artifact.Destroy(), "remove acceptance output image") })
		t.Logf("output image UUID: %s", artifact.Id())
	}
	require.NoError(t, err)
	require.NotNil(t, artifact)
	require.Empty(t, b.config.InstanceUuid, "temporary VM must be expunged")
	require.Empty(t, artifact.State("cleanup_warnings"))
	// Existing input is preserved; an image imported by this build is removed.
	d, err := b.config.AccessConfig.driver(ctx, time.Minute)
	require.NoError(t, err)
	if imageUUID != "" {
		source, err := d.GetImage(imageUUID)
		require.NoError(t, err)
		require.Equal(t, "Ready", source.Status)
	} else {
		sources, err := d.QueryImage(name + "-source")
		require.NoError(t, err)
		require.Empty(t, sources, "temporary imported source image must be expunged")
	}
}

type acceptanceHook struct {
	ssh          bool
	expectedUser string
	after        func(context.Context) error
}

func (h *acceptanceHook) Run(ctx context.Context, name string, ui packersdk.Ui, comm packersdk.Communicator, data interface{}) error {
	if name != packersdk.HookProvision {
		return nil
	}
	if h.ssh {
		var output strings.Builder
		cmd := &packersdk.RemoteCmd{Command: "id -un && uname -m && test -r /etc/os-release", Stdout: &output}
		if err := cmd.RunWithUi(ctx, comm, ui); err != nil {
			return err
		}
		if cmd.ExitStatus() != 0 {
			return fmt.Errorf("acceptance command exited %d", cmd.ExitStatus())
		}
		if !strings.HasPrefix(output.String(), h.expectedUser+"\n") {
			return fmt.Errorf("SSH connected as an unexpected user")
		}
	}
	if h.after != nil {
		return h.after(ctx)
	}
	return nil
}
