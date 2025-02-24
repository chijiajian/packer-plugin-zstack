# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

packer {
  required_plugins {
    zstack = {
      version = "v0.0.1"
      source  = "github.com/zstackio/zstack"
    }
  }
}

source "zstack" "test" {
  zstack_host       = "172.30.3.3"
  port              = 8080
#  account_name      = "admin"
#  account_password  = "password"
  access_key_id     = "uxPnrlvM0RK7H53os1Gn"
  access_key_secret = "1UQ3bfz9qS4vG9CmKLtqeNISwOMai1aByPElOBjN"
  source_image = "C88"
  source_image_url = "http://192.168.200.100/mirror/jiajian.chi/martketplace/prometheus-grafana/prometheus-grafana-v4-x86.qcow2"
  format = "qcow2"
  platform = "Linux"
  
  network_name = "public-net"
  instance_offering_name = "min"
  backup_storage_name = "bs"
  instance_name = "packer-test"
  image_name = "image-by-packer-chi-testing"
  ssh_username = "root"
  ssh_password = "ZStack@123"
}

build {
  sources = ["source.zstack.test"]

  provisioner "shell" {
    inline = [
    "mkdir /mnt/cdrom",
    "mount /dev/cdrom /mnt/cdrom",
    "cd /mnt/cdrom/",
    "bash ./zs-tools-install.sh"  
]
  
  }
}
