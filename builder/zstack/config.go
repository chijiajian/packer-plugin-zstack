package zstack

import (
	"errors"
	"os"

	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
)

type Config struct {
	common.PackerConfig    `mapstructure:",squash"`
	commonsteps.HTTPConfig `mapstructure:",squash"`
	Comm                   communicator.Config `mapstructure:",squash"`
	AccessConfig           `mapstructure:",squash"`

	//	DataVolumeConfig    `mapstructure:",squash"`
	ImageConfig    `mapstructure:",squash"`
	NetworkConfig  `mapstructure:",squash"`
	InstanceConfig `mapstructure:",squash"`
	//ZoneConfig          `mapstructure:",squash"`
	BackupStorageConfig `mapstructure:",squash"`
	ExportImageResult   `mapstructure:",squash"`
	DebugMode           string `mapstructure:"debug_mode"`
}

type ImageConfig struct {
	ImageName   string `mapstructure:"image_name"`
	SourceImage string `mapstructure:"source_image"`
	//	GuestOsType        string   `mapstructure:"guest_os_type"`
	//	Url                string   `mapstructure:"url"`
	//	Format             string   `mapstructure:"format"`
	//	BackupStorageUuids []string `mapstructure:"backup_storage_uuids"`
	ImageUuid string `mapstructure:"image_uuid"`

	// Platform           string   `mapstructure:"platform"`
}

type NetworkConfig struct {
	L3NetworkName string `mapstructure:"network_name"`
	L3NetworkUuid string `mapstructure:"network_uuid"`
}

type InstanceConfig struct {
	InstanceName         string `mapstructure:"instance_name"`
	InstanceUuid         string `mapstructure:"instance_uuid"`
	InstanceOfferingName string `mapstructure:"instance_offering_name"`
	InstanceOfferingUuid string `mapstructure:"instance_offering_uuid"`
	MemorySize           int64  `mapstructure:"memory_size"`
	CpuNum               int64  `mapstructure:"cpu_num"`
	SshKey               string `mapstructure:"sshkey"`
	UserData             string `mapstructure:"userdata"`
	RootVolumeUuid       string `mapstructure:"root_volume_uuid"`
	IP                   string `mapstructure:"ip"`
}

type BackupStorageConfig struct {
	BackupStorageUuid string `mapstructure:"backup_storage_uuid"`
	BackupStorageName string `mapstructure:"backup_storage_name"`
}

type ExportImageResult struct {
	ImageUrl string `mapstructure:"image_url"`
	Success  bool   `mapstructure:"export_image_result"`
}

/*
	type ExportImageFromBackupStorageResultView struct {
	    ImageUrl     string `json:"imageUrl"`     //导出的镜像的URL
	    ExportMd5Sum string `json:"exportMd5Sum"` //导出的镜像的MD5值
	    Success      bool   `json:"success"`      //导出是否成功
	    Error        string `json:"error"`        //导出失败的错误信息
	}

	type DataVolumeConfig struct {
		Name               string `mapstructure:"name"`
		DiskSize           int64  `mapstructure:"disk_size"`
		DiskOfferingUuid   string `mapstructure:"disk_offering_uuid"`
		PrimaryStorageUuid string `mapstructure:"primary_storage_uuid"`
	}

	type ImageConfig struct {
		ImageName          string   `mapstructure:"image_name"`
		GuestOsType        string   `mapstructure:"guest_os_type"`
		Url                string   `mapstructure:"url"`
		Format             string   `mapstructure:"format"`
		BackupStorageUuids []string `mapstructure:"backup_storage_uuids"`
		ImageUuid          string   `mapstructure:"image_uuid"`
		Platform           string   `mapstructure:"platform"`
	}

	type ZoneConfig struct {
		ZoneUuid string `mapstructure:"zone_uuid"`
	}
*/
func (c *Config) Prepare(raws ...interface{}) error {
	err := config.Decode(c, &config.DecodeOpts{
		Interpolate:        true,
		InterpolateContext: &c.ctx,
	}, raws...)
	if err != nil {
		return err
	}

	var errs *packersdk.MultiError

	if c.Host == "" {
		c.Host = os.Getenv("ZSTACK_HOST")
	}

	if c.AccessKeyId == "" {
		c.AccessKeyId = os.Getenv("ZSTACK_ACCESS_KEY_ID")
	}

	if c.AccessKeySecret == "" {
		c.AccessKeySecret = os.Getenv("ZSTACK_ACCESS_KEY_SECRET")
	}

	if c.AccountName == "" {
		c.AccountName = os.Getenv("ZSTACK_ACCOUNT_NAME")
	}

	if c.AccountPassword == "" {
		c.AccountPassword = os.Getenv("ZSTACK_ACCOUNT_PASSWORD")
	}

	if c.Host == "" {
		errs = packersdk.MultiErrorAppend(errs, errors.New("a host of cloud mn ip must be sepcified"))
	}
	/*
		if c.ZoneUuid == "" {
			errs = packersdk.MultiErrorAppend(errs, errors.New("a zone uuid must be sepcified"))
		}
	*/
	if errs != nil && len(errs.Errors) > 0 {
		return errs
	}
	return nil

}
