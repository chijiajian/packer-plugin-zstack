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

	ui.Say("Waiting for VM instance to become running...")

	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			err := fmt.Errorf("timeout waiting for instance to become running")
			state.Put("error", err)
			ui.Errorf(err.Error())
			return multistep.ActionHalt
		case <-ticker.C:
			instance, err := driver.GetVmInstance(config.InstanceUuid)
			if err != nil {
				state.Put("error", err)
				ui.Errorf(err.Error())
				return multistep.ActionHalt
			}

			if instance.State == "Running" {
				ui.Say("Instance is now running!")
				return multistep.ActionContinue
			}
			ui.Message(fmt.Sprintf("Instance state is %s, waiting...", instance.State))

		case <-ctx.Done():
			return multistep.ActionHalt
		}
	}
}

func (s *StepWaitForRunning) Cleanup(state multistep.StateBag) {

}
