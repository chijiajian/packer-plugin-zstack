package zstack

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
)

type StepAddImage struct {
}

func (s *StepAddImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	if config.SourceImageUrl == "" || config.SourceImage == "" {
		ui.Error("Source image URL or name is empty")
		log.Printf("[ERROR] Source image URL or name is empty")
		return multistep.ActionHalt
	}

	ui.Say("Starting image addition process...")
	log.Printf("[INFO] Adding image '%s' from URL '%s'", config.SourceImage, config.SourceImageUrl)

	description := config.ImageDescription
	if description == "" {
		description = "Image added via Packer build process"
	}

	imageParam := param.AddImageParam{
		BaseParam: param.BaseParam{},
		Params: param.AddImageParamDetail{
			Name:        config.SourceImage,
			Description: &description,
			Url:         config.SourceImageUrl,
			MediaType:   strPtr("RootVolumeTemplate"),
			GuestOsType: strPtr(config.GuestOsType),
			System:      false,
			Format:      strPtr(config.Format),
			Platform:    strPtr(config.Platform),
		},
	}
	if config.BackupStorageUuid != "" {
		imageParam.Params.BackupStorageUuids = []string{config.BackupStorageUuid}
	}

	img, err := driver.AddImage(imageParam)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to add image: %s", err))
		log.Printf("[ERROR] Failed to add image: %v", err)
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Successfully added image '%s' (UUID: %s)", config.SourceImage, img.UUID))
	log.Printf("[INFO] Image added successfully with UUID: %s", img.UUID)

	config.ImageUuid = img.UUID
	state.Put("config", config)

	return multistep.ActionContinue
}

func (s *StepAddImage) Cleanup(state multistep.StateBag) {}
