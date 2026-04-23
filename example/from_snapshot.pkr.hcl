# Build an image directly from an existing ZStack volume snapshot (no VM/SSH/provisioning).
variable "zstack_host" {
  type    = string
  default = env("ZSTACK_HOST")
}

variable "account_name" {
  type    = string
  default = env("ZSTACK_ACCOUNT_NAME")
}

variable "account_password" {
  type      = string
  sensitive = true
  default   = env("ZSTACK_ACCOUNT_PASSWORD")
}

variable "snapshot_uuid" {
  type        = string
  description = "UUID of the volume snapshot to convert into an image"
}

packer {
  required_plugins {
    zstack = {
      version = ">= 2.0.0"
      source  = "github.com/zstackio/zstack"
    }
  }
}

source "zstack" "from_snapshot" {
  zstack_host      = var.zstack_host
  account_name     = var.account_name
  account_password = var.account_password

  source_volume_snapshot_uuid = var.snapshot_uuid

  image_name          = "packer-from-snapshot-image"
  image_description   = "Built with Packer from a ZStack volume snapshot"
  platform            = "Linux"
  architecture        = "x86_64"
  backup_storage_name = "local-backup"
}

build {
  sources = ["source.zstack.from_snapshot"]
}
