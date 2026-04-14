package zstack

import (
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/zstackio/packer-plugin-zstack/builder/zstack/utils"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
	"golang.org/x/net/context"
)

type StepCreateVMInstance struct {
	//vm *view.VmInstanceInventoryView
}

func strPtr(s string) *string {
	return &s
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
		userData := strings.TrimSpace(config.UserData)
		/*
			if userData[len(userData)-1] == '\n' {
				userData = userData[:len(userData)-1]
			}
		*/
		if _, err := base64.StdEncoding.DecodeString(userData); err != nil {
			//log.Printf("[DEBUG] base64 encoding user data...")
			userData = base64.StdEncoding.EncodeToString([]byte(userData))
		}
		//log.Printf("[DEBUG] userdata: %s", userData)
		systemtags = addSystemTags(systemtags, fmt.Sprintf("userdata::%s", userData))
	}

	createVmInstanceParam := param.CreateVmInstanceParam{
		BaseParam: param.BaseParam{
			SystemTags: systemtags,
		},
		Params: param.CreateVmInstanceParamDetail{
			Name:           config.InstanceName,
			Description:    strPtr("Auto created by packer-plugin-zstack"),
			ImageUuid:      strPtr(config.ImageUuid),
			L3NetworkUuids: []string{config.L3NetworkUuid},
		},
	}

	if config.InstanceOfferingUuid != "" {
		createVmInstanceParam.Params.InstanceOfferingUuid = strPtr(config.InstanceOfferingUuid)
	} else {
		if config.CPUNum > 0 {
			cpuNum := int(config.CPUNum)
			createVmInstanceParam.Params.CpuNum = &cpuNum
		}
		if config.MemorySize > 0 {
			memSize := utils.MBToBytes(config.MemorySize)
			createVmInstanceParam.Params.MemorySize = &memSize
		}
	}

	instance, err := driver.CreateVmInstance(createVmInstanceParam)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to create VM instance: %v", err))
		log.Printf("[ERROR] Failed to create VM instance: %v", err)
		state.Put("error", err)
		return multistep.ActionHalt
	}

	config.InstanceUuid = instance.UUID
	config.RootVolumeUuid = instance.RootVolumeUuid
	config.IP = instance.VmNics[0].Ip

	state.Put("config", config)
	log.Printf("[INFO] Successfully created VM instance (UUID: %s, IP: %s)", instance.UUID, config.IP)
	ui.Say(fmt.Sprintf("Successfully created VM instance '%s' (UUID: %s, IP: %s)",
		config.InstanceName, instance.UUID, config.IP))

	return multistep.ActionContinue
}

func (s *StepCreateVMInstance) Cleanup(state multistep.StateBag) {
	// AC-004-06: Only cleanup on failure — successful builds use StepExpungeVmInstance
	_, hasError := state.GetOk("error")
	if !hasError {
		return
	}

	config := state.Get("config").(*Config)
	if config.InstanceUuid == "" {
		return
	}

	ui := state.Get("ui").(packersdk.Ui)
	driver := state.Get("driver").(Driver)

	// AC-004-04: Log cleanup action
	ui.Say("Cleaning up: destroying temporary VM instance...")
	log.Printf("[INFO] Cleaning up VM instance '%s' after build failure", config.InstanceUuid)

	if err := driver.DestroyVmInstance(config.InstanceUuid); err != nil {
		// AC-004-05: Cleanup failure logs warning, doesn't panic
		log.Printf("[WARN] Failed to destroy VM instance during cleanup: %v", err)
		ui.Error(fmt.Sprintf("Warning: failed to destroy VM instance during cleanup: %v", err))
		return
	}

	if err := driver.DeleteVmInstance(config.InstanceUuid); err != nil {
		log.Printf("[WARN] Failed to expunge VM instance during cleanup: %v", err)
		ui.Error(fmt.Sprintf("Warning: failed to expunge VM instance during cleanup: %v", err))
		return
	}

	log.Printf("[INFO] Successfully cleaned up VM instance '%s'", config.InstanceUuid)
	ui.Say("Successfully cleaned up temporary VM instance")
}
