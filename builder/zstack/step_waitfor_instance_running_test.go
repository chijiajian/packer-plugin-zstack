package zstack

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

func TestStepWaitForRunning_Run(t *testing.T) {
	t.Run("EmptyInstanceUuidHalts", func(t *testing.T) {
		config := &Config{}
		state := testStateBag(config, &MockDriver{})

		action := (&StepWaitForRunning{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		_, ok := state.GetOk("error")
		assert.True(t, ok)
	})

	t.Run("SuccessWhenRunning", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		config.pollInterval = 5 * time.Millisecond
		driver := &MockDriver{GetVmInstanceResult: &view.VmInstanceInventoryView{
			BaseInfoView: view.BaseInfoView{UUID: "vm-1"},
			State:        "Running",
		}}
		state := testStateBag(config, driver)

		action := (&StepWaitForRunning{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.GetVmInstanceCalled)
	})

	t.Run("GetVmInstanceErrorHalts", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		config.pollInterval = 5 * time.Millisecond
		driver := &MockDriver{GetVmInstanceErr: errors.New("vm query failed")}
		state := testStateBag(config, driver)

		action := (&StepWaitForRunning{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Contains(t, errVal.(error).Error(), "vm query failed")
	})

	t.Run("ContextCancelHalts", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		config.pollInterval = time.Hour
		driver := &MockDriver{GetVmInstanceResult: &view.VmInstanceInventoryView{
			BaseInfoView: view.BaseInfoView{UUID: "vm-1"},
			State:        "Starting",
		}}
		state := testStateBag(config, driver)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		action := (&StepWaitForRunning{}).Run(ctx, state)
		assert.Equal(t, multistep.ActionHalt, action)
	})

	t.Run("TimeoutHalts", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		config.vmRunningTimeout = 10 * time.Millisecond
		config.pollInterval = time.Hour
		driver := &MockDriver{GetVmInstanceResult: &view.VmInstanceInventoryView{
			State: "Starting",
		}}
		state := testStateBag(config, driver)

		action := (&StepWaitForRunning{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Contains(t, errVal.(error).Error(), "timeout")
	})
}

func TestStepWaitForRunning_Cleanup_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		(&StepWaitForRunning{}).Cleanup(testStateBag(&Config{}, &MockDriver{}))
	})
}
