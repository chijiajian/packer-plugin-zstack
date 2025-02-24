package zstack

import (
	"os"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
	"github.com/stretchr/testify/assert"
)

func TestAccessConfig_Prepare(t *testing.T) {
	tests := []struct {
		name          string
		config        AccessConfig
		environ       map[string]string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid_access_key_config",
			config: AccessConfig{
				Host:            "",
				Port:            8080,
				AccessKeyId:     "",
				AccessKeySecret: "",
			},
			expectError: false,
		},
		{
			name: "valid_account_config",
			config: AccessConfig{
				Host:            "",
				Port:            8080,
				AccountName:     "admin",
				AccountPassword: "password",
			},
			expectError: false,
		},
		{
			name: "missing_host",
			config: AccessConfig{
				AccessKeyId:     "",
				AccessKeySecret: "",
			},
			expectError:   true,
			errorContains: "host is required",
		},
		{
			name: "missing_credentials",
			config: AccessConfig{
				Host: "192.168.1.100",
				Port: 8080,
			},
			expectError:   true,
			errorContains: "either accountname + accountpassword or accesskeyid + accesskeysecret is required",
		},
		{
			name: "from_environment",
			config: AccessConfig{
				Port: 8080,
			},
			environ: map[string]string{
				"ZSTACK_HOST":            "192.168.1.100",
				"ZSTACK_ACCESSKEYID":     "env-key",
				"ZSTACK_ACCESSKEYSECRET": "env-secret",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for the test
			for k, v := range tt.environ {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Create a copy of the config
			c := tt.config

			// Initialize interpolate context
			c.ctx = interpolate.Context{}

			// Run Prepare
			errs := c.Prepare()

			// Check for expected errors
			if tt.expectError {
				assert.NotNil(t, errs, "Expected errors but got none")
				if tt.errorContains != "" {
					found := false
					for _, err := range errs {
						if err != nil && assert.Contains(t, err.Error(), tt.errorContains) {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error containing '%s'", tt.errorContains)
				}
			} else {
				assert.Nil(t, errs, "Expected no errors but got: %v", errs)
			}

			// Additional checks for environment variables
			if tt.environ != nil {
				if envHost := tt.environ["ZSTACK_HOST"]; envHost != "" {
					assert.Equal(t, envHost, c.Host, "Host should be set from environment")
				}
				if envKeyId := tt.environ["ZSTACK_ACCESSKEYID"]; envKeyId != "" {
					assert.Equal(t, envKeyId, c.AccessKeyId, "AccessKeyId should be set from environment")
				}
			}
		})
	}
}

func TestAccessConfig_CreateClient(t *testing.T) {
	tests := []struct {
		name        string
		config      AccessConfig
		expectError bool
	}{
		{
			name: "valid_access_key_config",
			config: AccessConfig{
				Host:            "192.168.1.100",
				Port:            8080,
				AccessKeyId:     "test-key",
				AccessKeySecret: "test-secret",
			},
			expectError: false,
		},
		{
			name: "valid_account_config",
			config: AccessConfig{
				Host:            "192.168.1.100",
				Port:            8080,
				AccountName:     "admin",
				AccountPassword: "password",
			},
			expectError: false,
		},
		{
			name: "invalid_config",
			config: AccessConfig{
				Host: "192.168.1.100",
				Port: 8080,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := tt.config.CreateClient()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestAccessConfig_Driver(t *testing.T) {
	config := AccessConfig{
		Host:            "192.168.1.100",
		Port:            8080,
		AccessKeyId:     "test-key",
		AccessKeySecret: "test-secret",
	}

	driver, err := config.Driver()
	assert.NoError(t, err)
	assert.NotNil(t, driver)
	assert.NotNil(t, driver.client)
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		envVar       string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "env_var_exists",
			envVar:       "TEST_VAR",
			envValue:     "test_value",
			defaultValue: "default",
			expected:     "test_value",
		},
		{
			name:         "env_var_not_exists",
			envVar:       "NON_EXISTENT_VAR",
			defaultValue: "default",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envVar, tt.envValue)
				defer os.Unsetenv(tt.envVar)
			}

			result := getEnvOrDefault(tt.envVar, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}

}
