// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"context"
	"errors"
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
	ctx, cancel := context.WithTimeout(ctx, config.ImageReadyTimeout())
	defer cancel()
	driver := stepDriver(ctx, state)

	if config.ImageUuid == "" {
		err := fmt.Errorf("image UUID is empty, cannot wait for ready status")
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	ui.Say("Waiting for image status to become ready...")

	ticker := time.NewTicker(config.PollInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			image, err := driver.GetImage(config.ImageUuid)
			if err != nil {
				ui.Error(err.Error())
				log.Printf("[ERROR] Failed to get image %s: %v", config.ImageUuid, err)
				state.Put("error", err)
				return multistep.ActionHalt
			}

			if image == nil {
				return halt(state, fmt.Errorf("empty image inventory"))
			}
			log.Printf("[DEBUG] Image %s status: %s", config.ImageUuid, image.Status)

			if image.Status == "Ready" {
				if r := resources(state); r.Image.UUID == config.ImageUuid && r.Image.Created {
					r.ImageReady = true
				}
				ui.Say("Image status is now ready!")
				return multistep.ActionContinue
			}
			if image.Status == "Deleted" || image.Status == "Failed" {
				return halt(state, fmt.Errorf("image entered terminal status %s", image.Status))
			}
			ui.Message(fmt.Sprintf("image status is %s, waiting...", image.Status))

		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.DeadlineExceeded) {
				err = fmt.Errorf("timeout waiting for image %s status to become ready: %w", config.ImageUuid, err)
			}
			state.Put("error", err)
			ui.Error(err.Error())
			log.Printf("[INFO] Waiting for image %s ended: %v", config.ImageUuid, err)
			return multistep.ActionHalt
		}
	}
}

func (s *StepWaitForImageReady) Cleanup(state multistep.StateBag) {}
