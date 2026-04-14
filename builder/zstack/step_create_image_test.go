package zstack

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

type createImageCaptureDriver struct {
	*MockDriver
	RootVolumeUuid string
	Params         param.CreateRootVolumeTemplateFromRootVolumeParam
}

func (d *createImageCaptureDriver) CreateImage(rootVolumeUuid string, params param.CreateRootVolumeTemplateFromRootVolumeParam) (*view.ImageInventoryView, error) {
	d.CreateImageCalled = true
	d.RootVolumeUuid = rootVolumeUuid
	d.Params = params
	return d.CreateImageResult, d.CreateImageErr
}

func TestStepCreateImage_Run(t *testing.T) {
	t.Run("CreateImageSuccess", func(t *testing.T) {
		config := &Config{
			ImageConfig:         ImageConfig{ImageName: "img-name"},
			InstanceConfig:      InstanceConfig{RootVolumeUuid: "root-vol-1"},
			BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "bs-1"},
		}
		driver := &createImageCaptureDriver{MockDriver: &MockDriver{CreateImageResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "img-uuid-1"}}}}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.CreateImageCalled)
		assert.Equal(t, "img-uuid-1", config.ImageUuid)
		assert.Equal(t, "root-vol-1", driver.RootVolumeUuid)
	})

	t.Run("CreateImageNoBackupStorage", func(t *testing.T) {
		config := &Config{
			ImageConfig:    ImageConfig{ImageName: "img-name"},
			InstanceConfig: InstanceConfig{RootVolumeUuid: "root-vol-1"},
		}
		driver := &createImageCaptureDriver{MockDriver: &MockDriver{CreateImageResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "img-uuid-1"}}}}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.CreateImageCalled)
		assert.Nil(t, driver.Params.Params.BackupStorageUuids)
	})

	t.Run("CreateImageWithDescription", func(t *testing.T) {
		config := &Config{
			ImageConfig:    ImageConfig{ImageName: "img-name", ImageDescription: "custom desc"},
			InstanceConfig: InstanceConfig{RootVolumeUuid: "root-vol-1"},
		}
		driver := &createImageCaptureDriver{MockDriver: &MockDriver{CreateImageResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "img-uuid-1"}}}}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		if assert.NotNil(t, driver.Params.Params.Description) {
			assert.Equal(t, "custom desc", *driver.Params.Params.Description)
		}
	})

	t.Run("CreateImageDefaultDescription", func(t *testing.T) {
		config := &Config{
			ImageConfig:    ImageConfig{ImageName: "img-name"},
			InstanceConfig: InstanceConfig{RootVolumeUuid: "root-vol-1"},
		}
		driver := &createImageCaptureDriver{MockDriver: &MockDriver{CreateImageResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "img-uuid-1"}}}}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		if assert.NotNil(t, driver.Params.Params.Description) {
			assert.Equal(t, "Auto created by packer-plugin-zstack", *driver.Params.Params.Description)
		}
	})

	t.Run("CreateImageError", func(t *testing.T) {
		expectedErr := errors.New("create image failed")
		config := &Config{
			ImageConfig:    ImageConfig{ImageName: "img-name"},
			InstanceConfig: InstanceConfig{RootVolumeUuid: "root-vol-1"},
		}
		driver := &createImageCaptureDriver{MockDriver: &MockDriver{CreateImageErr: expectedErr}}
		state := testStateBag(config, driver)

		action := (&StepCreateImage{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Equal(t, expectedErr, errVal)
	})
}
