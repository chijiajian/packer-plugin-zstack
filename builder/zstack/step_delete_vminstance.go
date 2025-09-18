package zstack

import (
	"context"

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

	if err := driver.DestroyVmInstance(instanceUuid); err != nil {
		ui.Error("Failed to destroy VM instance: " + err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	if err := driver.DeleteVmInstance(instanceUuid); err != nil {
		ui.Error("Failed to delete VM instance: " + err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	config.InstanceUuid = ""
	ui.Say("Expunge vm instance success...")
	return multistep.ActionContinue
}

func (s *StepExpungeVmInstance) Cleanup(state multistep.StateBag) {}
