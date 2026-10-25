#!/usr/bin/env python3
"""CLI/MCP regressions: Linux + bwrap + qemu-img + QEMU required.
Set NIDO_TEST_BINARY (absolute candidate path), NIDO_TEST_STORAGE (scratch disk),
and optionally NIDO_TEST_EVIDENCE. No existing VM or application state is exposed.
"""
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile

INSIDE = len(sys.argv) == 3 and sys.argv[1] == '--inside'
ROOT = pathlib.Path(os.environ['NIDO_SAFETY_ROOT']) if INSIDE else pathlib.Path(tempfile.mkdtemp(prefix='nido-safety-', dir=os.environ['NIDO_TEST_STORAGE']))
HOME_DIR = pathlib.Path.home()/'.nido'
NIDO = str(HOME_DIR/'bin'/'nido')
CASES = ['path-traversal', 'stale-pid', 'missing-info', 'cache-direct', 'cache-chain', 'cache-relative', 'config-port', 'mcp-config-stale', 'storage-errors', 'controls']
LOG = []

def run(argv, **kw):
    p = subprocess.run(argv, capture_output=True, text=True, timeout=25, **kw)
    LOG.append({'argv': argv, 'stdin': kw.get('input'), 'exit': p.returncode, 'stdout': p.stdout, 'stderr': p.stderr})
    return p

def cli(*args):
    return run([NIDO, *args, '--json'])

def mcp(tool, **args):
    req = {'jsonrpc':'2.0','id':1,'method':'tools/call','params':{'name':tool,'arguments':args}}
    p = run([NIDO, 'mcp'], input=json.dumps(req)+'\n')
    response = json.loads(p.stdout)
    result = response['result']
    return result, json.loads(result['content'][0]['text'])

def state(name, **values):
    value = dict(name=name, arch="amd64", pid=0, ssh_port=51022, ssh_user='audit', memory_mb=128, vcpus=1)
    value.update(values)
    (HOME_DIR/'run'/f'{name}.json').write_text(json.dumps(value))

def read_state(name):
    return json.loads((HOME_DIR/'run'/f'{name}.json').read_text())

def disk(path, backing=None):
    args = ['/usr/bin/qemu-img', 'create', '-f', 'qcow2']
    if backing is not None:
        args += ['-F', 'qcow2', '-b', str(backing)]
    args += [str(path), '1M']
    assert run(args).returncode == 0

def healthy(path):
    return run(['/usr/bin/qemu-img','info','--backing-chain','--output=json',str(path)]).returncode == 0

def setup():
    for d in ['vms','run','images','backups']:
        (HOME_DIR/d).mkdir(exist_ok=True)
    (HOME_DIR/'config.env').write_text(f'IMAGE_DIR={HOME_DIR}/images\nBACKUP_DIR={HOME_DIR}/backups\nSSH_USER=audit\nPORT_RANGE_START=31000\nPORT_RANGE_END=31020\n')
    (HOME_DIR/'images'/'.catalog.json').write_text(json.dumps({'schema_version':'1.0','updated_at':'2026-10-19T00:00:00Z','images':[]}))

