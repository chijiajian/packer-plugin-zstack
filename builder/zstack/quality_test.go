package zstack

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/stretchr/testify/require"

	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

func TestFailedBuildNeverReturnsSourceArtifact(t *testing.T) {
	for _, step := range []multistep.Step{&StepPreValidate{}, &StepInstanceOfferingValidate{}, &StepSourceImageValidate{}} {
		cfg := &Config{ImageConfig: ImageConfig{ImageUuid: "user-source"}, InstanceConfig: InstanceConfig{InstanceOfferingName: "missing"}}
		if _, ok := step.(*StepSourceImageValidate); ok {
			cfg.ImageUuid = ""
		}
		d := &MockDriver{}
		state := testStateBag(cfg, d)
		(&multistep.BasicRunner{Steps: []multistep.Step{step}}).Run(context.Background(), state)
		b := Builder{config: *cfg}
		a, err := b.collectArtifact(state, d)
		require.Error(t, err)
		require.Nil(t, a)
		require.False(t, d.DeleteImageCalled)
	}
	for _, key := range []string{multistep.StateHalted, multistep.StateCancelled, "no_output"} {
		cfg := &Config{ImageConfig: ImageConfig{ImageUuid: "user-source"}}
		d := &MockDriver{}
		state := testStateBag(cfg, d)
		state.Put(key, true)
		b := Builder{config: *cfg}
		a, e := b.collectArtifact(state, d)
		require.Error(t, e)
		require.Nil(t, a)
	}
	a := &Artifact{config: Config{ImageConfig: ImageConfig{ImageUuid: "user-source"}}, driver: &MockDriver{}}
	require.NoError(t, a.Destroy())
	require.False(t, a.driver.(*MockDriver).DeleteImageCalled)
}

func TestCancellationCleansOwnedVM(t *testing.T) {
	cfg := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "owned"}}
	d := &MockDriver{}
	state := testStateBag(cfg, d)
	resources(state).VM = ownedResource{UUID: "owned", Created: true}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Equal(t, multistep.ActionHalt, (&StepWaitForRunning{}).Run(ctx, state))
	(&StepCreateVMInstance{}).Cleanup(state)
	require.True(t, d.DestroyVmInstanceCalled)
	require.True(t, d.DeleteVmInstanceCalled)
	// Config input alone never grants ownership.
	d = &MockDriver{}
	state = testStateBag(&Config{InstanceConfig: InstanceConfig{InstanceUuid: "external"}}, d)
	state.Put("error", errors.New("failed"))
	(&StepCreateVMInstance{}).Cleanup(state)
	require.False(t, d.DestroyVmInstanceCalled)
}

func TestImageOwnershipAndRetention(t *testing.T) {
	for _, keep := range []bool{false, true} {
		cfg := &Config{KeepImageOnError: keep, KeepSourceImage: keep}
		d := &MockDriver{}
		state := testStateBag(cfg, d)
		r := resources(state)
		r.ImportedImage = ownedResource{UUID: "temporary", Created: true}
		(&StepAddImage{}).Cleanup(state)
		require.Equal(t, !keep, d.DeleteImageCalled)
		d.DeleteImageCalled = false
		r.Image = ownedResource{UUID: "output", Created: true}
		state.Put(multistep.StateCancelled, true)
		(&StepCreateImage{}).Cleanup(state)
		require.Equal(t, !keep, d.DeleteImageCalled)
	}
	d := &MockDriver{ExpungeImageErr: errors.New("busy")}
	a := &Artifact{owned: true, driver: d, config: Config{ImageConfig: ImageConfig{ImageUuid: "output"}}}
	require.Error(t, a.Destroy())
	d.ExpungeImageErr = nil
	d.DeleteImageCalled = false
	require.NoError(t, a.Destroy())
	require.False(t, d.DeleteImageCalled, "retry expunge without deleting twice")
}

