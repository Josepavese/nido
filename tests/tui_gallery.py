#!/usr/bin/env python3
"""Render TestScreenGallery JSON using the same terminal observer as PTY tests.

NIDO_TUI_GALLERY_JSON=/external/scratch/gallery.json go test ./internal/tui/app -run TestScreenGallery -count=1
PYTHONPATH=/observer/with/pyte python3 -B tests/tui_gallery.py /external/scratch/gallery.json /external/scratch/gallery
The result contains offline page states, not evidence of real provider operations.
"""
import collections
import html
import json
import pathlib
import sys
from tui_terminal import Observer


def render(source, destination):
    target = pathlib.Path(destination)
    target.mkdir(parents=True, exist_ok=True)
    groups = collections.defaultdict(list)
    for frame in json.loads(pathlib.Path(source).read_text()):
        observer = Observer()
        observer.screen.resize(lines=frame['Height'], columns=frame['Width'])
        observer.feed(frame['Content'].replace('\n', '\r\n'))
        groups[frame['Name']].append(observer.snapshot(frame['Name']))
    style = ('<!doctype html><meta charset="utf-8"><style>'
             'body{background:#080a10;color:#cdd6f4;font-family:sans-serif;margin:12px}'
             'nav{display:flex;gap:8px;flex-wrap:wrap}a{color:#89b4fa}'
             'h1{font-size:20px}h2{font-size:12px}pre{font:9px/1.25 "DejaVu Sans Mono",monospace;'
             'white-space:pre;background:#0f1216;padding:8px;width:max-content}'
             'main{display:flex;flex-wrap:wrap;gap:14px}section{width:max-content}</style>')
    links = '<details><summary>Browse screen states</summary><nav>' + ''.join(f'<a href="screen-{i:02}.html">{html.escape(name)}</a>' for i, name in enumerate(groups)) + '</nav></details>'
    for i, (name, frames) in enumerate(groups.items()):
        page = style + links + f'<h1>{html.escape(name)} - offline page rendering</h1><main>'
        page += ''.join(f'<section>{frame}</section>' for frame in frames[:4])
        (target/f'screen-{i:02}.html').write_text(page + '</main>')
    (target/'index.html').write_text(style + '<h1>Nido screen gallery</h1>' + links)
    print(f'Gallery: {len(groups)} states exported to {target}')


if __name__ == '__main__':
    render(*sys.argv[1:])
