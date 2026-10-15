# ZStack Packer Plugin

Build custom ZStack images from an existing VM image, a volume snapshot, or an unattended operating-system installation from ISO. The output is a root-volume template stored in ZStack; supported backup storage can also provide an export URL.

## Installation

Declare `github.com/chijiajian/zstack` in Packer's `required_plugins` block, select a plugin version containing the features you need, and run `packer init`. Use v1.2.0 or later for ISO builds and v1.2.1 or later for the dedicated snapshot builder.

```hcl
packer {
  required_plugins {
    zstack = {
      source  = "github.com/chijiajian/zstack"
      version = ">= 1.2.1"
    }
  }
}
```

Authenticate with either `ZSTACK_ACCESS_KEY_ID`/`ZSTACK_ACCESS_KEY_SECRET` or `ZSTACK_ACCOUNT_NAME`/`ZSTACK_ACCOUNT_PASSWORD`, together with `ZSTACK_HOST`. Explicit HCL values override environment defaults. Load local `.env` files yourself; Packer does not load them automatically.

## Builders

- [ZStack](/packer/integrations/chijiajian/zstack/latest/components/builder/zstack) (`zstack`): provision a VM from an existing image and create a template from its root disk.
- [ZStack ISO](/packer/integrations/chijiajian/zstack/latest/components/builder/iso) (`zstack-iso`): install an OS onto a blank root disk using HTTP/CD answer files, verify hard-disk boot, then provision and create a template. ZStack userdata is not used for installation.

- [ZStack Snapshot](/packer/integrations/chijiajian/zstack/latest/components/builder/snapshot) (`zstack-snapshot`): convert an existing root-volume snapshot directly into an image, without starting a VM or running provisioners. Available starting with v1.2.1. The old `zstack` snapshot input remains compatible but is deprecated.

## Examples and requirements

The [examples directory](https://github.com/chijiajian/packer-plugin-zstack/tree/main/examples) includes authentication, resource UUIDs, snapshot builds, placement and unattended ISO installation. Choose one template rather than loading all top-level examples together.

ISO examples cover Debian, Ubuntu Server, Rocky ARM and Windows Server. Configure the installer credentials, firmware, root-disk offering and an installation check for your OS. The runner must reach SSH/WinRM and, for boot commands, the host's VNC console; HTTP/CD delivery must be reachable from the guest and backup storage.

Only the root disk is included in the resulting image. Build-owned temporary VMs, data disks and imported media are cleaned by default; existing source resources are retained. See each builder's configuration for retention settings, timeouts and export behavior. Keep credentials and rendered answer files in ignored local files.
