// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zstack

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/require"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

func isoRaw() map[string]any {
	return map[string]any{
		"zstack_host": "127.0.0.1", "access_key_id": "test-id", "access_key_secret": "test-secret",
		"image_uuid": "installer", "image_name": "output", "instance_name": "iso-vm", "network_uuid": "net", "backup_storage_uuid": "bs",
		"cpu_num": 2, "memory_size": 2048, "root_disk_offering_uuid": "disk", "communicator": "ssh", "ssh_username": "packer", "ssh_password": "test-only",
		"installation_check_command": "test -f /etc/installed-marker", "boot_command": []string{"auto url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg<enter>"},
	}
}
func TestISOBuilderValidation(t *testing.T) {
	b := &ISOBuilder{}
	_, _, err := b.Prepare(isoRaw())
	require.NoError(t, err)
	require.True(t, b.config.ISOInstall)
	require.Equal(t, 30*time.Minute, b.config.Comm.SSHTimeout)
	require.Contains(t, b.config.FlatBootCommand(), "{{ .HTTPIP }}")
	for _, change := range []map[string]any{
		{"boot_mode": "BIOS"}, {"http_ip": "0.0.0.0"}, {"userdata": "#cloud-config"}, {"sshkey": "public-key"}, {"communicator": "none"}, {"root_disk_offering_uuid": ""},
		{"installation_check_command": ""}, {"ssh_password": ""}, {"http_directory": "/missing", "http_content": map[string]string{"/file": "value"}},
	} {
		raw := isoRaw()
		for k, v := range change {
			raw[k] = v
		}
		_, _, err := (&ISOBuilder{}).Prepare(raw)
		require.Error(t, err)
	}
	raw := isoRaw()
	delete(raw, "image_uuid")
	raw["source_image"] = "installer"
	raw["source_image_url"] = "http://example.test/install.iso"
	b = &ISOBuilder{}
	_, _, err = b.Prepare(raw)
	require.NoError(t, err)
	require.Equal(t, "iso", b.config.Format)
}
func TestISOImportsMediaTypeAndNoMetadata(t *testing.T) {
	c := &Config{ISOConfig: ISOConfig{ISOInstall: true, BootMode: "UEFI"}, ImageConfig: ImageConfig{ImageUuid: "boot", SourceImage: "installer", SourceImageUrl: "http://example.test/install.iso", Format: "iso"}, InstanceConfig: InstanceConfig{RootDiskOfferingUuid: "disk", SshKey: "must-not-inject", UserData: "must-not-inject"}}
	c.Comm.SSHHost = "127.0.0.1"
	d := &MockDriver{AddImageResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "boot"}}, CreateVmInstanceResult: &view.VmInstanceInventoryView{BaseInfoView: view.BaseInfoView{UUID: "vm"}}}
	state := testStateBag(c, d)
	state.Put("answer_iso_uuid", "answer")
	require.Equal(t, multistep.ActionContinue, (&StepAddImage{}).Run(context.Background(), state))
	require.Equal(t, "ISO", *d.AddImageParam.Params.MediaType)
	require.Contains(t, d.AddImageParam.SystemTags, "bootMode::UEFI")
	require.Equal(t, multistep.ActionContinue, (&StepCreateVMInstance{}).Run(context.Background(), state))
	require.Contains(t, d.CreateVmInstanceParam.SystemTags, "cdroms::boot::answer::None")
	require.Contains(t, d.CreateVmInstanceParam.SystemTags, "bootMode::UEFI")
	require.Equal(t, "CreateStopped", *d.CreateVmInstanceParam.Params.Strategy)
	for _, tag := range d.CreateVmInstanceParam.SystemTags {
		require.False(t, strings.HasPrefix(tag, "userdata::") || strings.HasPrefix(tag, "sshkey::"))
	}
}

type isoMock struct {
	*MockDriver
	calls []string
	fail  string
}

