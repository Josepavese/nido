#!/usr/bin/env python3
"""Isolated CLI/MCP/PTY architecture contracts; optional actual ARM64 guest boot.

NIDO_TEST_QEMU_ARM=1 requires qemu-system-aarch64, UEFI, network and up to
2 GiB scratch. It proves software emulation on this host, not native ARM/KVM.
"""
import json
import os
import pathlib
import shutil
import subprocess
import tempfile
import time

from events_integration import MCP
from tui_terminal import TUI, Observer

BINARY = str(pathlib.Path(os.environ['NIDO_TEST_BINARY']).resolve())
REPO = pathlib.Path(__file__).resolve().parents[1]


def run():
    with tempfile.TemporaryDirectory(prefix='isa-', dir=os.environ.get('NIDO_TEST_STORAGE')) as directory:
        root = pathlib.Path(directory)
        home = root/'nido'
        for name in ['run', 'vms', 'images', 'backups']:
            (home/name).mkdir(parents=True)
        image_dir=pathlib.Path(os.environ.get('NIDO_TEST_IMAGE_CACHE',str(home/'images')))
        image_dir.mkdir(parents=True,exist_ok=True)
        (root/'registry').mkdir()
        shutil.copy(REPO/'registry/images.json', root/'registry/images.json')
        shutil.copy(REPO/'registry/images.json', image_dir/'.catalog.json')
        (home/'config.env').write_text(f'IMAGE_DIR={image_dir}\nBACKUP_DIR={home}/backups\nSSH_USER=debian\nLINKED_CLONES=true\n')
        environment = dict(os.environ, NIDO_HOME=str(home), TERM='xterm-256color')
        environment.pop('NIDO_JSON', None)

        def cli(*args, timeout=20):
            result = subprocess.run([BINARY, *args, '--json'], cwd=root, env=environment,
                                    capture_output=True, text=True, timeout=timeout)
            assert result.returncode == 0, (args, result.stdout, result.stderr)
            data = json.loads(result.stdout)
            assert data['status'] == 'ok', data
            return data['data']

        client = MCP(environment, root)
        tui = None
        owned = []
        try:
            for arch in ['amd64', 'arm64']:
                detail = cli('images', 'info', 'ubuntu:24.04', '--arch', arch)
                assert detail['image']['version']['arch'] == arch, detail
                remote = client.resource(f'nido://image/ubuntu:24.04?arch={arch}')
                assert remote['image']['arch'] == arch, remote
            # Native omission must resolve to exactly one supported variant.
            native = cli('images', 'info', 'ubuntu:24.04')['image']['version']['arch']
            assert native in ['amd64', 'arm64'], native
            observer=Observer() if os.environ.get("NIDO_TUI_VISUAL_DIR") else None
            tui = TUI(environment, root,observer=observer)
            tui.text('FLEET')
            tui.key(b'\x1b[C')
            tui.text('SPAWN')
            tui.key(b'\r')
            tui.text('Arch')
            tui.key(b'arm-proof')
            tui.key(b'\t')
            tui.key(b'\r')
            tui.text('amd64')
            tui.key(b'\r')
            tui.text('arm64')
            if observer:
                time.sleep(.2)
                output=pathlib.Path(os.environ['NIDO_TUI_VISUAL_DIR']);output.mkdir(parents=True,exist_ok=True)
                (output/'architecture.html').write_text('<!doctype html><meta charset="utf-8">'+observer.snapshot('ARM64 selection - actual PTY'))
            tui.resize(90, 30)
            tui.text('arm64')
            tui.close()
            tui = None
            print('PASS: CLI native/explicit ISA, MCP variant resource, actual PTY selector/resize', flush=True)
            if os.environ.get('NIDO_TEST_QEMU_ARM') != '1':
                return
            owned.append('arm-proof')
            print('ARM64 guest: acquire verified Debian cloud image and spawn through CLI', flush=True)
            cli('spawn', 'arm-proof', '--image', 'debian:13', '--arch', 'arm64', '--memory', '768', '--cpus', '2', timeout=1200)
            detail = cli('info', 'arm-proof')['vm']
            assert detail['arch'] == 'arm64', detail
            assert detail['virtualization_capability']['accelerator'] == 'tcg', detail
            deadline = time.monotonic() + 240
            while time.monotonic() < deadline:
                result = subprocess.run([BINARY, 'ssh', 'arm-proof', 'uname', '-m'], cwd=root,
                                        env=environment, capture_output=True, text=True, timeout=15)
                if result.returncode == 0 and 'aarch64' in result.stdout:
                    print('PASS: real guest SSH reports aarch64 under TCG', flush=True)
                    break
                time.sleep(2)
            else:
                serial = home/'run/arm-proof.serial.log'
                raise AssertionError('ARM guest readiness deadline: ' + (serial.read_text(errors='replace')[-6000:] if serial.exists() else 'no serial'))
            cli('stop', 'arm-proof')
            cli('start', 'arm-proof')
            assert cli('info', 'arm-proof')['vm']['arch'] == 'arm64'
            cli('stop', 'arm-proof')
            template = cli('template', 'create', 'arm-proof', 'arm-template', timeout=120)
            path = pathlib.Path(template['action']['path'])
            assert json.loads(path.with_name(path.name+'.nido.json').read_text())['arch'] == 'arm64'
            owned.append('arm-clone')
            cli('spawn', 'arm-clone', 'arm-template', timeout=40)
            assert cli('info', 'arm-clone')['vm']['arch'] == 'arm64'
            assert (home/'vms/arm-clone.qcow2.uefi.fd').exists()
            print('PASS: restart retains ISA; template clone inherits ARM64 and private UEFI variables', flush=True)
        finally:
            if tui is not None:
                tui.close()
            for name in reversed(owned):
                subprocess.run([BINARY, 'delete', name], cwd=root, env=environment,
                               stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, timeout=30, check=True)
            client.close()


if __name__ == '__main__':
    run()
