package zstack

import (
	"context"
	"fmt"
	"os"
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
	for _, key := range []string{"ZSTACK_ACC_IMAGE_UUID", "ZSTACK_ACC_NETWORK_UUID", "ZSTACK_ACC_BACKUP_STORAGE_UUID"} {
		require.NotEmpty(t, os.Getenv(key), key)
	}
	name := fmt.Sprintf("packer-quality-%d", time.Now().Unix())
	comm := os.Getenv("ZSTACK_ACC_COMMUNICATOR")
	if comm == "" {
		comm = "none"
	}
	raw := map[string]interface{}{
		"image_uuid":          os.Getenv("ZSTACK_ACC_IMAGE_UUID"),
		"network_uuid":        os.Getenv("ZSTACK_ACC_NETWORK_UUID"),
		"backup_storage_uuid": os.Getenv("ZSTACK_ACC_BACKUP_STORAGE_UUID"),
		"instance_name":       name, "image_name": name,
		"cpu_num": 1, "memory_size": 1024,
		"communicator": comm, "vm_running_timeout": "10m", "image_ready_timeout": "10m",
		"api_timeout": "10m", "cleanup_timeout": "5m",
		"skip_export": os.Getenv("ZSTACK_ACC_SKIP_EXPORT") == "1",
	}
	if comm == "ssh" {
		raw["ssh_username"] = os.Getenv("ZSTACK_ACC_SSH_USERNAME")
		raw["ssh_timeout"] = "3m"
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
	hook := &acceptanceHook{ssh: comm == "ssh"}
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
	// Input remains available after the temporary VM was expunged.
	d, err := b.config.AccessConfig.driver(ctx, time.Minute)
	require.NoError(t, err)
	source, err := d.GetImage(os.Getenv("ZSTACK_ACC_IMAGE_UUID"))
	require.NoError(t, err)
	require.Equal(t, "Ready", source.Status)
}

type acceptanceHook struct {
	ssh   bool
	after func(context.Context) error
}

func (h *acceptanceHook) Run(ctx context.Context, name string, ui packersdk.Ui, comm packersdk.Communicator, data interface{}) error {
	if name != packersdk.HookProvision {
		return nil
	}
	if h.ssh {
		cmd := &packersdk.RemoteCmd{Command: "uname -m && test -r /etc/os-release"}
		if err := cmd.RunWithUi(ctx, comm, ui); err != nil {
			return err
		}
		if cmd.ExitStatus() != 0 {
			return fmt.Errorf("acceptance command exited %d", cmd.ExitStatus())
		}
	}
	if h.after != nil {
		return h.after(ctx)
	}
	return nil
}
