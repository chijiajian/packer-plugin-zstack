# AccessKey/Secret authentication with URL-based source image
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

packer {
  required_plugins {
    zstack = {
      version = ">= 2.0.0"
      source  = "github.com/zstackio/zstack"
    }
  }
}

source "zstack" "aksk" {
  zstack_host       = var.zstack_host
  access_key_id     = var.access_key_id
  access_key_secret = var.access_key_secret

  source_image     = "ubuntu-22.04-from-url"
  source_image_url = "http://mirrors.example.com/images/ubuntu-22.04-server.qcow2"
  format           = "qcow2"
  platform         = "Linux"
  guest_os_type    = "Ubuntu"

  network_name        = "default-network"
  backup_storage_name = "local-backup"

  # Use cpu_num and memory_size instead of instance_offering_name
  cpu_num     = 2
  memory_size = 4096  # in MB

  instance_name     = "packer-aksk"
  image_name        = "packer-aksk-image"
  image_description = "Built with Packer - AK/SK + URL image example"

  ssh_username = "root"
  ssh_password = "your-ssh-password"
}

build {
  sources = ["source.zstack.aksk"]

  provisioner "shell" {
    inline = [
      "echo 'Provisioning from URL-based image'",
    ]
  }
}
