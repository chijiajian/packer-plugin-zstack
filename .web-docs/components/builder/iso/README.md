
Type: `zstack-iso`

Installs an operating system from ISO onto a blank root disk, provisions the installed guest and creates a ZStack root-volume template. Use installer answer files over HTTP or an auxiliary CD to configure the OS, account and SSH/WinRM. Installation does not depend on ZStack userdata or automatic cloud SSH-key injection.

## Required configuration

Use the authentication, networking, compute and backup-storage settings from the [ZStack builder](/packer/integrations/chijiajian/zstack/latest/components/builder/zstack), plus:

| Setting | Purpose |
| --- | --- |
| `root_disk_offering_uuid` | Blank installation disk; choose sufficient capacity for the OS |
| `image_uuid` or `source_image` | Existing Enabled/Ready ISO, selected by UUID or name |
| `source_image_url` with `source_image` and `format = "iso"` | Alternative: let ZStack download the installer to backup storage |
| `communicator` | `ssh` or `winrm`; `none` is not allowed |
| Installation credential | SSH password/private key or WinRM credential configured by the answer file |
| `installation_check_command` | Command that succeeds only in the installed OS; check an installation marker, expected OS and firmware |

Also set the output `image_name` and build `instance_name`. Snapshot sources, `userdata` and `sshkey` are incompatible with ISO installation. The default `zstack` builder can also enable this mode with `iso_install = true`.

## Firmware and answer files

| Setting | Behavior |
| --- | --- |
| `boot_mode` | `Legacy` or `UEFI`; unset uses the image/cloud default. Applied to installer import and VM creation |
| `http_directory` / `http_content` | Serve installer answer files through Packer's HTTP server; choose one source |
| `http_ip` | Explicit reachable runner IP when automatic selection is unsuitable |
| `http_bind_address`, `http_port_min`, `http_port_max` | HTTP listener address and port range |
| `cd_files` / `cd_content` / `cd_label` | Generate an auxiliary answer ISO from local files/content and give it the installer's required label |
| `boot_command` / `boot_wait` | Installer-specific VNC keys and initial delay |
| `boot_key_interval` / `boot_keygroup_interval` | Delay between keys or key groups |

Boot commands require access to the host's TCP VNC console. HTTP answers must be reachable from the guest. Generated answer CDs require a local ISO creation tool and a separate temporary HTTP publisher reachable by backup storage; that publisher uses an automatically allocated port. Source URLs must also be reachable by ZStack's backup storage.

ZStack rejected imports below 1 MiB in the tested environment. The plugin automatically pads generated answer ISOs to at least 1 MiB with trailing zeros, preserving the files and volume label. Users do not need to enlarge small answer files themselves. This padding applies to generated answer CDs, not arbitrary source URLs.

Answer syntax and boot commands belong to the installer: Debian Preseed, Ubuntu autoinstall, Rocky Kickstart and Windows Autounattend need different settings. Ubuntu's `cidata` CD is read by the installer itself; it is not a ZStack userdata payload. Match CD device names/drive letters, disk partitioning and ISO labels to your media.

## Build and cleanup

1. Import or select the installer ISO, create a VM with a blank root disk, and attach any generated answer CD.
2. Send boot commands and wait for the installer-configured SSH/WinRM connection.
3. Run `installation_check_command`, stop the VM, detach both installation media, select hard-disk boot, and reconnect/check again.
4. Run provisioners, stop the VM, create the root-volume template, wait for Ready and export when supported.
5. Clean build-owned temporary resources. Existing installer images are retained; `keep_source_image = true` retains a URL-imported installer. Generated answer CDs are always cleaned.

The two installation checks prevent an installer shell or VM Running status from being mistaken for a successful OS installation. Use the regular builder's API/readiness/cleanup timeouts and retention options; increase `image_ready_timeout` for large ISO imports. Cleanup errors include resource UUIDs for investigation.

## Examples and tested combinations

See the [ISO examples](https://github.com/chijiajian/packer-plugin-zstack/tree/main/examples/iso) for runnable templates and answer files:

| OS | Architecture / firmware | Connection |
| --- | --- | --- |
| Debian 13.7 | amd64 / BIOS and UEFI | SSH |
| Ubuntu Server 24.04 LTS | amd64 / UEFI | SSH |
| Rocky Linux 9.4 | ARM64 / UEFI | SSH |
| Windows Server 2022 | amd64 / UEFI | HTTPS WinRM |

Real acceptance covered unattended installation, hard-disk reboot, file transfer, root-volume template creation, export deletion and a new VM booted from the output template without a firmware override. Debian/Ubuntu/Windows used Ceph; Rocky ARM used LocalStorage with ImageStore. These results do not certify other OS/storage combinations or Windows Sysprep/generalize.

Windows requires compatible signed VirtIO storage/network drivers. The example uses HTTPS WinRM and restricts its firewall rule to the runner. Rotate installation credentials and perform any required image generalization before distributing templates. Do not commit rendered answer files containing credentials.
