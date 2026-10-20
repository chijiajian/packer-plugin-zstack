#!/usr/bin/env python3
"""Opt-in DHCP generalization for Helix/Kylin virtio Ethernet guests."""
import argparse
import configparser
import fcntl
import hashlib
import json
import os
import signal
import time
from pathlib import Path
import subprocess
import uuid

BASE = Path('/etc/sysconfig/network-scripts')
STATE = Path('/var/lib/zstack-network')
RUN = Path('/run/zstack-network')
KEYFILES = Path('/etc/NetworkManager/system-connections')
CONFIG = Path('/etc/zstack-network.json')

def foreign_profile(name, mac, owned):
    """Do not take over a user-defined interface, including inactive profiles."""
    for p in BASE.glob('ifcfg-*'):
        if p.name in owned:
            continue
        values = {}
        for line in p.read_text().splitlines():
            if '=' in line and not line.lstrip().startswith('#'):
                k, v = line.split('=', 1)
                values[k.strip()] = v.strip().strip('"\'')
        if (values.get('DEVICE') == name or
                values.get('HWADDR', '').lower() == mac or
                values.get('NAME') == name):
            return True
        if values.get('TYPE') == 'Ethernet' and not values.get('DEVICE') and not values.get('HWADDR'):
            return True
    for p in KEYFILES.glob('*'):
        if not p.is_file():
            continue
        profile = configparser.ConfigParser(interpolation=None)
        profile.read(str(p))
        interface = profile.get('connection', 'interface-name', fallback='')
        address = profile.get('ethernet', 'mac-address', fallback='').lower()
        # An unbound Ethernet profile could intentionally target any NIC.
        kind = profile.get('connection', 'type', fallback='')
        if interface == name or address == mac or (
                kind in ('ethernet', '802-3-ethernet') and not interface and not address):
            return True
    return False

def command(*args):
    return subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                          universal_newlines=True, timeout=60)

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest() if path.exists() else None

def retire_dhclient(profile):
    """Stop only the legacy DHCP client for an owned, removed interface."""
    name = next(line.split('=', 1)[1] for line in profile.read_text().splitlines()
                if line.startswith('DEVICE='))
    pidfile = Path('/run/dhclient-{}.pid'.format(name))
    if not pidfile.exists():
        return
    pid = int(pidfile.read_text().strip())
    proc = Path('/proc') / str(pid) / 'cmdline'
    if not proc.exists():
        pidfile.unlink()
        return
    argv = proc.read_bytes().decode().strip('\0').split('\0')
    if Path(argv[0]).name != 'dhclient' or name not in argv or str(pidfile) not in argv:
        raise RuntimeError('Unexpected process in DHCP pidfile: ' + str(pidfile))
    os.kill(pid, signal.SIGTERM)
    for _ in range(50):
        if not proc.exists() or not proc.read_bytes():
            if pidfile.exists():
                pidfile.unlink()
            return
        time.sleep(0.1)
    raise RuntimeError('DHCP client did not stop for ' + name)

def write(path, text):
    if path.exists() and path.read_text() == text:
        return
    tmp = path.with_suffix('.tmp')
    tmp.write_text(text)
    os.chmod(str(tmp), 0o600)
    tmp.replace(path)

def devices():
    result = []
    for p in Path('/sys/class/net').iterdir():
        driver = p / 'device/driver'
        if driver.exists() and driver.resolve().name == 'virtio_net' and (p / 'type').read_text().strip() == '1':
            result.append((str((p / 'device').resolve()), p.name, (p / 'address').read_text().strip().lower()))
    return sorted(result)

def reconcile(prepare=False):
    STATE.mkdir(parents=True, exist_ok=True)
    RUN.mkdir(parents=True, exist_ok=True)
    with (RUN / 'lock').open('w') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        statefile = STATE / 'connections.json'
        state = json.loads(statefile.read_text()) if statefile.exists() else {}
        config = CONFIG
        primary = json.loads(config.read_text()).get('primary_mac', '').lower() if config.exists() else ''
        nics = devices()
        current = {mac for _, _, mac in nics}
        nm = command('systemctl', 'is-active', 'NetworkManager').returncode == 0
        # Retire only unchanged profiles owned by this service, never user edits.
        for mac, entry in list(state.items()):
            p = BASE / entry['file']
            if mac not in current and digest(p) == entry['sha256']:
                if nm and not prepare:
                    command('nmcli', 'connection', 'delete', 'uuid', entry['uuid'])
                elif not prepare:
                    retire_dhclient(p)
                # NetworkManager's ifcfg-rh plugin may already remove the file.
                if p.exists():
                    p.unlink()
                del state[mac]
        for index, (_, name, mac) in enumerate(nics):
            entry = state.get(mac)
            if entry and digest(BASE / entry['file']) != entry['sha256']:
                continue  # Explicit user modification transfers ownership to user.
            # An unmanaged profile for this device/MAC takes precedence.
            owned = {e['file'] for e in state.values()
                     if digest(BASE / e['file']) == e['sha256']}
            if foreign_profile(name, mac, owned):
                continue
            if nm and not entry:
                active = command('nmcli', '-g', 'GENERAL.CONNECTION', 'device', 'show', name).stdout.strip()
                if active and active != '--':
                    continue
            uid = entry['uuid'] if entry else str(uuid.uuid4())
            filename = 'ifcfg-zstack-' + mac.replace(':', '')
            metric = 100 if primary == mac else (200 + index if primary else 100 + index)
            text = ('# Managed by zstack-network; editing this file opts out.\n'
                    'TYPE=Ethernet\nNAME=zstack-{mac}\nUUID={uid}\nDEVICE={name}\nHWADDR={mac}\n'
                    'ONBOOT=yes\nBOOTPROTO=dhcp\nDEFROUTE={route}\nPEERDNS={dns}\n'
                    'IPV4_ROUTE_METRIC={metric}\nMETRIC={metric}\nIPV6INIT=no\n').format(
                        mac=mac, uid=uid, name=name, metric=metric,
                        route='yes' if not primary or primary == mac else 'no',
                        dns='yes' if primary == mac or (not primary and index == 0) else 'no')
            p = BASE / filename
            changed = not p.exists() or p.read_text() != text
            if changed:
                write(p, text)
            state[mac] = {'file': filename, 'uuid': uid, 'sha256': digest(p)}
            write(statefile, json.dumps(state, indent=2))
            activated = RUN / mac.replace(':', '')
            if not prepare and (changed or not activated.exists()):
                if not changed:
                    address = command('ip', '-4', '-o', 'address', 'show', 'dev', name, 'scope', 'global')
                    if address.returncode == 0 and address.stdout.strip():
                        activated.touch()
                        continue
                if nm:
                    loaded = command('nmcli', 'connection', 'load', str(p))
                    if loaded.returncode:
                        raise RuntimeError(loaded.stderr)
                    r = command('nmcli', '--wait', '15', 'connection', 'up', 'uuid', uid, 'ifname', name)
                else:
                    r = command('/sbin/ifup', str(p))
                if r.returncode:
                    print('Activation pending for {}: {}'.format(name, r.stderr), flush=True)
                else:
                    activated.touch()
        write(statefile, json.dumps(state, indent=2))

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--prepare', action='store_true')
    reconcile(parser.parse_args().prepare)
