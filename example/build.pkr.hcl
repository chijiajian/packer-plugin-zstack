# Copyright (c) ZStack.io, Inc.
# SPDX-License-Identifier: MPL-2.0

packer {
  required_plugins {
    zstack = {
      version = "v1.0.1"
      source  = "github.com/chijiajian/zstack"
    }
  }
}

source "zstack" "test" {
  zstack_host       = "172.30.3.2"
  port              = 8080
  account_name      = "admin"
  account_password  = "password"
 # or using access key and secret to auth
 # access_key_id     = "key id"
 # access_key_secret = "secret"
  source_image = "C88"
  source_image_url = "http://image_url/mirror/images/martketplace/prometheus-grafana/prometheus-grafana-v4-x86.qcow2"
  format = "qcow2"
  platform = "Linux"
  
  network_name = "test"
  instance_offering_name = "min"
  backup_storage_name = "test"
  instance_name = "packer-test"
  image_name = "image-by-packer-chi-testing"
  ssh_username = "root"
  ssh_password = "ssh password"
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
