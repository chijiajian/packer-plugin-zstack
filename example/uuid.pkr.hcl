# Copyright ZStack.io, Inc. 2013, 2026
# SPDX-License-Identifier: MPL-2.0

# UUID passthrough - skip all name-based lookups
variable "zstack_host" {
  type    = string
  default = env("ZSTACK_HOST")
}

variable "access_key_id" {
  type    = string
  default = env("ZSTACK_ACCESS_KEY_ID")
}

variable "access_key_secret" {
  type      = string
  sensitive = true
  default   = env("ZSTACK_ACCESS_KEY_SECRET")
}

variable "source_image_uuid" {
  type = string
}

variable "network_uuid" {
  type = string
}

variable "instance_offering_uuid" {
  type = string
}

packer {
  required_plugins {
    zstack = {
      version = ">= 2.0.0"
      source  = "github.com/zstackio/zstack"
    }
  }
}

source "zstack" "uuid" {
  zstack_host       = var.zstack_host
  access_key_id     = var.access_key_id
  access_key_secret = var.access_key_secret

  # Direct UUID references - no API queries needed
  image_uuid             = var.source_image_uuid
  network_uuid           = var.network_uuid
  instance_offering_uuid = var.instance_offering_uuid

  instance_name     = "packer-uuid"
  image_name        = "packer-uuid-image"
  image_description = "Built with Packer - UUID passthrough example"

  backup_storage_name = "local-backup"

  # Backup storage is required because the builder now creates the final image via snapshot -> image.
  ssh_username = "root"
  ssh_password = "your-ssh-password"
}

build {
  sources = ["source.zstack.uuid"]

  provisioner "shell" {
    inline = [
      "echo 'Built using UUID passthrough'",
    ]
  }
}
