#!/usr/bin/env node
'use strict';

// Thin launcher for the Nido binary. Downloads and verifies the right release
// asset on first use (lazy, so `--ignore-scripts` and pnpm build policies do
// not break the install). MCP traffic stays on stdout: only the launcher's own
// errors go to stderr.

const { spawnSync } = require('node:child_process');
const { ensureBinary } = require('../lib/install');

async function main() {
  let binary;
  try {
    binary = await ensureBinary();
  } catch (error) {
    console.error(`nido: ${error.message}`);
    process.exit(1);
  }

  const result = spawnSync(binary, process.argv.slice(2), { stdio: 'inherit' });
  if (result.error) {
    console.error(`nido: cannot start ${binary}: ${result.error.message}`);
    process.exit(1);
  }
  if (result.signal) {
    process.kill(process.pid, result.signal);
    return;
  }
  process.exit(result.status === null ? 1 : result.status);
}

main();
