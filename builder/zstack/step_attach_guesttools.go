package zstack

import (
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"golang.org/x/net/context"
)

type StepAttachGuestTools struct {
}

func (s *StepAttachGuestTools) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	instanceUuid := config.InstanceUuid
	if instanceUuid == "" {
		err := fmt.Errorf("instance UUID is required but not provided")
		ui.Error(err.Error())
		log.Printf("[ERROR] %v", err)
		return multistep.ActionHalt
	}
	log.Printf("[INFO] Starting guest tools attachment for VM: %s", instanceUuid)
	ui.Say("Starting guest tools attachment process...")

	vm, _ := driver.GetVmInstance(instanceUuid)
	err := driver.AttachGuestToolsToVm(instanceUuid)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to attach guest tools: %s", err))
		log.Printf("[ERROR] Failed to attach guest tools to VM %s: %v", instanceUuid, err)
		return multistep.ActionHalt
	}
	log.Printf("[INFO] Successfully attached guest tools to VM %s", instanceUuid)
	ui.Say(fmt.Sprintf("Successfully attached guest tools to VM '%s'", vm.Name))

	return multistep.ActionContinue
}

func (s *StepAttachGuestTools) Cleanup(state multistep.StateBag) {}
