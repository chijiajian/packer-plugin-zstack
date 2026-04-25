// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"context"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepExpungeVmInstance struct {
}

func (s *StepExpungeVmInstance) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	instanceUuid := config.InstanceUuid
	if instanceUuid == "" {
		ui.Say("No VM instance to expunge, skipping...")
		return multistep.ActionContinue
	}

	ui.Say("Expunging VM instance...")

	// Image is already created at this point; treat cleanup failures as
	// non-fatal warnings so the build still surfaces a usable artifact.
	if err := driver.DestroyVmInstance(instanceUuid); err != nil {
		ui.Say("Warning: failed to destroy VM instance (non-fatal): " + err.Error())
		log.Printf("[WARN] Failed to destroy VM instance %s during expunge: %v", instanceUuid, err)
		return multistep.ActionContinue
	}

	if err := driver.DeleteVmInstance(instanceUuid); err != nil {
		ui.Say("Warning: failed to expunge VM instance (non-fatal): " + err.Error())
		log.Printf("[WARN] Failed to expunge VM instance %s: %v", instanceUuid, err)
		return multistep.ActionContinue
	}

	config.InstanceUuid = ""
	ui.Say("Expunge vm instance success...")
	return multistep.ActionContinue
}

func (s *StepExpungeVmInstance) Cleanup(state multistep.StateBag) {}
