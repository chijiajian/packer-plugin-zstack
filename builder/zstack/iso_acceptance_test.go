// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zstack

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/stretchr/testify/require"
)

// The private config supplies a standard installer ISO and an OS-specific answer
// file. No cloud metadata or manual guest installation substitutes for this run.
func TestAcceptanceISO(t *testing.T) {
	if os.Getenv("ZSTACK_ACC_ISO") != "1" {
		t.Skip("set ZSTACK_ACC_ISO=1 and ZSTACK_ACC_ISO_CONFIG_FILE to run real ISO acceptance")
	}
	data, err := os.ReadFile(os.Getenv("ZSTACK_ACC_ISO_CONFIG_FILE"))
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	b := &ISOBuilder{}
	_, _, err = b.Prepare(raw)
	require.NoError(t, err)
	require.Empty(t, b.config.UserData)
	require.Empty(t, b.config.SshKey)
	require.Contains(t, []string{"ssh", "winrm"}, b.config.Comm.Type)
	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	ctx, cancel := context.WithTimeout(signalCtx, 50*time.Minute)
	defer cancel()
	ui := &packersdk.BasicUi{Reader: strings.NewReader(""), Writer: os.Stdout, ErrorWriter: os.Stderr}
	hook := &isoAcceptanceHook{config: &b.config, expectedUser: b.config.Comm.SSHUsername, expectedArch: b.config.Architecture, winrm: b.config.Comm.Type == "winrm", winrmUser: b.config.Comm.WinRMUser}
	artifact, err := b.Run(ctx, ui, hook)
	if b.config.RootVolumeUuid != "" {
		t.Logf("ISO build root volume UUID: %s", b.config.RootVolumeUuid)
	}
	if artifact != nil {
		t.Cleanup(func() { require.NoError(t, artifact.Destroy()) })
		t.Logf("ISO output image UUID: %s", artifact.Id())
	}
	require.NoError(t, err)
	require.NotNil(t, artifact)
	require.True(t, hook.called)
	require.Empty(t, artifact.State("cleanup_warnings"))
	d, err := b.config.AccessConfig.driver(ctx, b.config.APITimeout())
	require.NoError(t, err)
	image, err := d.GetImage(artifact.Id())
	require.NoError(t, err)
	require.Equal(t, "Ready", image.Status)
	require.Equal(t, "RootVolumeTemplate", image.MediaType)
	require.NotEqual(t, "iso", image.Format)
	// Boot a new VM from the resulting template with no ISO and no userdata.
	clone := b.config
	clone.ISOInstall = false
	clone.InstanceUuid = ""
	clone.RootVolumeUuid = ""
	clone.IP = ""
	clone.InstanceName += "-artifact-check"
	clone.ImageUuid = artifact.Id()
	clone.BootMode = "" // Verify the output image carries the required firmware itself.
	clone.Comm.SSHHost = ""
	clone.Comm.WinRMHost = ""
	state := new(multistep.BasicStateBag)
	state.Put("config", &clone)
	state.Put("driver", d)
	state.Put("ui", ui)
	t.Cleanup(func() { cleanupVM(state) })
	require.Equal(t, multistep.ActionContinue, (&StepCreateVMInstance{}).Run(ctx, state))
	t.Logf("artifact verification VM UUID: %s", clone.InstanceUuid)
	t.Logf("artifact verification root volume UUID: %s", clone.RootVolumeUuid)
	require.Equal(t, multistep.ActionContinue, (&StepWaitForRunning{}).Run(ctx, state))
	connection := &communicator.StepConnect{Config: &clone.Comm, Host: commHost(clone.IP), SSHConfig: clone.Comm.SSHConfigFunc()}
	require.Equal(t, multistep.ActionContinue, connection.Run(ctx, state))
	defer connection.Cleanup(state)
	require.Equal(t, multistep.ActionContinue, (&StepVerifyInstallation{}).Run(ctx, state))
	require.NoError(t, (&isoAcceptanceHook{config: &clone, expectedUser: clone.Comm.SSHUsername, expectedArch: clone.Architecture, winrm: clone.Comm.Type == "winrm", winrmUser: clone.Comm.WinRMUser}).Run(ctx, packersdk.HookProvision, ui, state.Get("communicator").(packersdk.Communicator), nil))
	cleanupVM(state)
	require.Empty(t, state.Get("cleanup_warnings"))
	if source := raw["source_image"]; source != nil {
		images, err := d.QueryImage(source.(string))
		require.NoError(t, err)
		require.Empty(t, images, "temporary installer image must be removed")
	}
	if !b.config.SkipExport {
		url, ok := artifact.State("image_url").(string)
		require.True(t, ok)
		require.NotEmpty(t, url, "ISO acceptance requires a real export URL when skip_export is false")
		require.NoError(t, acceptanceExportProbe(ctx, url, true))
		require.NoError(t, artifact.Destroy())
		require.NoError(t, acceptanceExportProbe(ctx, url, false))
		t.Log("ISO export URL readable before Destroy and returns HTTP 404/410 afterward")
	}
	t.Log("ISO install, answer-file delivery, hard-disk reboot, provisioning, output-template boot and lifecycle assertions passed")
}

type isoAcceptanceHook struct {
	config       *Config
	expectedUser string
	expectedArch string
	winrm        bool
	winrmUser    string
	called       bool
}

func (h *isoAcceptanceHook) Run(ctx context.Context, name string, ui packersdk.Ui, comm packersdk.Communicator, _ any) error {
	if name != packersdk.HookProvision {
		return nil
	}
	h.called = true
	if h.winrm {
		return (&acceptanceHook{winrm: true, expectedUser: h.winrmUser}).Run(ctx, name, ui, comm, nil)
	}
	if h.expectedArch == "" && h.config != nil {
		h.expectedArch = h.config.Architecture
	}
	if h.expectedArch == "" {
		h.expectedArch = "x86_64"
	}
	path := "/tmp/packer-iso-" + newResource().UUID
	if err := comm.Upload(path, strings.NewReader("iso-transfer-ok"), nil); err != nil {
		return err
	}
	var output strings.Builder
	cmd := &packersdk.RemoteCmd{Command: "id -un && uname -m && test -f /etc/packer-installed && test \"$(cat " + path + ")\" = iso-transfer-ok && rm -f " + path, Stdout: &output}
	if err := cmd.RunWithUi(ctx, comm, ui); err != nil {
		return err
	}
	if cmd.ExitStatus() != 0 {
		return fmt.Errorf("ISO guest command failed: %d", cmd.ExitStatus())
	}
	if strings.TrimSpace(output.String()) != h.expectedUser+"\n"+h.expectedArch {
		return fmt.Errorf("unexpected ISO guest identity/architecture")
	}
	return nil
}
