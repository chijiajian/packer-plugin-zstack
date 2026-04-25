// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

type StepInstanceOfferingValidate struct {
}

func (s *StepInstanceOfferingValidate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	ui.Say("Validating instance offering...")

	// AC-002-03: Skip query when instance_offering_uuid is provided
	if config.InstanceConfig.InstanceOfferingUuid != "" {
		log.Printf("[INFO] Using provided instance offering UUID: %s", config.InstanceConfig.InstanceOfferingUuid)
		ui.Say(fmt.Sprintf("Using provided instance offering UUID: %s", config.InstanceConfig.InstanceOfferingUuid))
		return multistep.ActionContinue
	}

	instanceOfferings, err := validateInstanceOffering(state)
	if err != nil {
		ui.Errorf("Instance offering validation failed: %s, please use cpu_num and memory_size instead", err)
		return multistep.ActionHalt
	}

	config.InstanceConfig.InstanceOfferingUuid = instanceOfferings[0].UUID
	ui.Say("Instance offering validated")
	return multistep.ActionContinue
}

func validateInstanceOffering(state multistep.StateBag) ([]view.InstanceOfferingInventoryView, error) {
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	instanceOffering, err := driver.QueryInstanceOffering(config.InstanceOfferingName)
	if err != nil {
		return nil, fmt.Errorf("error querying instance offering: %s", err)
	}
	if len(instanceOffering) == 0 {
		return nil, fmt.Errorf("instance offering '%s' not found", config.InstanceOfferingName)
	}
	return instanceOffering, nil
}

func (s *StepInstanceOfferingValidate) Cleanup(state multistep.StateBag) {}
