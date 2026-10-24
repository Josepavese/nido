#!/usr/bin/env python3
"""Independent-process CLI/MCP and actual two-PTY TUI event qualification.

Uses only a private fixture home. No VM process, downloads or real user data.
PTY checks run on Unix; native transport contract tests run on every CI host.
"""
import json
import os
import pathlib
import pty
import queue
import select
import struct
import subprocess
import tempfile
import threading
import time
import fcntl
import termios

BINARY = str(pathlib.Path(os.environ['NIDO_TEST_BINARY']).resolve())
STORAGE = os.environ.get('NIDO_TEST_STORAGE', tempfile.gettempdir())
TIMEOUT = 8


class MCP:
    def __init__(self, environment, cwd):
        self.process = subprocess.Popen([BINARY, 'mcp'], env=environment, cwd=cwd,
                                        stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                        stderr=subprocess.PIPE, text=True)
        self.frames = queue.Queue()
        self.notifications = []
        self.ident = 0
        self.thread = threading.Thread(target=self.read, daemon=True)
        self.thread.start()
        self.call('initialize', {'protocolVersion': '2024-11-05', 'capabilities': {},
                                'clientInfo': {'name': 'event-qualification', 'version': '1'}})
        self.send({'jsonrpc': '2.0', 'method': 'notifications/initialized'})

    def read(self):
        try:
            for line in self.process.stdout:
                self.frames.put(json.loads(line))
        except Exception as error:
            self.frames.put(error)

    def send(self, frame):
        self.process.stdin.write(json.dumps(frame) + '\n')
        self.process.stdin.flush()

    def call(self, method, params):
        self.ident += 1
        self.send(dict(jsonrpc='2.0', id=self.ident, method=method, params=params))
        deadline = time.monotonic() + TIMEOUT
        while time.monotonic() < deadline:
            frame = self.frames.get(timeout=max(.01, deadline - time.monotonic()))
            assert isinstance(frame, dict), frame
            if 'method' in frame:
                self.notifications.append(frame)
            elif frame.get('id') == self.ident:
                assert 'error' not in frame, frame
                return frame['result']
        raise AssertionError('MCP response deadline')

    def tool(self, **arguments):
        result = self.call('tools/call', {'name': 'nido_events', 'arguments': arguments})
        assert not result.get('isError'), result
        return json.loads(result['content'][0]['text'])

    def resource(self, uri):
        return json.loads(self.call('resources/read', {'uri': uri})['contents'][0]['text'])

    def collect(self, duration):
        deadline = time.monotonic() + duration
        while time.monotonic() < deadline:
            try:
                frame = self.frames.get(timeout=max(.001, deadline - time.monotonic()))
            except queue.Empty:
                break
            assert isinstance(frame, dict), frame
            assert 'method' in frame, frame
            self.notifications.append(frame)

    def notified(self, uri, timeout=TIMEOUT):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            for i, frame in enumerate(self.notifications):
                assert frame['method'] == 'notifications/resources/updated', frame
                assert set(frame['params']) == {'uri'}, frame
                if frame['params']['uri'] == uri:
                    self.notifications.pop(i)
                    return
            self.collect(min(.2, max(.001, deadline - time.monotonic())))
        raise AssertionError(f'No resource notification: {uri}')

    def close(self):
        self.process.stdin.close()
        self.process.wait(timeout=TIMEOUT)
        self.thread.join(timeout=TIMEOUT)
        assert self.process.returncode == 0, self.process.stderr.read()


class TUI:
    def __init__(self, environment, cwd):
        self.master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack('HHHH', 45, 150, 0, 0))
        self.process = subprocess.Popen([BINARY, 'gui'], env=environment, cwd=cwd,
                                        stdin=slave, stdout=slave, stderr=slave,
                                        start_new_session=True)
        os.close(slave)
        self.output = bytearray()
        self.stop = threading.Event()
        self.thread = threading.Thread(target=self.read, daemon=True)
        self.thread.start()

    def read(self):
        while not self.stop.is_set():
            if select.select([self.master], [], [], .1)[0]:
                try:
                    data = os.read(self.master, 65536)
                except OSError:
                    return
                if not data:
                    return
                self.output.extend(data)
                assert len(self.output) < 2 * 1024 * 1024, 'unbounded PTY trace'

    def key(self, key):
        os.write(self.master, key)

    def text(self, value, after=0):
        deadline = time.monotonic() + TIMEOUT
        while time.monotonic() < deadline:
            if value.encode() in self.output[after:]:
                return
            assert self.process.poll() is None, self.output.decode(errors='replace')[-2000:]
            time.sleep(.025)
        raise AssertionError(f'TUI did not render {value!r}: ' + self.output.decode(errors='replace')[-4000:])

    def close(self):
        self.key(b'\x03')
        try:
            self.process.wait(timeout=TIMEOUT)
        finally:
            if self.process.poll() is None:
                self.process.kill()
                self.process.wait()
            self.stop.set()
            self.thread.join(timeout=TIMEOUT)
            os.close(self.master)
        assert self.process.returncode == 0, self.output.decode(errors='replace')[-2000:]