def inside(case):
    setup()
    observations = {}
    if case == 'path-traversal':
        for surface in ['cli','mcp']:
            target = HOME_DIR/'images'/f'protected-{surface}.qcow2'
            target.write_bytes(b'ONLY DISPOSABLE AUDIT SENTINEL')
            if surface == 'cli':
                response = cli('delete', f'../images/protected-{surface}')
                ok = response.returncode == 0
            else:
                response, _ = mcp('nido_vm', action='delete', name=f'../images/protected-{surface}')
                ok = not response.get('isError', False)
            observations[surface] = {'reported_success':ok,'out_of_vm_directory_file_deleted':not target.exists()}
            assert not ok and target.exists()
    elif case == 'stale-pid':
        for surface in ['cli','mcp']:
            worker = subprocess.Popen(['/usr/bin/sleep','120'])
            try:
                state('fixture', pid=worker.pid)
                (HOME_DIR/'vms'/'fixture.qcow2').write_bytes(b'not a bootable VM')
                info = cli('info','fixture') if surface == 'cli' else mcp('nido_vm',action='info',name='fixture')
                if surface == 'cli':
                    stop = cli('stop','fixture')
                    ok = stop.returncode == 0
                else:
                    stop, _ = mcp('nido_vm',action='stop',name='fixture')
                    ok = not stop.get('isError',False)
                observations[surface] = {'non_qemu_process_killed':worker.poll() is not None,'returncode':worker.poll(),'reported_success':ok}
                assert not ok and worker.poll() is None
                if surface == 'cli': assert json.loads(info.stdout)['data']['vm']['state'] == 'stopped'
                else: assert info[1]['vm']['State'] == 'stopped'
            finally:
                if worker.poll() is None: worker.terminate()
                worker.wait(timeout=2)
    elif case == 'missing-info':
        p = cli('info','does-not-exist')
        r, b = mcp('nido_vm',action='info',name='does-not-exist')
        observations['cli'] = {'exit':p.returncode,'status':json.loads(p.stdout)['status']}
        observations['mcp'] = {'isError':r.get('isError',False),'error':b.get('error')}
        assert p.returncode != 0 and r.get('isError',False)
        (HOME_DIR/'vms'/'corrupt.qcow2').write_bytes(b'disposable')
        (HOME_DIR/'run'/'corrupt.json').write_text('{invalid JSON')
        p = cli('info','corrupt')
        r, b = mcp('nido_vm',action='info',name='corrupt')
        observations['corrupt_state'] = {'cli_success':p.returncode==0,'mcp_success':not r.get('isError',False),'mcp_error':b.get('error')}
        assert p.returncode != 0 and r.get('isError',False)
    elif case == 'cache-direct':
        base = HOME_DIR/'images'/'audit-v1-amd64.qcow2'
        vm = HOME_DIR/'vms'/'fixture.qcow2'
        disk(base); disk(vm, base)
        assert healthy(vm)
        blocked = cli('cache','remove','audit:v1')
        assert blocked.returncode != 0 and base.exists()
        r, _ = mcp('nido_image',action='cache_remove',image='audit:v1')
        broken = not healthy(vm)
        observations = {'cli_refuses_used_image':True,'mcp_reported_success':not r.get('isError',False),'base_removed':not base.exists(),'vm_backing_chain_broken':broken}
        assert r.get('isError',False) and base.exists() and not broken
        base2 = HOME_DIR/'images'/'audit-v2-amd64.qcow2'
        vm2 = HOME_DIR/'vms'/'fixture2.qcow2'
        disk(base2); disk(vm2,base2)
        blocked2 = cli('images','remove','audit:v2')
        assert blocked2.returncode != 0 and base2.exists()
        r, _ = mcp('nido_image',action='remove',image='audit:v2')
        observations['remove_alias'] = {'cli_refuses':blocked2.returncode!=0,'mcp_reported_success':not r.get('isError',False),'backing_chain_broken':not healthy(vm2)}
        assert r.get('isError',False) and base2.exists() and not observations['remove_alias']['backing_chain_broken']
    elif case == 'cache-chain':
        for surface in ['cli','mcp']:
            base = HOME_DIR/'images'/f'base-{surface}.qcow2'
            middle = HOME_DIR/'backups'/f'mid-{surface}.compact.qcow2'
            vm = HOME_DIR/'vms'/f'{surface}.qcow2'
            disk(base); disk(middle, base); disk(vm, middle)
            assert healthy(vm)
            if surface == 'cli':
                p = cli('cache','prune'); ok = p.returncode == 0
            else:
                r, _ = mcp('nido_image',action='cache_prune',unused_only=True)
                ok = not r.get('isError',False)
            broken = not healthy(vm)
            observations[surface] = {'reported_success':ok,'transitive_base_deleted':not base.exists(),'vm_backing_chain_broken':broken}
            assert ok and base.exists() and not broken
    elif case == 'cache-relative':
        for surface in ['cli','mcp']:
            base = HOME_DIR/'images'/f'base-{surface}.qcow2'
            vm = HOME_DIR/'vms'/f'{surface}.qcow2'
            disk(base); disk(vm, f'../images/base-{surface}.qcow2')
            assert healthy(vm)
            if surface == 'cli':
                p = cli('cache','prune'); ok = p.returncode == 0
            else:
                r, _ = mcp('nido_image',action='cache_prune',unused_only=True)
                ok = not r.get('isError',False)
            broken = not healthy(vm)
            observations[surface] = {'reported_success':ok,'relative_base_deleted':not base.exists(),'vm_backing_chain_broken':broken}
            assert ok and base.exists() and not broken
    elif case == 'config-port':
        for surface in ['cli','mcp']:
            name = f'port-{surface}'
            disk(HOME_DIR/'vms'/f'{name}.qcow2')
            state(name, raw_qemu_args=['-accel','tcg','-S'])
            if surface == 'cli':
                p = cli('config',name,'--port=http:80'); assert p.returncode == 0
                p = cli('start',name); assert p.returncode == 0
            else:
                r, _ = mcp('nido_vm',action='config_update',name=name,ports=['http:80']); assert not r.get('isError',False)
                r, _ = mcp('nido_vm',action='start',name=name); assert not r.get('isError',False)
            try:
                saved = read_state(name)
                cmdline = pathlib.Path(f'/proc/{saved["pid"]}/cmdline').read_bytes().decode().split('\0')
                netdev = cmdline[cmdline.index('-netdev')+1]
                observations[surface] = {'saved_host_port':saved['forwarding'][0]['host_port'],'runtime_netdev':netdev,'guest80_missing':':80' not in netdev}
                assert saved['forwarding'][0]['host_port'] > 0 and f"hostfwd=tcp:127.0.0.1:{saved['forwarding'][0]['host_port']}-:80" in netdev
            finally:
                cli('stop',name)
            if surface == 'cli':
                assert cli('config',name,'--port=http:80:31080').returncode == 0
                assert cli('start',name).returncode == 0
            else:
                r, _ = mcp('nido_vm',action='config_update',name=name,ports=['http:80:31080']); assert not r.get('isError',False)
                r, _ = mcp('nido_vm',action='start',name=name); assert not r.get('isError',False)
            try:
                saved = read_state(name)
                cmdline = pathlib.Path(f'/proc/{saved["pid"]}/cmdline').read_bytes().decode().split('\0')
                netdev = cmdline[cmdline.index('-netdev')+1]
                observations[surface]['explicit_port_control'] = 'hostfwd=tcp:127.0.0.1:31080-:80' in netdev
                assert observations[surface]['explicit_port_control']
            finally:
                cli('stop',name)
    elif case == 'mcp-config-stale':
        requests = [dict(action='config_get'),dict(action='config_set',key='SSH_USER',value='new-audit-user'),dict(action='config_get')]
        new_images = HOME_DIR/'images-new'
        new_images.mkdir()
        (new_images/'unused.qcow2').write_bytes(b'only a disposable cache entry')
        requests += [dict(action='config_set',key='IMAGE_DIR',value=str(new_images))]
        payload = ''.join(json.dumps({'jsonrpc':'2.0','id':i,'method':'tools/call','params':{'name':'nido_system','arguments':a}})+'\n' for i,a in enumerate(requests))
        payload += json.dumps({'jsonrpc':'2.0','id':9,'method':'tools/call','params':{'name':'nido_image','arguments':{'action':'cache_prune','unused_only':True}}})+'\n'
        p = run([NIDO,'mcp'],input=payload)
        assert not (new_images/'unused.qcow2').exists()
        bodies = [json.loads(json.loads(line)['result']['content'][0]['text']) for line in p.stdout.splitlines()]
        observations = {'before':bodies[0]['config']['SSHUser'],'set_response':bodies[1]['config']['SSHUser'],'after_same_session':bodies[2]['config']['SSHUser'],'file_contains_new_value':'SSH_USER=new-audit-user' in (HOME_DIR/'config.env').read_text()}
        assert observations == {'before':'audit','set_response':'new-audit-user','after_same_session':'new-audit-user','file_contains_new_value':True}
        control = cli('config')
        observations['storage_operation_uses_new_path'] = True
        observations['fresh_cli_reads_new_value'] = json.loads(control.stdout)['data']['config']['ssh_user'] == 'new-audit-user'
        assert observations['fresh_cli_reads_new_value']
    elif case == 'storage-errors':
        unused = HOME_DIR/'images'/'unused-v1.qcow2'
        disk(unused)
        (HOME_DIR/'vms'/'unreadable.qcow2').write_bytes(b'QFI\xfb\x00\x00\x00\x03')
        p = cli('cache','remove','unused:v1')
        r, _ = mcp('nido_image',action='cache_remove',image='unused:v1')
        assert p.returncode != 0 and r.get('isError',False) and unused.exists()
        p = cli('cache','prune')
        r, _ = mcp('nido_image',action='cache_prune',unused_only=True)
        assert p.returncode != 0 and r.get('isError',False) and unused.exists()
        observations = {'inspection_failure_preserves_cache':True}
    elif case == 'controls':
        state('fixture', raw_qemu_args=['-name','initial'])
        assert cli('config','fixture','--qemu-arg=-S').returncode == 0
        assert read_state('fixture')['raw_qemu_args'] == ['-S']
        assert cli('config','fixture','--cpus=2').returncode == 0
        assert read_state('fixture')['raw_qemu_args'] == ['-S']
        r, _ = mcp('nido_vm',action='config_update',name='fixture',raw_qemu_args=[])
        assert not r.get('isError',False) and read_state('fixture').get('raw_qemu_args',[]) == []
        before = (HOME_DIR/'run'/'fixture.json').read_bytes()
        r, _ = mcp('nido_vm',action='config_update',name='fixture',raw_qemu_args=None)
        assert r.get('isError',False) and before == (HOME_DIR/'run'/'fixture.json').read_bytes()
        p = cli('config','fixture','--memory=127')
        assert p.returncode != 0 and before == (HOME_DIR/'run'/'fixture.json').read_bytes()
        base = HOME_DIR/'images'/'protected-v1.qcow2'
        vm = HOME_DIR/'vms'/'linked.qcow2'
        disk(base); disk(vm,base)
        assert cli('cache','prune').returncode == 0 and base.exists() and healthy(vm)
        observations = {'raw_qemu_args_replace_omit_clear':True,'invalid_updates_preserve_state':True,'cli_prune_keeps_direct_absolute_backing':True}
    return observations

