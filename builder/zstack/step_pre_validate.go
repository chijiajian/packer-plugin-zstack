package zstack

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"zstack.io/zstack-sdk-go/pkg/view"
)

type StepPreValidate struct {
}

func (s *StepPreValidate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	ui.Say("Validating configuration...")

	images, err := validateImage(state)
	if err != nil || images[0].Status != "Ready" || images[0].State != "Enabled" {
		ui.Errorf("Image validation failed: %s", err)
		return multistep.ActionHalt
	}

	config.ImageConfig.ImageUuid = images[0].UUID
	ui.Say("Source Image validated")

	networks, err := validateNetwork(state)
	if err != nil {
		ui.Errorf("Network validation failed: %s", err)
		return multistep.ActionHalt
	}
	//state.Put("l3_network_uuid", networks[0].UUID)
	config.NetworkConfig.L3NetworkUuid = networks[0].UUID
	ui.Say("L3 network validated")

	instanceOfferings, err := validateInstanceOffering(state)
	if err != nil {
		ui.Errorf("Instance Offering validation failed: %s", err)
		return multistep.ActionHalt
	}

	config.InstanceConfig.InstanceOfferingUuid = instanceOfferings[0].UUID
	ui.Say("instance offering validated")

	backupStoarges, err := validateBackupStorage(state)
	if err != nil {
		ui.Errorf("image storage validation failed: ", err)
		return multistep.ActionHalt
	}
	//state.Put("backup_storage_uuid", backupStoarges[0].UUID)
	config.BackupStorageConfig.BackupStorageUuid = backupStoarges[0].UUID
	//state.Put("config", &config)
	ui.Say("image storage validated")

	state.Put("config", config)

	return multistep.ActionContinue
}

func validateImage(state multistep.StateBag) ([]view.ImageView, error) {
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	images, err := driver.QueryImage(config.SourceImage)
	if err != nil {
		return nil, fmt.Errorf("error querying image: %s", err)
	}

	if images == nil {
		return nil, fmt.Errorf("image not found")
	}

	return images, nil
}

func validateNetwork(state multistep.StateBag) ([]view.L3NetworkInventoryView, error) {
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	networks, err := driver.QueryL3Network(config.L3NetworkName)
	if err != nil {
		return nil, fmt.Errorf("error querying L3 Network: %s", err)
	}

	if networks == nil {
		return nil, fmt.Errorf("network not found")
	}

	return networks, nil
}

func validateInstanceOffering(state multistep.StateBag) ([]view.InstanceOfferingInventoryView, error) {
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	instanceOffering, err := driver.QueryInstanceOffering(config.InstanceOfferingName)
	if err != nil {
		return nil, fmt.Errorf("error querying L3 Network: %s", err)
	}

	if instanceOffering == nil {
		return nil, fmt.Errorf("instanceOffering not found")
	}

	return instanceOffering, nil
}

func validateBackupStorage(state multistep.StateBag) ([]view.BackupStorageInventoryView, error) {
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	backupStorage, err := driver.QueryBackStorage(config.BackupStorageName)
	if err != nil {
		return nil, fmt.Errorf("error quering image storagtes: %s", err)
	}

	if backupStorage == nil {
		return nil, fmt.Errorf("image storage not fount")
	}

	return backupStorage, nil
}

func (s *StepPreValidate) Cleanup(state multistep.StateBag) {}
