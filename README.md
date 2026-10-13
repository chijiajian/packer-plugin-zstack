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

The plugin also supports building an image directly from an existing **volume snapshot** by setting `source_volume_snapshot_uuid`. In this mode no VM is created and SSH/provisioners are skipped; the builder creates the image template from that existing snapshot — see [`example/from_snapshot.pkr.hcl`](example/from_snapshot.pkr.hcl).

See the [`example/`](example) directory for ready-to-run HCL examples covering account/password auth, AK/SK auth, and UUID passthrough.

## E2E Test Method

Use the local E2E template and shell script to verify the full build flow (image import, VM create, SSH provision, VM stop, image create from root volume):

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

### Backup Storage / Export Behavior

- `backup_storage_name` or `backup_storage_uuid` is required for both normal VM builds and `source_volume_snapshot_uuid` builds, because the generated image is stored in backup storage and may be exported from it.
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

## Quality fixes and upgrade (2026-10)

This development version uses **Go 1.26.9**, **ZStack SDK v0.0.10**, and **Packer Plugin SDK v0.6.12**. See the [task and acceptance record](docs/stories/STORY-PKR-QUALITY-20261013.md). The `go-cty` replacement remains necessary for the SDK's HCL generator.

Environment variables are defaults; explicit configuration wins. Use one authentication method. Copy `.env.example` to `.env`, restrict it with `chmod 600 .env`, and load it with `set -a; source .env; set +a`. Packer does not load dotenv files itself. `.env` and `.acceptance/` are ignored by Git.

Failed or cancelled builds now return no artifact. Only a newly created, ready output image is owned by the artifact; existing source images and snapshots are never deleted. `Files()` is empty for remote images; use `Id()`, `State("image_uuid")`, and `State("image_url")`. Temporary VMs and their newly created data disks are destroyed and expunged. Imported source images are cleaned unless `keep_source_image = true`; failed output images are cleaned unless `keep_image_on_error = true`. Cleanup failures report resource UUIDs and appear in `State("cleanup_warnings")`. A server-side asynchronous create can outlive cancellation: inspect the reported UUID before retrying.

`api_timeout` defaults to `30m`; `cleanup_timeout` defaults to `5m`. `image_ready_timeout` and `vm_running_timeout` default to `5m` and now also bound their corresponding create requests. Increase these for large images. The plugin keeps SDK models but uses cancellable HTTP requests and job polling because the SDK transport does not bind its context. `skip_export = true` disables export. Only explicit unsupported-export errors are downgraded; permission and storage errors fail the build.

Optional placement and disk settings are `zone_uuid`, `cluster_uuid`, `host_uuid`, `root_disk_offering_uuid`, and `data_disk_offering_uuids`. `network_uuids` enables multiple NICs and is exclusive with the singular network selectors; `default_network_uuid` chooses the default NIC. Output templates contain the root disk only. Required inputs, conflicting sources and duplicate resource names are validated early. Snapshot sources must be Ready, Enabled root-volume snapshots, with communicator `none`.

The generated HCL schema includes the full SSH/WinRM options, including bastion and WinRM TLS/port. Use `ssh_clear_authorized_keys = true` to remove the temporary public key before creating the image. Existing private-key and agent authentication are preserved.

Run `make test plugin-check` locally and `go generate ./...` to regenerate the HCL schema. Cloud acceptance is opt-in: set `ZSTACK_ACC=1`, `ZSTACK_ACC_IMAGE_UUID`, `ZSTACK_ACC_NETWORK_UUID`, `ZSTACK_ACC_BACKUP_STORAGE_UUID`, and the usual credentials, then run `go test ./builder/zstack -run '^TestAcceptanceCloud$' -v -count=1 -timeout=35m`. This creates temporary cloud resources and deletes its output image. Set `ZSTACK_ACC_COMMUNICATOR=ssh` and `ZSTACK_ACC_SSH_USERNAME` to exercise SSH; otherwise the communicator is disabled.

For URL-import acceptance, set `ZSTACK_ACC_IMAGE_URL` instead of `ZSTACK_ACC_IMAGE_UUID` (exactly one is required). With SSH enabled, `ZSTACK_ACC_SSH_PASSWORD` selects password authentication; leave it empty to exercise generated-key authentication. Keep image URLs and credentials in the ignored local `.env`. The test checks the actual SSH user, runs system commands through the provision hook, and verifies that its imported source image is removed.

Generated-key authentication requires a guest image that processes the cloud's SSH-key injection (for example, enabled and correctly configured cloud-init) and the matching SSH user. A successful password login does not establish that generated-key injection works. Use an existing password/private key for images with cloud-init disabled.