def run():
    with tempfile.TemporaryDirectory(prefix='ne-', dir=STORAGE) as directory:
        root = pathlib.Path(directory)
        home = root/'.nido'
        for name in ['run', 'vms', 'images', 'backups']:
            (home/name).mkdir(parents=True)
        (root/'registry').mkdir()
        catalog = json.dumps(dict(schema_version='1.0', updated_at='2026-10-24T00:00:00Z', images=[]))
        (root/'registry/images.json').write_text(catalog)
        (home/'images/.catalog.json').write_text(catalog)
        (home/'backups/template-a.compact.qcow2').write_bytes(b'private template metadata fixture')
        (home/'config.env').write_text(f'SSH_USER=fixture\nIMAGE_DIR={home}/images\nBACKUP_DIR={home}/backups\n')
        vm = 'event-vm'
        state = dict(name=vm, pid=0, memory_mb=512, vcpus=1, ssh_port=51022, ssh_user='fixture')
        (home/f'vms/{vm}.qcow2').write_bytes(b'private metadata fixture, no VM launch')
        state_path = home/f'run/{vm}.json'
        state_path.write_text(json.dumps(state))
        environment = dict(os.environ, HOME=str(root), USERPROFILE=str(root), TERM='xterm-256color')
        environment.pop('NIDO_JSON', None)
        processes = []
        owner = subprocess.Popen([BINARY, 'events', '--json'], env=environment, cwd=root,
                                  stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        try:
            first = json.loads(owner.stdout.readline())['data']
            assert first['kind'] == 'stream.resync', first
            active, silent = MCP(environment, root), MCP(environment, root)
            processes += [active, silent]
            uri = f'nido://vm/{vm}'
            active.call('resources/subscribe', {'uri': uri})
            a, b = TUI(environment, root), TUI(environment, root)
            processes += [a, b]
            a.text(vm)
            b.text(vm)
            a.text('512')
            b.text('512')
            # Move the second real terminal to another page while mutations arrive.
            route_cut = len(b.output)
            b.key(b'\x1b[C')
            b.text('SPAWN', route_cut)
            b.key(b'\r')
            b.key(b'draft-vm')
            b.text('draft-vm')

            def configure(memory):
                result = subprocess.run([BINARY, 'config', vm, '--memory', str(memory), '--json'],
                                        env=environment, cwd=root, capture_output=True, text=True,
                                        timeout=TIMEOUT, check=True)
                assert json.loads(result.stdout)['status'] == 'ok', result.stdout

            cuts = [len(a.output), len(b.output)]
            configure(640)
            active.notified(uri)
            detail = active.resource(uri)["vm"]
            assert detail['memory_mb'] == 640, detail
            a.text('640', cuts[0])
            active.call('resources/subscribe', {'uri': 'nido://fleet/templates'})
            (home/'backups/template-b.compact.qcow2').write_bytes(b'another private template metadata fixture')
            active.notified('nido://fleet/templates')
            draft_cut = len(b.output)
            b.key(b'x')
            b.text('draft-vmx', draft_cut)
            active.call('resources/unsubscribe', {'uri': 'nido://fleet/templates'})
            escape_cut = len(b.output)
            b.key(b'\x1b')
            b.text('bird-name', escape_cut)
            b.key(b'\x1b[D')
            b.text('640', cuts[1])
            active.collect(.6)
            active.notifications.clear()
            active.tool(action='mute', uri=uri)
            configure(768)
            active.collect(.8)
            assert not active.notifications, active.notifications
            silent.collect(.1)
            assert not silent.notifications, silent.notifications
            active.tool(action='unmute', uri=uri)
            active.notified(uri)
            assert active.resource(uri)['vm']['memory_mb'] == 768
            # Kill only this test's broker to prove live-session takeover.
            before = active.tool(action='status')['epoch']
            owner.kill()
            owner.wait(timeout=TIMEOUT)
            active.notified(uri)
            after = active.tool(action='status')['epoch']
            assert before != after, (before, after)
            cuts = [len(a.output), len(b.output)]
            state['memory_mb'] = 896
            temporary = state_path.with_suffix('.stage')
            temporary.write_text(json.dumps(state))
            temporary.replace(state_path)
            active.notified(uri)
            assert active.resource(uri)['vm']['memory_mb'] == 896
            a.text('896', cuts[0])
            b.text('896', cuts[1])
            cuts = [len(a.output), len(b.output)]
            state_path.unlink()
            (home/f'vms/{vm}.qcow2').unlink()
            active.notified(uri)
            assert not active.resource('nido://fleet/vms')['vms']
            a.text('Select VM', cuts[0])
            b.text('Select VM', cuts[1])
            print(json.dumps(dict(result='PASS', scope='Linux real PTY and independent processes',
                                  two_tuis=True, background_route=True, draft_preserved=True, cli_to_mcp=True,
                                  no_unsolicited_notifications=True, mute_unmute=True,
                                  external_atomic_update=True, owner_crash_takeover=True,
                                  external_delete=True)))
        finally:
            for process in reversed(processes):
                process.close()
            if owner.poll() is None:
                owner.terminate()
                owner.wait(timeout=TIMEOUT)
        assert not (home/'events/bus.sock').exists(), 'last consumer left event socket'


if __name__ == '__main__':
    run()
