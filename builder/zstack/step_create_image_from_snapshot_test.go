package zstack

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

func TestStepCreateImageFromSnapshot_Run(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		config := &Config{
			ImageConfig: ImageConfig{
				ImageName:                "img-from-snap",
				SourceVolumeSnapshotUuid: "snap-uuid-1",
				Platform:                 "Linux",
				GuestOsType:              "Ubuntu 22.04",
				Architecture:             "x86_64",
				ImageDescription:         "from snapshot",
			},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{
			GetVolumeSnapshotResult:       &view.VolumeSnapshotInventoryView{BaseInfoView: view.BaseInfoView{UUID: "snap-uuid-1"}, Status: "Ready", State: "Enabled"},
			CreateImageFromSnapshotResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "img-uuid-1"}},
		}
		state := testStateBag(config, driver)

		action := (&StepCreateImageFromSnapshot{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.GetVolumeSnapshotCalled)
		assert.Equal(t, "snap-uuid-1", driver.GetVolumeSnapshotUuid)
		assert.True(t, driver.CreateImageFromSnapshotCalled)
		assert.Equal(t, "snap-uuid-1", driver.CreateImageFromSnapshotSnapshotUuid)
		assert.Equal(t, []string{"bs-1"}, driver.CreateImageFromSnapshotParam.Params.BackupStorageUuids)
		assert.Equal(t, "img-from-snap", driver.CreateImageFromSnapshotParam.Params.Name)
		if assert.NotNil(t, driver.CreateImageFromSnapshotParam.Params.Platform) {
			assert.Equal(t, "Linux", *driver.CreateImageFromSnapshotParam.Params.Platform)
		}
		if assert.NotNil(t, driver.CreateImageFromSnapshotParam.Params.GuestOsType) {
			assert.Equal(t, "Ubuntu 22.04", *driver.CreateImageFromSnapshotParam.Params.GuestOsType)
		}
		if assert.NotNil(t, driver.CreateImageFromSnapshotParam.Params.Architecture) {
			assert.Equal(t, "x86_64", *driver.CreateImageFromSnapshotParam.Params.Architecture)
		}
		if assert.NotNil(t, driver.CreateImageFromSnapshotParam.Params.Description) {
			assert.Equal(t, "from snapshot", *driver.CreateImageFromSnapshotParam.Params.Description)
		}
		assert.Equal(t, "img-uuid-1", config.ImageUuid)
	})

	t.Run("MissingSnapshotUuid", func(t *testing.T) {
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{}
		state := testStateBag(config, driver)

		action := (&StepCreateImageFromSnapshot{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		_, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.False(t, driver.CreateImageFromSnapshotCalled)
	})

	t.Run("MissingBackupStorage", func(t *testing.T) {
		config := &Config{
			ImageConfig: ImageConfig{ImageName: "img", SourceVolumeSnapshotUuid: "snap-1"},
		}
		driver := &MockDriver{}
		state := testStateBag(config, driver)

		action := (&StepCreateImageFromSnapshot{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		_, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.False(t, driver.CreateImageFromSnapshotCalled)
	})

	t.Run("GetSnapshotError", func(t *testing.T) {
		expectedErr := errors.New("snapshot not found")
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img", SourceVolumeSnapshotUuid: "snap-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{GetVolumeSnapshotErr: expectedErr}
		state := testStateBag(config, driver)

		action := (&StepCreateImageFromSnapshot{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Equal(t, expectedErr, errVal)
		assert.False(t, driver.CreateImageFromSnapshotCalled)
	})

	t.Run("CreateImageError", func(t *testing.T) {
		expectedErr := errors.New("create failed")
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img", SourceVolumeSnapshotUuid: "snap-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{
			GetVolumeSnapshotResult:    &view.VolumeSnapshotInventoryView{BaseInfoView: view.BaseInfoView{UUID: "snap-1"}, Status: "Ready"},
			CreateImageFromSnapshotErr: expectedErr,
		}
		state := testStateBag(config, driver)

		action := (&StepCreateImageFromSnapshot{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Equal(t, expectedErr, errVal)
	})

	t.Run("DefaultDescription", func(t *testing.T) {
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img", SourceVolumeSnapshotUuid: "snap-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{
			GetVolumeSnapshotResult:       &view.VolumeSnapshotInventoryView{BaseInfoView: view.BaseInfoView{UUID: "snap-1"}, Status: "Ready"},
			CreateImageFromSnapshotResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "img-uuid-1"}},
		}
		state := testStateBag(config, driver)

		action := (&StepCreateImageFromSnapshot{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		if assert.NotNil(t, driver.CreateImageFromSnapshotParam.Params.Description) {
			assert.Equal(t, "Auto created by packer-plugin-zstack from volume snapshot", *driver.CreateImageFromSnapshotParam.Params.Description)
		}
	})
}

func TestConfigPrepare_SnapshotMode(t *testing.T) {
	t.Run("MissingImageName", func(t *testing.T) {
		c := &Config{
			AccessConfig: AccessConfig{Host: "h", AccountName: "a", AccountPassword: "p"},
			ImageConfig:  ImageConfig{SourceVolumeSnapshotUuid: "snap-1"},
		}
		err := c.Prepare()
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "image_name")
		}
	})

	t.Run("MissingBackupStorage", func(t *testing.T) {
		c := &Config{
			AccessConfig: AccessConfig{Host: "h", AccountName: "a", AccountPassword: "p"},
			ImageConfig:  ImageConfig{SourceVolumeSnapshotUuid: "snap-1", ImageName: "img"},
		}
		err := c.Prepare()
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "backup_storage")
		}
	})

	t.Run("Valid", func(t *testing.T) {
		c := &Config{
			AccessConfig:        AccessConfig{Host: "h", AccountName: "a", AccountPassword: "p"},
			ImageConfig:         ImageConfig{SourceVolumeSnapshotUuid: "snap-1", ImageName: "img"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageName: "bs"},
		}
		err := c.Prepare()
		assert.NoError(t, err)
		assert.Equal(t, "Linux", c.Platform)
	})
}
