package zstack

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepWaitForImageReady struct{}

func (s *StepWaitForImageReady) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	ui.Say("Waiting for image status to become ready...")

	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			err := fmt.Errorf("timeout waiting for image status to become ready")
			state.Put("error", err)
			ui.Errorf(err.Error())
			return multistep.ActionHalt
		case <-ticker.C:
			image, err := driver.GetImage(config.ImageUuid)
			if err != nil {
				state.Put("error", err)
				ui.Errorf(err.Error())
				return multistep.ActionHalt
			}

			if image.Status == "Ready" {
				ui.Say("Image status is now ready!")
				return multistep.ActionContinue
			}
			ui.Message(fmt.Sprintf("image status is %s, waiting...", image.Status))

		case <-ctx.Done():
			return multistep.ActionHalt
		}
	}
}

func (s *StepWaitForImageReady) Cleanup(state multistep.StateBag) {

}
