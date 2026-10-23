'use strict';

// Resolves, downloads, verifies and caches the Nido binary shipped in GitHub
// Releases. No npm dependencies: Node >= 18 only.
//
// Asset contract (published by bin/build-release.sh):
//   nido-linux-amd64.tar.gz, nido-linux-arm64.tar.gz,
//   nido-darwin-amd64.tar.gz, nido-darwin-arm64.tar.gz,
//   nido-windows-amd64.zip, SHA256SUMS
// Each archive contains a top-level nido-<os>-<arch>/ directory with the
// nido (or nido.exe) binary plus the bundled registry.

const crypto = require('node:crypto');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const { Readable } = require('node:stream');
const { pipeline } = require('node:stream/promises');

const REPO = 'Josepavese/nido';
const pkg = require('../package.json');

function verbose() {
  return process.env.NIDO_NPM_VERBOSE === '1';
}

function log(...args) {
  if (verbose()) console.error('[nido]', ...args);
}

function target() {
  const platform = process.platform === 'win32' ? 'windows' : process.platform;
  if (!['linux', 'darwin', 'windows'].includes(platform)) {
    throw new Error(`unsupported platform: ${process.platform}`);
  }
  let arch = process.arch;
  if (arch === 'x64') arch = 'amd64';
  if (arch !== 'amd64' && arch !== 'arm64') {
    throw new Error(`unsupported architecture: ${process.arch}`);
  }
  const ext = platform === 'windows' ? 'zip' : 'tar.gz';
  return {
    platform,
    arch,
    ext,
    asset: `nido-${platform}-${arch}.${ext}`,
    binName: platform === 'windows' ? 'nido.exe' : 'nido',
    folder: `nido-${platform}-${arch}`,
  };
}

function cacheRoot() {
  if (process.env.NIDO_NPM_CACHE) return process.env.NIDO_NPM_CACHE;
  if (process.platform === 'win32') {
    return path.join(process.env.LOCALAPPDATA || path.join(os.homedir(), 'AppData', 'Local'), 'nido-npm');
  }
  const base = process.env.XDG_CACHE_HOME || path.join(os.homedir(), '.cache');
  return path.join(base, 'nido-npm');
}

async function resolveVersion() {
  const override = process.env.NIDO_NPM_VERSION;
  if (override) return override.trim().replace(/^v/, '');
  if (/^\d+\.\d+\.\d+$/.test(pkg.version)) return pkg.version;
  // Development fallback: ask GitHub for the latest published release.
  const res = await fetch(`https://api.github.com/repos/${REPO}/releases/latest`, {
    headers: { 'User-Agent': 'nido-mcp' },
  });
  if (!res.ok) throw new Error(`cannot resolve latest Nido release (${res.status})`);
  const body = await res.json();
  if (!body.tag_name) throw new Error('latest Nido release has no tag_name');
  return String(body.tag_name).replace(/^v/, '');
}

async function download(url, dest) {
  log('download', url);
  const res = await fetch(url, { redirect: 'follow' });
  if (!res.ok) throw new Error(`download failed (${res.status}) ${url}`);
  await pipeline(Readable.fromWeb(res.body), fs.createWriteStream(dest));
}

async function sha256(file) {
  const hash = crypto.createHash('sha256');
  await pipeline(fs.createReadStream(file), hash);
  return hash.digest('hex');
}

async function expectedChecksum(baseUrl, asset) {
  const res = await fetch(`${baseUrl}SHA256SUMS`, { redirect: 'follow' });
  if (!res.ok) throw new Error(`cannot fetch SHA256SUMS (${res.status})`);
  const text = await res.text();
  for (const line of text.split('\n')) {
    const match = line.trim().match(/^([a-f0-9]{64})\s+\*?(.+)$/i);
    if (match && path.basename(match[2].trim()) === asset) return match[1].toLowerCase();
  }
  throw new Error(`no SHA256 entry for ${asset}`);
}

function extract(archive, destination, t) {
  if (t.ext === 'zip') {
    const script = `Expand-Archive -LiteralPath '${archive.replace(/'/g, "''")}' -DestinationPath '${destination.replace(/'/g, "''")}' -Force`;
    const result = spawnSync('powershell.exe', ['-NoProfile', '-NonInteractive', '-Command', script], {
      stdio: verbose() ? 'inherit' : 'pipe',
    });
    if (result.error) throw new Error(`zip extraction requires PowerShell: ${result.error.message}`);
    if (result.status !== 0) throw new Error('zip extraction failed');
    return;
  }
  const result = spawnSync('tar', ['-xzf', archive, '-C', destination], {
    stdio: verbose() ? 'inherit' : 'pipe',
  });
  if (result.error) throw new Error(`tar is required to extract Nido: ${result.error.message}`);
  if (result.status !== 0) throw new Error('tar extraction failed');
}

function findBinary(dir, t) {
  const direct = path.join(dir, t.folder, t.binName);
  if (fs.existsSync(direct)) return direct;
  const stack = [dir];
  while (stack.length) {
    const current = stack.pop();
    for (const entry of fs.readdirSync(current, { withFileTypes: true })) {
      const full = path.join(current, entry.name);
      if (entry.isDirectory()) stack.push(full);
      if (entry.isFile() && entry.name === t.binName) return full;
    }
  }
  throw new Error(`binary ${t.binName} not found in archive`);
}

async function ensureBinary() {
  if (process.env.NIDO_BINARY) return process.env.NIDO_BINARY;

  const t = target();
  const version = await resolveVersion();
  const installDir = path.join(cacheRoot(), `v${version}`, `${t.platform}-${t.arch}`);
  const binary = path.join(installDir, t.binName);
  if (fs.existsSync(binary)) return binary;

  fs.mkdirSync(cacheRoot(), { recursive: true });
  const workDir = fs.mkdtempSync(path.join(cacheRoot(), 'tmp-'));
  const archive = path.join(workDir, t.asset);
  const baseUrl = `https://github.com/${REPO}/releases/download/v${version}/`;

  try {
    await download(baseUrl + t.asset, archive);
    const actual = await sha256(archive);
    const expected = await expectedChecksum(baseUrl, t.asset);
    if (actual !== expected) {
      throw new Error(`checksum mismatch for ${t.asset}: expected ${expected}, got ${actual}`);
    }
    const extracted = path.join(workDir, 'extracted');
    fs.mkdirSync(extracted, { recursive: true });
    extract(archive, extracted, t);
    const stagedBinary = findBinary(extracted, t);

    fs.mkdirSync(path.dirname(installDir), { recursive: true });
    fs.rmSync(installDir, { recursive: true, force: true });
    fs.renameSync(path.dirname(stagedBinary), installDir);
    if (process.platform !== 'win32') fs.chmodSync(path.join(installDir, t.binName), 0o755);
    log('installed', path.join(installDir, t.binName));
    return path.join(installDir, t.binName);
  } finally {
    fs.rmSync(workDir, { recursive: true, force: true });
  }
}

module.exports = { ensureBinary, target, resolveVersion };