func (d *isoMock) WithContext(context.Context) Driver { return d }
func (d *isoMock) SetBootOrder(_ string, o []string) error {
	d.calls = append(d.calls, "order:"+strings.Join(o, ","))
	if d.fail == "order" {
		return errors.New("order failed")
	}
	return nil
}
func (d *isoMock) StartVM(string) (*view.VmInstanceInventoryView, error) {
	d.calls = append(d.calls, "start")
	if d.fail == "start" {
		return nil, errors.New("start failed")
	}
	return &view.VmInstanceInventoryView{BaseInfoView: view.BaseInfoView{UUID: "vm"}, State: "Running"}, nil
}
func (d *isoMock) DetachISO(_ string, id string) error {
	d.calls = append(d.calls, "detach:"+id)
	if d.fail == "detach" {
		return errors.New("detach failed")
	}
	return nil
}
func (d *isoMock) Console(string) (*consoleAddress, error) { return nil, errors.New("not expected") }
func TestISOHardDiskBootOrderAndFailure(t *testing.T) {
	for _, failure := range []string{"", "detach", "order", "start"} {
		c := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm"}}
		d := &isoMock{MockDriver: &MockDriver{StopVminstanceResult: &view.VmInstanceInventoryView{State: "Stopped"}}, fail: failure}
		state := testStateBag(c, d)
		state.Put("boot_iso_uuid", "boot")
		state.Put("answer_iso_uuid", "answer")
		action := (&StepISOHardDiskBoot{}).Run(context.Background(), state)
		if failure == "" {
			require.Equal(t, multistep.ActionContinue, action)
			require.Equal(t, []string{"detach:boot", "detach:answer", "order:HardDisk", "start"}, d.calls)
		} else {
			require.Equal(t, multistep.ActionHalt, action)
		}
	}
}
func TestAnswerCDPublicationAndCleanup(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "answer*.iso")
	require.NoError(t, err)
	_, err = f.WriteString("test-cd-bytes")
	require.NoError(t, err)
	f.Close()
	c := &Config{ISOConfig: ISOConfig{ISOInstall: true, HTTPIP: "127.0.0.1"}}
	c.HTTPAddress = "127.0.0.1"
	d := &MockDriver{AddImageResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "answer"}}, GetImageResult: &view.ImageInventoryView{Status: "Ready"}}
	state := testStateBag(c, d)
	state.Put("cd_path", f.Name())
	step := &StepAnswerISO{}
	require.Equal(t, multistep.ActionContinue, step.Run(context.Background(), state))
	response, err := http.Get(d.AddImageParam.Params.Url)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.GreaterOrEqual(t, response.ContentLength, int64(1<<20))
	response.Body.Close()
	response, err = http.Get("http://" + step.listener.Addr().String() + "/.env")
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
	response.Body.Close()
	step.Cleanup(state)
	require.True(t, d.DeleteImageCalled)
	require.True(t, d.ExpungeImageCalled)
}
func TestISOBootCommandCancellation(t *testing.T) {
	c := &Config{ISOConfig: ISOConfig{BootConfig: bootcommand.BootConfig{BootWait: time.Hour, BootCommand: []string{"<enter>"}}}}
	state := testStateBag(c, &MockDriver{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Equal(t, multistep.ActionHalt, (&StepISOBootCommand{}).Run(ctx, state))
	require.ErrorIs(t, state.Get("error").(error), context.Canceled)
}
func TestISODriverWirePaths(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.RequestURI())
		if strings.HasSuffix(r.URL.Path, "console-addresses") {
			w.Write([]byte(`{"hostIp":"127.0.0.1","port":5900,"protocol":"vnc"}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "console-passwords") {
			w.Write([]byte(`{"password":"test"}`))
			return
		}
		w.Write([]byte(`{"inventory":{"uuid":"vm","state":"Running"}}`))
	}))
	defer server.Close()
	client, err := newAPIClient(&AccessConfig{Host: server.URL})
	require.NoError(t, err)
	d := &ZStackDriver{client: client}
	require.NoError(t, d.SetBootOrder("vm", []string{"HardDisk"}))
	_, err = d.StartVM("vm")
	require.NoError(t, err)
	require.NoError(t, d.DetachISO("vm", "answer"))
	console, err := d.Console("vm")
	require.NoError(t, err)
	require.Equal(t, "test", console.Password)
	require.Contains(t, methods, "DELETE /zstack/v1/vm-instances/vm/iso?isoUuid=answer")
}

func TestExistingISOSourceValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		image *view.ImageInventoryView
		arch  string
		ok    bool
	}{
		{"ready", &view.ImageInventoryView{MediaType: "ISO", Status: "Ready", State: "Enabled", Architecture: "aarch64"}, "", true},
		{"wrong-media", &view.ImageInventoryView{MediaType: "RootVolumeTemplate", Status: "Ready", State: "Enabled"}, "", false},
		{"disabled", &view.ImageInventoryView{MediaType: "ISO", Status: "Ready", State: "Disabled"}, "", false},
		{"wrong-architecture", &view.ImageInventoryView{MediaType: "ISO", Status: "Ready", State: "Enabled", Architecture: "aarch64"}, "x86_64", false},
		{"empty", nil, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Config{ISOConfig: ISOConfig{ISOInstall: true}, ImageConfig: ImageConfig{ImageUuid: "existing", Architecture: tc.arch}}
			d := &MockDriver{GetImageResult: tc.image}
			state := testStateBag(c, d)
			action := (&StepSourceImageValidate{}).Run(context.Background(), state)
			if tc.ok {
				require.Equal(t, multistep.ActionContinue, action)
				require.Equal(t, "aarch64", c.Architecture)
			} else {
				require.Equal(t, multistep.ActionHalt, action)
			}
			(&StepSourceImageValidate{}).Cleanup(state)
			require.False(t, d.DeleteImageCalled)
		})
	}
}
