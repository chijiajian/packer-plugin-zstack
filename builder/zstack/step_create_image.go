package zstack

import (
	"context"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"zstack.io/zstack-sdk-go/pkg/param"
)

type StepCreateImage struct {
	// vm *param.CreateVmInstanceParam
}

func (s *StepCreateImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	rootVolumeUuid := config.RootVolumeUuid

	createImageFromRootVolumeParam := param.CreateRootVolumeTemplateFromRootVolumeParam{
		BaseParam:      param.BaseParam{},
		RootVolumeUuid: rootVolumeUuid,
		Params: param.CreateRootVolumeTemplateFromRootVolumeDetailParam{
			Name:               config.ImageName,
			Description:        "Auto created by packer-plugin-zstack",
			BackupStorageUuids: []string{config.BackupStorageConfig.BackupStorageUuid},
		},
	}

	image, err := driver.CreateImage(createImageFromRootVolumeParam)

	//vms, _ := driver.GetVmInstance(instanceUuid)
	ui.Say("Create Image from vm instance root volume...")

	if err != nil {
		ui.Say("failt to create image from vm instance root volume")
		return multistep.ActionHalt
	}
	config.ImageUuid = image.UUID
	ui.Say("created image from vm instance ")
	return multistep.ActionContinue
}

func (s *StepCreateImage) Cleanup(state multistep.StateBag) {}
