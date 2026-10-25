#!/usr/bin/env python3
"""Native resize/focus/mouse regression, with optional visual HTML from pyte.

NIDO_TEST_BINARY=/external/scratch/nido python3 tests/tui_resize.py
Set NIDO_TUI_VISUAL_DIR to export bounded, reviewable terminal snapshots.
Visual export needs pyte in the observer environment, never in product runtime.
All data is private metadata; this never launches, edits or deletes a real VM.
"""
import hashlib
import json
import os
import pathlib
import tempfile
import time

from tui_terminal import TUI, Observer

VISUAL = os.environ.get('NIDO_TUI_VISUAL_DIR')




def run():
    snapshots = []
    with tempfile.TemporaryDirectory(prefix='nido-resize-', dir=os.environ.get('NIDO_TEST_STORAGE')) as directory:
        root = pathlib.Path(directory)
        home = root / '.nido'
        for name in ['run', 'vms', 'images', 'backups']:
            (home/name).mkdir(parents=True)
        (root/'registry').mkdir()
        catalog = json.dumps(dict(schema_version='1.0', updated_at='2026-10-25T00:00:00Z', images=[]))
        (root/'registry/images.json').write_text(catalog)
        (home/'images/.catalog.json').write_text(catalog)
        config = home/'config.env'
        config.write_text(f'SSH_USER=fixture\nIMAGE_DIR={home}/images\nBACKUP_DIR={home}/backups\n')
        disk = home/'vms/robin.qcow2'
        disk.write_bytes(b'private metadata fixture; never boot')
        state = home/'run/robin.json'
        state.write_text(json.dumps(dict(name='robin', pid=0, memory_mb=512,
                                        vcpus=1, ssh_port=51022, ssh_user='fixture')))
        (home/'backups/template-a.compact.qcow2').write_bytes(b'private template metadata fixture')
        protected = {path: hashlib.sha256(path.read_bytes()).hexdigest() for path in [disk, state, config]}
        environment = dict(os.environ, HOME=str(root), USERPROFILE=str(root),
                           TERM='xterm-256color', COLORTERM='truecolor',
                           COLORFGBG='15;0', NIDO_THEME=os.environ.get('NIDO_TUI_TEST_THEME','dark'))
        for key in ['NO_COLOR', 'NIDO_JSON']:
            environment.pop(key, None)
        observer = Observer() if VISUAL else None
        tui = TUI(environment, root, size=(120, 36), observer=observer)

        def capture(name):
            # Wait for the bounded Bubble Tea renderer flush, not a product task.
            time.sleep(.18)
            assert tui.process.poll() is None, tui.output.decode(errors='replace')[-2000:]
            assert b'panic:' not in tui.output
            if observer:
                snapshots.append(observer.snapshot(name))

        def key(data, name=None):
            cut = len(tui.output)
            tui.key(data)
            if name:
                tui.text(name, cut)
            time.sleep(.06)

        def resize(w, h, label):
            tui.resize(w, h)
            capture(label)

        try:
            tui.text('robin')
            capture('Fleet - wide')
            resize(80, 24, 'Fleet - compact')
            resize(60, 15, 'Fleet - list')
            key(b'\r')  # Compact Enter opens detail, never toggles power.
            capture('Fleet - detail')
            key(b'\x1b[6~')
            capture('Fleet - paged detail')
            key(b'\x1b')
            key(b'\x1b[C')
            key(b'\r')
            key(b'egg-draft', 'egg-draft')
            resize(40, 10, 'Hatchery - minimum, draft')
            # Caret movement must stay in the input, then insert in the middle.
            key(b'\x1b[D'); key(b'X', 'egg-drafXt')
            capture('Hatchery - caret editing')
            resize(12, 3, 'Resting state')
            key(b'BAD\r')  # Hidden inputs are suspended.
            resize(60, 15, 'Hatchery - recovered draft')
            assert b'egg-drafXtBAD' not in tui.output
            for _ in range(7):
                key(b'\t')
            capture('Hatchery - final controls')
            # Mouse route uses the compact SYSTEM tab's actual rendered position.
            key(b'\x1b[<0;30;1M')
            capture('System - clicked tab')
            key(b'\r')
            resize(40, 10, 'System - minimum form')
            for _ in range(4):
                key(b'\t')
            capture('System - save reachable')
            key(b'\r', 'Confirm Configuration')
            capture('System - confirmation')
            resize(120,36,'System - expanded confirmation')
            if observer:
                # Locate the displayed No button; use real SGR mouse input.
                for y, line in enumerate(observer.screen.display):
                    if 'Yes' in line and 'No' in line:
                        key(f'\x1b[<0;{line.index("No")+1};{y+1}M'.encode())
                        break
                else:
                    raise AssertionError('visible cancellation button missing')
            else:
                key(b'\x1b')
            resize(40,10,'System - cancelled confirmation')  # Cancel. Never apply fixture configuration.
            key(b'\x1b')
            key(b'\x1b[B'); key(b'\r'); key(b'\r', 'Select Theme')
            capture('Theme picker - minimum')
            resize(120, 36, 'Theme picker - wide')
            key(b'\x1b'); key(b'\x1b'); key(b'\x1b[C')
            capture('About - wide')
            resize(40, 10, 'About - minimum')
            key(b'\x1b[6~')
            capture('About - scroll')
            key(b'\x1b[D'); key(b'\x1b[D')
            capture('Registry - minimum list')
            key(b'\r')
            capture('Registry - minimum detail')
            resize(120,36,'Registry - wide recovery')
            # Audit every System detail without executing its action.
            key(b'\x1b[C')
            key(b'\x1b[H')
            for page in ['Core', 'Rice', 'Evolution', 'Health', 'Artifacts', 'Self Destruct']:
                key(b'\r')
                for width, height in [(120, 36), (80, 24), (60, 15), (40, 10)]:
                    resize(width, height, f'System {page} - audit')
                if page == 'Self Destruct':
                    key(b'\r', 'SELF DESTRUCT')
                    capture('Uninstall - cancel only')
                    key(b'\x1b')
                key(b'\x1b')
                key(b'\x1b[B')
            # Fleet template text and both actions survive a resize round trip.
            key(b'\x1b[C'); key(b'\x1b[C')
            key(b't', 'Create Template')
            key('città-bird'.encode())
            for width, height in [(120, 36), (80, 24), (60, 15), (40, 10)]:
                resize(width, height, 'Template - editing')
            key(b'\t')
            key(b'q')  # A dialog button must not turn q into a hidden quit.
            capture('Template - confirm reachable')
            key(b'\t'); capture('Template - cancel reachable')
            key(b'\x1b')
            assert b'Spawning ' not in tui.output, 'audit activated a spawn action'
            # Start from a known focus state for the forwarding journey.
            tui.close()
            observer = Observer() if VISUAL else None
            tui = TUI(environment, root, size=(120, 36), observer=observer)
            tui.text('robin')
            key(b'\x1b[C'); key(b'\r')
            for _ in range(6):
                key(b'\t')
            key(b'\r', 'Add Port Forwarding')
            for width, height in [(120, 36), (80, 24), (60, 15), (40, 10)]:
                resize(width, height, 'Forwarding - editing')
            key(b'\t'); key(b'\t'); key(b'8080')
            key(b'\t'); key(b'\t'); capture('Forwarding - confirm reachable')
            key(b'\t'); capture('Forwarding - cancel reachable')
            key(b'\x1b')
            assert b'Spawning ' not in tui.output, 'audit activated a spawn action'
        finally:
            tui.close(b'\x1b\x03')  # Immediate Esc + Ctrl+C must still exit.
        for path, digest in protected.items():
            assert hashlib.sha256(path.read_bytes()).hexdigest() == digest, f'Unexpected fixture mutation: {path}'
        assert not (home/'events/bus.sock').exists(), 'terminal left event owner running'

    if VISUAL:
        target = pathlib.Path(VISUAL)
        target.mkdir(parents=True, exist_ok=True)
        style = ('<!doctype html><meta charset="utf-8"><style>'
                 'body{background:#080a10;color:#cdd6f4;font-family:sans-serif}'
                 'h2{font-size:15px}pre{font:14px/1.3 "DejaVu Sans Mono",monospace;'
                 'white-space:pre;background:#0f1216;padding:16px;width:max-content}'
                 'section{display:inline-block;vertical-align:top;margin:12px}</style>')
        links = '<nav>' + ' '.join(f'<a href="frame-{i:02}.html">{i:02}</a>' for i in range(len(snapshots))) + '</nav>'
        for i, snapshot in enumerate(snapshots):
            (target/f'frame-{i:02}.html').write_text(style + links + snapshot)
        (target/'index.html').write_text(style + links + ''.join(f'<section>{s}</section>' for s in snapshots))
    print('PASS: real PTY resize, caret, focus, mouse route, modal cancel, recovery; fixture bytes unchanged')


if __name__ == '__main__':
    run()
