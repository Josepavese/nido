'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const canonical = require('../../internal/build/targets.json');
const projection = require('./targets.json');
const { targetFor, target } = require('./install');

test('published npm targets match the release owner', () => {
  assert.deepEqual(projection, canonical);
  for (const value of canonical) {
    const resolved = targetFor(value.os, value.arch);
    assert.equal(resolved.asset, `nido-${value.os}-${value.arch}.${value.extension}`);
  }
});
test('host spellings normalize without substituting unsupported targets', () => {
  assert.equal(targetFor('win32', 'ARM64').arch, 'arm64');
  assert.equal(targetFor('linux', 'aarch64').arch, 'arm64');
  assert.equal(targetFor('darwin', 'x86_64').arch, 'amd64');
  assert.throws(() => targetFor('linux', 'armv7l'), /unsupported/);
  assert.throws(() => targetFor('freebsd', 'amd64'), /unsupported/);
  assert.ok(canonical.some(value => target().asset === `nido-${value.os}-${value.arch}.${value.extension}`));
});
