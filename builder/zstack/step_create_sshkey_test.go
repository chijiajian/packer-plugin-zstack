package zstack

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
)

func TestStepCreateSSHKey_Run(t *testing.T) {
	t.Run("PasswordAuthSkipsKeyGeneration", func(t *testing.T) {
		config := &Config{}
		state := testStateBag(config, &MockDriver{})

		step := &StepCreateSSHKey{Password: "mypass"}
		action := step.Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.Empty(t, config.Comm.SSHPrivateKey)
		assert.Empty(t, config.Comm.SSHPublicKey)
		assert.Empty(t, config.SshKey)
	})

	t.Run("ExistingPrivateKeyFileUsed", func(t *testing.T) {
		tmpFile, err := os.CreateTemp(t.TempDir(), "ssh-private-key-*.pem")
		assert.NoError(t, err)

		privateKey := []byte("-----BEGIN RSA PRIVATE KEY-----\ndummy\n-----END RSA PRIVATE KEY-----")
		_, err = tmpFile.Write(privateKey)
		assert.NoError(t, err)
		assert.NoError(t, tmpFile.Close())

		config := &Config{}
		config.Comm.SSHPrivateKeyFile = tmpFile.Name()
		state := testStateBag(config, &MockDriver{})

		step := &StepCreateSSHKey{}
		action := step.Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.Equal(t, privateKey, config.Comm.SSHPrivateKey)
	})

	t.Run("ExistingPrivateKeyFileNotFound", func(t *testing.T) {
		config := &Config{}
		config.Comm.SSHPrivateKeyFile = t.TempDir() + "/missing.pem"
		state := testStateBag(config, &MockDriver{})

		step := &StepCreateSSHKey{}
		action := step.Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		assert.NotNil(t, state.Get("error"))
	})

	t.Run("GenerateNewSSHKey", func(t *testing.T) {
		config := &Config{}
		state := testStateBag(config, &MockDriver{})

		step := &StepCreateSSHKey{}
		action := step.Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.NotEmpty(t, config.Comm.SSHPrivateKey)
		assert.NotEmpty(t, config.Comm.SSHPublicKey)
		assert.Equal(t, string(config.Comm.SSHPublicKey), config.SshKey)
	})

	t.Run("GeneratedKeyNoTrailingNewline", func(t *testing.T) {
		config := &Config{}
		state := testStateBag(config, &MockDriver{})

		step := &StepCreateSSHKey{}
		action := step.Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.NotEmpty(t, config.Comm.SSHPublicKey)
		assert.False(t, strings.HasSuffix(string(config.Comm.SSHPublicKey), "\n"))
	})
}

func TestStepCreateSSHKey_Cleanup(t *testing.T) {
	config := &Config{}
	state := testStateBag(config, &MockDriver{})

	step := &StepCreateSSHKey{}
	assert.NotPanics(t, func() {
		step.Cleanup(state)
	})
}
