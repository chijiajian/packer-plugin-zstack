package zstack

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

type slowExportDriver struct {
	*MockDriver
	delay time.Duration
}

func (d *slowExportDriver) ExportImage(backupStorageUuid string, params param.ExportImageFromBackupStorageParam) (*view.ExportImageFromBackupStorageEventView, error) {
	time.Sleep(d.delay)
	d.ExportImageCalled = true
	d.ExportImageBackupStorageUuid = backupStorageUuid
	d.ExportImageParam = params
	return d.ExportImageResult, d.ExportImageErr
}

func TestStepExportImage_Run(t *testing.T) {
	t.Run("ExportImageSuccess", func(t *testing.T) {
		config := &Config{
			ImageConfig:         ImageConfig{ImageUuid: "img-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{ExportImageResult: &view.ExportImageFromBackupStorageEventView{ImageUrl: "backup://image.qcow2"}}
		state := testStateBag(config, driver)

		action := (&StepExportImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.ExportImageCalled)
		assert.Equal(t, "backup://image.qcow2", config.ImageUrl)
	})

	t.Run("ExportImageSkipNoBackupStorage", func(t *testing.T) {
		config := &Config{ImageConfig: ImageConfig{ImageUuid: "img-1"}}
		driver := &MockDriver{}
		state := testStateBag(config, driver)

		action := (&StepExportImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.False(t, driver.ExportImageCalled)
	})

	t.Run("ExportImageEmptyImageUuid", func(t *testing.T) {
		config := &Config{BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"}}
		driver := &MockDriver{}
		state := testStateBag(config, driver)

		action := (&StepExportImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		assert.False(t, driver.ExportImageCalled)
		_, ok := state.GetOk("error")
		assert.True(t, ok)
	})

	t.Run("ExportImageError", func(t *testing.T) {
		expectedErr := errors.New("export failed")
		config := &Config{
			ImageConfig:         ImageConfig{ImageUuid: "img-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{ExportImageErr: expectedErr}
		state := testStateBag(config, driver)

		action := (&StepExportImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		assert.True(t, driver.ExportImageCalled)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Equal(t, expectedErr, errVal)
	})

	t.Run("ExportImageUnsupportedBackupStorageTypeSkips", func(t *testing.T) {
		expectedErr := errors.New("No service deals with message: APIExportImageFromBackupStorageMsg")
		config := &Config{
			ImageConfig:         ImageConfig{ImageUuid: "img-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &MockDriver{ExportImageErr: expectedErr}
		state := testStateBag(config, driver)

		action := (&StepExportImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.ExportImageCalled)
		_, ok := state.GetOk("error")
		assert.False(t, ok)
	})

	t.Run("ContextCancelHalts", func(t *testing.T) {
		config := &Config{
			ImageConfig:         ImageConfig{ImageUuid: "img-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		// Make the SDK call block briefly so the context can cancel first.
		slow := &slowExportDriver{MockDriver: &MockDriver{
			ExportImageResult: &view.ExportImageFromBackupStorageEventView{ImageUrl: "backup://x.qcow2"},
		}, delay: 50 * time.Millisecond}
		state := testStateBag(config, slow)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		action := (&StepExportImage{}).Run(ctx, state)

		assert.Equal(t, multistep.ActionHalt, action)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.ErrorIs(t, errVal.(error), context.Canceled)
	})
}
