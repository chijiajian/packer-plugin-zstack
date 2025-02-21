package zstack

import (
	"context"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"zstack.io/zstack-sdk-go/pkg/param"
)

type StepExportImage struct {
}

func (s *StepExportImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	backupStorageUuid := config.BackupStorageUuid
	imageUuid := config.ImageUuid
	//rootVolumeUuid := config.RootVolumeUuid  param.ExportImageFromBackupStorageParam

	exportImageParam := param.ExportImageFromBackupStorageParam{
		BackupStorageUuid: backupStorageUuid,
		ExportImageFromBackupStorage: param.ExportImageFromBackupStorageDetailParam{
			ImageUuid: imageUuid,
		},
	}

	exportImageResult, err := driver.ExportImage(exportImageParam)

	//vms, _ := driver.GetVmInstance(instanceUuid)
	ui.Say("Export Image from...")

	if err != nil {
		ui.Say("failt to export image")
		return multistep.ActionHalt
	}
	config.ImageUrl = exportImageResult.ImageUrl
	ui.Say("Export Image from..." + config.ImageUrl)
	return multistep.ActionContinue
}

func (s *StepExportImage) Cleanup(state multistep.StateBag) {}
