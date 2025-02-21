package zstack

import (
	"context"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepStopVmInstance struct {
}

func (s *StepStopVmInstance) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	instanceUuid := config.InstanceUuid

	vmInstance, err := driver.StopVminstance(instanceUuid)

	ui.Say("stop vm instances..." + instanceUuid)

	if err != nil {
		ui.Say("failt to stop  vm instance")
		return multistep.ActionHalt
	}
	ui.Say("stopped... " + vmInstance.Name)
	return multistep.ActionContinue
}

func (s *StepStopVmInstance) Cleanup(state multistep.StateBag) {}
