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
	//	ui.Say("Create vm instance...image uuid: " + config.ImageUuid + " ... " + config.L3NetworkUuid + " ... " + config.InstanceOfferingUuid)

	ui.Say("Get from state image uuid1 ... ")
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

	ui.Message(config.ImageUuid + " " + config.InstanceOfferingUuid + " " + config.L3NetworkUuid)

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
			//MemorySize:           config.MemorySize,
			//CpuNum:               config.CpuNum,
		},
	}

	instance, err := driver.CreateVmInstance(createVmInstanceParam)
	if err != nil {

	}
	ui.Say("vm instance has been created")
	ui.Message(fmt.Sprintf("Instance UUID: %s", instance.UUID))

	config.InstanceUuid = instance.UUID
	config.RootVolumeUuid = instance.RootVolumeUUID
	config.IP = instance.VMNics[0].IP

	state.Put("config", config)

	return multistep.ActionContinue
}

func (s *StepCreateVMInstance) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	driver := state.Get("driver").(Driver)

	if state.Get("debug_mode").(bool) {
		return
	}
	if config.InstanceUuid == "" {
		return
	}

	ui.Say("Cleaning up VM instance...")
	if err := driver.DeleteVmInstance(config.InstanceUuid); err != nil {
		ui.Error(fmt.Sprintf("Error cleaning up VM instance: %s", err))
	}
	config.InstanceUuid = ""
	state.Remove("config")
}
