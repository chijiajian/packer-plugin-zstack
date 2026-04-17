# Packer Plugin ZStack

The `ZStack` multi-component plugin can be used with HashiCorp [Packer](https://www.packer.io)
to create custom images from ZStack VM instances. For the full list of available features for this plugin see [docs](docs).

## Installation

### Using pre-built releases

#### Using the `packer init` command

Starting from version 1.7, Packer supports a new `packer init` command allowing
automatic installation of Packer plugins. Read the
[Packer documentation](https://www.packer.io/docs/commands/init) for more information.

To install this plugin, copy and paste this code into your Packer configuration. Then, run [`packer init`](https://www.packer.io/docs/commands/init).

```hcl
packer {
  required_plugins {
    zstack = {
      version = ">= 2.0.0"
      source  = "github.com/zstackio/zstack"
    }
  }
}
```

## Quick Start

The plugin supports two authentication methods:

- Account/password authentication with `ZSTACK_HOST`, `ZSTACK_ACCOUNT_NAME`, and `ZSTACK_ACCOUNT_PASSWORD`
- Access key/secret authentication with `ZSTACK_HOST`, `ZSTACK_ACCESS_KEY_ID`, and `ZSTACK_ACCESS_KEY_SECRET`

You can build from resource names or use UUID passthrough with `image_uuid`, `network_uuid`, and `instance_offering_uuid` to skip name-based lookups.

Optional backup storage settings let you export the resulting image when `backup_storage_name` or `backup_storage_uuid` is configured; if neither is set, the export step is skipped.

Use `image_description` to set a custom description for the generated image.

See the [`example/`](example) directory for ready-to-run HCL examples covering account/password auth, AK/SK auth, and UUID passthrough.

## E2E Test Method

Use the local E2E template and shell script to verify the full build flow (image import, VM create, SSH provision, image create):

- Template: [`example/local-dev.pkr.hcl`](example/local-dev.pkr.hcl)
- Provisioner script: [`example/load_images.sh`](example/load_images.sh)

### Recommended Test Config

For `source_image_url` imports, configure backup storage by **name** whenever possible. The plugin resolves `backup_storage_name` to UUID automatically, so users do not need to memorize UUID values.

```hcl
source "zstack" "e2e-test" {
  zstack_host      = "cloud ip address"
  port             = 8080
  account_name     = "admin"
  account_password = "pwd"

  source_image     = "docker-by-packer-image-compressed"
  source_image_url = "http://imageUrl/packer/docker-by-packer-image-compressed.qcow2"
  format           = "qcow2"
  platform         = "Linux"

  network_name           = "l3-public"
  instance_offering_name = "medium-vm"
  backup_storage_name    = "sftp-bs"

  instance_name = "packer-e2e-test"
  image_name    = "packer-e2e-test-image"
  ssh_username  = "root"
  ssh_password  = "pwd"
}

build {
  sources = ["source.zstack.e2e-test"]

  provisioner "shell" {
    script = "example/load_images.sh"
  }
}
```

### Run

```bash
PACKER_PLUGIN_PATH=$(pwd) packer validate example/local-dev.pkr.hcl
PACKER_PLUGIN_PATH=$(pwd) packer build example/local-dev.pkr.hcl
```

### Export Behavior

- If `backup_storage_name`/`backup_storage_uuid` is missing while `source_image_url` is set, validation fails early.
- If backup storage does not support image export, export is skipped with a warning instead of failing the entire build.

### Configuration

For more information on how to configure the plugin, please read the
documentation located in the [`docs/`](docs) directory.


## Contributing

* If you think you've found a bug in the code or you have a question regarding
  the usage of this software, please reach out to us by opening an issue in
  this GitHub repository.
* Contributions to this project are welcome: if you want to add a feature or a
  fix a bug, please do so by opening a Pull Request in this GitHub repository.
  In case of feature contribution, we kindly ask you to open an issue to
  discuss it beforehand.
