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

	var err error

	err = driver.DestroyVmInstance(instanceUuid)
	if err != nil {
		return multistep.ActionHalt
	}
	err = driver.DeleteVmInstance(instanceUuid)
	if err != nil {
		return multistep.ActionHalt
	}

	ui.Say("Expunge vm instance...")

	config.InstanceUuid = ""
	ui.Say("Expunge vm instance success...")
	return multistep.ActionContinue
}

func (s *StepExpungeVmInstance) Cleanup(state multistep.StateBag) {}
