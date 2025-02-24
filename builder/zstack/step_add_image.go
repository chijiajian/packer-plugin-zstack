package zstack

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"zstack.io/zstack-sdk-go/pkg/param"
)

type StepAddImage struct {
}

func (s *StepAddImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	ui.Say("Starting image addition process...")
	log.Printf("[INFO] Beginning image addition step")

	platform := config.Platform
	format := config.Format
	guestOsTyep := config.GuestOsType
	sourceImageUrl := config.SourceImageUrl
	backupStorage := config.BackupStorageUuid
	imageName := config.SourceImage

	imageParam := param.AddImageParam{
		BaseParam: param.BaseParam{},
		Params: param.AddImageDetailParam{
			Name:               imageName,
			Description:        "Image added via Packer build process",
			Url:                sourceImageUrl,
			MediaType:          param.RootVolumeTemplate,
			GuestOsType:        guestOsTyep,
			System:             false,
			Format:             param.ImageFormat(format),
			Platform:           platform,
			BackupStorageUuids: []string{backupStorage},
		},
	}
	log.Printf("[DEBUG] Adding image with configuration: %+v", imageParam)
	ui.Say(fmt.Sprintf("Adding image '%s' to storage...", imageName))

	img, err := driver.AddImage(imageParam)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to add image: %s", err))
		log.Printf("[ERROR] Failed to add image: %v", err)
		return multistep.ActionHalt
	}
	log.Printf("[INFO] Successfully added image with UUID: %s", img.UUID)
	ui.Say(fmt.Sprintf("Successfully added image '%s' (UUID: %s)", imageName, img.UUID))

	config.ImageUuid = img.UUID
	state.Put("config", config)

	return multistep.ActionContinue
}

func (s *StepAddImage) Cleanup(state multistep.StateBag) {}
