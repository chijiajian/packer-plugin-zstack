# Build from an existing image. Credentials use the ZSTACK_* environment defaults.
packer {
  required_plugins {
    zstack = {
      version = ">= 1.1.0"
      source  = "github.com/chijiajian/zstack"
    }
  }
}
variable "image_uuid" { type = string }
variable "network_uuids" { type = list(string) }
variable "backup_storage_uuid" { type = string }
variable "ssh_username" { default = "ubuntu" }

source "zstack" "advanced" {
  image_uuid                = var.image_uuid
  network_uuids             = var.network_uuids
  default_network_uuid      = var.network_uuids[0]
  backup_storage_uuid       = var.backup_storage_uuid
  instance_name             = "packer-advanced"
  image_name                = "packer-advanced-output"
  cpu_num                   = 2
  memory_size               = 2048
  ssh_username              = var.ssh_username
  ssh_clear_authorized_keys = true
  api_timeout               = "30m"
  image_ready_timeout       = "20m"
  vm_running_timeout        = "10m"
  cleanup_timeout           = "5m"
  keep_source_image         = false
  keep_image_on_error       = false
  skip_export               = false
  # Optional placement / disks:
  # cluster_uuid = "..."
  # host_uuid = "..."
  # root_disk_offering_uuid = "..."
  # data_disk_offering_uuids = ["..."]
}
build {
  sources = ["source.zstack.advanced"]
  provisioner "shell" { inline = ["uname -m"] }
}
