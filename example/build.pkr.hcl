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
  host              = "172.30.3.3"
  port              = 8080
  account_name      = "admin"
  account_password  = "password"
  access_key_id     = "uxPnrlvM0RK7H53os1Gn"
  access_key_secret = "1UQ3bfz9qS4vG9CmKLtqeNISwOMai1aByPElOBjN"
}

build {
  sources = ["source.zstack.test"]
}