func TestImportArchitectureAndPlacement(t *testing.T) {
	cfg := &Config{ImageConfig: ImageConfig{Architecture: "aarch64", SourceImage: "arm", SourceImageUrl: "https://example.invalid/arm"}}
	d := &MockDriver{AddImageResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "imported"}}}
	state := testStateBag(cfg, d)
	require.Equal(t, multistep.ActionContinue, (&StepAddImage{}).Run(context.Background(), state))
	require.Equal(t, "aarch64", *d.AddImageParam.Params.Architecture)
	require.NotEmpty(t, *d.AddImageParam.Params.ResourceUuid)
	cfg.Comm.Type = "none"
	cfg.L3NetworkUuids = []string{"net1", "net2"}
	cfg.DefaultL3NetworkUuid = "net2"
	cfg.ZoneUuid = "zone"
	cfg.ClusterUuid = "cluster"
	cfg.HostUuid = "host"
	cfg.RootDiskOfferingUuid = "root-offer"
	cfg.DataDiskOfferingUuids = []string{"data-offer"}
	d.CreateVmInstanceResult = &view.VmInstanceInventoryView{BaseInfoView: view.BaseInfoView{UUID: "vm"}, VmNics: []view.VmNicInventoryView{{Ip: "10.0.0.1", L3NetworkUuid: "net1"}, {Ip: "10.0.1.1", L3NetworkUuid: "net2"}}}
	require.Equal(t, multistep.ActionContinue, (&StepCreateVMInstance{}).Run(context.Background(), state))
	p := d.CreateVmInstanceParam.Params
	require.Equal(t, cfg.L3NetworkUuids, p.L3NetworkUuids)
	require.Equal(t, "cluster", *p.ClusterUuid)
	require.Equal(t, "host", *p.HostUuid)
	require.Equal(t, "root-offer", *p.RootDiskOfferingUuid)
	require.Equal(t, "zone", *p.ZoneUuid)
	require.Equal(t, []string{"data-offer"}, p.DataDiskOfferingUuids)
	require.Equal(t, "10.0.1.1", cfg.IP)
}

func TestExportErrorsAndMetadata(t *testing.T) {
	for _, msg := range []string{"APIExportImageFromBackupStorageMsg: permission denied", "APIExportImageFromBackupStorageMsg: disk full", "unsupported credential"} {
		require.False(t, isUnsupportedExportError(errors.New(msg)))
	}
	cfg := &Config{ImageConfig: ImageConfig{ImageUuid: "output"}, BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs"}}
	d := &MockDriver{ExportImageResult: &view.ExportImageFromBackupStorageEventView{ImageUrl: "https://example.invalid/output"}}
	state := testStateBag(cfg, d)
	require.Equal(t, multistep.ActionContinue, (&StepExportImage{}).Run(context.Background(), state))
	r := resources(state)
	r.Image = ownedResource{UUID: "output", Created: true}
	r.ImageReady = true
	b := Builder{config: *cfg}
	a, err := b.collectArtifact(state, d)
	require.NoError(t, err)
	require.Equal(t, cfg.ImageUrl, a.State("image_url"))
	require.Empty(t, a.Files())
}

func TestStrictConfigAndFullHCLSchema(t *testing.T) {
	cfg := &Config{}
	require.NotEmpty(t, cfg.validateBuild())
	cfg.SourceVolumeSnapshotUuid = "snapshot"
	cfg.SourceImage = "source"
	require.NotEmpty(t, cfg.validateBuild())
	cfg = &Config{ImageConfig: ImageConfig{ImageName: "target", SourceVolumeSnapshotUuid: "snapshot"}}
	require.Empty(t, cfg.validateBuild())
	spec := (&Builder{}).ConfigSpec()
	for _, key := range []string{"ssh_bastion_host", "winrm_host", "winrm_port", "winrm_use_ssl", "ssh_clear_authorized_keys", "api_timeout", "cleanup_timeout", "network_uuids"} {
		require.Contains(t, spec, key)
	}
	for _, key := range []string{"instance_uuid", "root_volume_uuid", "image_url"} {
		require.NotContains(t, spec, key)
	}
	state := testStateBag(&Config{}, &MockDriver{QueryImageResult: []view.ImageInventoryView{{}, {}}})
	_, err := validateImage(context.Background(), state)
	require.ErrorContains(t, err, "multiple")
}

func httpDriver(t *testing.T, handler http.HandlerFunc) *ZStackDriver {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	u, err := url.Parse(server.URL)
	require.NoError(t, err)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)
	cli, err := newAPIClient(&AccessConfig{Host: u.Hostname(), Port: port, AccessKeyId: "fake-ak", AccessKeySecret: "fake-sk"})
	require.NoError(t, err)
	return &ZStackDriver{client: cli, timeout: time.Second}
}
func TestDriverHTTPContracts(t *testing.T) {
	t.Run("stop inventory envelope", func(t *testing.T) {
		d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "PUT", r.Method)
			require.Equal(t, "/zstack/v1/vm-instances/vm/actions", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"inventory":{"uuid":"vm","state":"Stopped"}}`)
		})
		vm, err := d.StopVminstance("vm")
		require.NoError(t, err)
		require.Equal(t, "Stopped", vm.State)
	})
	t.Run("request deadline", func(t *testing.T) {
		d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
			case <-time.After(time.Second):
			}
		})
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		start := time.Now()
		_, err := d.WithContext(ctx).GetImage("image")
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Less(t, time.Since(start), 500*time.Millisecond)
	})
	t.Run("empty inventory", func(t *testing.T) {
		d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"inventory":{}}`)
		})
		_, err := d.StopVminstance("vm")
		require.ErrorContains(t, err, "empty inventory")
	})
}

