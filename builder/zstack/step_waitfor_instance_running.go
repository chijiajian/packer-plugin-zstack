package zstack

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepWaitForRunning struct{}

func (s *StepWaitForRunning) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	if config.InstanceUuid == "" {
		err := fmt.Errorf("Instance UUID is empty, cannot wait for running")
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	ui.Say("Waiting for VM instance to become running...")

	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			err := fmt.Errorf("timeout waiting for instance %s to become running", config.InstanceUuid)
			state.Put("error", err)
			ui.Errorf(err.Error())
			return multistep.ActionHalt
		case <-ticker.C:
			instance, err := driver.GetVmInstance(config.InstanceUuid)
			if err != nil {
				state.Put("error", err)
				ui.Errorf("Failed to get VM instance: %v", err)
				return multistep.ActionHalt
			}

			if instance.State == "Running" {
				ui.Say("Instance is now running!")
				return multistep.ActionContinue
			}
			ui.Message(fmt.Sprintf("[%s] Instance state is %s, waiting...", time.Now().Format(time.RFC3339), instance.State))

		case <-ctx.Done():
			ui.Error("Context cancelled while waiting for VM running")
			return multistep.ActionHalt
		}
	}
}

func (s *StepWaitForRunning) Cleanup(state multistep.StateBag) {

}
