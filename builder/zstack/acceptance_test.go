package zstack

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/stretchr/testify/require"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
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
	var artifact packersdk.Artifact
	if scenario == "key-cleanup" {
		require.Equal(t, "ssh", comm)
		require.Empty(t, os.Getenv("ZSTACK_ACC_SSH_PASSWORD"), "this scenario must authenticate with the generated key")
		d, e := b.config.AccessConfig.driver(ctx, b.config.APITimeout())
		require.NoError(t, e)
		d.secrets = append(d.secrets, b.config.SourceImageUrl)
		hook.after = func(ctx context.Context) error {
			return hook.checkTemporaryKey(ctx, ui, string(b.config.Comm.SSHPublicKey), true)
		}
		observer := &acceptanceStopObserver{Driver: d, beforeStop: func(ctx context.Context) error {
			if e := hook.checkTemporaryKey(ctx, ui, string(b.config.Comm.SSHPublicKey), false); e != nil {
				return e
			}
			t.Log("temporary key absent from user and root authorized_keys before image creation")
			return nil
		}}
		artifact, err = b.runWithDriver(ctx, ui, hook, observer)
	} else {
		artifact, err = b.Run(ctx, ui, hook)
	}
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
	comm         packersdk.Communicator
}

func (h *acceptanceHook) Run(ctx context.Context, name string, ui packersdk.Ui, comm packersdk.Communicator, data interface{}) error {
	if name != packersdk.HookProvision {
		return nil
	}
	if h.ssh {
		h.comm = comm
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

// Inspect the guest after the real cleanup step, before shutdown seals its disk.
// This scenario requires Python 3 and passwordless sudo in the test image.
func (h *acceptanceHook) checkTemporaryKey(ctx context.Context, ui packersdk.Ui, key string, present bool) error {
	fields := strings.Fields(key)
	if len(fields) < 2 || h.comm == nil {
		return fmt.Errorf("generated key and SSH communicator are required")
	}
	// Match key material, including copies whose comments were changed by the guest.
	key = fields[1]
	script := fmt.Sprintf(`import pathlib, pwd
key = %q
home = pathlib.Path(pwd.getpwnam(%q).pw_dir)
def contains(path):
    return path.exists() and key in path.read_text()
user_key = contains(home / '.ssh/authorized_keys')
root_key = contains(pathlib.Path('/root/.ssh/authorized_keys'))
assert user_key if %s else not (user_key or root_key), 'unexpected temporary key state'
`, key, h.expectedUser, map[bool]string{true: "True", false: "False"}[present])
	encoded := base64.StdEncoding.EncodeToString([]byte(script))
	cmd := &packersdk.RemoteCmd{Command: "sudo -n python3 -c \"import base64; exec(base64.b64decode('" + encoded + "'))\""}
	if present {
		cmd.Command = "sudo -n timeout 180 cloud-init status --wait && " + cmd.Command
	}
	if err := cmd.RunWithUi(ctx, h.comm, ui); err != nil {
		return err
	}
	if cmd.ExitStatus() != 0 {
		return fmt.Errorf("temporary key check (present=%t) exited %d", present, cmd.ExitStatus())
	}
	return nil
}

type acceptanceStopObserver struct {
	Driver
	ctx        context.Context
	beforeStop func(context.Context) error
}

func (d *acceptanceStopObserver) WithContext(ctx context.Context) Driver {
	return &acceptanceStopObserver{Driver: d.Driver.WithContext(ctx), ctx: ctx, beforeStop: d.beforeStop}
}

func (d *acceptanceStopObserver) StopVminstance(id string) (*view.VmInstanceInventoryView, error) {
	if err := d.beforeStop(d.ctx); err != nil {
		return nil, err
	}
	return d.Driver.StopVminstance(id)
}
