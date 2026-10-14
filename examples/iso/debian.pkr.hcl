# Requires a locally installed build of this branch or a release containing zstack-iso.
# Install Debian from a standard BIOS installer ISO. Verify its SHA256 against
# the publisher's checksums before making it available to ZStack.
packer {
  required_plugins {
    zstack = {
      source  = "github.com/chijiajian/zstack"
      version = "= 1.0.1-dev"
    }
  }
}

variable "iso_url" { type = string }
variable "network_uuid" { type = string }
variable "backup_storage_uuid" { type = string }
variable "root_disk_offering_uuid" { type = string }
variable "http_ip" { type = string }
variable "ssh_password" {
  type      = string
  sensitive = true
}
# SHA-512 crypt hash of the same installation password; generate locally.
variable "root_password_hash" {
  type      = string
  sensitive = true
}

source "zstack-iso" "debian" {
  source_image_url        = var.iso_url
  source_image            = "packer-debian-installer"
  instance_name           = "packer-debian-install"
  image_name              = "packer-debian-template"
  format                  = "iso"
  platform                = "Linux"
  architecture            = "x86_64"
  network_uuid            = var.network_uuid
  backup_storage_uuid     = var.backup_storage_uuid
  root_disk_offering_uuid = var.root_disk_offering_uuid
  cpu_num                 = 2
  memory_size             = 2048

  # The guest must reach this address; backup storage also needs it for cd_content.
  http_ip = var.http_ip
  http_content = {
    "/preseed.cfg" = templatefile("preseed.cfg.pkrtpl", {
      root_password_hash = var.root_password_hash
    })
  }
  cd_label          = "PACKER_ANSWER"
  cd_files          = ["${path.root}/post-install.sh"]
  boot_wait         = "10s"
  boot_key_interval = "30ms"
  boot_command = [
    "<esc><wait>auto auto=true priority=critical ",
    "url=http://{{ .HTTPIP }}:{{ .HTTPPort }}/preseed.cfg ",
    "netcfg/choose_interface=auto netcfg/get_hostname=packer-debian netcfg/get_domain=local --- <enter>"
  ]
  installation_check_command = "test -f /etc/packer-installed && test -f /etc/packer-answer-cd-used && test \"$(findmnt -n -o FSTYPE /)\" = ext4"
  communicator               = "ssh"
  ssh_username               = "root"
  ssh_password               = var.ssh_password
  ssh_timeout                = "35m"
  image_ready_timeout        = "20m"
}

build {
  sources = ["source.zstack-iso.debian"]
  provisioner "shell" {
    inline = ["cat /etc/os-release", "uname -m"]
  }
}
