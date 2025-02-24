
The zstack builder is used to create ZStack Image by VM Instance


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

- `source_image_url` (String) - URL of the source image, supports HTTP/HTTPS protocols.

- `format` (String) - Image format, supports formats like "qcow2", "raw", etc.

- `platform` (String) - Operating system platform type, such as "Linux", "Windows", etc.

- `image_name` (String) - Name of the target image to be created.

**Network Parameters**

- `network_name` (String) - L3 network name for VM network configuration.

**Instance Parameters**

- `instance_offering_name` (String) - Computing specification name that defines VM resources like CPU and memory.

- `instance_name` (String) - Name of the VM instance to be created.

**Storage Parameters**
- `backup_storage_name` (String) - Name of the backup storage for storing created images.

**SSH Parameters**
- `ssh_username` (String) - SSH username for connecting to the created VM instance.

- `ssh_password` (String) - SSH password for authentication.

<!--
  A basic example on the usage of the builder. Multiple examples
  can be provided to highlight various build configurations.

-->
### Example Usage


```hcl
source "zstack" "test" {
  zstack_host       = "10.3.3.3"
  port             = 8080
  access_key_id     = "aaaaaaa"
  access_key_secret = "1UQ3bfz9qS4vaaaaaaaaa"
  
  source_image      = "Centos8"
  source_image_url  = "http://example.com/image.qcow2"
  format           = "qcow2"
  platform         = "Linux"
  
  network_name     = "public-net"
  instance_offering_name = "min"
  backup_storage_name = "bs"
  instance_name    = "packer-test"
  image_name       = "image-by-packer"
  
  ssh_username     = "root"
  ssh_password     = "password"
}

```
