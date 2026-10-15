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
variable "admin_password" {
  type      = string
  sensitive = true
}
# Directory must be named drivers, containing viostor/vioscsi/NetKVM subdirectories.
variable "virtio_drivers_directory" { type = string }

source "zstack-iso" "windows" {
  source_image_url        = var.iso_url
  source_image            = "packer-windows-installer"
  instance_name           = "packer-windows-install"
  image_name              = "packer-windows-template"
  format                  = "iso"
  platform                = "WindowsVirtio"
  architecture            = "x86_64"
  boot_mode               = "UEFI"
  network_uuid            = var.network_uuid
  backup_storage_uuid     = var.backup_storage_uuid
  root_disk_offering_uuid = var.root_disk_offering_uuid
  cpu_num                 = 2
  memory_size             = 4096
  http_ip                 = var.http_ip
  boot_wait               = "1s"
  boot_key_interval       = "15ms"
  image_ready_timeout     = "20m"
  communicator            = "winrm"
  winrm_username          = "Administrator"
  winrm_password          = var.admin_password
  winrm_use_ssl           = true
  winrm_use_ntlm          = true
  winrm_insecure          = true
  winrm_timeout           = "40m"
  cd_label                = "PACKER_ANSWER"
  cd_files                = [var.virtio_drivers_directory]
  cd_content = {
    "Autounattend.xml" = templatefile("../windows-Autounattend.xml.pkrtpl", {
      admin_password_xml   = replace(replace(replace(var.admin_password, "&", "&amp;"), "<", "&lt;"), ">", "&gt;")
      winrm_client_address = var.http_ip
    })
    "setup.ps1"          = file("../windows-setup.ps1")
    "verify-install.ps1" = file("../windows-verify-install.ps1")
  }
  boot_command               = ["<spacebar><wait><spacebar><wait><spacebar><wait><spacebar><wait><spacebar><wait><spacebar><wait><spacebar><wait><spacebar><wait><spacebar><wait><spacebar><wait><spacebar><wait><spacebar><wait><spacebar><wait><spacebar><wait><spacebar><wait><enter>"]
  installation_check_command = "powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass -File C:\\ProgramData\\Packer\\verify-install.ps1"
}

build {
  sources = ["source.zstack-iso.windows"]
}
