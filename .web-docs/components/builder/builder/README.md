
<!-- Builder Configuration Fields -->

**Required**

- `zstack_host` (string) - The ZStack Management Node  endpoint to connect to.


<!--
  Optional Configuration Fields

  Configuration options that are not required or have reasonable defaults
  should be listed under the optionals section. Defaults values should be
  noted in the description of the field
-->

**Optional**

- `access_key_id` (String) AccessKey ID for ZStack API. Create AccessKey ID from MN,  Operational Management->Access Control->AccessKey Management. May also be provided via ZSTACK_ACCESS_KEY_ID environment variable. Required if using AccessKey authentication. Mutually exclusive with `account_name` and `account_password`.
- `access_key_secret` (String, Sensitive) AccessKey Secret for ZStack API. May also be provided via ZSTACK_ACCESS_KEY_SECRET environment variable. Required if using AccessKey authentication. Mutually exclusive with `account_name` and `account_password`.
- `account_name` (String) Username for ZStack API. May also be provided via ZSTACK_ACCOUN_TNAME environment variable. Required if using Account authentication.  Only supports the platform administrator account (`admin`). Mutually exclusive with `access_key_id` and `access_key_secret`. Using `access_key_id` and `access_key_secret` is the recommended approach for authentication, as it provides more flexibility and security.
- `account_password` (String, Sensitive) Password for ZStack API. May also be provided via ZSTACK_ACCOUNT_PASSWORD environment variable.Required if using Account authentication.  Only supports the platform administrator account (`admin`). Mutually exclusive with `access_key_id` and `access_key_secret`. Using `access_key_id` and `access_key_secret` is the recommended approach for authentication, as it provides more flexibility and security.
- `port` (Number) ZStack Cloud MN API port. May also be provided via ZSTACK_PORT environment variable.


**Image Parameters**

- `source_image` (String) - Name of the source image used to create VM instance.

- `image_uuid` (String) - UUID of the source image. When provided, skips name-based image lookup.

- `source_image_url` (String) - URL of the source image, supports HTTP/HTTPS protocols.

- `format` (String) - Image format, supports formats like "qcow2", "raw", etc.

- `platform` (String) - Operating system platform type, such as "Linux", "Windows", etc.

- `guest_os_type` (String) - Guest OS type, such as "Ubuntu", "CentOS", etc.

- `image_name` (String) - Name of the target image to be created.

- `image_description` (String) - Description for the created image. Defaults to the image name if not set.

**Network Parameters**

- `network_name` (String) - L3 network name for VM network configuration.

- `network_uuid` (String) - UUID of the L3 network. When provided, skips name-based network lookup.

**Instance Parameters**

- `instance_offering_name` (String) - Computing specification name that defines VM resources like CPU and memory.

- `instance_offering_uuid` (String) - UUID of the instance offering. When provided, skips name-based lookup.

- `cpu_num` (Number) - Number of CPU cores for the VM instance. Use instead of instance_offering_name for custom sizing.

- `memory_size` (Number) - Memory size in MB for the VM instance. Use instead of instance_offering_name for custom sizing.

- `instance_name` (String) - Name of the VM instance to be created.

**Storage Parameters**
- `backup_storage_name` (String) - Name of the backup storage for storing created images.

- `backup_storage_uuid` (String) - UUID of the backup storage. Optional - when neither backup_storage_name nor backup_storage_uuid is specified, the image export step is skipped.

**SSH Parameters**
- `ssh_username` (String) - SSH username for connecting to the created VM instance.

- `ssh_password` (String) - SSH password for authentication.

<!--
  A basic example on the usage of the builder. Multiple examples
  can be provided to highlight various build configurations.

-->
### Example Usage


```hcl
source "zstack" "example" {
  zstack_host       = "zstack.example.com"
  access_key_id     = env("ZSTACK_ACCESS_KEY_ID")
  access_key_secret = env("ZSTACK_ACCESS_KEY_SECRET")

  source_image     = "ubuntu-22.04-from-url"
  source_image_url = "http://mirrors.example.com/images/ubuntu-22.04-server.qcow2"
  format           = "qcow2"
  platform         = "Linux"
  guest_os_type    = "Ubuntu"

  network_name        = "default-network"
  cpu_num             = 2
  memory_size         = 4096
  backup_storage_name = "local-backup"

  instance_name     = "packer-example"
  image_name        = "packer-example-image"
  image_description = "Built with Packer using AK/SK authentication"

  ssh_username = "root"
  ssh_password = "your-ssh-password"
}

```
