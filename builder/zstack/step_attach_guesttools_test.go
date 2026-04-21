package zstack

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/view"
)

func TestStepAttachGuestTools_Run(t *testing.T) {
	t.Run("EmptyUuidHalts", func(t *testing.T) {
		config := &Config{}
		driver := &MockDriver{}
		state := testStateBag(config, driver)

		action := (&StepAttachGuestTools{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		assert.False(t, driver.GetVmInstanceCalled)
		assert.False(t, driver.AttachGuestToolsCalled)
	})

	t.Run("GetVmErrorHalts", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		driver := &MockDriver{GetVmInstanceErr: errors.New("get vm failed")}
		state := testStateBag(config, driver)

		action := (&StepAttachGuestTools{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		assert.True(t, driver.GetVmInstanceCalled)
		assert.False(t, driver.AttachGuestToolsCalled)
		_, ok := state.GetOk("error")
		assert.True(t, ok)
	})

	t.Run("SuccessAttaches", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		driver := &MockDriver{GetVmInstanceResult: &view.VmInstanceInventoryView{
			BaseInfoView: view.BaseInfoView{UUID: "vm-1", Name: "vm-name"},
		}}
		state := testStateBag(config, driver)

		action := (&StepAttachGuestTools{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.AttachGuestToolsCalled)
		assert.Equal(t, "vm-1", driver.AttachGuestToolsVmUuid)
	})

	t.Run("AttachErrorIsNonFatal", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		driver := &MockDriver{
			GetVmInstanceResult: &view.VmInstanceInventoryView{
				BaseInfoView: view.BaseInfoView{UUID: "vm-1", Name: "vm-name"},
			},
			AttachGuestToolsErr: errors.New("enterprise only"),
		}
		state := testStateBag(config, driver)

		action := (&StepAttachGuestTools{}).Run(context.Background(), state)

		// Non-fatal: continues even on failure
		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.AttachGuestToolsCalled)
		_, hasErr := state.GetOk("error")
		assert.False(t, hasErr, "attach failure should not halt build")
	})
}

func TestStepAttachGuestTools_Cleanup_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		(&StepAttachGuestTools{}).Cleanup(testStateBag(&Config{}, &MockDriver{}))
	})
}
