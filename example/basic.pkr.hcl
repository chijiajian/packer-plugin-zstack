# Account/password authentication with source image name lookup
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

variable "ssh_password" {
  type      = string
  sensitive = true
}

packer {
  required_plugins {
    zstack = {
      version = ">= 2.0.0"
      source  = "github.com/zstackio/zstack"
    }
  }
}

source "zstack" "basic" {
  zstack_host      = var.zstack_host
  account_name     = var.account_name
  account_password = var.account_password

  source_image           = "CentOS7.9"
  network_name           = "default-network"
  instance_offering_name = "min-offering"
  instance_name          = "packer-basic"
  image_name             = "packer-basic-image"
  image_description      = "Built with Packer - basic example"

  backup_storage_name = "local-backup"

  ssh_username = "root"
  ssh_password = var.ssh_password
}

build {
  sources = ["source.zstack.basic"]

  provisioner "shell" {
    inline = [
      "echo 'Hello from Packer!'",
      "yum update -y",
    ]
  }
}
