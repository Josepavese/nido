#!/usr/bin/env bash
# Build release binaries for all platforms

set -euo pipefail

VERSION=${1:?Usage: build-release.sh VERSION [OUTPUT_DIR]}
OUTPUT_DIR=${2:-"${RUNNER_TEMP:-${TMPDIR:-/tmp}}/nido-release-${VERSION}"}
mkdir -p "$OUTPUT_DIR"
OUTPUT_DIR=$(cd "$OUTPUT_DIR" && pwd)
case "$OUTPUT_DIR/" in "$PWD/"*) echo "Release output must be outside the repository" >&2; exit 1;; esac
for tool in go tar zip jq; do command -v "$tool" >/dev/null || { echo "Missing build tool: $tool" >&2; exit 1; }; done
bash bin/verify-release.sh --metadata server.json
# Each invocation owns a private staging directory; never purge unrelated outputs.
STAGING_DIR=$(mktemp -d "$OUTPUT_DIR/.staging.XXXXXX")
trap 'rm -rf "$STAGING_DIR"' EXIT
BUNDLE_DIR="$STAGING_DIR/bundle"
mkdir -p "$BUNDLE_DIR"
BINARIES=()
CHECKSUM_FILES=()
while IFS=$'\t' read -r target_os target_arch archive_ext; do
    name="nido-${target_os}-${target_arch}"
    suffix=""; [ "$target_os" != windows ] || suffix=".exe"
    pkg_dir="$STAGING_DIR/$name"
    mkdir -p "$pkg_dir"
    for command_name in nido nido-validator; do
        GOOS="$target_os" GOARCH="$target_arch" go build -ldflags "-X github.com/Josepavese/nido/v5/internal/build.Version=$VERSION" -o "$pkg_dir/$command_name$suffix" "./cmd/$command_name"
    done
    # Flat executable names remain unique in the MCP bundle.
    bundle_binary="$name$suffix"
    cp "$pkg_dir/nido$suffix" "$BUNDLE_DIR/$bundle_binary"
    BINARIES+=("$bundle_binary")
    cp -R registry "$pkg_dir/registry"
    rm -f "$OUTPUT_DIR/$name.$archive_ext"
    if [ "$archive_ext" = zip ]; then
        (cd "$STAGING_DIR" && zip -qr "$OUTPUT_DIR/$name.zip" "$name")
    else
        (cd "$STAGING_DIR" && tar -czf "$OUTPUT_DIR/$name.tar.gz" "$name")
    fi
    rm -rf "$pkg_dir"
    CHECKSUM_FILES+=("$name.$archive_ext")
done < <(jq -r '.[] | [.os,.arch,.extension] | @tsv' internal/build/targets.json)
echo ""
echo "🧠 Generating MCP Assets..."

# Launchers select one native executable inside the local bundle.
cat > "$BUNDLE_DIR/nido" <<'LAUNCHER'
#!/bin/sh
set -eu
case "$(uname -s)" in Linux) target_os=linux;; Darwin) target_os=darwin;; *) exit 1;; esac
case "$(uname -m)" in x86_64|amd64) target_arch=amd64;; arm64|aarch64) target_arch=arm64;; *) exit 1;; esac
if [ "$target_os" = darwin ] && [ "$(sysctl -n hw.optional.arm64 2>/dev/null || true)" = 1 ]; then target_arch=arm64; fi
exec "$(dirname "$0")/nido-$target_os-$target_arch" "$@"
LAUNCHER
cat > "$BUNDLE_DIR/nido.cmd" <<'LAUNCHER'
@echo off
setlocal
set "nido_arch=%PROCESSOR_ARCHITECTURE%"
if defined PROCESSOR_ARCHITEW6432 set "nido_arch=%PROCESSOR_ARCHITEW6432%"
if /I "%nido_arch%"=="AMD64" (set "nido_arch=amd64") else if /I "%nido_arch%"=="ARM64" (set "nido_arch=arm64") else (exit /b 1)
"%~dp0nido-windows-%nido_arch%.exe" %*
LAUNCHER
chmod 0755 "$BUNDLE_DIR/nido"
BINARIES+=(nido nido.cmd)
# The MCP bundle is a flat tarball of runnable server binaries.
(cd "$BUNDLE_DIR" && tar -czf "$OUTPUT_DIR/nido.mcpb" "${BINARIES[@]}")
echo "  Created nido.mcpb"

# Generate server.json with SHA256 injection.
MCPB_PATH="${OUTPUT_DIR}/nido.mcpb"
if command -v sha256sum >/dev/null 2>&1; then
    SHA=$(sha256sum "$MCPB_PATH" | awk '{print $1}')
else
    SHA=$(shasum -a 256 "$MCPB_PATH" | awk '{print $1}')
fi
REPO=${GITHUB_REPOSITORY:-"Josepavese/nido"}
STRICT_VERSION=${VERSION#v}
URL="https://github.com/${REPO}/releases/download/${VERSION}/nido.mcpb"

echo "  Injecting version ($STRICT_VERSION) and SHA into server.json..."
jq --arg v "${STRICT_VERSION}" --arg s "$SHA" --arg u "$URL" \
  '.version = $v
   | (.packages[] | select(.registryType == "mcpb")) |= (.version = $v | .fileSha256 = $s | .identifier = $u)
   | (.packages[] | select(.registryType == "npm")) |= (.version = $v)' \
  server.json > "${OUTPUT_DIR}/server.json"
echo "  Created dist/server.json"

cp installers/quick-install.sh "${OUTPUT_DIR}/install.sh"
cp installers/quick-install.ps1 "${OUTPUT_DIR}/install.ps1"
chmod 0644 "${OUTPUT_DIR}/install.ps1"
chmod 0755 "${OUTPUT_DIR}/install.sh"
echo "  Created installer assets"

CHECKSUM_FILES+=(nido.mcpb server.json install.sh install.ps1)

if command -v sha256sum &> /dev/null; then
    (cd "$OUTPUT_DIR" && sha256sum "${CHECKSUM_FILES[@]}" > SHA256SUMS)
    echo "  Created SHA256SUMS"
elif command -v shasum &> /dev/null; then
    (cd "$OUTPUT_DIR" && shasum -a 256 "${CHECKSUM_FILES[@]}" > SHA256SUMS)
    echo "  Created SHA256SUMS"
fi

echo ""
echo "✅ Build & Package complete! Archives in $OUTPUT_DIR/"
ls -lh "$OUTPUT_DIR"/*.{tar.gz,zip,mcpb,json,sh,ps1} "$OUTPUT_DIR"/SHA256SUMS 2>/dev/null || true
