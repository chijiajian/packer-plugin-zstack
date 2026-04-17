package zstack

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

type StepPreValidate struct {
}

func (s *StepPreValidate) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	ui.Say("Validating configuration...")

	// AC-002-02: Skip network query when network_uuid is provided
	if config.NetworkConfig.L3NetworkUuid != "" {
		log.Printf("[INFO] Using provided network UUID: %s", config.NetworkConfig.L3NetworkUuid)
		ui.Say(fmt.Sprintf("Using provided network UUID: %s", config.NetworkConfig.L3NetworkUuid))
	} else {
		networks, err := validateNetwork(state)
		if err != nil {
			ui.Errorf("Network validation failed: %s", err)
			return multistep.ActionHalt
		}
		config.NetworkConfig.L3NetworkUuid = networks[0].UUID
		ui.Say("L3 network validated")
	}

	// source_image_url imports require backup storage for image download/import.
	if config.SourceImageUrl != "" && config.BackupStorageConfig.BackupStorageUuid == "" && config.BackupStorageConfig.BackupStorageName == "" {
		err := fmt.Errorf("backup_storage_name or backup_storage_uuid is required when source_image_url is set")
		ui.Errorf("Backup storage validation failed: %s", err)
		return multistep.ActionHalt
	}

	// AC-002-04: Skip backup storage query when backup_storage_uuid is provided
	// AC-003-01: Backup storage is optional for non-import flows.
	if config.BackupStorageConfig.BackupStorageUuid != "" {
		log.Printf("[INFO] Using provided backup storage UUID: %s", config.BackupStorageConfig.BackupStorageUuid)
		ui.Say(fmt.Sprintf("Using provided backup storage UUID: %s", config.BackupStorageConfig.BackupStorageUuid))
	} else if config.BackupStorageConfig.BackupStorageName != "" {
		backupStorages, err := validateBackupStorage(state)
		if err != nil {
			ui.Errorf("Backup storage validation failed: %s", err)
			return multistep.ActionHalt
		}
		config.BackupStorageConfig.BackupStorageUuid = backupStorages[0].UUID
		ui.Say("Backup storage validated")
	} else {
		log.Printf("[INFO] No backup storage configured, image export will be skipped")
		ui.Say("No backup storage configured, image export will be skipped")
	}

	state.Put("config", config)
	return multistep.ActionContinue
}

func validateNetwork(state multistep.StateBag) ([]view.L3NetworkInventoryView, error) {
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	networks, err := driver.QueryL3Network(config.L3NetworkName)
	if err != nil {
		return nil, fmt.Errorf("error querying L3 Network: %s", err)
	}
	if len(networks) == 0 {
		return nil, fmt.Errorf("network '%s' not found", config.L3NetworkName)
	}
	return networks, nil
}

func validateBackupStorage(state multistep.StateBag) ([]view.BackupStorageInventoryView, error) {
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	backupStorage, err := driver.QueryBackStorage(config.BackupStorageName)
	if err != nil {
		return nil, fmt.Errorf("error querying backup storage: %s", err)
	}
	if len(backupStorage) == 0 {
		return nil, fmt.Errorf("backup storage '%s' not found", config.BackupStorageName)
	}
	return backupStorage, nil
}

func (s *StepPreValidate) Cleanup(state multistep.StateBag) {}
