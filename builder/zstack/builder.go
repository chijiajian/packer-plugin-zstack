// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

const BuilderId = "packer.zstack"

type Builder struct {
	config          Config
	snapshotBuilder bool
	runner          multistep.Runner
	ui              packersdk.Ui
}

func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(raws ...any) ([]string, []string, error) {

	errs := b.config.Prepare(raws...)

	if b.config.Comm.Type == "" {
		if b.config.SourceVolumeSnapshotUuid != "" {
			b.config.Comm.Type = "none"
		} else {
			b.config.Comm.Type = "ssh"
		}
	}

	//var errs *packer.MultiError
	if es := b.config.Comm.Prepare(&b.config.ctx); len(es) > 0 {
		errs = packersdk.MultiErrorAppend(errs, es...)
	}

	if errs != nil {
		return nil, nil, errs
	}
	if b.config.SourceVolumeSnapshotUuid != "" && !b.snapshotBuilder {
		return nil, []string{"Using source_volume_snapshot_uuid with zstack is deprecated; use zstack-snapshot. Snapshot builds do not execute provisioners."}, nil
	}
	return nil, nil, nil
}

func (b *Builder) Run(ctx context.Context, ui packersdk.Ui, hook packersdk.Hook) (packersdk.Artifact, error) {
	b.ui = ui
	log.Printf("[DEBUG] Starting build with %s", b.config.RedactedSummary())

	driver, err := b.config.AccessConfig.driver(ctx, b.config.APITimeout())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ZStack Driver: %v", err)
	}

	driver.secrets = append(driver.secrets, b.config.SourceImageUrl, b.config.UserData, encodeUserData(b.config.UserData), b.config.Comm.SSHPassword, b.config.Comm.WinRMPassword)
	for _, content := range b.config.HTTPContent {
		driver.secrets = append(driver.secrets, content)
	}
	for _, content := range b.config.CDContent {
		driver.secrets = append(driver.secrets, content)
	}
	return b.runWithDriver(ctx, ui, hook, driver)
}