func TestRedactionPreservesCause(t *testing.T) {
	cause := fmt.Errorf("password=private-value userdata=cloud-secret URL=https://user:pass@host/image?token=secret: %w", context.Canceled)
	err := redactError(cause, []string{"private-value", "cloud-secret"})
	require.ErrorIs(t, err, context.Canceled)
	for _, secret := range []string{"private-value", "cloud-secret", "user:pass", "token=secret"} {
		require.NotContains(t, err.Error(), secret)
	}
}

func TestBuilderVMAndSnapshotFlows(t *testing.T) {
	for _, snapshot := range []bool{false, true} {
		t.Run(fmt.Sprintf("snapshot=%t", snapshot), func(t *testing.T) {
			cfg := Config{ImageConfig: ImageConfig{ImageName: "output", ImageUuid: "source"}, BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs"}, NetworkConfig: NetworkConfig{L3NetworkUuid: "net"}, pollInterval: time.Millisecond}
			cfg.Comm.Type = "none"
			if snapshot {
				cfg.ImageUuid = ""
				cfg.SourceVolumeSnapshotUuid = "snapshot"
			}
			d := &MockDriver{
				CreateVmInstanceResult:        &view.VmInstanceInventoryView{BaseInfoView: view.BaseInfoView{UUID: "vm"}, RootVolumeUuid: "root"},
				GetVmInstanceResult:           &view.VmInstanceInventoryView{State: "Running"},
				StopVminstanceResult:          &view.VmInstanceInventoryView{State: "Stopped"},
				CreateImageResult:             &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "output"}},
				CreateImageFromSnapshotResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "output"}},
				GetImageResult:                &view.ImageInventoryView{Status: "Ready"},
				GetVolumeSnapshotResult:       &view.VolumeSnapshotInventoryView{Status: "Ready", State: "Enabled", VolumeType: "Root"},
				ExportImageResult:             &view.ExportImageFromBackupStorageEventView{ImageUrl: "https://example.invalid/output"},
			}
			b := Builder{config: cfg}
			a, err := b.runWithDriver(context.Background(), &packersdk.MockUi{}, &packersdk.MockHook{}, d)
			require.NoError(t, err)
			require.Equal(t, "output", a.Id())
			require.Equal(t, !snapshot, d.DestroyVmInstanceCalled)
			require.False(t, d.DeleteImageCalled)
			require.NoError(t, a.Destroy())
			require.Equal(t, "output", d.DeleteImageUuid)
		})
	}
}

