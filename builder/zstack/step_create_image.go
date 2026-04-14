package zstack

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
)

type StepCreateImage struct {
}

func (s *StepCreateImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	ui.Say(fmt.Sprintf("Creating image '%s' from VM root volume...", config.ImageName))
	log.Printf("[INFO] Starting image creation from root volume: %s", config.RootVolumeUuid)

	description := config.ImageDescription
	if description == "" {
		description = "Auto created by packer-plugin-zstack"
	}

	createImageFromRootVolumeParam := param.CreateRootVolumeTemplateFromRootVolumeParam{
		BaseParam: param.BaseParam{},
		Params: param.CreateRootVolumeTemplateFromRootVolumeParamDetail{
			Name:        config.ImageName,
			Description: &description,
		},
	}

	// AC-003-03: Only include BackupStorageUuids when configured
	if config.BackupStorageConfig.BackupStorageUuid != "" {
		createImageFromRootVolumeParam.Params.BackupStorageUuids = []string{config.BackupStorageConfig.BackupStorageUuid}
	}

	image, err := driver.CreateImage(config.RootVolumeUuid, createImageFromRootVolumeParam)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to create image: %v", err))
		log.Printf("[ERROR] Failed to create image: %v", err)
		state.Put("error", err)
		return multistep.ActionHalt
	}

	config.ImageUuid = image.UUID
	state.Put("config", config)

	log.Printf("[INFO] Successfully created image with UUID: %s", image.UUID)
	ui.Say(fmt.Sprintf("Successfully created image '%s' (UUID: %s)", config.ImageName, image.UUID))
	return multistep.ActionContinue
}

func (s *StepCreateImage) Cleanup(state multistep.StateBag) {}
