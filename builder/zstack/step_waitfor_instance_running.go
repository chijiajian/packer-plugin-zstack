// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepWaitForRunning struct{}

func (s *StepWaitForRunning) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	ctx, cancel := context.WithTimeout(ctx, config.VmRunningTimeout())
	defer cancel()
	driver := stepDriver(ctx, state)

	if config.InstanceUuid == "" {
		err := fmt.Errorf("Instance UUID is empty, cannot wait for running")
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	ui.Say("Waiting for VM instance to become running...")

	ticker := time.NewTicker(config.PollInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			instance, err := driver.GetVmInstance(config.InstanceUuid)
			if err != nil {
				state.Put("error", err)
				ui.Error(fmt.Sprintf("Failed to get VM instance: %v", err))
				return multistep.ActionHalt
			}

			if instance == nil {
				return halt(state, fmt.Errorf("empty VM inventory"))
			}
			if instance.State == "Running" {
				ui.Say("Instance is now running!")
				return multistep.ActionContinue
			}
			ui.Message(fmt.Sprintf("[%s] Instance state is %s, waiting...", time.Now().Format(time.RFC3339), instance.State))

		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.DeadlineExceeded) {
				err = fmt.Errorf("timeout waiting for instance %s to become running: %w", config.InstanceUuid, err)
			}
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}
}

func (s *StepWaitForRunning) Cleanup(state multistep.StateBag) {}
