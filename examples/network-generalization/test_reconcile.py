"""Offline ownership/lifecycle checks; real DHCP is covered by lab regression."""
import importlib.util
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('reconcile', Path(__file__).with_name('reconcile.py'))
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

class ReconcileTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        root = Path(self.tmp.name)
        self.config = patch.object(module, 'CONFIG', root / 'config.json')
        self.config.start()
        self.addCleanup(self.config.stop)
        for key in ('BASE', 'STATE', 'RUN', 'KEYFILES'):
            p = root / key
            p.mkdir()
            original = getattr(module, key)
            setattr(module, key, p)
            self.addCleanup(setattr, module, key, original)
        self.nics = [('pci1', 'ens3', 'fa:00:00:00:00:01'), ('pci2', 'ens4', 'fa:00:00:00:00:02')]
        self.calls = []
        def command(*args):
            self.calls.append(args)
            return SimpleNamespace(returncode=1 if args[0] == 'systemctl' else 0, stdout='', stderr='')
        self.command = patch.object(module, 'command', side_effect=command)
        self.command.start()
        self.addCleanup(self.command.stop)
        self.devices = patch.object(module, 'devices', side_effect=lambda: self.nics)
        self.devices.start()
        self.addCleanup(self.devices.stop)

    def test_prepare_then_activate_and_idempotent(self):
        module.reconcile(True)
        self.assertFalse(any(c[0] == '/sbin/ifup' for c in self.calls))
        module.reconcile()
        self.assertEqual(sum(c[0] == '/sbin/ifup' for c in self.calls), 2)
        self.calls.clear()
        module.reconcile()
        self.assertFalse(any(c[0] == '/sbin/ifup' for c in self.calls))

    def test_pci_rename_same_mac(self):
        module.reconcile()
        state = json.loads((module.STATE / 'connections.json').read_text())
        self.nics = [('pci1', 'enp1s0', self.nics[0][2]), ('pci2', 'enp2s0', self.nics[1][2])]
        module.reconcile()
        for _, name, mac in self.nics:
            p = module.BASE / state[mac]['file']
            self.assertIn('DEVICE='+name, p.read_text())
            self.assertIn('UUID='+state[mac]['uuid'], p.read_text())

    def test_hotplug_replaces_profile(self):
        module.reconcile()
        self.nics.pop()
        module.reconcile()
        self.assertEqual(len(list(module.BASE.glob('ifcfg-*'))), 1)
        self.nics.append(('pci2', 'ens4', 'fa:00:00:00:00:03'))
        module.reconcile()
        self.assertEqual(len(list(module.BASE.glob('ifcfg-*'))), 2)

    def test_user_edit_is_preserved(self):
        module.reconcile()
        p = next(module.BASE.glob('ifcfg-*'))
        p.write_text(p.read_text().replace('BOOTPROTO=dhcp', 'BOOTPROTO=none\nIPADDR=192.0.2.5'))
        expected = p.read_text()
        self.nics = []
        module.reconcile()
        self.assertEqual(p.read_text(), expected)

    def test_user_edit_blocks_replacement_nic_with_same_name(self):
        self.nics = self.nics[:1]
        module.reconcile()
        p = next(module.BASE.glob('ifcfg-*'))
        p.write_text(p.read_text().replace('BOOTPROTO=dhcp', 'BOOTPROTO=none\nIPADDR=192.0.2.5'))
        self.nics = [('pci1', 'ens3', 'fa:00:00:00:00:03')]
        module.reconcile()
        self.assertEqual(list(module.BASE.glob('ifcfg-*')), [p])

    def test_static_ifcfg_and_inactive_nm_profile_are_preserved(self):
        (module.BASE / 'ifcfg-custom').write_text('DEVICE="ens3"\nBOOTPROTO=none\nIPADDR=192.0.2.5\n')
        (module.KEYFILES / 'custom.nmconnection').write_text('[connection]\ntype=ethernet\ninterface-name=ens4\n')
        module.reconcile()
        self.assertEqual(list(module.BASE.glob('ifcfg-zstack-*')), [])

    def test_management_mac_controls_routes_and_dns(self):
        module.CONFIG.write_text(json.dumps({'primary_mac': self.nics[1][2]}))
        module.reconcile(True)
        state = json.loads((module.STATE / 'connections.json').read_text())
        first = (module.BASE / state[self.nics[0][2]]['file']).read_text()
        second = (module.BASE / state[self.nics[1][2]]['file']).read_text()
        self.assertIn('DEFROUTE=no\nPEERDNS=no', first)
        self.assertIn('DEFROUTE=yes\nPEERDNS=yes', second)

    def test_nm_activation_failure_is_retried(self):
        failed = True
        def nm_command(*args):
            self.calls.append(args)
            return SimpleNamespace(returncode=int(failed and 'up' in args), stdout='', stderr='test')
        with patch.object(module, 'command', side_effect=nm_command):
            module.reconcile()
            self.assertEqual(len(list(module.RUN.glob('fa*'))), 0)
            failed = False
            module.reconcile()
            self.assertEqual(len(list(module.RUN.glob('fa*'))), 2)
            self.assertEqual(sum('load' in c for c in self.calls), 4)

    def test_stale_pidfile_does_not_kill_unrelated_process(self):
        root = Path(self.tmp.name)
        pidfile = root / 'dhclient-ens4.pid'
        pidfile.write_text('123')
        proc = root / 'proc' / '123'
        proc.mkdir(parents=True)
        (proc / 'cmdline').write_bytes(b'/usr/bin/python3\0application.py\0')
        profile = root / 'profile'
        profile.write_text('DEVICE=ens4\n')
        def path(value):
            if value == '/run/dhclient-ens4.pid':
                return pidfile
            if value == '/proc':
                return root / 'proc'
            return Path(value)
        with patch.object(module, 'Path', side_effect=path), patch.object(module.os, 'kill') as kill:
            with self.assertRaisesRegex(RuntimeError, 'Unexpected process'):
                module.retire_dhclient(profile)
            kill.assert_not_called()

    def test_nm_deletes_profile_before_reconciler(self):
        module.reconcile(True)
        removed = self.nics.pop()[2]
        entries = json.loads((module.STATE / 'connections.json').read_text())
        def nm_command(*args):
            if 'delete' in args:
                (module.BASE / entries[removed]['file']).unlink()
            return SimpleNamespace(returncode=0, stdout='', stderr='')
        with patch.object(module, 'command', side_effect=nm_command):
            module.reconcile()
        self.assertNotIn(removed, json.loads((module.STATE / 'connections.json').read_text()))

if __name__ == '__main__':
    unittest.main()
