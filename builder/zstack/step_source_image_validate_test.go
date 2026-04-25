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

func TestStepSourceImageValidate_Run(t *testing.T) {
	tests := []struct {
		name              string
		config            *Config
		driver            *MockDriver
		expectedAction    multistep.StepAction
		expectedImageUUID string
		assertions        func(t *testing.T, driver *MockDriver)
	}{
		{
			name: "ImageUuidPassthrough",
			config: &Config{
				ImageConfig: ImageConfig{ImageUuid: "image-uuid", SourceImage: "ignored-image-name"},
			},
			driver:            &MockDriver{},
			expectedAction:    multistep.ActionContinue,
			expectedImageUUID: "image-uuid",
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.False(t, driver.QueryImageCalled)
			},
		},
		{
			name: "ImageNameQuery",
			config: &Config{
				ImageConfig: ImageConfig{SourceImage: "source-image-name"},
			},
			driver: &MockDriver{
				QueryImageResult: []view.ImageInventoryView{{
					BaseInfoView: view.BaseInfoView{UUID: "image-uuid"},
					Status:       "Ready",
					State:        "Enabled",
				}},
			},
			expectedAction:    multistep.ActionContinue,
			expectedImageUUID: "image-uuid",
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.True(t, driver.QueryImageCalled)
				assert.Equal(t, "source-image-name", driver.QueryImageName)
			},
		},
		{
			name: "ImageNotReady",
			config: &Config{
				ImageConfig: ImageConfig{SourceImage: "source-image-name"},
			},
			driver: &MockDriver{
				QueryImageResult: []view.ImageInventoryView{{
					BaseInfoView: view.BaseInfoView{UUID: "image-uuid"},
					Status:       "Downloading",
					State:        "Enabled",
				}},
			},
			expectedAction: multistep.ActionHalt,
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.True(t, driver.QueryImageCalled)
			},
		},
		{
			name: "ImageQueryError",
			config: &Config{
				ImageConfig: ImageConfig{SourceImage: "source-image-name"},
			},
			driver:         &MockDriver{QueryImageErr: errors.New("boom")},
			expectedAction: multistep.ActionHalt,
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.True(t, driver.QueryImageCalled)
				assert.Equal(t, "source-image-name", driver.QueryImageName)
			},
		},
		{
			name: "ImageNotFound",
			config: &Config{
				ImageConfig: ImageConfig{SourceImage: "missing-image"},
			},
			driver:         &MockDriver{},
			expectedAction: multistep.ActionHalt,
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.True(t, driver.QueryImageCalled)
				assert.Equal(t, "missing-image", driver.QueryImageName)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := testStateBag(test.config, test.driver)
			step := &StepSourceImageValidate{}

			action := step.Run(context.Background(), state)

			assert.Equal(t, test.expectedAction, action)
			assert.Equal(t, test.expectedImageUUID, test.config.ImageConfig.ImageUuid)
			if test.assertions != nil {
				test.assertions(t, test.driver)
			}
		})
	}
}
