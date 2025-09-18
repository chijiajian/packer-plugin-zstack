package zstack

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/terraform-zstack-modules/zstack-sdk-go/pkg/param"
)

type StepExportImage struct {
}

func (s *StepExportImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	if config.BackupStorageUuid == "" || config.ImageUuid == "" {
		err := fmt.Errorf("backup storage UUID or image UUID is empty")
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	ui.Say("Exporting image to backup storage...")

	exportImageParam := param.ExportImageFromBackupStorageParam{
		BackupStorageUuid: config.BackupStorageUuid,
		ExportImageFromBackupStorage: param.ExportImageFromBackupStorageDetailParam{
			ImageUuid: config.ImageUuid,
		},
	}

	exportImageResult, err := driver.ExportImage(exportImageParam)
	if err != nil {
		ui.Error("Failed to export image: " + err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	config.ImageUrl = exportImageResult.ImageUrl
	state.Put("config", config)
	ui.Say("Successfully exported image: " + config.ImageUrl)
	return multistep.ActionContinue
}

func (s *StepExportImage) Cleanup(state multistep.StateBag) {}
