# Copyright ZStack.io, Inc. 2013, 2026
# SPDX-License-Identifier: MPL-2.0

packer {
  required_plugins {
    zstack = {
      version = ">= 1.1.0"
      source  = "github.com/chijiajian/zstack"
    }
  }
}

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
  default   = env("ZSTACK_ACCESS_KEY_SECRET")
  sensitive = true
}

variable "source_image_url" {
  type = string
}

variable "network_uuid" {
  type = string
}

variable "backup_storage_uuid" {
  type = string
}

variable "image_name" {
  type = string
}

variable "ssh_username" {
  type    = string
  default = "root"
}

variable "ssh_password" {
  type      = string
  sensitive = true
}

variable "build_interface" {
  type    = string
  default = "ens3"
}

variable "host_uuid" {
  type    = string
  default = null
}

source "zstack" "generalized" {
  zstack_host         = var.zstack_host
  access_key_id       = var.access_key_id
  access_key_secret   = var.access_key_secret
  source_image_url    = var.source_image_url
  source_image        = "${var.image_name}-source"
  image_name          = var.image_name
  instance_name       = "${var.image_name}-build"
  network_uuid        = var.network_uuid
  backup_storage_uuid = var.backup_storage_uuid
  host_uuid           = var.host_uuid
  cpu_num             = 4
  memory_size         = 8192
  format              = "qcow2"
  platform            = "Linux"
  architecture        = "x86_64"
  ssh_username        = var.ssh_username
  ssh_password        = var.ssh_password
  ssh_timeout         = "15m"
  api_timeout         = "45m"
  image_ready_timeout = "45m"
  skip_export         = true
}

build {
  sources = ["source.zstack.generalized"]

  provisioner "shell" {
    inline = ["install -d -m 700 /tmp/packer-network-generalization"]
  }

  provisioner "file" {
    source      = "${path.root}/network-generalization/"
    destination = "/tmp/packer-network-generalization/"
  }

  # Keep this as the last guest provisioning step. Root login is required.
  provisioner "shell" {
    environment_vars = ["BUILD_INTERFACE=${var.build_interface}"]
    inline           = ["bash /tmp/packer-network-generalization/install.sh", "rm -rf /tmp/packer-network-generalization"]
  }
}
