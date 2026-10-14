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
		if config.ISOInstall {
			image, err := stepDriver(ctx, state).GetImage(config.ImageUuid)
			if err != nil {
				return halt(state, err)
			}
			if image == nil || image.MediaType != "ISO" || image.Status != "Ready" || image.State != "Enabled" {
				return halt(state, fmt.Errorf("ISO source must be an Enabled Ready ISO"))
			}
			if err := config.checkISOSource(image); err != nil {
				return halt(state, err)
			}
		}
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

	if config.ISOInstall && images[0].MediaType != "ISO" {
		return halt(state, fmt.Errorf("source image must have mediaType ISO"))
	}
	if config.ISOInstall {
		if err := config.checkISOSource(&images[0]); err != nil {
			return halt(state, err)
		}
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

func (c *Config) checkISOSource(image *view.ImageInventoryView) error {
	if c.Platform == "" {
		c.Platform = image.Platform
	}
	if c.Architecture == "" {
		c.Architecture = image.Architecture
		return nil
	}
	if image.Architecture != "" && c.Architecture != image.Architecture {
		return fmt.Errorf("configured architecture does not match the source ISO")
	}
	return nil
}
