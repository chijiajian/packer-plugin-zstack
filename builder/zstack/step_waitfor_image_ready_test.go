// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

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

func TestStepWaitForImageReady_Run(t *testing.T) {
	t.Run("EmptyImageUuidHalts", func(t *testing.T) {
		config := &Config{}
		driver := &MockDriver{}
		state := testStateBag(config, driver)

		action := (&StepWaitForImageReady{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		_, ok := state.GetOk("error")
		assert.True(t, ok)
	})

	t.Run("SuccessWhenReady", func(t *testing.T) {
		config := &Config{ImageConfig: ImageConfig{ImageUuid: "img-1"}}
		config.pollInterval = 5 * time.Millisecond
		driver := &MockDriver{GetImageResult: &view.ImageInventoryView{
			BaseInfoView: view.BaseInfoView{UUID: "img-1"},
			Status:       "Ready",
		}}
		state := testStateBag(config, driver)

		action := (&StepWaitForImageReady{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionContinue, action)
		assert.True(t, driver.GetImageCalled)
	})

	t.Run("GetImageErrorHalts", func(t *testing.T) {
		config := &Config{ImageConfig: ImageConfig{ImageUuid: "img-1"}}
		config.pollInterval = 5 * time.Millisecond
		driver := &MockDriver{GetImageErr: errors.New("query failed")}
		state := testStateBag(config, driver)

		action := (&StepWaitForImageReady{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Contains(t, errVal.(error).Error(), "query failed")
	})

	t.Run("ContextCancelHalts", func(t *testing.T) {
		config := &Config{ImageConfig: ImageConfig{ImageUuid: "img-1"}}
		config.pollInterval = time.Hour // don't let ticker win
		// Return non-Ready so the step stays in the polling loop.
		driver := &MockDriver{GetImageResult: &view.ImageInventoryView{
			BaseInfoView: view.BaseInfoView{UUID: "img-1"},
			Status:       "Downloading",
		}}
		state := testStateBag(config, driver)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		action := (&StepWaitForImageReady{}).Run(ctx, state)
		assert.Equal(t, multistep.ActionHalt, action)
	})

	t.Run("TimeoutHalts", func(t *testing.T) {
		config := &Config{ImageConfig: ImageConfig{ImageUuid: "img-1"}}
		config.imageReadyTimeout = 10 * time.Millisecond
		config.pollInterval = time.Hour // ensure timeout wins

		driver := &MockDriver{GetImageResult: &view.ImageInventoryView{Status: "Downloading"}}
		state := testStateBag(config, driver)

		action := (&StepWaitForImageReady{}).Run(context.Background(), state)

		assert.Equal(t, multistep.ActionHalt, action)
		errVal, ok := state.GetOk("error")
		assert.True(t, ok)
		assert.Contains(t, errVal.(error).Error(), "timeout")
	})
}

func TestStepWaitForImageReady_Cleanup_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		(&StepWaitForImageReady{}).Cleanup(testStateBag(&Config{}, &MockDriver{}))
	})
}
