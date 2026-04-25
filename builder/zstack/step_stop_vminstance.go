// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"context"
	"fmt"

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
	ui.Say(fmt.Sprintf("Stopping VM instance '%s'...", instanceUuid))

	vmInstance, err := driver.StopVminstance(instanceUuid)

	if err != nil {
		// AC-004-03: Write error to state so Cleanup can detect failure
		err = fmt.Errorf("failed to stop VM instance '%s': %v", instanceUuid, err)
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Successfully stopped VM instance '%s'", vmInstance.Name))
	return multistep.ActionContinue
}

func (s *StepStopVmInstance) Cleanup(state multistep.StateBag) {}
