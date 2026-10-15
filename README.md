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
      version = ">= 1.1.0"
      source  = "github.com/chijiajian/zstack"
    }
  }
}
```

## Quick Start

The plugin supports two authentication methods:

- Account/password authentication with `ZSTACK_HOST`, `ZSTACK_ACCOUNT_NAME`, and `ZSTACK_ACCOUNT_PASSWORD`
- Access key/secret authentication with `ZSTACK_HOST`, `ZSTACK_ACCESS_KEY_ID`, and `ZSTACK_ACCESS_KEY_SECRET`

You can build from resource names or use UUID passthrough with `image_uuid`, `network_uuid`, and `instance_offering_uuid` to skip name-based lookups.

Backup storage is required for image creation and export. In the normal VM build path, the builder stops the VM and creates the image template directly from the stopped root volume. When `source_volume_snapshot_uuid` is set, the builder creates the template from that existing volume snapshot instead.

Use `image_description` to set a custom description for the generated image.

The `zstack-snapshot` builder creates an image directly from an existing **root-volume snapshot** using `source_volume_snapshot_uuid`. This new builder requires a local build until released; v1.2.0 does not include it. The legacy `zstack` snapshot input remains supported with a deprecation warning in newer builds. In this mode no VM is created and SSH/provisioners are skipped; the builder creates the image template from that existing snapshot — see [`examples/from_snapshot.pkr.hcl`](examples/from_snapshot.pkr.hcl).

See the [`examples/`](examples) directory for ready-to-run HCL examples covering account/password auth, AK/SK auth, and UUID passthrough.

## E2E Test Method

Use the basic template to verify VM creation, SSH provisioning and image creation from the root volume. The optional shell script checks the guest environment; the configuration below also demonstrates URL-based image import:

- Template: [`examples/basic.pkr.hcl`](examples/basic.pkr.hcl)
- Provisioner script: [`examples/load_images.sh`](examples/load_images.sh)

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
    script = "examples/load_images.sh"
  }
}
```

### Run

```bash
PACKER_PLUGIN_PATH=$(pwd) packer validate examples/basic.pkr.hcl
PACKER_PLUGIN_PATH=$(pwd) packer build examples/basic.pkr.hcl
```

### Backup Storage / Export Behavior

- `backup_storage_name` or `backup_storage_uuid` is required for both normal VM builds and `source_volume_snapshot_uuid` builds, because the generated image is stored in backup storage and may be exported from it.
- If backup storage does not support image export, export is skipped with a warning instead of failing the entire build.

### Configuration

For more information on how to configure the plugin, please read the
documentation located in the [`docs/`](docs) directory.

## Configuration and lifecycle

See the [ZStack builder guide](docs/builders/zstack.mdx) for authentication, SSH/WinRM, networking, placement, timeouts and resource cleanup. Output images contain the root disk only; existing source images and snapshots are retained.

## Installing an operating system from ISO

Use `source "zstack-iso"` with a blank root-disk offering and installer answer files over HTTP or CD. The installer configures SSH/WinRM credentials; ZStack userdata is not used. The builder verifies the installed OS again after detaching installation media and restarting from disk.

See the [ISO builder guide](docs/builders/iso.mdx) and [unattended installation examples](examples/iso/README.md) for Debian, Ubuntu, Rocky ARM and Windows.

## Development and testing

Run `make test plugin-check` for local tests and plugin validation. Run `make generate` to regenerate the HCL schema and public web documentation. For opt-in tests against a real cloud, see the [acceptance guide](docs/testing.md).

Version-specific changes are listed in [GitHub Releases](https://github.com/chijiajian/packer-plugin-zstack/releases).

## Contributing

* If you think you've found a bug in the code or you have a question regarding
  the usage of this software, please reach out to us by opening an issue in
  this GitHub repository.
* Contributions to this project are welcome: if you want to add a feature or a
  fix a bug, please do so by opening a Pull Request in this GitHub repository.
  In case of feature contribution, we kindly ask you to open an issue to
  discuss it beforehand.