func (b *Builder) runWithDriver(ctx context.Context, ui packersdk.Ui, hook packersdk.Hook, driver Driver) (packersdk.Artifact, error) {
	state := new(multistep.BasicStateBag)
	state.Put("ui", ui)
	state.Put("driver", driver)
	state.Put("config", &b.config)
	state.Put("hook", hook)
	state.Put("communicator_config", &b.config.Comm)

	var steps []multistep.Step

	if b.config.SourceVolumeSnapshotUuid != "" {
		log.Printf("[INFO] Using snapshot-only build path (source_volume_snapshot_uuid=%s)", b.config.SourceVolumeSnapshotUuid)
		steps = []multistep.Step{
			&StepPreValidate{},
			&StepCreateImageFromSnapshot{},
			&StepWaitForImageReady{},
			&StepExportImage{},
		}

		log.Printf("[DEBUG] Build steps prepared with %s", b.config.RedactedSummary())

		b.runner = commonsteps.NewRunner(steps, b.config.PackerConfig, ui)
		b.runner.Run(ctx, state)

		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return b.collectArtifact(state, driver)
	}

	baseSteps := []multistep.Step{
		&StepPreValidate{},
	}

	if b.config.SourceImageUrl != "" {
		imageSteps := []multistep.Step{
			&StepAddImage{},
			&StepWaitForImageReady{},
		}
		baseSteps = append(baseSteps, imageSteps...)
	} else {
		imageSteps := []multistep.Step{
			&StepSourceImageValidate{},
		}
		baseSteps = append(baseSteps, imageSteps...)
	}

	if b.config.InstanceConfig.InstanceOfferingName != "" || b.config.InstanceConfig.InstanceOfferingUuid != "" {
		log.Printf("[DEBUG] InstanceOffering validate...")
		instanceOfferingSteps := []multistep.Step{
			&StepInstanceOfferingValidate{},
		}
		baseSteps = append(baseSteps, instanceOfferingSteps...)
	}
	remainingSteps := []multistep.Step{}

	// AC-005-06: Only generate SSH key for SSH communicator
	if b.config.Comm.Type == "ssh" && !b.config.ISOInstall {
		remainingSteps = append(remainingSteps, &StepCreateSSHKey{
			Password:     b.config.Comm.SSHPassword,
			Debug:        b.config.PackerDebug,
			DebugKeyPath: fmt.Sprintf("zstack_%s.pem", b.config.PackerBuildName),
		})
	}

	newConnect := func() *communicator.StepConnect {
		return &communicator.StepConnect{Config: &b.config.Comm, Host: commHost(b.communicatorHost()), SSHConfig: b.config.Comm.SSHConfigFunc()}
	}
	if b.config.ISOInstall {
		remainingSteps = append(remainingSteps,
			commonsteps.HTTPServerFromHTTPConfig(&b.config.HTTPConfig),
			&commonsteps.StepCreateCD{Files: b.config.CDFiles, Content: b.config.CDContent, Label: b.config.CDLabel},
			&StepAnswerISO{}, &StepCreateVMInstance{}, &StepISOBoot{}, &StepWaitForRunning{}, &StepISOBootCommand{},
		)
		firstConnect := newConnect()
		remainingSteps = append(remainingSteps, firstConnect, &StepVerifyInstallation{}, &StepISOHardDiskBoot{Connect: firstConnect}, &StepWaitForRunning{}, newConnect(), &StepVerifyInstallation{}, &StepAttachGuestTools{})
	} else {
		remainingSteps = append(remainingSteps, &StepCreateVMInstance{}, &StepWaitForRunning{}, &StepAttachGuestTools{}, newConnect())
	}
	remainingSteps = append(remainingSteps, &commonsteps.StepProvision{}, &commonsteps.StepCleanupTempKeys{Comm: &b.config.Comm}, &StepStopVmInstance{}, &StepCreateImage{}, &StepWaitForImageReady{}, &StepExpungeVmInstance{}, &StepExportImage{})

	steps = append(baseSteps, remainingSteps...)

	log.Printf("[DEBUG] Build steps prepared with %s", b.config.RedactedSummary())

	b.runner = commonsteps.NewRunner(steps, b.config.PackerConfig, ui)
	b.runner.Run(ctx, state)

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return b.collectArtifact(state, driver)
}

func (b *Builder) collectArtifact(state multistep.StateBag, driver Driver) (packersdk.Artifact, error) {
	if rawErr, ok := state.GetOk("error"); ok {
		if err, ok := rawErr.(error); ok {
			return nil, err
		}
		return nil, fmt.Errorf("unexpected error type in state: %T", rawErr)
	}
	if _, ok := state.GetOk(multistep.StateCancelled); ok {
		return nil, context.Canceled
	}
	if _, ok := state.GetOk(multistep.StateHalted); ok {
		return nil, fmt.Errorf("build halted before completion")
	}
	r := resources(state)
	if !r.Image.Created || !r.ImageReady || r.Image.Removed || r.Image.UUID == "" {
		return nil, fmt.Errorf("build did not produce a ready target image")
	}
	// Artifact carries output metadata, never source-image identifiers or secrets.
	artifact := &Artifact{config: Config{
		ImageConfig:         ImageConfig{ImageUuid: r.Image.UUID},
		ExportImageResult:   ExportImageResult{ImageUrl: b.config.ImageUrl},
		BackupStorageConfig: b.config.BackupStorageConfig,
		cleanupTimeout:      b.config.CleanupTimeout(),
	}, driver: driver, owned: true}
	artifact.warnings, _ = state.Get("cleanup_warnings").([]string)
	return artifact, nil
}

func commHost(host string) func(multistep.StateBag) (string, error) {
	return func(state multistep.StateBag) (string, error) {
		if host != "" {
			return host, nil
		}

		config, ok := state.Get("config").(*Config)
		if !ok || config == nil {
			return "", fmt.Errorf("IP address not found")
		}

		ip := config.IP

		return ip, nil
	}
}

func (b *Builder) communicatorHost() string {
	if b.config.Comm.Type == "winrm" {
		return b.config.Comm.WinRMHost
	}
	return b.config.Comm.SSHHost
}
