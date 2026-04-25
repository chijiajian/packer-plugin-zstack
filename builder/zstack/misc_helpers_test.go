// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHostIp(t *testing.T) {
	config := &Config{InstanceConfig: InstanceConfig{IP: "10.0.0.1"}}
	state := testStateBag(config, &MockDriver{})
	ip, err := GetHostIp(state)
	assert.NoError(t, err)
	if assert.NotNil(t, ip) {
		assert.Equal(t, "10.0.0.1", *ip)
	}
}

func TestGetVmUuid(t *testing.T) {
	config := &Config{InstanceConfig: InstanceConfig{InstanceUuid: "vm-abc"}}
	state := testStateBag(config, &MockDriver{})
	assert.Equal(t, "vm-abc", GetVmUuid(state))
}

func TestBuilder_ConfigSpec(t *testing.T) {
	b := &Builder{}
	spec := b.ConfigSpec()
	assert.NotNil(t, spec)
	assert.NotEmpty(t, spec)
}

func TestFlatConfigMapping(t *testing.T) {
	flat := new(FlatConfig).HCL2Spec()
	assert.NotNil(t, flat)
	assert.Contains(t, flat, "zstack_host")
	assert.Contains(t, flat, "clean_traffic")
	assert.Contains(t, flat, "image_ready_timeout")
	assert.Contains(t, flat, "vm_running_timeout")
}

func TestFlatMapstructure(t *testing.T) {
	assert.NotNil(t, (&Config{}).FlatMapstructure())
}

// Exercise no-op Cleanup implementations so they register as covered.
func TestCleanups_NoOp(t *testing.T) {
	state := testStateBag(&Config{}, &MockDriver{})
	assert.NotPanics(t, func() {
		(&StepPreValidate{}).Cleanup(state)
		(&StepSourceImageValidate{}).Cleanup(state)
		(&StepInstanceOfferingValidate{}).Cleanup(state)
		(&StepAddImage{}).Cleanup(state)
		(&StepCreateImage{}).Cleanup(state)
		(&StepStopVmInstance{}).Cleanup(state)
		(&StepExportImage{}).Cleanup(state)
		(&StepExpungeVmInstance{}).Cleanup(state)
		(&StepAttachGuestTools{}).Cleanup(state)
		(&StepCreateSSHKey{}).Cleanup(state)
	})
}
