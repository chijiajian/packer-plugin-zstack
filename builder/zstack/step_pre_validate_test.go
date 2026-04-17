package zstack

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

func TestStepPreValidate_Run(t *testing.T) {
	tests := []struct {
		name                      string
		config                    *Config
		driver                    *MockDriver
		expectedAction            multistep.StepAction
		expectedNetworkUUID       string
		expectedBackupStorageUUID string
		assertions                func(t *testing.T, driver *MockDriver)
	}{
		{
			name: "NetworkUuidPassthrough",
			config: &Config{
				NetworkConfig: NetworkConfig{L3NetworkUuid: "network-uuid"},
			},
			driver:              &MockDriver{},
			expectedAction:      multistep.ActionContinue,
			expectedNetworkUUID: "network-uuid",
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.False(t, driver.QueryL3NetworkCalled)
			},
		},
		{
			name: "NetworkNameQuery",
			config: &Config{
				NetworkConfig: NetworkConfig{L3NetworkName: "network-name"},
			},
			driver: &MockDriver{
				QueryL3NetworkResult: []view.L3NetworkInventoryView{{
					BaseInfoView: view.BaseInfoView{UUID: "network-uuid"},
				}},
			},
			expectedAction:      multistep.ActionContinue,
			expectedNetworkUUID: "network-uuid",
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.True(t, driver.QueryL3NetworkCalled)
				assert.Equal(t, "network-name", driver.QueryL3NetworkName)
			},
		},
		{
			name: "NetworkQueryError",
			config: &Config{
				NetworkConfig: NetworkConfig{L3NetworkName: "network-name"},
			},
			driver:         &MockDriver{QueryL3NetworkErr: errors.New("boom")},
			expectedAction: multistep.ActionHalt,
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.True(t, driver.QueryL3NetworkCalled)
			},
		},
		{
			name: "NetworkNotFound",
			config: &Config{
				NetworkConfig: NetworkConfig{L3NetworkName: "missing-network"},
			},
			driver:         &MockDriver{},
			expectedAction: multistep.ActionHalt,
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.True(t, driver.QueryL3NetworkCalled)
				assert.Equal(t, "missing-network", driver.QueryL3NetworkName)
			},
		},
		{
			name: "BackupStorageUuidPassthrough",
			config: &Config{
				NetworkConfig:       NetworkConfig{L3NetworkUuid: "network-uuid"},
				BackupStorageConfig: BackupStorageConfig{BackupStorageUuid: "backup-storage-uuid"},
			},
			driver:                    &MockDriver{},
			expectedAction:            multistep.ActionContinue,
			expectedNetworkUUID:       "network-uuid",
			expectedBackupStorageUUID: "backup-storage-uuid",
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.False(t, driver.QueryBackStorageCalled)
			},
		},
		{
			name: "BackupStorageNameQuery",
			config: &Config{
				NetworkConfig:       NetworkConfig{L3NetworkUuid: "network-uuid"},
				BackupStorageConfig: BackupStorageConfig{BackupStorageName: "backup-storage-name"},
			},
			driver: &MockDriver{
				QueryBackStorageResult: []view.BackupStorageInventoryView{{
					BaseInfoView: view.BaseInfoView{UUID: "backup-storage-uuid"},
				}},
			},
			expectedAction:            multistep.ActionContinue,
			expectedNetworkUUID:       "network-uuid",
			expectedBackupStorageUUID: "backup-storage-uuid",
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.True(t, driver.QueryBackStorageCalled)
				assert.Equal(t, "backup-storage-name", driver.QueryBackStorageName)
			},
		},
		{
			name: "BackupStorageOptionalSkip",
			config: &Config{
				NetworkConfig: NetworkConfig{L3NetworkUuid: "network-uuid"},
			},
			driver:              &MockDriver{},
			expectedAction:      multistep.ActionContinue,
			expectedNetworkUUID: "network-uuid",
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.False(t, driver.QueryBackStorageCalled)
			},
		},
		{
			name: "SourceImageUrlRequiresBackupStorage",
			config: &Config{
				NetworkConfig: NetworkConfig{L3NetworkUuid: "network-uuid"},
				ImageConfig:   ImageConfig{SourceImageUrl: "https://example.com/image.qcow2"},
			},
			driver:              &MockDriver{},
			expectedAction:      multistep.ActionHalt,
			expectedNetworkUUID: "network-uuid",
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.False(t, driver.QueryBackStorageCalled)
			},
		},
		{
			name: "BackupStorageQueryError",
			config: &Config{
				NetworkConfig:       NetworkConfig{L3NetworkUuid: "network-uuid"},
				BackupStorageConfig: BackupStorageConfig{BackupStorageName: "backup-storage-name"},
			},
			driver:              &MockDriver{QueryBackStorageErr: errors.New("boom")},
			expectedAction:      multistep.ActionHalt,
			expectedNetworkUUID: "network-uuid",
			assertions: func(t *testing.T, driver *MockDriver) {
				assert.True(t, driver.QueryBackStorageCalled)
				assert.Equal(t, "backup-storage-name", driver.QueryBackStorageName)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := testStateBag(test.config, test.driver)
			step := &StepPreValidate{}

			action := step.Run(context.Background(), state)

			assert.Equal(t, test.expectedAction, action)
			assert.Equal(t, test.expectedNetworkUUID, test.config.NetworkConfig.L3NetworkUuid)
			assert.Equal(t, test.expectedBackupStorageUUID, test.config.BackupStorageConfig.BackupStorageUuid)
			if test.assertions != nil {
				test.assertions(t, test.driver)
			}
		})
	}
}
