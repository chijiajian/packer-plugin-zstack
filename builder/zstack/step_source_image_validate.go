package zstack

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/terraform-zstack-modules/zstack-sdk-go/pkg/view"
)

type StepSourceImageValidate struct {
}

func (s *StepSourceImageValidate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	ui.Say("Validating configuration...")

	images, err := validateImage(state)
	if err != nil || images[0].Status != "Ready" || images[0].State != "Enabled" {
		ui.Errorf("Image validation failed: %s", err)
		return multistep.ActionHalt
	}

	config.ImageConfig.ImageUuid = images[0].UUID
	ui.Say("Source Image validated")
	state.Put("config", config)

	return multistep.ActionContinue
}
func validateImage(state multistep.StateBag) ([]view.ImageView, error) {
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	images, err := driver.QueryImage(config.SourceImage)
	if err != nil {
		return nil, fmt.Errorf("error querying image: %s", err)
	}

	if images == nil {
		return nil, fmt.Errorf("image not found")
	}

	return images, nil
}

func (s *StepSourceImageValidate) Cleanup(state multistep.StateBag) {}
