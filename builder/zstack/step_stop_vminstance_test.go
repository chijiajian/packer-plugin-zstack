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

func TestStepStopVmInstance_Run(t *testing.T) {
	t.Run("StopVMSuccess", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		driver := &MockDriver{StopVminstanceResult: &view.VmInstanceInventoryView{BaseInfoView: view.BaseInfoView{Name: "vm-name"}}}
		state := testStateBag(config, driver)

		action := (&StepStopVmInstance{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.StopVminstanceCalled)
		assert.Equal(t, "vm-1", driver.StopVminstanceUuid)
	})

	t.Run("StopVMError", func(t *testing.T) {
		config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-1"}}
		driver := &MockDriver{StopVminstanceErr: errors.New("stop failed")}
		state := testStateBag(config, driver)

		action := (&StepStopVmInstance{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		assert.True(t, driver.StopVminstanceCalled)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Error(t, errVal.(error))
		assert.Contains(t, errVal.(error).Error(), "failed to stop VM instance 'vm-1'")
	})
}
