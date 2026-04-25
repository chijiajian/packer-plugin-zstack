// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
)

type StepCreateImageFromSnapshot struct{}

func (s *StepCreateImageFromSnapshot) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	snapshotUuid := config.SourceVolumeSnapshotUuid
	if snapshotUuid == "" {
		err := fmt.Errorf("source_volume_snapshot_uuid is empty, cannot create image from snapshot")
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	if config.BackupStorageUuid == "" {
		err := fmt.Errorf("backup storage UUID is empty, cannot create image from snapshot")
		ui.Error(err.Error())
		state.Put("error", err)
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Validating volume snapshot '%s'...", snapshotUuid))
	snapshot, err := driver.GetVolumeSnapshot(snapshotUuid)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to get volume snapshot: %v", err))
		state.Put("error", err)
		return multistep.ActionHalt
	}
	log.Printf("[INFO] Snapshot %s status=%s state=%s volumeType=%s", snapshot.UUID, snapshot.Status, snapshot.State, snapshot.VolumeType)

	ui.Say(fmt.Sprintf("Creating image '%s' from volume snapshot '%s'...", config.ImageName, snapshotUuid))

	description := config.ImageDescription
	if description == "" {
		description = "Auto created by packer-plugin-zstack from volume snapshot"
	}

	createParam := param.CreateRootVolumeTemplateFromVolumeSnapshotParam{
		BaseParam: param.BaseParam{},
		Params: param.CreateRootVolumeTemplateFromVolumeSnapshotParamDetail{
			Name:               config.ImageName,
			Description:        &description,
			BackupStorageUuids: []string{config.BackupStorageUuid},
		},
	}
	if config.GuestOsType != "" {
		gos := config.GuestOsType
		createParam.Params.GuestOsType = &gos
	}
	if config.Platform != "" {
		platform := config.Platform
		createParam.Params.Platform = &platform
	}
	if config.Architecture != "" {
		arch := config.Architecture
		createParam.Params.Architecture = &arch
	}

	image, err := driver.CreateImageFromVolumeSnapshot(snapshotUuid, createParam)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to create image from snapshot: %v", err))
		log.Printf("[ERROR] Failed to create image from snapshot %s: %v", snapshotUuid, err)
		state.Put("error", err)
		return multistep.ActionHalt
	}

	config.ImageUuid = image.UUID

	log.Printf("[INFO] Successfully created image with UUID: %s from snapshot %s", image.UUID, snapshotUuid)
	ui.Say(fmt.Sprintf("Successfully created image '%s' (UUID: %s) from snapshot %s", config.ImageName, image.UUID, snapshotUuid))
	return multistep.ActionContinue
}

func (s *StepCreateImageFromSnapshot) Cleanup(state multistep.StateBag) {}
