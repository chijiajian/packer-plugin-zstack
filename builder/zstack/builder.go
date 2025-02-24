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
)

const BuilderId = "packer.zstack"

type Builder struct {
	config Config
	runner multistep.Runner
	ui     packersdk.Ui
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

	baseSteps := []multistep.Step{
		&StepPreValidate{},
	}

	if b.config.SourceImageUrl != "" && b.config.Format != "" && b.config.Platform != "" {
		imageSteps := []multistep.Step{
			&StepAddImage{},
			&StepWaitForImageReady{},
		}
		baseSteps = append(baseSteps, imageSteps...)
	} else {
		imageSteps := []multistep.Step{
			&StepSourceImageValidate{},
		}
		baseSteps = append(baseSteps, imageSteps...)
	}

	remainingSteps := []multistep.Step{
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
		&StepExpungeVmInstance{},
		&StepExportImage{},
	}

	steps := append(baseSteps, remainingSteps...)

	log.Printf("[DEBUG] Completed Pre Step with config: %+v", b.config)

	b.runner = commonsteps.NewRunner(steps, b.config.PackerConfig, ui)
	b.runner.Run(ctx, state)

	p, _ := state.GetOk("image_url")
	if p == nil {
		p = []string{}
	}

	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	artifact := &Artifact{
		config:    b.config,
		exportUrl: p.([]string),
	}
	return artifact, nil
}

func commHost(host string) func(multistep.StateBag) (string, error) {
	return func(state multistep.StateBag) (string, error) {
		if host != "" {
			return host, nil
		}

		config, ok := state.Get("config").(*Config)
		if !ok || config == nil {
			return "", fmt.Errorf("IP address not found")
		}

		ip := config.IP

		return ip, nil
	}
}
