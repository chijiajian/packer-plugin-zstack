// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"context"
	"fmt"
	"log"
	"time"

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

	if config.BackupStorageConfig.BackupStorageUuid == "" {
		err := fmt.Errorf("backup storage UUID is empty, cannot create image from snapshot")
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	snapshotName := fmt.Sprintf("packer-%s-snapshot", config.ImageName)
	snapshotDescription := "Auto created by packer-plugin-zstack before image creation"
	createSnapshotParam := param.CreateVolumeSnapshotParam{
		BaseParam: param.BaseParam{},
		Params: param.CreateVolumeSnapshotParamDetail{
			Name:        snapshotName,
			Description: &snapshotDescription,
		},
	}

	ui.Say(fmt.Sprintf("Creating snapshot from root volume '%s'...", config.RootVolumeUuid))
	snapshot, err := driver.CreateVolumeSnapshot(config.RootVolumeUuid, createSnapshotParam)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to create snapshot from root volume: %v", err))
		log.Printf("[ERROR] Failed to create snapshot from root volume %s: %v", config.RootVolumeUuid, err)
		state.Put("error", err)
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Waiting for snapshot '%s' to become ready...", snapshot.UUID))
	timeout := time.After(config.ImageReadyTimeout())
	ticker := time.NewTicker(config.PollInterval())
	defer ticker.Stop()

	ready := false
	for !ready {
		select {
		case <-timeout:
			err := fmt.Errorf("timeout waiting for snapshot %s status to become ready", snapshot.UUID)
			state.Put("error", err)
			ui.Error(err.Error())
			log.Printf("[ERROR] %v", err)
			return multistep.ActionHalt
		case <-ticker.C:
			current, getErr := driver.GetVolumeSnapshot(snapshot.UUID)
			if getErr != nil {
				ui.Error(getErr.Error())
				log.Printf("[ERROR] Failed to get snapshot %s: %v", snapshot.UUID, getErr)
				state.Put("error", getErr)
				return multistep.ActionHalt
			}
			if current == nil {
				err := fmt.Errorf("snapshot %s not found while waiting for ready status", snapshot.UUID)
				ui.Error(err.Error())
				log.Printf("[ERROR] %v", err)
				state.Put("error", err)
				return multistep.ActionHalt
			}

			log.Printf("[DEBUG] Snapshot %s status=%s state=%s", snapshot.UUID, current.Status, current.State)
			if current.Status == "Ready" && current.State == "Enabled" {
				ui.Say("Snapshot status is now ready!")
				ready = true
				continue
			}
			ui.Message(fmt.Sprintf("snapshot status is %s/%s, waiting...", current.Status, current.State))
		case <-ctx.Done():
			log.Printf("[INFO] Context cancelled while waiting for snapshot %s", snapshot.UUID)
			return multistep.ActionHalt
		}
	}

	log.Printf("[INFO] Successfully created volume snapshot with UUID: %s", snapshot.UUID)
	ui.Say(fmt.Sprintf("Creating image '%s' from snapshot '%s'...", config.ImageName, snapshot.UUID))

	description := config.ImageDescription
	if description == "" {
		description = "Auto created by packer-plugin-zstack"
	}

	createImageFromSnapshotParam := param.CreateRootVolumeTemplateFromVolumeSnapshotParam{
		BaseParam: param.BaseParam{},
		Params: param.CreateRootVolumeTemplateFromVolumeSnapshotParamDetail{
			Name:               config.ImageName,
			Description:        &description,
			BackupStorageUuids: []string{config.BackupStorageConfig.BackupStorageUuid},
		},
	}
	if config.GuestOsType != "" {
		gos := config.GuestOsType
		createImageFromSnapshotParam.Params.GuestOsType = &gos
	}
	if config.Platform != "" {
		platform := config.Platform
		createImageFromSnapshotParam.Params.Platform = &platform
	}
	if config.Architecture != "" {
		arch := config.Architecture
		createImageFromSnapshotParam.Params.Architecture = &arch
	}

	image, err := driver.CreateImageFromVolumeSnapshot(snapshot.UUID, createImageFromSnapshotParam)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to create image: %v", err))
		log.Printf("[ERROR] Failed to create image: %v", err)
		state.Put("error", err)
		return multistep.ActionHalt
	}

	config.ImageUuid = image.UUID

	log.Printf("[INFO] Successfully created image with UUID: %s", image.UUID)
	ui.Say(fmt.Sprintf("Successfully created image '%s' (UUID: %s)", config.ImageName, image.UUID))
	return multistep.ActionContinue
}

func (s *StepCreateImage) Cleanup(state multistep.StateBag) {}
