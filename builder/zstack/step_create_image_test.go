// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

func TestStepCreateImage_Run(t *testing.T) {
	t.Run("CreateImageFromStoppedRootVolumeSuccess", func(t *testing.T) {
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img-name"},
			InstanceConfig:      InstanceConfig{RootVolumeUuid: "root-vol-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{
			CreateImageResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "img-uuid-1"}},
		}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.CreateImageCalled)
		assert.Equal(t, "root-vol-1", driver.CreateImageRootVolumeUuid)
		assert.Equal(t, []string{"bs-1"}, driver.CreateImageParam.Params.BackupStorageUuids)
		assert.False(t, driver.CreateVolumeSnapshotCalled)
		assert.False(t, driver.CreateImageFromSnapshotCalled)
		assert.Equal(t, "img-uuid-1", config.ImageUuid)
	})

	t.Run("CreateImageWithMetadata", func(t *testing.T) {
		config := &Config{
			ImageConfig: ImageConfig{
				ImageName:        "img-name",
				ImageDescription: "custom desc",
				Platform:         "Linux",
				GuestOsType:      "Ubuntu 22.04",
				Architecture:     "x86_64",
			},
			InstanceConfig:      InstanceConfig{RootVolumeUuid: "root-vol-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{
			CreateImageResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "img-uuid-1"}},
		}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.CreateImageCalled)
		assert.Equal(t, "img-name", driver.CreateImageParam.Params.Name)
		assert.Equal(t, []string{"bs-1"}, driver.CreateImageParam.Params.BackupStorageUuids)
		if assert.NotNil(t, driver.CreateImageParam.Params.Description) {
			assert.Equal(t, "custom desc", *driver.CreateImageParam.Params.Description)
		}
		if assert.NotNil(t, driver.CreateImageParam.Params.Platform) {
			assert.Equal(t, "Linux", *driver.CreateImageParam.Params.Platform)
		}
		if assert.NotNil(t, driver.CreateImageParam.Params.GuestOsType) {
			assert.Equal(t, "Ubuntu 22.04", *driver.CreateImageParam.Params.GuestOsType)
		}
		if assert.NotNil(t, driver.CreateImageParam.Params.Architecture) {
			assert.Equal(t, "x86_64", *driver.CreateImageParam.Params.Architecture)
		}
		assert.False(t, driver.CreateVolumeSnapshotCalled)
		assert.False(t, driver.CreateImageFromSnapshotCalled)
	})

	t.Run("CreateImageDefaultDescription", func(t *testing.T) {
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img-name"},
			InstanceConfig:      InstanceConfig{RootVolumeUuid: "root-vol-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{
			CreateImageResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "img-uuid-1"}},
		}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		if assert.NotNil(t, driver.CreateImageParam.Params.Description) {
			assert.Equal(t, "Auto created by packer-plugin-zstack", *driver.CreateImageParam.Params.Description)
		}
	})

	t.Run("CreateImageNoBackupStorage", func(t *testing.T) {
		config := &Config{
			ImageConfig:    ImageConfig{ImageName: "img-name"},
			InstanceConfig: InstanceConfig{RootVolumeUuid: "root-vol-1"},
		}
		driver := &MockDriver{}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Contains(t, errVal.(error).Error(), "backup storage UUID")
		assert.False(t, driver.CreateImageCalled)
		assert.False(t, driver.CreateVolumeSnapshotCalled)
		assert.False(t, driver.CreateImageFromSnapshotCalled)
	})

	t.Run("CreateImageError", func(t *testing.T) {
		expectedErr := errors.New("create image failed")
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img-name"},
			InstanceConfig:      InstanceConfig{RootVolumeUuid: "root-vol-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{
			CreateImageErr: expectedErr,
		}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Equal(t, expectedErr, errVal)
		assert.True(t, driver.CreateImageCalled)
		assert.False(t, driver.CreateVolumeSnapshotCalled)
		assert.False(t, driver.CreateImageFromSnapshotCalled)
	})
}
