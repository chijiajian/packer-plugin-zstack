# Requires plugin v1.2.0 or later with ISO/boot_mode support.
packer {
  required_plugins {
    zstack = {
      source  = "github.com/chijiajian/zstack"
      version = "= 1.2.0"
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
variable "password_hash" {
  type      = string
  sensitive = true
}

source "zstack-iso" "rocky" {
  source_image_url        = var.iso_url
  source_image            = "packer-rocky-installer"
  instance_name           = "packer-rocky-install"
  image_name              = "packer-rocky-template"
  format                  = "iso"
  platform                = "Linux"
  architecture            = "aarch64"
  boot_mode               = "UEFI"
  network_uuid            = var.network_uuid
  backup_storage_uuid     = var.backup_storage_uuid
  root_disk_offering_uuid = var.root_disk_offering_uuid
  cpu_num                 = 2
  memory_size             = 4096
  http_ip                 = var.http_ip
  boot_wait               = "15s"
  boot_key_interval       = "15ms"
  image_ready_timeout     = "20m"
  communicator            = "ssh"
  ssh_username            = "root"
  ssh_password            = var.ssh_password
  ssh_timeout             = "35m"
  http_content = {
    "/ks.cfg" = templatefile("../rocky.ks.pkrtpl", { root_password_hash = var.password_hash })
  }
  boot_command               = ["c<wait>linux /images/pxeboot/vmlinuz inst.ks=http://{{ .HTTPIP }}:{{ .HTTPPort }}/ks.cfg inst.stage2=hd:LABEL=Rocky-9-4-aarch64-dvd ip=dhcp console=tty0<enter><wait>initrd /images/pxeboot/initrd.img<enter><wait>boot<enter>"]
  installation_check_command = "test -f /etc/packer-installed && test -d /sys/firmware/efi"
}

build {
  sources = ["source.zstack-iso.rocky"]
}
