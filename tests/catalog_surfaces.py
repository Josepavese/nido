#!/usr/bin/env python3
"""Read-only catalog CLI/MCP contract tests in a networkless disposable home.
Linux + bubblewrap required. NIDO_TEST_BINARY and NIDO_TEST_STORAGE select
candidate bytes and scratch filesystem; NIDO_TEST_CATALOG can select a catalog.
No existing home, VM, image cache or session is writable or exposed to Nido.
"""
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile


def probe(binary, catalog):
    calls = 0

    def run(*args, request=None):
        nonlocal calls
        p = subprocess.run([binary, *args], input=request, text=True,
                           capture_output=True, timeout=20, check=True)
        assert not p.stderr, p.stderr
        calls += 1
        return json.loads(p.stdout)

    def mcp(method, params):
        response = run('mcp', request=json.dumps({'jsonrpc': '2.0', 'id': 1,
                       'method': method, 'params': params})+'\n')
        result = response['result']
        assert not result.get('isError'), result
        if method == 'resources/read':
            return json.loads(result['contents'][0]['text'])
        return json.loads(result['content'][0]['text'])

    expected = json.loads(pathlib.Path(catalog).read_text())['images']
    # Distinct sentinels detect any accidental write into VM/data directories.
    nido_home = pathlib.Path(os.environ['NIDO_HOME'])
    for name in ['vms', 'backups', 'run']:
        (nido_home/name).mkdir(parents=True, exist_ok=True)
        (nido_home/name/'sentinel').write_bytes(b'PRESERVE CATALOG TEST DATA')
    cli_list = run('images', 'list', '--json')['data']['images']
    tool_list = mcp('tools/call', {'name': 'nido_image', 'arguments': {'action': 'list'}})['images']
    resource_list = mcp('resources/read', {'uri': 'nido://catalog/images'})
    if isinstance(resource_list, dict):
        resource_list = resource_list['images']
    tags = {(i['name'], v['version'],v['arch']) for i in expected for v in i['versions']}
    for result in [cli_list, tool_list, resource_list]:
        actual = {(i['name'], i['version'],i['arch']) for i in result if i['registry'] != 'blueprint'}
        assert actual == tags, (actual, tags)
    checks = 0
    for image in expected:
        for version in image['versions']:
            refs = [version['version'], *version.get('aliases', [])]
            if version == image['versions'][0]:
                refs.append('')
            for ref in refs:
                tag = image['name']+(':'+ref if ref else '')
                cli = run('images', 'info', tag, '--arch',version['arch'], '--json')['data']['image']['version']
                tool = mcp('tools/call', {'name': 'nido_image', 'arguments': {'action': 'info', 'image': tag,'arch':version['arch']}})['image']
                resource = mcp('resources/read', {'uri': 'nido://image/'+tag+'?arch='+version['arch']})['image']
                for actual in [cli, tool, resource]:
                    assert actual['version'] == version['version'], (tag, actual)
                    for key in ['url', 'checksum', 'checksum_type', 'size_bytes']:
                        assert actual[key] == version[key], (tag, key, actual)
                checks += 1
    # Native HAL inventory consumes a private read-only sysfs, with one isolated
    # device and one device missing an IOMMU group. No physical host PCI is used.
    cli_devices = run('accel', 'list', '--json')['data']['devices']
    tool_devices = mcp('tools/call', {'name': 'nido_system', 'arguments': {'action': 'accel_list'}})['devices']
    resource_devices = mcp('resources/read', {'uri': 'nido://system/accelerators'})['devices']
    assert cli_devices == tool_devices == resource_devices
    assert len(cli_devices) == 2, cli_devices
    assert cli_devices[0]['IsSafe'] is True, cli_devices
    assert cli_devices[1]['IsSafe'] is False and not cli_devices[1]['IOMMUGroup'], cli_devices
    # Invalid references stay errors on both machine interfaces.
    for tag in ['missing-distribution:999', 'ubuntu:missing-version']:
        p = subprocess.run([binary, 'images', 'info', tag, '--json'],
                           text=True, capture_output=True, timeout=20)
        assert p.returncode != 0 and json.loads(p.stdout)['status'] == 'error'
        request = {'jsonrpc': '2.0', 'id': 1, 'method': 'tools/call',
                   'params': {'name': 'nido_image', 'arguments': {'action': 'info', 'image': tag,'arch':version['arch']}}}
        response = run('mcp', request=json.dumps(request)+'\n')
        assert response['result'].get('isError'), response
    for name in ['vms', 'backups', 'run']:
        assert (nido_home/name/'sentinel').read_bytes() == b'PRESERVE CATALOG TEST DATA'
        assert len(list((nido_home/name).iterdir())) == 1
    assert not list((nido_home/'images').glob('*.qcow2'))
    print(json.dumps({'result': 'PASS', 'tags': len(tags), 'references': checks,
                      'cli_mcp_calls': calls, 'network': 'disabled', 'vm_writes': 0}))


def main():
    if len(sys.argv) == 4 and sys.argv[1] == '--inside':
        probe(sys.argv[2], sys.argv[3])
        return
    binary = str(pathlib.Path(os.environ['NIDO_TEST_BINARY']).resolve())
    catalog = pathlib.Path(os.environ.get('NIDO_TEST_CATALOG',
              pathlib.Path(__file__).resolve().parents[1]/'registry/images.json')).resolve()
    with tempfile.TemporaryDirectory(prefix='nido-catalog-surfaces-', dir=os.environ['NIDO_TEST_STORAGE']) as temp:
        root = pathlib.Path(temp)
        home = root/'home'
        cache = home/'.nido/images'
        cache.mkdir(parents=True)
        shutil.copy2(catalog, cache/'.catalog.json')
        shutil.copy2(catalog, root/'catalog.json')
        (root/'tmp').mkdir()
        sysfs = root/'sys'
        devices = sysfs/'bus/pci/devices'
        for slot in ['0000:01:00.0', '0000:02:00.0']:
            device = devices/slot
            device.mkdir(parents=True)
            for name, value in {'class': '0x030000', 'vendor': '0x1234',
                                'device': '0x5678', 'boot_vga': '0'}.items():
                (device/name).write_text(value)
        (devices/'0000:01:00.0/iommu_group').symlink_to('/sys/kernel/iommu_groups/7')
        command = ['bwrap', '--ro-bind', '/', '/', '--ro-bind', str(sysfs), '/sys', '--dev', '/dev', '--unshare-user',
                   '--unshare-pid', '--unshare-net', '--proc', '/proc', '--tmpfs',
                   str(pathlib.Path.home()), '--bind', temp, temp, '--setenv', 'NIDO_HOME',
                   str(cache.parent), '--setenv', 'TMPDIR', str(root/'tmp'), '--chdir', temp,
                   '--', sys.executable, str(pathlib.Path(__file__).resolve()),
                   '--inside', binary, str(root/'catalog.json')]
        # The binary and test script may live below the hidden original home.
        # Bind only these two files into scratch, without exposing that home.
        shutil.copy2(binary, root/'nido')
        shutil.copy2(__file__, root/'probe.py')
        command[-5:] = [sys.executable, str(root/'probe.py'), '--inside', str(root/'nido'), str(root/'catalog.json')]
        subprocess.run(command, check=True, timeout=180)


if __name__ == '__main__':
    main()
