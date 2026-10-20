// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zstack

import (
	"fmt"

	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
)

// SnapshotBuilder converts an existing root-volume snapshot without creating a VM.
type SnapshotBuilder struct{ Builder }

//go:generate go tool packer-sdc mapstructure-to-hcl2 -type SnapshotConfig

type SnapshotConfig struct {
	common.PackerConfig      `mapstructure:",squash"`
	AccessConfig             `mapstructure:",squash"`
	BackupStorageConfig      `mapstructure:",squash"`
	SourceVolumeSnapshotUuid string `mapstructure:"source_volume_snapshot_uuid"`
	ImageName                string `mapstructure:"image_name"`
	ImageDescription         string `mapstructure:"image_description"`
	Platform                 string `mapstructure:"platform"`
	Architecture             string `mapstructure:"architecture"`
	GuestOsType              string `mapstructure:"guest_os_type"`
	Communicator             string `mapstructure:"communicator"`
	APITimeoutRaw            string `mapstructure:"api_timeout"`
	CleanupTimeoutRaw        string `mapstructure:"cleanup_timeout"`
	ImageReadyTimeoutRaw     string `mapstructure:"image_ready_timeout"`
	KeepImageOnError         bool   `mapstructure:"keep_image_on_error"`
	SkipExport               bool   `mapstructure:"skip_export"`
}

func (b *SnapshotBuilder) ConfigSpec() hcldec.ObjectSpec {
	return new(SnapshotConfig).FlatMapstructure().HCL2Spec()
}

func (b *SnapshotBuilder) Prepare(raws ...any) ([]string, []string, error) {
	// Decode the restricted schema first, including for legacy JSON templates.
	var c SnapshotConfig
	if err := config.Decode(&c, nil, raws...); err != nil {
		return nil, nil, err
	}
	if c.SourceVolumeSnapshotUuid == "" {
		return nil, nil, fmt.Errorf("zstack-snapshot requires source_volume_snapshot_uuid")
	}
	b.snapshotBuilder = true
	return b.Builder.Prepare(raws...)
}
