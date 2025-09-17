package zstack

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/terraform-zstack-modules/zstack-sdk-go/pkg/view"
)

type StepInstanceOfferingValidate struct {
}

func (s *StepInstanceOfferingValidate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	ui.Say("InstanceOffering  validate...")

	instanceOfferings, err := validateInstanceOffering(state)
	if err != nil {
		ui.Errorf("Instance Offering validation failed: %s, pls using cpu_num and memory_size", err)
		return multistep.ActionHalt
	}

	config.InstanceConfig.InstanceOfferingUuid = instanceOfferings[0].UUID
	ui.Say("instance offering validated")

	state.Put("config", config)
	return multistep.ActionContinue
}

func validateInstanceOffering(state multistep.StateBag) ([]view.InstanceOfferingInventoryView, error) {
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	instanceOffering, err := driver.QueryInstanceOffering(config.InstanceOfferingName)
	if err != nil {
		return nil, fmt.Errorf("error querying L3 Network: %s", err)
	}

	if instanceOffering == nil {
		return nil, fmt.Errorf("instanceOffering not found")
	}

	return instanceOffering, nil
}

func (s *StepInstanceOfferingValidate) Cleanup(state multistep.StateBag) {}
