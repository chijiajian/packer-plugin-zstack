// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func clearZStackEnvVars() {
	os.Unsetenv("ZSTACK_HOST")
	os.Unsetenv("ZSTACK_PORT")
	os.Unsetenv("ZSTACK_ACCOUNT_NAME")
	os.Unsetenv("ZSTACK_ACCOUNT_PASSWORD")
	os.Unsetenv("ZSTACK_ACCESS_KEY_ID")
	os.Unsetenv("ZSTACK_ACCESS_KEY_SECRET")
}

func TestAccessConfig_Prepare(t *testing.T) {
	originalHost := os.Getenv("ZSTACK_HOST")
	originalPort := os.Getenv("ZSTACK_PORT")
	originalAccountName := os.Getenv("ZSTACK_ACCOUNT_NAME")
	originalAccountPassword := os.Getenv("ZSTACK_ACCOUNT_PASSWORD")
	originalAccessKeyId := os.Getenv("ZSTACK_ACCESS_KEY_ID")
	originalAccessKeySecret := os.Getenv("ZSTACK_ACCESS_KEY_SECRET")

	defer func() {
		os.Setenv("ZSTACK_HOST", originalHost)
		os.Setenv("ZSTACK_PORT", originalPort)
		os.Setenv("ZSTACK_ACCOUNT_NAME", originalAccountName)
		os.Setenv("ZSTACK_ACCOUNT_PASSWORD", originalAccountPassword)
		os.Setenv("ZSTACK_ACCESS_KEY_ID", originalAccessKeyId)
		os.Setenv("ZSTACK_ACCESS_KEY_SECRET", originalAccessKeySecret)
	}()

	t.Run("CustomPortPreserved", func(t *testing.T) {
		clearZStackEnvVars()
		c := &AccessConfig{
			Host:            "example.com",
			Port:            9090,
			AccountName:     "admin",
			AccountPassword: "password",
		}
		errs := c.Prepare()
		assert.Empty(t, errs)
		assert.Equal(t, 9090, c.Port)
	})

	t.Run("DefaultPortWhenNotSet", func(t *testing.T) {
		clearZStackEnvVars()
		c := &AccessConfig{
			Host:            "example.com",
			AccountName:     "admin",
			AccountPassword: "password",
		}
		errs := c.Prepare()
		assert.Empty(t, errs)
		assert.Equal(t, 0, c.Port)
	})

	t.Run("MissingAuthReturnsError", func(t *testing.T) {
		clearZStackEnvVars()
		c := &AccessConfig{
			Host: "example.com",
		}
		errs := c.Prepare()
		assert.NotEmpty(t, errs)
		hasAuthErr := false
		for _, e := range errs {
			if e.Error() == "either account_name + account_password or access_key_id + access_key_secret is required" {
				hasAuthErr = true
			}
		}
		assert.True(t, hasAuthErr, "Should have auth error")
	})

	t.Run("AccessKeyEnvVars", func(t *testing.T) {
		clearZStackEnvVars()
		os.Setenv("ZSTACK_HOST", "key-host.com")
		os.Setenv("ZSTACK_ACCESS_KEY_ID", "myKeyId")
		os.Setenv("ZSTACK_ACCESS_KEY_SECRET", "myKeySecret")

		c := &AccessConfig{}
		errs := c.Prepare()
		assert.Empty(t, errs)
		assert.Equal(t, "key-host.com", c.Host)
		assert.Equal(t, "myKeyId", c.AccessKeyId)
		assert.Equal(t, "myKeySecret", c.AccessKeySecret)
	})

	t.Run("AccountEnvVars", func(t *testing.T) {
		clearZStackEnvVars()
		os.Setenv("ZSTACK_HOST", "acc-host.com")
		os.Setenv("ZSTACK_ACCOUNT_NAME", "envAccount")
		os.Setenv("ZSTACK_ACCOUNT_PASSWORD", "envPassword")

		c := &AccessConfig{}
		errs := c.Prepare()
		assert.Empty(t, errs)
		assert.Equal(t, "acc-host.com", c.Host)
		assert.Equal(t, "envAccount", c.AccountName)
		assert.Equal(t, "envPassword", c.AccountPassword)
	})

	t.Run("ConfigFromParameters", func(t *testing.T) {
		clearZStackEnvVars()
		c := &AccessConfig{
			Host:            "example.com",
			Port:            8888,
			AccountName:     "testAccount",
			AccountPassword: "testPassword",
		}
		errs := c.Prepare()
		assert.Empty(t, errs)
		assert.Equal(t, "example.com", c.Host)
		assert.Equal(t, 8888, c.Port)
		assert.Equal(t, "testAccount", c.AccountName)
		assert.Equal(t, "testPassword", c.AccountPassword)
	})

	t.Run("EnvironmentOverridesEmptyConfig", func(t *testing.T) {
		clearZStackEnvVars()
		os.Setenv("ZSTACK_HOST", "override-example.com")
		os.Setenv("ZSTACK_PORT", "7777")
		os.Setenv("ZSTACK_ACCOUNT_NAME", "envOverride")
		os.Setenv("ZSTACK_ACCOUNT_PASSWORD", "envPass")

		c := &AccessConfig{}
		errs := c.Prepare()
		assert.Empty(t, errs)
		assert.Equal(t, "override-example.com", c.Host)
		assert.Equal(t, 7777, c.Port)
	})

	t.Run("MissingHostAndAuth", func(t *testing.T) {
		clearZStackEnvVars()
		c := &AccessConfig{}
		errs := c.Prepare()
		assert.NotEmpty(t, errs)
		assert.Len(t, errs, 2)
	})

	t.Run("InvalidPortEnvReturnsError", func(t *testing.T) {
		clearZStackEnvVars()
		os.Setenv("ZSTACK_HOST", "example.com")
		os.Setenv("ZSTACK_PORT", "abc")
		os.Setenv("ZSTACK_ACCOUNT_NAME", "admin")
		os.Setenv("ZSTACK_ACCOUNT_PASSWORD", "password")
		defer os.Unsetenv("ZSTACK_PORT")

		c := &AccessConfig{}
		errs := c.Prepare()
		if assert.NotEmpty(t, errs) {
			var found bool
			for _, e := range errs {
				if strings.Contains(e.Error(), "ZSTACK_PORT") {
					found = true
				}
			}
			assert.True(t, found, "expected ZSTACK_PORT parse error")
		}
	})
}

func TestGetEnvOrDefault(t *testing.T) {
	originalVar := os.Getenv("TEST_ENV_VAR")
	defer os.Setenv("TEST_ENV_VAR", originalVar)

	os.Setenv("TEST_ENV_VAR", "env_value")
	result := getEnvOrDefault("TEST_ENV_VAR", "default_value")
	assert.Equal(t, "env_value", result)

	os.Unsetenv("TEST_ENV_VAR")
	result = getEnvOrDefault("TEST_ENV_VAR", "default_value")
	assert.Equal(t, "default_value", result)
}
