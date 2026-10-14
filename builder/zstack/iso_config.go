// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0
package zstack

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/bootcommand"
)

type ISOConfig struct {
	BootKeyInterval          time.Duration `mapstructure:"boot_key_interval"`
	ISOInstall               bool          `mapstructure:"iso_install"`
	bootcommand.BootConfig   `mapstructure:",squash"`
	CDFiles                  []string          `mapstructure:"cd_files"`
	CDContent                map[string]string `mapstructure:"cd_content"`
	CDLabel                  string            `mapstructure:"cd_label"`
	HTTPIP                   string            `mapstructure:"http_ip"`
	InstallationCheckCommand string            `mapstructure:"installation_check_command"`
}

// ISOBuilder exposes zstack-iso while retaining the shared lifecycle implementation.
type ISOBuilder struct{ Builder }

func (b *ISOBuilder) Prepare(raws ...any) ([]string, []string, error) {
	raws = append(raws, map[string]any{"iso_install": true})
	return b.Builder.Prepare(raws...)
}
func (c *Config) prepareISO() []error {
	var errs []error
	add := func(s string) { errs = append(errs, fmt.Errorf("ISO build: %s", s)) }
	if !c.ISOInstall {
		if strings.EqualFold(c.Format, "iso") {
			add("format = iso requires iso_install or zstack-iso")
		}
		if len(c.CDFiles) > 0 || len(c.CDContent) > 0 || len(c.BootCommand) > 0 {
			add("cd_files/cd_content/boot_command require iso_install or zstack-iso")
		}
		return errs
	}
	if c.Comm.SSHTimeout == 0 {
		c.Comm.SSHTimeout = 30 * time.Minute
	}
	if c.Comm.WinRMTimeout == 0 {
		c.Comm.WinRMTimeout = 30 * time.Minute
	}
	if c.Platform == "" && c.SourceImageUrl != "" {
		c.Platform = "Linux"
	}
	if c.BootGroupInterval < 0 {
		add("boot_keygroup_interval must not be negative")
	}
	if c.BootKeyInterval < 0 {
		add("boot_key_interval must not be negative")
	}
	for path := range c.CDContent {
		clean := filepath.Clean(path)
		if filepath.IsAbs(path) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			add("cd_content paths must stay inside the CD root")
		}
	}
	if c.SourceVolumeSnapshotUuid != "" {
		add("snapshot source is incompatible")
	}
	if c.RootDiskOfferingUuid == "" {
		add("root_disk_offering_uuid is required for the blank installation disk")
	}
	if c.UserData != "" || c.SshKey != "" {
		add("userdata and sshkey are not used; configure installation credentials in the answer file")
	}
	if c.Comm.Type == "none" {
		add("communicator must be ssh or winrm")
	}
	if c.Comm.Type == "" || c.Comm.Type == "ssh" {
		if c.Comm.SSHPassword == "" && c.Comm.SSHPrivateKeyFile == "" {
			add("ssh_password or ssh_private_key_file is required; the installer must configure this credential")
		}
	}
	if c.InstallationCheckCommand == "" {
		add("installation_check_command is required to distinguish the installed OS from the installer")
	}
	if c.SourceImageUrl != "" && c.Format != "iso" {
		add("format must be iso")
	}
	if c.HTTPIP != "" {
		ip := net.ParseIP(c.HTTPIP)
		if ip == nil || ip.IsUnspecified() || ip.IsMulticast() {
			add("http_ip must be a usable IP address reachable by the VM and backup storage")
		}
	}
	for _, path := range c.CDFiles {
		if _, err := os.Stat(path); err != nil { // globs are handled by StepCreateCD
			matches, _ := filepath.Glob(path)
			if len(matches) == 0 {
				add("cd_files path does not exist: " + path)
			}
		}
	}
	errs = append(errs, c.HTTPConfig.Prepare(&c.ctx)...)
	if c.HTTPNetworkProtocol != "tcp" && c.HTTPNetworkProtocol != "tcp4" && c.HTTPNetworkProtocol != "tcp6" {
		add("http_network_protocol must use TCP")
	}
	errs = append(errs, c.BootConfig.Prepare(&c.ctx)...)
	return errs
}
