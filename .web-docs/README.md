
The ZStack Packer plugin allows you to create custom VM images on ZStack Cloud.
It supports creating images from existing VM instances, managing instance lifecycle,
and exporting images to backup storage.

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

- [builder](/packer/integrations/hashicorp/zstack/latest/components/builder/zstack) - Provides the capability to build customized images based on an existing ZStack VM instance.




