#!/usr/bin/env python3
"""Automatic, one-time upgrade through actual CLI/MCP processes and private data."""
import concurrent.futures
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import tempfile

from events_integration import BINARY, MCP, STORAGE


def run():
    with tempfile.TemporaryDirectory(prefix='upgrade-', dir=STORAGE) as directory:
        root = pathlib.Path(directory)
        home = root/'home'
        for path in [home/'run', home/'vms', root/'images', root/'templates']:
            path.mkdir(parents=True)
        (home/'config.env').write_text(f'IMAGE_DIR={root}/images\nBACKUP_DIR={root}/templates\n')
        environment = dict(os.environ, NIDO_HOME=str(home))
        environment.pop('NIDO_JSON', None)
        base = root/'images/old-base.qcow2'
        disk = root/'actual-disk.qcow2'
        qemu = shutil.which('qemu-img')
        if qemu:
            subprocess.run([qemu, 'create', '-f', 'qcow2', str(base), '64M'], check=True, capture_output=True)
            subprocess.run([qemu, 'create', '-f', 'qcow2', '-F', 'qcow2', '-b', str(base), str(disk)], check=True, capture_output=True)
            chain = subprocess.check_output([qemu, 'info', '--backing-chain', '--output=json', str(disk)])
        else:
            base.write_bytes(b'BASE PAYLOAD'); disk.write_bytes(b'DISK PAYLOAD')
        vm = home/'vms/legacy.qcow2'
        vm.symlink_to(disk)
        original = {'name': 'legacy', 'pid': 0, 'ssh_port': 31001, 'vnc_port': 31002,
                    'ssh_user': 'preserve', 'ssh_password': 'fixture-only',
                    'forwarding': [{'host_port': 31003, 'guest_port': 80}],
                    'unknown': {'null': None, 'nested': ['preserve', 7]}}
        state = home/'run/legacy.json'
        original_bytes = json.dumps(original).encode()
        state.write_bytes(original_bytes); state.chmod(0o600)
        blocker = home/'run/legacy.json.pre-architecture'
        blocker.mkdir()

        def cli(*args):
            return subprocess.run([BINARY, *args, '--json'], env=environment, cwd=root,
                                  capture_output=True, text=True, timeout=20)

        def snapshot(path):
            st = path.stat()
            return (st.st_ino, st.st_size, st.st_mtime_ns, st.st_mode,
                    hashlib.sha256(path.read_bytes()).hexdigest())

        before = {p: snapshot(p) for p in [base, disk, home/'config.env']}
        first = cli('version')
        assert first.returncode != 0, first.stdout
        assert state.read_bytes() == original_bytes
        assert not (home/'.state-schema').exists()
        blocker.rmdir()
        # Simultaneous first commands resume the failed conversion together.
        with concurrent.futures.ThreadPoolExecutor(max_workers=6) as pool:
            attempts = list(pool.map(lambda _: cli('version'), range(6)))
        assert all(x.returncode == 0 for x in attempts), [(x.stdout, x.stderr) for x in attempts]
        assert all(json.loads(x.stdout)['status'] == 'ok' for x in attempts)
        assert json.loads(state.read_text()) == dict(original, arch='amd64')
        assert blocker.read_bytes() == original_bytes
        assert state.stat().st_mode & 0o777 == 0o600
        assert vm.is_symlink() and vm.resolve() == disk
        assert all(snapshot(p) == value for p, value in before.items())
        if qemu:
            assert subprocess.check_output([qemu, 'info', '--backing-chain', '--output=json', str(disk)]) == chain
        marker = home/'.state-schema'
        prepared = snapshot(marker)
        client = MCP(environment, root)
        try:
            fleet = client.resource('nido://fleet/vms')
            assert len(fleet['vms']) == 1 and fleet['vms'][0]['arch'] == 'amd64', fleet
            client.collect(.1)
            assert not client.notifications
        finally:
            client.close()
        assert snapshot(marker) == prepared, 'ordinary startup repeated conversion'
        assert cli('ls').returncode == 0
        assert snapshot(marker) == prepared
        assert not list(home.glob('.state-schema.lock'))
        assert not list(home.rglob('.nido-metadata-*'))
        print('PASS: automatic/resumable CLI upgrade, concurrent processes, local MCP, original metadata, payloads, symlink and backing chain preserved')


if __name__ == '__main__':
    run()
