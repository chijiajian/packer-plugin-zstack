# Examples

Choose one template for your build. The top-level HCL files are independent examples and repeat variable names; run Packer against an individual file rather than the entire `examples/` directory.

| Template | Purpose |
| --- | --- |
| [basic.pkr.hcl](basic.pkr.hcl) | Account/password authentication, an existing source image, and shell provisioning |
| [aksk.pkr.hcl](aksk.pkr.hcl) | Access-key authentication and URL-based image import |
| [uuid.pkr.hcl](uuid.pkr.hcl) | Select resources by UUID |
| [from_snapshot.pkr.hcl](from_snapshot.pkr.hcl) | Use `zstack-snapshot` to convert a root-volume snapshot (requires v1.2.1 or later) |
| [advanced.pkr.hcl](advanced.pkr.hcl) | Multiple networks, placement, timeouts and cleanup settings |
| [helix-kylin.pkr.hcl](helix-kylin.pkr.hcl) | Opt-in Helix/Kylin DHCP network generalization; see [scope and lifecycle](network-generalization/README.md) |
| [iso/](iso/README.md) | Install Debian, Ubuntu, Rocky ARM or Windows from ISO using unattended answer files |
| [load_images.sh](load_images.sh) | Optional shell provisioner for checking the guest environment |

Replace resource names/UUIDs with values from your environment. Load credentials from an ignored local `.env`, and supply template variables through `PKR_VAR_*` or an ignored variable file. Plugin version requirements are declared in each template; features added in the current source require a compatible local build or release.

For example, run these commands from the repository root after supplying the required variables:

```sh
packer init examples/basic.pkr.hcl
packer validate examples/basic.pkr.hcl
packer build examples/basic.pkr.hcl
```

`examples/local-dev.pkr.hcl` is an optional private local template and is ignored by Git. Follow [iso/README.md](iso/README.md) for ISO-specific requirements and templates.

## Advanced lifecycle and placement

`advanced.pkr.hcl` demonstrates multiple networks, temporary-key cleanup, bounded API/cleanup timeouts and retention settings. These fields require plugin v1.2.0 or later; run `packer init examples/advanced.pkr.hcl` to install a compatible release.

Load credentials from an ignored local `.env` before running Packer. Supply `image_uuid`, `network_uuids`, and `backup_storage_uuid` as variables. Set `ssh_username` to the source image's cloud-init user. The output contains only the root disk; any data disks created through `data_disk_offering_uuids` are temporary and are deleted during cleanup.
