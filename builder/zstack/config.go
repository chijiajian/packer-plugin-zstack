package zstack

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/multistep/commonsteps"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
)

const (
	defaultImageReadyTimeout = 5 * time.Minute
	defaultVmRunningTimeout  = 5 * time.Minute
)

type Config struct {
	common.PackerConfig    `mapstructure:",squash"`
	commonsteps.HTTPConfig `mapstructure:",squash"`
	Comm                   communicator.Config `mapstructure:",squash"`
	AccessConfig           `mapstructure:",squash"`

	ImageConfig    `mapstructure:",squash"`
	NetworkConfig  `mapstructure:",squash"`
	InstanceConfig `mapstructure:",squash"`

	BackupStorageConfig `mapstructure:",squash"`
	ExportImageResult   `mapstructure:",squash"`

	CleanTraffic           bool   `mapstructure:"clean_traffic"`
	ImageReadyTimeoutRaw   string `mapstructure:"image_ready_timeout"`
	VmRunningTimeoutRaw    string `mapstructure:"vm_running_timeout"`

	imageReadyTimeout time.Duration
	vmRunningTimeout  time.Duration

	pollInterval time.Duration
}

func (c *Config) PollInterval() time.Duration {
	if c.pollInterval > 0 {
		return c.pollInterval
	}
	return 5 * time.Second
}

func (c *Config) ImageReadyTimeout() time.Duration {
	if c.imageReadyTimeout > 0 {
		return c.imageReadyTimeout
	}
	return defaultImageReadyTimeout
}

func (c *Config) VmRunningTimeout() time.Duration {
	if c.vmRunningTimeout > 0 {
		return c.vmRunningTimeout
	}
	return defaultVmRunningTimeout
}

type ImageConfig struct {
	ImageName          string   `mapstructure:"image_name"`
	ImageDescription   string   `mapstructure:"image_description"`
	SourceImage        string   `mapstructure:"source_image"`
	GuestOsType        string   `mapstructure:"guest_os_type"`
	SourceImageUrl     string   `mapstructure:"source_image_url"`
	Format             string   `mapstructure:"format"`
	BackupStorageUuids []string `mapstructure:"backup_storage_uuids"`
	ImageUuid          string   `mapstructure:"image_uuid"`
	Platform           string   `mapstructure:"platform"`
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
	SshKey               string `mapstructure:"sshkey"`
	UserData             string `mapstructure:"userdata"`
	RootVolumeUuid       string `mapstructure:"root_volume_uuid"`
	IP                   string `mapstructure:"ip"`
	CPUNum               int64  `mapstructure:"cpu_num"`
	MemorySize           int64  `mapstructure:"memory_size"`
}

type BackupStorageConfig struct {
	BackupStorageUuid string `mapstructure:"backup_storage_uuid"`
	BackupStorageName string `mapstructure:"backup_storage_name"`
}

type ExportImageResult struct {
	ImageUrl string `mapstructure:"image_url"`
	Success  bool   `mapstructure:"export_image_result"`
}

func (c *Config) RedactedSummary() string {
	authMode := "none"
	switch {
	case c.AccountName != "" || c.AccountPassword != "":
		authMode = "account"
	case c.AccessKeyId != "" || c.AccessKeySecret != "":
		authMode = "access_key"
	}

	return fmt.Sprintf(
		"host=%q auth_mode=%s communicator=%q image_name=%q source_image=%q source_image_url_set=%t instance_name=%q network_name=%q network_uuid_set=%t instance_offering_name=%q instance_offering_uuid_set=%t backup_storage_name=%q backup_storage_uuid_set=%t ssh_host=%q ssh_port=%d ssh_username=%q ssh_password_set=%t ssh_private_key_file_set=%t user_data_set=%t packer_debug=%t",
		c.Host,
		authMode,
		c.Comm.Type,
		c.ImageName,
		c.SourceImage,
		c.SourceImageUrl != "",
		c.InstanceName,
		c.L3NetworkName,
		c.L3NetworkUuid != "",
		c.InstanceOfferingName,
		c.InstanceOfferingUuid != "",
		c.BackupStorageName,
		c.BackupStorageUuid != "",
		c.Comm.SSHHost,
		c.Comm.SSHPort,
		c.Comm.SSHUsername,
		c.Comm.SSHPassword != "",
		c.Comm.SSHPrivateKeyFile != "",
		c.UserData != "",
		c.PackerDebug,
	)
}

func (c *Config) Prepare(raws ...any) error {
	err := config.Decode(c, &config.DecodeOpts{
		Interpolate:        true,
		InterpolateContext: &c.ctx,
	}, raws...)
	if err != nil {
		log.Printf("[ERROR] Failed to decode configuration: %v", err)
		return err
	}

	var errs *packersdk.MultiError

	c.AccessConfig.applyEnvDefaults()
	for _, accessErr := range c.AccessConfig.validateCredentials() {
		errs = packersdk.MultiErrorAppend(errs, accessErr)
	}

	if c.SourceImageUrl != "" {
		log.Printf("[INFO] Configuring source image from URL: %s", c.SourceImageUrl)
		if c.Format == "" {
			c.Format = "qcow2"
			log.Printf("[DEBUG] Default image format set to: %s", c.Format)
		}
		if c.Platform == "" {
			c.Platform = "Linux"
			log.Printf("[DEBUG] Default platform set to: %s", c.Platform)
		}
		if c.SourceImage == "" {
			errs = packersdk.MultiErrorAppend(errs, errors.New("source image name must be specified when using source_image_url"))
		}
	}

	if c.ImageReadyTimeoutRaw != "" {
		d, err := time.ParseDuration(c.ImageReadyTimeoutRaw)
		if err != nil {
			errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("image_ready_timeout is invalid: %v", err))
		} else if d <= 0 {
			errs = packersdk.MultiErrorAppend(errs, errors.New("image_ready_timeout must be positive"))
		} else {
			c.imageReadyTimeout = d
		}
	}
	if c.VmRunningTimeoutRaw != "" {
		d, err := time.ParseDuration(c.VmRunningTimeoutRaw)
		if err != nil {
			errs = packersdk.MultiErrorAppend(errs, fmt.Errorf("vm_running_timeout is invalid: %v", err))
		} else if d <= 0 {
			errs = packersdk.MultiErrorAppend(errs, errors.New("vm_running_timeout must be positive"))
		} else {
			c.vmRunningTimeout = d
		}
	}

	if errs != nil && len(errs.Errors) > 0 {
		log.Printf("[ERROR] Configuration validation failed with %d error(s)", len(errs.Errors))
		return errs
	}

	log.Printf("[INFO] Configuration prepared successfully")
	return nil

}
