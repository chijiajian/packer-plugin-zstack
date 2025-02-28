package zstack

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"golang.org/x/net/context"
	"zstack.io/zstack-sdk-go/pkg/param"
)

type StepCreateVMInstance struct {
	//vm *view.VmInstanceInventoryView
}

func (s *StepCreateVMInstance) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	ui.Say(fmt.Sprintf("Creating VM instance '%s'...", config.InstanceName))

	var systemtags []string
	systemtags = addSystemTags(systemtags, "cdroms::Empty::None::None", "cleanTraffic::false")
	if config.SshKey != "" {
		systemtags = addSystemTags(systemtags, fmt.Sprintf("sshkey::%s", config.SshKey))
	}
	if config.UserData != "" {
		userData := config.UserData
		if userData[len(userData)-1] == '\n' {
			userData = userData[:len(userData)-1]
		}
		if _, err := base64.StdEncoding.DecodeString(userData); err != nil {
			log.Printf("[DEBUG] base64 encoding user data...")
			userData = base64.StdEncoding.EncodeToString([]byte(userData))
		}
		log.Printf("[DEBUG] userdata: %s", userData)
		systemtags = addSystemTags(systemtags, fmt.Sprintf("userdata::%s", userData))
	}

	createVmInstanceParam := param.CreateVmInstanceParam{
		BaseParam: param.BaseParam{
			SystemTags: systemtags,
			UserTags:   nil,
			RequestIp:  "",
		},
		Params: param.CreateVmInstanceDetailParam{
			Name:                 config.InstanceName,
			Description:          "Auto created by packer-plugin-zstack",
			InstanceOfferingUUID: config.InstanceOfferingUuid,
			ImageUUID:            config.ImageUuid,
			L3NetworkUuids:       []string{config.L3NetworkUuid},
		},
	}

	if config.InstanceOfferingUuid != "" {
		createVmInstanceParam.Params.InstanceOfferingUUID = config.InstanceOfferingUuid
	} else {
		if config.CPUNum > 0 {
			createVmInstanceParam.Params.CpuNum = config.CPUNum
		}
		if config.MemorySize > 0 {
			createVmInstanceParam.Params.MemorySize = config.MemorySize
		}
		if config.DiskSize > 0 {
			createVmInstanceParam.Params.RootDiskSize = &config.DiskSize
		}
	}

	instance, err := driver.CreateVmInstance(createVmInstanceParam)
	if err != nil {

	}

	config.InstanceUuid = instance.UUID
	config.RootVolumeUuid = instance.RootVolumeUUID
	config.IP = instance.VMNics[0].IP

	state.Put("config", config)
	log.Printf("[INFO] Successfully created VM instance (UUID: %s, IP: %s)", instance.UUID, config.IP)
	ui.Say(fmt.Sprintf("Successfully created VM instance '%s' (UUID: %s, IP: %s)",
		config.InstanceName, instance.UUID, config.IP))

	return multistep.ActionContinue
}

func (s *StepCreateVMInstance) Cleanup(state multistep.StateBag) {

}
