// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
)

func TestStepExpungeVmInstance_Run(t *testing.T) {
	t.Run("EmptyUuidSkips", func(t *testing.T) {
		config := &Config{}
		driver := &MockDriver{}
		state := testStateBag(config, driver)

		action := (&StepExpungeVmInstance{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.False(t, driver.DestroyVmInstanceCalled)
		assert.False(t, driver.DeleteVmInstanceCalled)
	})

	t.Run("SuccessDestroysAndDeletes", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		driver := &MockDriver{}
		state := testStateBag(config, driver)

		action := (&StepExpungeVmInstance{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.DestroyVmInstanceCalled)
		assert.Equal(t, "vm-1", driver.DestroyVmInstanceUuid)
		assert.True(t, driver.DeleteVmInstanceCalled)
		assert.Equal(t, "vm-1", driver.DeleteVmInstanceUuid)
		assert.Empty(t, config.InstanceUuid)
	})

	t.Run("DestroyErrorWarnsAndContinues", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		driver := &MockDriver{DestroyVmInstanceErr: errors.New("destroy fail")}
		state := testStateBag(config, driver)

		action := (&StepExpungeVmInstance{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.DestroyVmInstanceCalled)
		assert.False(t, driver.DeleteVmInstanceCalled)
		_, ok := state.GetOk("error")
		assert.False(t, ok, "expunge cleanup failures must not poison build state")
	})

	t.Run("DeleteErrorWarnsAndContinues", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		driver := &MockDriver{DeleteVmInstanceErr: errors.New("delete fail")}
		state := testStateBag(config, driver)

		action := (&StepExpungeVmInstance{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.DestroyVmInstanceCalled)
		assert.True(t, driver.DeleteVmInstanceCalled)
		_, ok := state.GetOk("error")
		assert.False(t, ok, "expunge cleanup failures must not poison build state")
	})
}

func TestStepExpungeVmInstance_Cleanup_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		(&StepExpungeVmInstance{}).Cleanup(testStateBag(&Config{}, &MockDriver{}))
	})
}
