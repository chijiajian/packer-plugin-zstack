package zstack

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	//	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

const BuilderId = "packer.zstack"

type Builder struct {
	//client *client.ZSClient    `mapstructure:",squash"`
	config Config
	runner multistep.Runner
	ui     packersdk.Ui
	//ctx    interpolate.Context `mapstructure:",squash"`
}

func (b *Builder) ConfigSpec() hcldec.ObjectSpec { return b.config.FlatMapstructure().HCL2Spec() }

func (b *Builder) Prepare(raws ...interface{}) ([]string, []string, error) {
	errs := b.config.Prepare(raws...)

	if b.config.Comm.Type == "" {
		b.config.Comm.Type = "ssh"
	}

	//var errs *packer.MultiError
	if es := b.config.Comm.Prepare(&b.config.ctx); len(es) > 0 {
		errs = packersdk.MultiErrorAppend(errs, es...)
	}

	if errs != nil {
		return nil, nil, errs
	}
	return nil, nil, nil
}

func (b *Builder) Run(ctx context.Context, ui packersdk.Ui, hook packersdk.Hook) (packersdk.Artifact, error) {
	b.ui = ui
	log.Printf("[DEBUG] Starting build with config: %+v", b.config)
	log.Printf("[DEBUG] Starting Prepare method")
	driver, err := b.config.AccessConfig.Driver()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ZStack Driver: %v", err)
	}

	state := new(multistep.BasicStateBag)
	state.Put("ui", ui)
	state.Put("driver", driver)
	state.Put("config", &b.config)
	state.Put("hook", hook)

	// 定义执行步骤
	steps := []multistep.Step{
		&StepPreValidate{},
		&StepCreateVMInstance{},
		&StepWaitForRunning{},
		&StepAttachGuestTools{},
		&communicator.StepConnect{
			Config:    &b.config.Comm,
			Host:      commHost(b.config.Comm.SSHHost),
			SSHConfig: b.config.Comm.SSHConfigFunc(),
		},
		&commonsteps.StepProvision{}, &StepStopVmInstance{},
		&StepCreateImage{},
		&StepExportImage{},
	}

	log.Printf("[DEBUG] Completed Pre Step with config: %+v", b.config)
	// 执行流程
	b.runner = commonsteps.NewRunner(steps, b.config.PackerConfig, ui)
	b.runner.Run(ctx, state)

	// 返回空 Artifact（仅测试用）
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}
	return nil, nil
}

func commHost(host string) func(multistep.StateBag) (string, error) {
	return func(state multistep.StateBag) (string, error) {
		if host != "" {
			return host, nil
		}

		// 从state中获取VM的IP地址
		config, ok := state.Get("config").(*Config)
		if !ok || config == nil {
			return "", fmt.Errorf("IP address not found")
		}

		ip := config.IP

		return ip, nil
	}
}
