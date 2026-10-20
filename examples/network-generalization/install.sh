#!/bin/bash
# Run as root, as the final provisioning step. Do not restart networking here.
set -euo pipefail
[ "$(id -u)" = 0 ] || { echo 'root required' >&2; exit 1; }
command -v python3 >/dev/null
command -v systemctl >/dev/null
: "${BUILD_INTERFACE:?Set BUILD_INTERFACE to the DHCP interface used during image creation}"
[[ "$BUILD_INTERFACE" =~ ^[a-zA-Z0-9_-]+$ ]]
profile=$(python3 - "$BUILD_INTERFACE" <<'PYSELECT'
import hashlib, json, sys
from pathlib import Path
base = Path('/etc/sysconfig/network-scripts')
state = Path('/var/lib/zstack-network/connections.json')
entries = json.loads(state.read_text()).values() if state.exists() else []
profile = base / ('ifcfg-' + sys.argv[1])
for entry in entries:
    candidate = base / entry['file']
    if not candidate.exists() or hashlib.sha256(candidate.read_bytes()).hexdigest() != entry['sha256']:
        raise SystemExit('Refusing to reseal user-modified generalized connections')
    if not profile.exists() and ('DEVICE=' + sys.argv[1]) in candidate.read_text().splitlines():
        profile = candidate
print(profile)
PYSELECT
)
# Validate before making changes; reject complex/static source profiles.
python3 - "$profile" "$BUILD_INTERFACE" <<'PY'
import sys
from pathlib import Path
p = Path(sys.argv[1])
if not p.exists():
    raise SystemExit('Build interface profile missing: ' + str(p))
interface = sys.argv[2]
driver = Path('/sys/class/net') / interface / 'device/driver'
if not driver.exists() or driver.resolve().name != 'virtio_net':
    raise SystemExit('Build interface must use virtio_net')
for prefix in ('route-', 'route6-', 'rule-', 'rule6-'):
    companion = p.with_name(prefix + interface)
    if companion.exists() and companion.read_text().strip():
        raise SystemExit('Refusing to discard custom routes/rules: ' + str(companion))
d = dict(line.split('=', 1) for line in p.read_text().splitlines() if '=' in line and not line.lstrip().startswith('#'))
d = {k.strip(): v.strip().strip('\"\'') for k, v in d.items()}
if d.get('BOOTPROTO', '').lower() != 'dhcp' or any(k.startswith(('IPADDR', 'GATEWAY', 'BRIDGE', 'MASTER', 'VLAN', 'BOND')) for k in d):
    raise SystemExit('Refusing to generalize static or composite build connection')
if d.get('DEVICE', interface) != interface or d.get('TYPE', 'Ethernet') != 'Ethernet':
    raise SystemExit('Build profile is not a plain Ethernet interface')
PY
install -d -m 700 /var/lib/zstack-network
install -m 600 "$profile" /var/lib/zstack-network/build-profile.backup
# Stop only a prior generalizer when resealing a generalized clone.
systemctl stop zstack-network.timer zstack-network.service 2>/dev/null || true
install -m 755 "$(dirname "$0")/reconcile.py" /usr/local/sbin/zstack-network
cat > /etc/systemd/system/zstack-network-prepare.service <<'EOF'
[Unit]
Description=Prepare generalized virtio DHCP profiles
After=local-fs.target systemd-udev-trigger.service
Before=network-pre.target NetworkManager.service network.service
Wants=network-pre.target
[Service]
Type=oneshot
ExecStart=/usr/local/sbin/zstack-network --prepare
[Install]
WantedBy=multi-user.target
EOF
cat > /etc/systemd/system/zstack-network.service <<'EOF'
[Unit]
Description=Reconcile generalized virtio DHCP profiles
After=network.target
[Service]
Type=oneshot
ExecStart=/usr/local/sbin/zstack-network
TimeoutStartSec=180
# network-scripts launches a persistent dhclient owned by the interface.
KillMode=process
EOF
cat > /etc/systemd/system/zstack-network.timer <<'EOF'
[Unit]
Description=Discover added virtio network devices
[Timer]
OnBootSec=15s
OnUnitInactiveSec=10s
AccuracySec=1s
Unit=zstack-network.service
[Install]
WantedBy=timers.target
EOF
# Remove only the explicitly selected build connection. Keep the live connection
# until Packer shuts down; the installed services start on the next boot.
python3 - <<'PYCLEAN'
import json
from pathlib import Path
state = Path('/var/lib/zstack-network/connections.json')
if state.exists():
    for entry in json.loads(state.read_text()).values():
        (Path('/etc/sysconfig/network-scripts') / entry['file']).unlink()
    state.unlink()
PYCLEAN
rm -f -- "$profile"
systemctl daemon-reload
systemctl enable zstack-network-prepare.service zstack-network.timer
