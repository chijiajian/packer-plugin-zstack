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

func (b *Builder) Prepare(raws ...any) ([]string, []string, error) {

	errs := b.config.Prepare(raws...)

	if b.config.Comm.Type == "" {
		if b.config.SourceVolumeSnapshotUuid != "" {
			b.config.Comm.Type = "none"
		} else {
			b.config.Comm.Type = "ssh"
		}
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
	log.Printf("[DEBUG] Starting build with %s", b.config.RedactedSummary())

	driver, err := b.config.AccessConfig.Driver()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ZStack Driver: %v", err)
	}

	state := new(multistep.BasicStateBag)
	state.Put("ui", ui)
	state.Put("driver", driver)
	state.Put("config", &b.config)
	state.Put("hook", hook)

	var steps []multistep.Step

	if b.config.SourceVolumeSnapshotUuid != "" {
		log.Printf("[INFO] Using snapshot-only build path (source_volume_snapshot_uuid=%s)", b.config.SourceVolumeSnapshotUuid)
		steps = []multistep.Step{
			&StepPreValidate{},
			&StepCreateImageFromSnapshot{},
			&StepWaitForImageReady{},
			&StepExportImage{},
		}

		log.Printf("[DEBUG] Build steps prepared with %s", b.config.RedactedSummary())

		b.runner = commonsteps.NewRunner(steps, b.config.PackerConfig, ui)
		b.runner.Run(ctx, state)

		return b.collectArtifact(state, driver)
	}

	baseSteps := []multistep.Step{
		&StepPreValidate{},
	}

	if b.config.SourceImageUrl != "" {
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

	if b.config.InstanceConfig.InstanceOfferingName != "" || b.config.InstanceConfig.InstanceOfferingUuid != "" {
		log.Printf("[DEBUG] InstanceOffering validate...")
		instanceOfferingSteps := []multistep.Step{
			&StepInstanceOfferingValidate{},
		}
		baseSteps = append(baseSteps, instanceOfferingSteps...)
	}
	remainingSteps := []multistep.Step{}

	// AC-005-06: Only generate SSH key for SSH communicator
	if b.config.Comm.Type == "ssh" {
		remainingSteps = append(remainingSteps, &StepCreateSSHKey{
			Password:     b.config.Comm.SSHPassword,
			Debug:        b.config.PackerDebug,
			DebugKeyPath: fmt.Sprintf("zstack_%s.pem", b.config.PackerBuildName),
		})
	}

	remainingSteps = append(remainingSteps,
		&StepCreateVMInstance{},
		&StepWaitForRunning{},
		&StepAttachGuestTools{},
		&communicator.StepConnect{
			Config:    &b.config.Comm,
			Host:      commHost(b.config.Comm.SSHHost),
			SSHConfig: b.config.Comm.SSHConfigFunc(),
		},
		&commonsteps.StepProvision{},
		&StepStopVmInstance{},
		&StepCreateImage{},
		&StepExpungeVmInstance{},
		&StepExportImage{},
	)

	steps = append(baseSteps, remainingSteps...)

	log.Printf("[DEBUG] Build steps prepared with %s", b.config.RedactedSummary())

	b.runner = commonsteps.NewRunner(steps, b.config.PackerConfig, ui)
	b.runner.Run(ctx, state)

	return b.collectArtifact(state, driver)
}

func (b *Builder) collectArtifact(state multistep.StateBag, driver Driver) (packersdk.Artifact, error) {
	var urls []string
	if v, ok := state.GetOk("image_url"); ok {
		switch val := v.(type) {
		case string:
			if val != "" {
				urls = []string{val}
			}
		case []string:
			urls = val
		case []any:
			for _, item := range val {
				if s, ok := item.(string); ok {
					urls = append(urls, s)
				}
			}
		default:
			log.Printf("[WARN] unexpected type for image_url in state: %T", v)
		}
	}

	if rawErr, ok := state.GetOk("error"); ok {
		if err, ok := rawErr.(error); ok {
			return nil, err
		}
		return nil, fmt.Errorf("unexpected error type in state: %T", rawErr)
	}

	artifact := &Artifact{
		config:    b.config,
		exportUrl: urls,
		driver:    driver,
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
