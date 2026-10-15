# Requires ZStack plugin v1.2.0 or later.
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

source "zstack-iso" "ubuntu" {
  source_image_url        = var.iso_url
  source_image            = "packer-ubuntu-installer"
  instance_name           = "packer-ubuntu-install"
  image_name              = "packer-ubuntu-template"
  format                  = "iso"
  platform                = "Linux"
  architecture            = "x86_64"
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
  ssh_username            = "packer"
  ssh_password            = var.ssh_password
  ssh_timeout             = "35m"
  cd_label                = "cidata"
  cd_content = {
    "user-data" = templatefile("../ubuntu-autoinstall.pkrtpl", { password_hash = var.password_hash })
    "meta-data" = "instance-id: packer-ubuntu-iso\nlocal-hostname: packer-ubuntu\n"
  }
  boot_command               = ["c<wait>linux /casper/vmlinuz autoinstall ds=nocloud ---<enter><wait>initrd /casper/initrd<enter><wait>boot<enter>"]
  installation_check_command = "test -f /etc/packer-installed && test -d /sys/firmware/efi && . /etc/os-release && test \"$ID\" = ubuntu && test \"$VERSION_ID\" = 24.04"
}

build {
  sources = ["source.zstack-iso.ubuntu"]
}
