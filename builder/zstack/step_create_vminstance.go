// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package zstack

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/zstackio/packer-plugin-zstack/builder/zstack/utils"
	"github.com/zstackio/zstack-sdk-go-v2/pkg/param"
)

type StepCreateVMInstance struct{}

func strPtr(s string) *string {
	return &s
}

func (s *StepCreateVMInstance) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packersdk.Ui)
	config := state.Get("config").(*Config)
	ctx, cancel := context.WithTimeout(ctx, config.VmRunningTimeout())
	defer cancel()
	driver := stepDriver(ctx, state)

	ui.Say(fmt.Sprintf("Creating VM instance '%s'...", config.InstanceName))

	var systemtags []string
	systemtags = append(systemtags, "cdroms::Empty::None::None", fmt.Sprintf("cleanTraffic::%t", config.CleanTraffic))
	if config.SshKey != "" {
		systemtags = append(systemtags, fmt.Sprintf("sshkey::%s", config.SshKey))
	}
	if config.UserData != "" {
		systemtags = append(systemtags, fmt.Sprintf("userdata::%s", encodeUserData(config.UserData)))
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

	r := resources(state)
	r.VM = newResource()
	createVmInstanceParam.Params.ResourceUuid = &r.VM.UUID
	if len(config.L3NetworkUuids) > 0 {
		createVmInstanceParam.Params.L3NetworkUuids = config.L3NetworkUuids
	}
	if config.DefaultL3NetworkUuid != "" {
		createVmInstanceParam.Params.DefaultL3NetworkUuid = &config.DefaultL3NetworkUuid
	}
	if config.ZoneUuid != "" {
		createVmInstanceParam.Params.ZoneUuid = &config.ZoneUuid
	}
	if config.ClusterUuid != "" {
		createVmInstanceParam.Params.ClusterUuid = &config.ClusterUuid
	}
	if config.HostUuid != "" {
		createVmInstanceParam.Params.HostUuid = &config.HostUuid
	}
	if config.RootDiskOfferingUuid != "" {
		createVmInstanceParam.Params.RootDiskOfferingUuid = &config.RootDiskOfferingUuid
	}
	createVmInstanceParam.Params.DataDiskOfferingUuids = config.DataDiskOfferingUuids
	ui.Say(fmt.Sprintf("Tracking VM resource UUID: %s", r.VM.UUID))
	instance, err := driver.CreateVmInstance(createVmInstanceParam)
	if err != nil {
		ui.Error(fmt.Sprintf("Failed to create VM instance: %v", err))
		log.Printf("[ERROR] Failed to create VM instance: %v", err)
		state.Put("error", err)
		return multistep.ActionHalt
	}

	if instance == nil || instance.UUID == "" {
		return halt(state, fmt.Errorf("CreateVmInstance returned empty inventory"))
	}
	r.VM.UUID = instance.UUID
	r.VM.Created = true
	for _, v := range instance.AllVolumes {
		if v.Type == "Data" {
			r.DataVolumes = append(r.DataVolumes, ownedResource{UUID: v.UUID, Created: true})
		}
	}
	config.InstanceUuid = instance.UUID
	state.Put("instance_id", instance.UUID)
	config.RootVolumeUuid = instance.RootVolumeUuid
	// Prefer the selected/default NIC. A supplied communicator host needs no NIC IP.
	for _, nic := range instance.VmNics {
		if nic.Ip != "" && (config.DefaultL3NetworkUuid == "" || nic.L3NetworkUuid == config.DefaultL3NetworkUuid) {
			config.IP = nic.Ip
			break
		}
	}
	if config.IP == "" && config.Comm.Type != "none" && config.Comm.SSHHost == "" && config.Comm.WinRMHost == "" {
		return halt(state, fmt.Errorf("VM instance %s has no usable network interface; cannot proceed", instance.UUID))
	}

	if config.Comm.Type == "ssh" && config.Comm.SSHHost == "" {
		config.Comm.SSHHost = config.IP
	}
	if config.Comm.Type == "winrm" && config.Comm.WinRMHost == "" {
		config.Comm.WinRMHost = config.IP
	}
	log.Printf("[INFO] Successfully created VM instance (UUID: %s, IP: %s)", instance.UUID, config.IP)
	ui.Say(fmt.Sprintf("Successfully created VM instance '%s' (UUID: %s, IP: %s)",
		config.InstanceName, instance.UUID, config.IP))

	return multistep.ActionContinue
}

func (s *StepCreateVMInstance) Cleanup(state multistep.StateBag) { cleanupVM(state) }

// encodeUserData returns the user data as base64. Input is always treated as
// plaintext. Callers that already hold base64-encoded content must decode it
// before passing it in.
func encodeUserData(raw string) string {
	return base64.StdEncoding.EncodeToString([]byte(strings.TrimSpace(raw)))
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
