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

func TestStepInstanceOfferingValidate_Run(t *testing.T) {
	tests := []struct {
		name                 string
		config               *Config
		driver               *MockDriver
		expectedAction       multistep.StepAction
		expectedOfferingUUID string
		assertions           func(t *testing.T, driver *MockDriver)
	}{
		{
			name: "OfferingUuidPassthrough",
			config: &Config{
				InstanceConfig: InstanceConfig{InstanceOfferingUuid: "offering-uuid"},
			},
			driver:               &MockDriver{},
			expectedAction:       multistep.ActionContinue,
			expectedOfferingUUID: "offering-uuid",
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.False(t, driver.QueryInstanceOfferingCalled)
			},
		},
		{
			name: "OfferingNameQuery",
			config: &Config{
				InstanceConfig: InstanceConfig{InstanceOfferingName: "offering-name"},
			},
			driver: &MockDriver{
				QueryInstanceOfferingResult: []view.InstanceOfferingInventoryView{{
					BaseInfoView: view.BaseInfoView{UUID: "offering-uuid"},
				}},
			},
			expectedAction:       multistep.ActionContinue,
			expectedOfferingUUID: "offering-uuid",
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.True(t, driver.QueryInstanceOfferingCalled)
				assert.Equal(t, "offering-name", driver.QueryInstanceOfferingName)
			},
		},
		{
			name: "OfferingQueryError",
			config: &Config{
				InstanceConfig: InstanceConfig{InstanceOfferingName: "offering-name"},
			},
			driver:         &MockDriver{QueryInstanceOfferingErr: errors.New("boom")},
			expectedAction: multistep.ActionHalt,
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.True(t, driver.QueryInstanceOfferingCalled)
				assert.Equal(t, "offering-name", driver.QueryInstanceOfferingName)
			},
		},
		{
			name: "OfferingNotFound",
			config: &Config{
				InstanceConfig: InstanceConfig{InstanceOfferingName: "missing-offering"},
			},
			driver:         &MockDriver{},
			expectedAction: multistep.ActionHalt,
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.True(t, driver.QueryInstanceOfferingCalled)
				assert.Equal(t, "missing-offering", driver.QueryInstanceOfferingName)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := testStateBag(test.config, test.driver)
			step := &StepInstanceOfferingValidate{}

			action := step.Run(context.Background(), state)

			assert.Equal(t, test.expectedAction, action)
			assert.Equal(t, test.expectedOfferingUUID, test.config.InstanceConfig.InstanceOfferingUuid)
			if test.assertions != nil {
				test.assertions(t, test.driver)
			}
		})
	}
}
