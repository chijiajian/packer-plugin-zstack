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
	t.Run("CreateSnapshotThenImageSuccess", func(t *testing.T) {
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img-name"},
			InstanceConfig:      InstanceConfig{RootVolumeUuid: "root-vol-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{
			CreateVolumeSnapshotResult:    &view.VolumeSnapshotInventoryView{BaseInfoView: view.BaseInfoView{UUID: "snap-uuid-1"}, Status: "Ready", State: "Enabled"},
			CreateImageFromSnapshotResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "img-uuid-1"}},
			GetVolumeSnapshotResult:       &view.VolumeSnapshotInventoryView{BaseInfoView: view.BaseInfoView{UUID: "snap-uuid-1"}, Status: "Ready", State: "Enabled"},
		}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.CreateVolumeSnapshotCalled)
		assert.Equal(t, "root-vol-1", driver.CreateVolumeSnapshotVolumeUuid)
		if assert.NotNil(t, driver.CreateVolumeSnapshotParam.Params.Description) {
			assert.Equal(t, "Auto created by packer-plugin-zstack before image creation", *driver.CreateVolumeSnapshotParam.Params.Description)
		}
		assert.True(t, driver.CreateImageFromSnapshotCalled)
		assert.Equal(t, "snap-uuid-1", driver.CreateImageFromSnapshotSnapshotUuid)
		assert.Equal(t, []string{"bs-1"}, driver.CreateImageFromSnapshotParam.Params.BackupStorageUuids)
		assert.Equal(t, "img-uuid-1", config.ImageUuid)
	})

	t.Run("CreateSnapshotError", func(t *testing.T) {
		expectedErr := errors.New("create snapshot failed")
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img-name"},
			InstanceConfig:      InstanceConfig{RootVolumeUuid: "root-vol-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{CreateVolumeSnapshotErr: expectedErr}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Equal(t, expectedErr, errVal)
		assert.False(t, driver.CreateImageFromSnapshotCalled)
	})

	t.Run("WaitSnapshotReadyError", func(t *testing.T) {
		expectedErr := errors.New("get snapshot failed")
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img-name"},
			InstanceConfig:      InstanceConfig{RootVolumeUuid: "root-vol-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{
			CreateVolumeSnapshotResult: &view.VolumeSnapshotInventoryView{BaseInfoView: view.BaseInfoView{UUID: "snap-uuid-1"}},
			GetVolumeSnapshotErr:       expectedErr,
		}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Equal(t, expectedErr, errVal)
		assert.False(t, driver.CreateImageFromSnapshotCalled)
	})

	t.Run("CreateImageSuccess", func(t *testing.T) {
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img-name"},
			InstanceConfig:      InstanceConfig{RootVolumeUuid: "root-vol-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{
			CreateVolumeSnapshotResult:    &view.VolumeSnapshotInventoryView{BaseInfoView: view.BaseInfoView{UUID: "snap-uuid-1"}, Status: "Ready", State: "Enabled"},
			CreateImageFromSnapshotResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "img-uuid-1"}},
			GetVolumeSnapshotResult:       &view.VolumeSnapshotInventoryView{BaseInfoView: view.BaseInfoView{UUID: "snap-uuid-1"}, Status: "Ready", State: "Enabled"},
		}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.CreateVolumeSnapshotCalled)
		assert.True(t, driver.CreateImageFromSnapshotCalled)
		assert.Equal(t, "img-uuid-1", config.ImageUuid)
		assert.Equal(t, "root-vol-1", driver.CreateVolumeSnapshotVolumeUuid)
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
		assert.False(t, driver.CreateVolumeSnapshotCalled)
		assert.False(t, driver.CreateImageFromSnapshotCalled)
	})

	t.Run("CreateImageWithDescription", func(t *testing.T) {
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img-name", ImageDescription: "custom desc"},
			InstanceConfig:      InstanceConfig{RootVolumeUuid: "root-vol-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{
			CreateVolumeSnapshotResult:    &view.VolumeSnapshotInventoryView{BaseInfoView: view.BaseInfoView{UUID: "snap-uuid-1"}, Status: "Ready", State: "Enabled"},
			CreateImageFromSnapshotResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "img-uuid-1"}},
			GetVolumeSnapshotResult:       &view.VolumeSnapshotInventoryView{BaseInfoView: view.BaseInfoView{UUID: "snap-uuid-1"}, Status: "Ready", State: "Enabled"},
		}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		if assert.NotNil(t, driver.CreateImageFromSnapshotParam.Params.Description) {
			assert.Equal(t, "custom desc", *driver.CreateImageFromSnapshotParam.Params.Description)
		}
	})

	t.Run("CreateImageDefaultDescription", func(t *testing.T) {
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img-name"},
			InstanceConfig:      InstanceConfig{RootVolumeUuid: "root-vol-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{
			CreateVolumeSnapshotResult:    &view.VolumeSnapshotInventoryView{BaseInfoView: view.BaseInfoView{UUID: "snap-uuid-1"}, Status: "Ready", State: "Enabled"},
			CreateImageFromSnapshotResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "img-uuid-1"}},
			GetVolumeSnapshotResult:       &view.VolumeSnapshotInventoryView{BaseInfoView: view.BaseInfoView{UUID: "snap-uuid-1"}, Status: "Ready", State: "Enabled"},
		}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		if assert.NotNil(t, driver.CreateImageFromSnapshotParam.Params.Description) {
			assert.Equal(t, "Auto created by packer-plugin-zstack", *driver.CreateImageFromSnapshotParam.Params.Description)
		}
	})

	t.Run("CreateImageError", func(t *testing.T) {
		expectedErr := errors.New("create image failed")
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img-name"},
			InstanceConfig:      InstanceConfig{RootVolumeUuid: "root-vol-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{
			CreateVolumeSnapshotResult: &view.VolumeSnapshotInventoryView{BaseInfoView: view.BaseInfoView{UUID: "snap-uuid-1"}, Status: "Ready", State: "Enabled"},
			CreateImageFromSnapshotErr: expectedErr,
			GetVolumeSnapshotResult:    &view.VolumeSnapshotInventoryView{BaseInfoView: view.BaseInfoView{UUID: "snap-uuid-1"}, Status: "Ready", State: "Enabled"},
		}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Equal(t, expectedErr, errVal)
	})
}
