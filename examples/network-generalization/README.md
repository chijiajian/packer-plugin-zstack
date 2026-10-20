# Helix / Kylin network generalization

This opt-in example builds a **DHCPv4 virtio Ethernet** image with network-scripts
(Helix) or NetworkManager's ifcfg-rh support (Kylin). It removes the selected build
connection's `ens3` binding and creates separate connections for the actual NICs
in each instance. No plugin Go changes are required.

## Build

Run from the repository root. Use `examples/helix-kylin.pkr.hcl` for either source
qcow2 URL. It requires a root SSH login, Python 3, systemd, and an existing plain
DHCP `ifcfg-<build_interface>` profile. It does not install packages or migrate
network managers. Cloud-init network rendering must be disabled when this
script owns networking; both tested Zaku source images already set
`network: {config: disabled}`. Other cloud-init functions are left unchanged.
Supply credentials through environment variables, never commit
them in a template:

```sh
export PKR_VAR_zstack_host="$ZSTACK_HOST"
export PKR_VAR_access_key_id="$ZSTACK_ACCESS_KEY_ID"
export PKR_VAR_access_key_secret="$ZSTACK_ACCESS_KEY_SECRET"
# Set these to the desired source and existing lab resources:
export PKR_VAR_source_image_url='http://image-server/path/to/source.qcow2'
export PKR_VAR_network_uuid='...'
export PKR_VAR_backup_storage_uuid='...'
export PKR_VAR_image_name='helix-generalized'
# Supply PKR_VAR_ssh_password securely in your local shell.
packer init examples/helix-kylin.pkr.hcl
packer validate examples/helix-kylin.pkr.hcl
packer build examples/helix-kylin.pkr.hcl
```

Optional `PKR_VAR_host_uuid` selects a build host. `build_interface` defaults to
`ens3`; it identifies **only the build VM's connection to remove**, not the
interface names expected in clones. The output is a registered root-volume
image (`skip_export = true`).

For existing templates, copy the directory with a file provisioner, then execute
`BUILD_INTERFACE=ens3 bash /tmp/packer-network-generalization/install.sh` as root
in the **last** shell provisioner. The tracked template demonstrates this flow.
The installer enables services without starting them or restarting networking,
so the build's SSH connection remains available until Packer shuts down the VM.
Resealing a previously generalized clone is supported: unchanged owned profiles
and their instance-specific state are removed before sealing. Resealing refuses
user-modified owned profiles. A backup of the selected source profile is kept outside network configuration
paths in `/var/lib/zstack-network/build-profile.backup`.

## Runtime behavior

* A boot service prepares profiles before network services start. A systemd timer
  reconciles NICs after boot and every 10 seconds after the preceding run ends,
  including hot additions and removals. This is not an instantaneous udev hook.
* Only devices whose sysfs driver is `virtio_net` and type is Ethernet qualify.
  Bridge, bond, VLAN, loopback and veth interfaces are excluded.
* Every generated profile has its own UUID and current-instance MAC. A machine
  type change updates the interface name while retaining that NIC's identity.
* On Helix, retiring an owned NIC also stops its verified per-interface legacy
  DHCP client, so reusing the device name does not inherit an old DHCP process.
  The oneshot service uses `KillMode=process` to allow interface DHCP clients to
  survive reconciliation; disabling the timer leaves those clients running.
* Only unchanged profiles recorded in the service's state may be rewritten or
  removed. Editing or deleting a generated profile opts that profile out.
  Matching foreign ifcfg/keyfile profiles, including inactive static profiles,
  take precedence. Unbound foreign Ethernet profiles conservatively block
  automatic configuration. This is not a manager for arbitrary business networks.
* The runtime explicitly loads/activates generated NM profiles; it does not
  depend on NetworkManager's automatic default-connection policy, or globally
  change `no-auto-default`. Existing unrelated connections are preserved.
* Generated profiles do not configure IPv6 (`IPV6INIT=no`); kernel link-local
  addresses may still appear. IPv6, static-only subnets, other NIC
  models, netplan/ifupdown, and NetworkManager without ifcfg-rh support are outside
  this example's supported scope.

### Default routes and management NIC identity

Without an instance-specific configuration, all eligible NICs use DHCPv4 and
accept DHCP routes with metrics 100, 101, ... in sysfs device-path order. Only the
first NIC accepts DHCP DNS. This order is **not a guarantee of ZStack's default
L3/management-network identity**. It is suitable for a single routed management
network plus isolated DHCP networks (the regression topology).

For multiple routed networks, inject `/etc/zstack-network.json` into each deployed
instance **before its first boot**, using that instance's management MAC:

```json
{"primary_mac": "fa:00:00:00:00:01"}
```

The designated NIC accepts default routes/DNS; generated secondary profiles use
`DEFROUTE=no` and `PEERDNS=no`. Do not bake the build VM's MAC into the image.
Platform metadata integration and deployment-time file injection are not
implemented here. If the configured MAC does not exist, secondary NICs do not
become a fallback default route. Do not use this setting until the correct MAC is
available.

## Validation and diagnosis

```sh
python3 examples/network-generalization/test_reconcile.py
bash -n examples/network-generalization/install.sh
# In a clone:
systemctl status zstack-network-prepare.service zstack-network.timer
journalctl -b -u zstack-network-prepare -u zstack-network
ip -br address
ip route
cat /var/lib/zstack-network/connections.json
```

Live acceptance must verify guest IPs against platform-assigned IPs by MAC,
actual i440fx/q35 machine types, changed boot IDs after reboot, dual/single NICs,
hot detach/attach, same-VM machine-type changes, and cross-VM connectivity.
Inventory NIC counts alone are insufficient. Validate application behavior and
multi-gateway routing separately for the deployment topology.

To stop automatic reconciliation on a deployed instance, disable both units:
`systemctl disable --now zstack-network.timer zstack-network-prepare.service`.
Generated connections remain in place; adjust them with the OS network manager.
Do not restore the build profile in a clone with a different NIC name or MAC.
