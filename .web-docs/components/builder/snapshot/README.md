
Type: `zstack-snapshot`

Convert an existing Enabled, Ready root-volume snapshot into a root-volume template. The source snapshot is retained. No VM is created, no SSH/WinRM connection is made, and provisioners are not executed. This builder does not create snapshots of existing VMs or capture their memory, additional disks or VM configuration.

Available starting with v1.2.1. Existing `source "zstack"` configurations using `source_volume_snapshot_uuid` continue to work, with a deprecation warning in newer builds. Change the source type and build reference to migrate, and remove unused VM/communicator settings.

```hcl
source "zstack-snapshot" "base" {
  # Authentication can be supplied through ZSTACK_* environment variables.
  source_volume_snapshot_uuid = "existing-root-snapshot-uuid"
  backup_storage_uuid        = "backup-storage-uuid"
  image_name                 = "snapshot-image"
  skip_export                = true
}

build {
  sources = ["source.zstack-snapshot.base"]
}
```

Required: `source_volume_snapshot_uuid`, `image_name`, and either `backup_storage_uuid` or `backup_storage_name`, plus valid ZStack authentication.

Optional output settings: `image_description`, `platform` (defaults to `Linux`), `architecture`, `guest_os_type`, `skip_export`, and `keep_image_on_error`. Set the platform explicitly for non-Linux snapshots. Export and cleanup behavior match the [ZStack builder](/packer/integrations/chijiajian/zstack/latest/components/builder/zstack).

Timeouts: `api_timeout`, `image_ready_timeout`, and `cleanup_timeout`. Authentication uses the same `zstack_host`, `port`, account/access-key fields and environment variables as the ZStack builder.

`communicator` defaults to `none` and accepts only `none`. VM, network, SSH/WinRM, ISO and userdata options are rejected. Do not attach provisioners to this source: there is no guest to provision. To customize the resulting image, use its UUID in a subsequent `zstack` build.