func TestSnapshotValidation(t *testing.T) {
	for _, snapshot := range []*view.VolumeSnapshotInventoryView{nil, {Status: "Ready", State: "Enabled", VolumeType: "Data"}, {Status: "Creating", State: "Enabled", VolumeType: "Root"}} {
		cfg := &Config{ImageConfig: ImageConfig{SourceVolumeSnapshotUuid: "snapshot"}, BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs"}}
		d := &MockDriver{GetVolumeSnapshotResult: snapshot}
		state := testStateBag(cfg, d)
		require.Equal(t, multistep.ActionHalt, (&StepCreateImageFromSnapshot{}).Run(context.Background(), state))
		require.False(t, d.CreateImageFromSnapshotCalled)
	}
}

func TestAgentDoesNotGenerateKey(t *testing.T) {
	cfg := &Config{}
	cfg.Comm.SSHAgentAuth = true
	cfg.SshKey = "existing-public-key"
	require.Equal(t, multistep.ActionContinue, (&StepCreateSSHKey{}).Run(context.Background(), testStateBag(cfg, &MockDriver{})))
	require.Empty(t, cfg.Comm.SSHPrivateKey)
	require.Equal(t, "existing-public-key", cfg.SshKey)
}

func TestTemporaryKeyCleanupIdentity(t *testing.T) {
	cfg := &Config{}
	require.Equal(t, multistep.ActionContinue, (&StepCreateSSHKey{}).Run(context.Background(), testStateBag(cfg, &MockDriver{})))
	require.True(t, strings.HasSuffix(cfg.SshKey, " "+cfg.Comm.SSHTemporaryKeyPairName))
	require.True(t, strings.HasPrefix(cfg.Comm.SSHTemporaryKeyPairName, "packer-"))
}

func TestDataVolumeCleanupRetriesAfterVMExpunge(t *testing.T) {
	d := &MockDriver{ExpungeDataVolumeErr: errors.New("busy")}
	state := testStateBag(&Config{}, d)
	r := resources(state)
	r.VM = ownedResource{UUID: "vm", Created: true}
	r.DataVolumes = []ownedResource{{UUID: "data", Created: true}}
	cleanupVM(state)
	require.True(t, r.VM.Removed)
	require.False(t, r.DataVolumes[0].Removed)
	d.ExpungeDataVolumeErr = nil
	d.DeleteVmInstanceCalled = false
	cleanupVM(state)
	require.False(t, d.DeleteVmInstanceCalled)
	require.True(t, r.DataVolumes[0].Removed)
	require.Equal(t, []string{"data"}, d.DeleteDataVolumeCalls)
	require.Equal(t, []string{"data", "data"}, d.ExpungeDataVolumeCalls)
}

func TestAsyncHTTPPolling(t *testing.T) {
	t.Run("poll configured origin with fresh signature", func(t *testing.T) {
		calls := 0
		d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			require.NotEmpty(t, r.Header.Get("Authorization"))
			if calls == 1 {
				require.Equal(t, "PUT", r.Method)
				w.WriteHeader(http.StatusAccepted)
				fmt.Fprint(w, `{"location":"http://untrusted.invalid/zstack/v1/api-jobs/job"}`)
			} else {
				require.Equal(t, "GET", r.Method)
				require.Equal(t, "/zstack/v1/api-jobs/job", r.URL.Path)
				if calls == 2 {
					w.WriteHeader(http.StatusAccepted)
					return
				}
				fmt.Fprint(w, `{"inventory":{"uuid":"vm","state":"Stopped"}}`)
			}
		})
		d.client.pollInterval = time.Millisecond
		vm, err := d.StopVminstance("vm")
		require.NoError(t, err)
		require.Equal(t, "Stopped", vm.State)
		require.Equal(t, 3, calls)
	})
	t.Run("cancel waiting job", func(t *testing.T) {
		d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			fmt.Fprint(w, `{"location":"/zstack/v1/api-jobs/job"}`)
		})
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		_, err := d.WithContext(ctx).StopVminstance("vm")
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})
	t.Run("redirect rejected", func(t *testing.T) {
		d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", "http://untrusted.invalid/")
			w.WriteHeader(http.StatusTemporaryRedirect)
		})
		_, err := d.GetImage("image")
		require.ErrorContains(t, err, "HTTP 307")
	})
}

func TestArtifactExportCleanupOrderAndRetry(t *testing.T) {
	d := &MockDriver{DeleteExportedImageErr: errors.New("busy")}
	a := &Artifact{owned: true, driver: d, config: Config{ImageConfig: ImageConfig{ImageUuid: "output"}, ExportImageResult: ExportImageResult{ImageUrl: "https://host/file?secret=token"}, BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs"}}}
	require.NotContains(t, a.String(), "secret")
	require.Error(t, a.Destroy())
	require.False(t, d.DeleteImageCalled)
	d.DeleteExportedImageErr = nil
	d.ExpungeImageErr = errors.New("busy")
	require.Error(t, a.Destroy())
	d.ExpungeImageErr = nil
	require.NoError(t, a.Destroy())
	require.Equal(t, 2, d.DeleteExportedImageCalls)
}

func TestAccessKeySigningAndSessionAuthentication(t *testing.T) {
	t.Run("access key signature", func(t *testing.T) {
		d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
			hash := hmac.New(sha1.New, []byte("fake-sk"))
			fmt.Fprintf(hash, "%s\n%s\n%s", r.Method, r.Header.Get("Date"), "/v1/images/image")
			require.Equal(t, "ZStack fake-ak:"+base64.StdEncoding.EncodeToString(hash.Sum(nil)), r.Header.Get("Authorization"))
			fmt.Fprint(w, `{"inventories":[{"uuid":"image","status":"Ready"}]}`)
		})
		_, err := d.GetImage("image")
		require.NoError(t, err)
	})
	t.Run("hashed account login and OAuth", func(t *testing.T) {
		d := httpDriver(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/zstack/v1/accounts/login" {
				require.Equal(t, "PUT", r.Method)
				require.Empty(t, r.Header.Get("Authorization"))
				var body map[string]map[string]string
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, fmt.Sprintf("%x", sha512.Sum512([]byte("fake-password"))), body["logInByAccount"]["password"])
				fmt.Fprint(w, `{"inventory":{"uuid":"fake-session"}}`)
			} else {
				require.Equal(t, "OAuth fake-session", r.Header.Get("Authorization"))
				fmt.Fprint(w, `{"inventories":[{"uuid":"image"}]}`)
			}
		})
		d.client.accessKey = ""
		d.client.account = "fake-account"
		d.client.password = "fake-password"
		_, err := d.client.Login(context.Background())
		require.NoError(t, err)
		_, err = d.GetImage("image")
		require.NoError(t, err)
	})
}
