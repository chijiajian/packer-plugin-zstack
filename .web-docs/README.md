<!--
  Include a short overview about the plugin.

  This document is a great location for creating a table of contents for each
  of the components the plugin may provide. This document should load automatically
  when navigating to the docs directory for a plugin.

-->

### Installation

To install this plugin, copy and paste this code into your Packer configuration, then run [`packer init`](https://www.packer.io/docs/commands/init).

```hcl
packer {
  required_plugins {
    zstack = {
      # source represents the GitHub URI to the plugin repository without the `packer-plugin-` prefix.
      source  = "github.com/chijiajian/packer-plugin-zstack"
      version = ">=1.0.0"
    }
  }
}
```

Alternatively, you can use `packer plugins install` to manage installation of this plugin.

```sh
$ packer plugins install github.com/chijiajian/packer-plugin-zstack
```

### Components


#### Builders

- [builder](/packer/integrations/hashicorp/zstack/latest/components/builder/builder-name) - The scaffolding builder is used to create endless Packer
  plugins using a consistent plugin structure.


