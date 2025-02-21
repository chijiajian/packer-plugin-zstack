package zstack

import (
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"zstack.io/zstack-sdk-go/pkg/view"
)

func GetHostIp(state multistep.StateBag) (*string, error) {
	vm := state.Get("config").(*Config)
	return &vm.IP, nil
}

func GetVmUuid(state multistep.StateBag) string {
	vm := state.Get("instance").(*view.VmInstanceInventoryView)
	return vm.UUID
}
