package zstack

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepWaitForImageReady struct{}

func (s *StepWaitForImageReady) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	if config.ImageUuid == "" {
		err := fmt.Errorf("image UUID is empty, cannot wait for ready status")
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	ui.Say("Waiting for image status to become ready...")

	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			err := fmt.Errorf("timeout waiting for image %s status to become ready", config.ImageUuid)
			state.Put("error", err)
			ui.Errorf(err.Error())
			log.Printf("[ERROR] %v", err)
			return multistep.ActionHalt
		case <-ticker.C:
			image, err := driver.GetImage(config.ImageUuid)
			if err != nil {
				ui.Errorf(err.Error())
				log.Printf("[ERROR] Failed to get image %s: %v", config.ImageUuid, err)
				state.Put("error", err)
				return multistep.ActionHalt
			}

			log.Printf("[DEBUG] Image %s status: %s", config.ImageUuid, image.Status)

			if image.Status == "Ready" {
				ui.Say("Image status is now ready!")
				return multistep.ActionContinue
			}
			ui.Message(fmt.Sprintf("image status is %s, waiting...", image.Status))

		case <-ctx.Done():
			log.Printf("[INFO] Context cancelled while waiting for image %s", config.ImageUuid)
			return multistep.ActionHalt
		}
	}
}

func (s *StepWaitForImageReady) Cleanup(state multistep.StateBag) {

}
