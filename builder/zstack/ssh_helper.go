// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

func GetHostIp(state multistep.StateBag) (*string, error) {
	vm := state.Get("config").(*Config)
	return &vm.IP, nil
}

func GetVmUuid(state multistep.StateBag) string {
	vm := state.Get("config").(*Config)
	return vm.InstanceUuid
}
