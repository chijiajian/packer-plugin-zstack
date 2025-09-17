package zstack

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccessConfig_Prepare(t *testing.T) {
	// Save original environment variables to restore after testing
	originalHost := os.Getenv("ZSTACK_HOST")
	originalPort := os.Getenv("ZSTACK_PORT")
	originalAccountName := os.Getenv("ZSTACK_ACCOUNTNAME")
	originalAccountPassword := os.Getenv("ZSTACK_ACCOUNTPASSWORD")
	originalAccessKeyId := os.Getenv("ZSTACK_ACCESSKEYID")
	originalAccessKeySecret := os.Getenv("ZSTACK_ACCESSKEYSECRET")

	// Restore environment variables after test
	defer func() {
		os.Setenv("ZSTACK_HOST", originalHost)
		os.Setenv("ZSTACK_PORT", originalPort)
		os.Setenv("ZSTACK_ACCOUNTNAME", originalAccountName)
		os.Setenv("ZSTACK_ACCOUNTPASSWORD", originalAccountPassword)
		os.Setenv("ZSTACK_ACCESSKEYID", originalAccessKeyId)
		os.Setenv("ZSTACK_ACCESSKEYSECRET", originalAccessKeySecret)
	}()

	// Test case 1: Configuration through parameters
	t.Run("ConfigFromParameters", func(t *testing.T) {
		// Clear environment variables to ensure test uses parameter values
		os.Unsetenv("ZSTACK_HOST")
		os.Unsetenv("ZSTACK_PORT")
		os.Unsetenv("ZSTACK_ACCOUNTNAME")
		os.Unsetenv("ZSTACK_ACCOUNTPASSWORD")
		os.Unsetenv("ZSTACK_ACCESSKEYID")
		os.Unsetenv("ZSTACK_ACCESSKEYSECRET")

		c := &AccessConfig{
			Host:            "example.com",
			Port:            8888,
			AccountName:     "testAccount",
			AccountPassword: "testPassword",
		}

		errs := c.Prepare()
		assert.Empty(t, errs, "Should have no errors")
		assert.Equal(t, "example.com", c.Host)
		assert.Equal(t, 8888, c.Port)
		assert.Equal(t, "testAccount", c.AccountName)
		assert.Equal(t, "testPassword", c.AccountPassword)
	})

	// Test case 2: Configuration through environment variables
	t.Run("ConfigFromEnvironment", func(t *testing.T) {
		os.Setenv("ZSTACK_HOST", "env-example.com")
		os.Setenv("ZSTACK_PORT", "9999")
		os.Setenv("ZSTACK_ACCOUNTNAME", "envAccount")
		os.Setenv("ZSTACK_ACCOUNTPASSWORD", "envPassword")

		c := &AccessConfig{}

		errs := c.Prepare()
		assert.Empty(t, errs, "Should have no errors")
		assert.Equal(t, "env-example.com", c.Host)
		assert.Equal(t, 9999, c.Port)
		assert.Equal(t, "envAccount", c.AccountName)
		assert.Equal(t, "envPassword", c.AccountPassword)
	})

	// Test case 3: Environment variables take precedence over configuration parameters
	t.Run("EnvironmentOverridesConfig", func(t *testing.T) {
		os.Setenv("ZSTACK_HOST", "override-example.com")
		os.Setenv("ZSTACK_PORT", "7777")

		c := &AccessConfig{
			Host:            "original.com",
			Port:            1234,
			AccountName:     "originalAccount",
			AccountPassword: "originalPassword",
		}

		errs := c.Prepare()
		assert.Empty(t, errs, "Should have no errors")
		assert.Equal(t, "override-example.com", c.Host)
		assert.Equal(t, 7777, c.Port)
		assert.Equal(t, "originalAccount", c.AccountName)
		assert.Equal(t, "originalPassword", c.AccountPassword)
	})

	// Test case 4: Missing required parameters
	t.Run("MissingRequiredFields", func(t *testing.T) {
		os.Unsetenv("ZSTACK_HOST")
		os.Unsetenv("ZSTACK_ACCOUNTNAME")
		os.Unsetenv("ZSTACK_ACCOUNTPASSWORD")
		os.Unsetenv("ZSTACK_ACCESSKEYID")
		os.Unsetenv("ZSTACK_ACCESSKEYSECRET")

		c := &AccessConfig{}

		errs := c.Prepare()
		assert.NotEmpty(t, errs, "Should have errors")
		assert.Len(t, errs, 2, "Should have two errors")
		assert.Contains(t, errs[0].Error(), "host is required")
		assert.Contains(t, errs[1].Error(), "either accountname + accountpassword or accesskeyid + accesskeysecret is required")
	})

	// Test case 5: Using access keys instead of account credentials
	t.Run("UsingAccessKeys", func(t *testing.T) {
		os.Unsetenv("ZSTACK_ACCOUNTNAME")
		os.Unsetenv("ZSTACK_ACCOUNTPASSWORD")
		os.Setenv("ZSTACK_HOST", "key-example.com")
		os.Setenv("ZSTACK_ACCESSKEYID", "testKeyId")
		os.Setenv("ZSTACK_ACCESSKEYSECRET", "testKeySecret")

		c := &AccessConfig{}

		errs := c.Prepare()
		assert.Empty(t, errs, "Should have no errors")
		assert.Equal(t, "key-example.com", c.Host)
		assert.Equal(t, "testKeyId", c.AccessKeyId)
		assert.Equal(t, "testKeySecret", c.AccessKeySecret)
		assert.Empty(t, c.AccountName)
		assert.Empty(t, c.AccountPassword)
	})
}

func TestGetEnvOrDefault(t *testing.T) {
	// Save original environment variable
	originalVar := os.Getenv("TEST_ENV_VAR")
	defer os.Setenv("TEST_ENV_VAR", originalVar)

	// Test when environment variable exists
	os.Setenv("TEST_ENV_VAR", "env_value")
	result := getEnvOrDefault("TEST_ENV_VAR", "default_value")
	assert.Equal(t, "env_value", result)

	// Test when environment variable doesn't exist
	os.Unsetenv("TEST_ENV_VAR")
	result = getEnvOrDefault("TEST_ENV_VAR", "default_value")
	assert.Equal(t, "default_value", result)
}
