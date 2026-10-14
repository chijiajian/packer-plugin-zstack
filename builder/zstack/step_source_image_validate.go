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

type StepSourceImageValidate struct {
}

func (s *StepSourceImageValidate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	ui.Say("Validating source image...")

	// AC-002-01: Skip image query when image_uuid is provided (UUID takes priority)
	if config.ImageConfig.ImageUuid != "" {
		log.Printf("[INFO] Using provided source image UUID: %s", config.ImageConfig.ImageUuid)
		ui.Say(fmt.Sprintf("Using provided source image UUID: %s", config.ImageConfig.ImageUuid))
		return multistep.ActionContinue
	}

	images, err := validateImage(ctx, state)
	if err != nil {
		state.Put("error", err)
		ui.Errorf("Image validation failed: %s", err)
		return multistep.ActionHalt
	}
	if images[0].Status != "Ready" || images[0].State != "Enabled" {
		return halt(state, fmt.Errorf("image %q is not ready (status=%s, state=%s)", config.SourceImage, images[0].Status, images[0].State))
	}

	config.ImageConfig.ImageUuid = images[0].UUID
	ui.Say("Source image validated")
	return multistep.ActionContinue
}

func validateImage(ctx context.Context, state multistep.StateBag) ([]view.ImageInventoryView, error) {
	config := state.Get("config").(*Config)
	driver := stepDriver(ctx, state)

	images, err := driver.QueryImage(config.SourceImage)
	if err != nil {
		return nil, fmt.Errorf("error querying image: %s", err)
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("image '%s' not found", config.SourceImage)
	}
	if len(images) > 1 {
		return nil, fmt.Errorf("multiple image resources match the name; specify UUID")
	}
	return images, nil
}

func (s *StepSourceImageValidate) Cleanup(state multistep.StateBag) {}
