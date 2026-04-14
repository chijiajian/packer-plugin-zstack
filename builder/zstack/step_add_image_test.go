package zstack

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

func TestStepAddImage_Run(t *testing.T) {
	newConfig := func() *Config {
		return &Config{
			ImageConfig: ImageConfig{
				SourceImage:      "source-image",
				SourceImageUrl:   "https://example.com/image.qcow2",
				GuestOsType:      "Linux",
				Format:           "qcow2",
				Platform:         "Linux",
				ImageDescription: "",
			},
			BackupStorageConfig: BackupStorageConfig{
				BackupStorageUuid: "backup-storage-uuid",
			},
		}
	}

	t.Run("AddImageSuccess", func(t *testing.T) {
		config := newConfig()
		driver := &MockDriver{AddImageResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "image-uuid"}}}
		state := testStateBag(config, driver)

		step := &StepAddImage{}
		action := step.Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.AddImageCalled)
		assert.Equal(t, "image-uuid", config.ImageUuid)
	})

	t.Run("AddImageWithDescription", func(t *testing.T) {
		config := newConfig()
		config.ImageDescription = "custom description"
		driver := &MockDriver{AddImageResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "image-uuid"}}}
		state := testStateBag(config, driver)

		step := &StepAddImage{}
		action := step.Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.AddImageCalled)
		assert.NotNil(t, driver.AddImageParam.Params.Description)
		assert.Equal(t, "custom description", *driver.AddImageParam.Params.Description)
	})

	t.Run("AddImageDefaultDescription", func(t *testing.T) {
		config := newConfig()
		driver := &MockDriver{AddImageResult: &view.ImageInventoryView{BaseInfoView: view.BaseInfoView{UUID: "image-uuid"}}}
		state := testStateBag(config, driver)

		step := &StepAddImage{}
		action := step.Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.AddImageCalled)
		assert.NotNil(t, driver.AddImageParam.Params.Description)
		assert.Equal(t, "Image added via Packer build process", *driver.AddImageParam.Params.Description)
	})

	t.Run("AddImageEmptySourceUrl", func(t *testing.T) {
		config := newConfig()
		config.SourceImageUrl = ""
		driver := &MockDriver{}
		state := testStateBag(config, driver)

		step := &StepAddImage{}
		action := step.Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		assert.False(t, driver.AddImageCalled)
	})

	t.Run("AddImageEmptySourceImage", func(t *testing.T) {
		config := newConfig()
		config.SourceImage = ""
		driver := &MockDriver{}
		state := testStateBag(config, driver)

		step := &StepAddImage{}
		action := step.Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		assert.False(t, driver.AddImageCalled)
	})

	t.Run("AddImageError", func(t *testing.T) {
		config := newConfig()
		driver := &MockDriver{AddImageErr: errors.New("add image failed")}
		state := testStateBag(config, driver)

		step := &StepAddImage{}
		action := step.Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		assert.True(t, driver.AddImageCalled)
		assert.Empty(t, config.ImageUuid)
	})
}