def main():
    if len(sys.argv) == 3 and sys.argv[1] == '--inside':
        case = sys.argv[2]
        try:
            observations = inside(case)
            result = 'PASS'
            (ROOT/'evidence'/f'{case}.json').write_text(json.dumps({'result':result,'observations':observations,'calls':LOG},indent=2))
            print(json.dumps({'case':case,'result':result,'observations':observations}))
        except BaseException:
            (ROOT/'evidence'/f'{case}.json').write_text(json.dumps({'result':'HARNESS_FAILED','calls':LOG},indent=2))
            raise
        return
    (ROOT/'tmp').mkdir()
    (ROOT/'evidence').mkdir()
    shutil.copy2(pathlib.Path(__file__).resolve(),ROOT/'probe.py')
    cases = sys.argv[1:] or CASES
    failed = False
    for case in cases:
        fixture = ROOT/f'fixture-{case}'
        assert case in CASES
        if fixture.exists(): shutil.rmtree(fixture)
        fixture.mkdir(exist_ok=True)
        (fixture/'bin').mkdir()
        cmd = ['bwrap','--ro-bind','/','/','--dev','/dev','--unshare-user','--unshare-pid','--unshare-net','--proc','/proc','--tmpfs',str(pathlib.Path.home()),'--bind',str(ROOT),str(ROOT),'--bind',str(fixture),str(HOME_DIR),'--ro-bind',os.environ['NIDO_TEST_BINARY'],str(HOME_DIR/'bin'/'nido'),'--setenv','NIDO_SAFETY_ROOT',str(ROOT),'--setenv','TMPDIR',str(ROOT/'tmp'),'--chdir',str(ROOT),'--','/usr/bin/python3',str(ROOT/'probe.py'),'--inside',case]
        p = subprocess.run(cmd,capture_output=True,text=True,timeout=100)
        print(p.stdout or p.stderr,flush=True)
        if p.returncode: failed = True
    output = os.environ.get('NIDO_TEST_EVIDENCE')
    if output:
        pathlib.Path(output).mkdir(parents=True,exist_ok=True)
        for proof in (ROOT/'evidence').glob('*.json'): shutil.copy2(proof,pathlib.Path(output)/proof.name)
    shutil.rmtree(ROOT)
    sys.exit(1 if failed else 0)

if __name__ == '__main__':
    try: main()
    finally:
        if not INSIDE and ROOT.exists(): shutil.rmtree(ROOT)
