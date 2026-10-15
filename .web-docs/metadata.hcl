# Copyright ZStack.io, Inc. 2013, 2026
# SPDX-License-Identifier: MPL-2.0


integration {
  name = "ZStack"
  description = "The ZStack Cloud plugin can be used with HashiCorp Packer to create custom images on ZStack Cloud."
  identifier = "packer/chijiajian/zstack"
  docs {
    # Render the generated pages on developer.hashicorp.com.
    process_docs = true
    readme_location = "./README.md"
  }
  license {
    type = "MPL-2.0"
    url = "https://github.com/chijiajian/packer-plugin-zstack/blob/main/LICENSE"
  }
  component {
    type = "builder"
    name = "ZStack ISO"
    slug = "iso"
  }
  component {
    type = "builder"
    name = "ZStack"
    slug = "zstack"
  }
}
