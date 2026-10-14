package zstack

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
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
	require.NotEmpty(t, os.Getenv("ZSTACK_ACC_BACKUP_STORAGE_UUID"), "ZSTACK_ACC_BACKUP_STORAGE_UUID")
	networks := strings.FieldsFunc(os.Getenv("ZSTACK_ACC_NETWORK_UUIDS"), func(r rune) bool { return r == ',' || r == ' ' })
	require.True(t, (len(networks) > 0) != (os.Getenv("ZSTACK_ACC_NETWORK_UUID") != ""), "set either NETWORK_UUID or NETWORK_UUIDS")
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
	if len(networks) > 0 {
		delete(raw, "network_uuid")
		raw["network_uuids"] = networks
	}
	for env, field := range map[string]string{
		"ZSTACK_ACC_DEFAULT_NETWORK_UUID": "default_network_uuid",
		"ZSTACK_ACC_ZONE_UUID":            "zone_uuid", "ZSTACK_ACC_CLUSTER_UUID": "cluster_uuid", "ZSTACK_ACC_HOST_UUID": "host_uuid",
	} {
		if value := os.Getenv(env); value != "" {
			raw[field] = value
		}
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
	if len(networks) > 0 || os.Getenv("ZSTACK_ACC_ZONE_UUID") != "" || os.Getenv("ZSTACK_ACC_CLUSTER_UUID") != "" || os.Getenv("ZSTACK_ACC_HOST_UUID") != "" {
		hook.verify = func(ctx context.Context) error {
			d, e := b.config.AccessConfig.driver(ctx, time.Minute)
			if e != nil {
				return e
			}
			vm, e := d.GetVmInstance(b.config.InstanceUuid)
			if e != nil {
				return e
			}
			if len(networks) > 0 {
				actual := map[string]string{}
				for _, nic := range vm.VmNics {
					actual[nic.L3NetworkUuid] = nic.Ip
				}
				if len(vm.VmNics) != len(networks) {
					return fmt.Errorf("unexpected NIC count: %d", len(vm.VmNics))
				}
				for _, n := range networks {
					if actual[n] == "" {
						return fmt.Errorf("missing NIC/IP for network %s", n)
					}
				}
				if expected := os.Getenv("ZSTACK_ACC_DEFAULT_NETWORK_UUID"); expected != "" {
					if vm.DefaultL3NetworkUuid != expected || b.config.IP != actual[expected] {
						return fmt.Errorf("default network or communicator IP mismatch")
					}
				}
			}
			for env, actual := range map[string]string{"ZSTACK_ACC_ZONE_UUID": vm.ZoneUuid, "ZSTACK_ACC_CLUSTER_UUID": vm.ClusterUuid, "ZSTACK_ACC_HOST_UUID": vm.HostUuid} {
				if expected := os.Getenv(env); expected != "" && expected != actual {
					return fmt.Errorf("placement mismatch for %s", env)
				}
			}
			t.Logf("verified %d NICs, default network %s, host %s", len(vm.VmNics), vm.DefaultL3NetworkUuid, vm.HostUuid)
			return nil
		}
	}
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
	if os.Getenv("ZSTACK_ACC_CHECK_EXPORT") == "1" {
		url, ok := artifact.State("image_url").(string)
		require.True(t, ok)
		require.NotEmpty(t, url, "export must produce a download URL")
		// Read only a few bytes; do not download a multi-GB image or log its URL.
		require.NoError(t, acceptanceExportProbe(ctx, url, true))
		require.NoError(t, artifact.Destroy())
		require.NoError(t, acceptanceExportProbe(ctx, url, false))
		t.Log("export URL readable before Destroy and absent afterward")
	}
}

func acceptanceExportProbe(ctx context.Context, url string, exists bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("invalid export URL")
	}
	req.Header.Set("Range", "bytes=0-3")
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("export URL request failed (exists=%t)", exists)
	}
	defer resp.Body.Close()
	if !exists {
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusGone {
			return fmt.Errorf("deleted export returned HTTP %d", resp.StatusCode)
		}
		return nil
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("export returned HTTP %d", resp.StatusCode)
	}
	var header [4]byte
	if _, err := io.ReadFull(resp.Body, header[:]); err != nil {
		return fmt.Errorf("export could not supply four bytes")
	}
	return nil
}

type acceptanceHook struct {
	ssh          bool
	expectedUser string
	after        func(context.Context) error
	verify       func(context.Context) error
	comm         packersdk.Communicator
}

func (h *acceptanceHook) Run(ctx context.Context, name string, ui packersdk.Ui, comm packersdk.Communicator, data interface{}) error {
	if name != packersdk.HookProvision {
		return nil
	}
	if h.verify != nil {
		if err := h.verify(ctx); err != nil {
			return err
		}
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
