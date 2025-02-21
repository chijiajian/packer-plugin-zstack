package zstack

import (
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
)

type FlatConfig struct {
	Host            *string `mapstructure:"zstack_host" cty:"zstack_host" hcl:"zstack_host"`
	Port            *int    `mapstructure:"port" cty:"port" hcl:"port"`
	AccountName     *string `mapstructure:"account_name" cty:"account_name" hcl:"account_name"`
	AccountPassword *string `mapstructure:"account_password" cty:"account_password" hcl:"account_password"`
	AccessKeyId     *string `mapstructure:"access_key_id" cty:"access_key_id" hcl:"access_key_id"`
	AccessKeySecret *string `mapstructure:"access_key_secret" cty:"access_key_secret" hcl:"access_key_secret"`

	SourceImage    *string `mapstructure:"source_image" cty:"source_image" hcl:"source_image"`
	ImageName      *string `mapstructure:"image_name" cty:"image_name" hcl:"image_name"`
	ImageUuid      *string `mapstructure:"image_uuid" cty:"image_uuid" hcl:"image_uuid"`
	RootVolumeUuid *string `mapstructure:"root_volume_uuid" cty:"root_volume_uuid" hcl:"root_volume_uuid"`
	ImageUrl       *string `mapstructure:"image_url" cty:"image_url" hcl:"image_url"`

	L3NetworkUuid *string `mapstructure:"network_uuid" cty:"network_uuid" hcl:"network_uuid"`
	L3NetworkName *string `mapstructure:"network_name" cty:"network_name" hcl:"network_name"`

	InstanceName         *string `mapstructure:"instance_name" cty:"instance_name" hcl:"instance_name"`
	InstanceOfferingName *string `mapstructure:"instance_offering_name" cty:"instance_offering_name" hcl:"instance_offering_name"`
	InstanceOfferingUuid *string `mapstructure:"instance_offering_uuid" cty:"instance_offering_uuid" hcl:"instance_offering_uuid"`
	InstanceUuid         *string `mapstructure:"instance_uuid" cty:"instance_uuid" hcl:"instance_uuid"`
	SshKey               *string `mapstructure:"sshkey" cty:"sshkey" hcl:"sshkey"`
	UserData             *string `mapstructure:"userdata" cty:"userdata" hcl:"userdata"`
	MemorySize           *string `mapstructure:"memory_size" cty:"memory_size" hcl:"memory_size"`
	CpuNum               *string `mapstructure:"cpu_num" cty:"cpu_num" hcl:"cpu_num"`
	BackupStorageName    *string `mapstructure:"backup_storage_name" cty:"backup_storage_name" hcl:"backup_storage_name"`
	BackupStorageUuid    *string `mapstructure:"backup_storage_uuid" cty:"backup_storage_uuid" hcl:"backup_storage_uuid"`

	SSHHost     *string `mapstructure:"ssh_host" cty:"ssh_host" hcl:"ssh_host"`
	SSHPort     *int    `mapstructure:"ssh_port" cty:"ssh_port" hcl:"ssh_port"`
	SSHUsername *string `mapstructure:"ssh_username" cty:"ssh_username" hcl:"ssh_username"`
	SSHPassword *string `mapstructure:"ssh_password" cty:"ssh_password" hcl:"ssh_password"`

	DebugMode *string `mapstructure:"debug_mode" cty:"debug_mode" hcl:"debug_mode"`
}

func (*Config) FlatMapstructure() interface{ HCL2Spec() map[string]hcldec.Spec } {
	return new(FlatConfig)
}

func (*FlatConfig) HCL2Spec() map[string]hcldec.Spec {
	s := map[string]hcldec.Spec{
		"zstack_host":       &hcldec.AttrSpec{Name: "zstack_host", Type: cty.String, Required: true},
		"port":              &hcldec.AttrSpec{Name: "port", Type: cty.Number, Required: false},
		"account_name":      &hcldec.AttrSpec{Name: "account_name", Type: cty.String, Required: false},
		"account_password":  &hcldec.AttrSpec{Name: "account_password", Type: cty.String, Required: false},
		"access_key_id":     &hcldec.AttrSpec{Name: "access_key_id", Type: cty.String, Required: false},
		"access_key_secret": &hcldec.AttrSpec{Name: "access_key_secret", Type: cty.String, Required: false},

		"source_image":     &hcldec.AttrSpec{Name: "source_image", Type: cty.String, Required: false},
		"image_name":       &hcldec.AttrSpec{Name: "image_name", Type: cty.String, Required: false},
		"image_uuid":       &hcldec.AttrSpec{Name: "image_uuid", Type: cty.String, Required: false},
		"root_volume_uuid": &hcldec.AttrSpec{Name: "root_volume_uuid", Type: cty.String, Required: false},
		"image_url":        &hcldec.AttrSpec{Name: "image_url", Type: cty.String, Required: false},

		"network_uuid": &hcldec.AttrSpec{Name: "network_uuid", Type: cty.String, Required: false},
		"network_name": &hcldec.AttrSpec{Name: "network_name", Type: cty.String, Required: false},

		"instance_name":          &hcldec.AttrSpec{Name: "instance_name", Type: cty.String, Required: false},
		"instance_uuid":          &hcldec.AttrSpec{Name: "instance_uuid", Type: cty.String, Required: false},
		"instance_offering_name": &hcldec.AttrSpec{Name: "instance_offering_name", Type: cty.String, Required: false},
		"instance_offering_uuid": &hcldec.AttrSpec{Name: "instance_offering_uuid", Type: cty.String, Required: false},
		"sshkey":                 &hcldec.AttrSpec{Name: "sshkey", Type: cty.String, Required: false},
		"userdata":               &hcldec.AttrSpec{Name: "userdata", Type: cty.String, Required: false},
		"memory_size":            &hcldec.AttrSpec{Name: "memory_size", Type: cty.String, Required: false},
		"cpu_num":                &hcldec.AttrSpec{Name: "cpu_num", Type: cty.String, Required: false},
		"backup_storage_name":    &hcldec.AttrSpec{Name: "backup_storage_name", Type: cty.String, Required: false},
		"backup_storage_uuid":    &hcldec.AttrSpec{Name: "backup_storage_uuid", Type: cty.String, Required: false},

		"ssh_host":     &hcldec.AttrSpec{Name: "ssh_host", Type: cty.String, Required: false},
		"ssh_port":     &hcldec.AttrSpec{Name: "ssh_port", Type: cty.Number, Required: false},
		"ssh_username": &hcldec.AttrSpec{Name: "ssh_username", Type: cty.String, Required: false},
		"ssh_password": &hcldec.AttrSpec{Name: "ssh_password", Type: cty.String, Required: false},

		"debug_mode": &hcldec.AttrSpec{Name: "debug_mode", Type: cty.String, Required: false},
	}

	return s
}
