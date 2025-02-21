package zstack

import (
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"golang.org/x/net/context"
)

type StepAttachGuestTools struct {
	// vm *param.CreateVmInstanceParam
}

func (s *StepAttachGuestTools) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	instanceUuid := config.InstanceUuid

	vms, _ := driver.GetVmInstance(instanceUuid)
	ui.Say("start attach guest tools..." + instanceUuid + vms.Name)

	err := driver.AttachGuestToolsToVm(instanceUuid)
	if err != nil {
		ui.Say("failt to attach guest tools")
		return multistep.ActionHalt
	}
	ui.Say("attach tools to " + instanceUuid)
	return multistep.ActionContinue
}

func (s *StepAttachGuestTools) Cleanup(state multistep.StateBag) {}
