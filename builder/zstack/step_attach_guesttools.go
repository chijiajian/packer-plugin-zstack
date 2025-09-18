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

	vm, err := driver.GetVmInstance(instanceUuid)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to get VM instance: %v", err))
		log.Printf("[ERROR] Failed to get VM instance %s: %v", instanceUuid, err)
		state.Put("error", err)
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Attaching guest tools to VM '%s'...", vm.Name))
	log.Printf("[INFO] Attaching guest tools to VM '%s' (UUID: %s)", vm.Name, instanceUuid)

	if err := driver.AttachGuestToolsToVm(instanceUuid); err != nil {
		ui.Error(fmt.Sprintf("Failed to attach guest tools: %v", err))
		log.Printf("[ERROR] Failed to attach guest tools to VM %s: %v", instanceUuid, err)
		state.Put("error", err)
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Successfully attached guest tools to VM '%s'", vm.Name))
	log.Printf("[INFO] Successfully attached guest tools to VM %s", instanceUuid)

	return multistep.ActionContinue
}

func (s *StepAttachGuestTools) Cleanup(state multistep.StateBag) {}
