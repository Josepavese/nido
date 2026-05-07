#!/usr/bin/env bash
# Build release binaries for all platforms

set -euo pipefail

VERSION=${1:-v3.0.0}
OUTPUT_DIR="dist"

echo "Building Nido $VERSION for all platforms..."

require_tool() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "Missing required build tool: $1" >&2
        exit 1
    fi
}

require_tool go
require_tool tar
require_tool zip
require_tool jq
if ! command -v sha256sum >/dev/null 2>&1 && ! command -v shasum >/dev/null 2>&1; then
    echo "Missing required SHA-256 tool: sha256sum or shasum" >&2
    exit 1
fi

mkdir -p "$OUTPUT_DIR"
rm -rf "$OUTPUT_DIR/staging" "$OUTPUT_DIR/mcp_staging"
rm -f "$OUTPUT_DIR"/nido-linux-amd64.tar.gz \
      "$OUTPUT_DIR"/nido-linux-arm64.tar.gz \
      "$OUTPUT_DIR"/nido-darwin-amd64.tar.gz \
      "$OUTPUT_DIR"/nido-darwin-arm64.tar.gz \
      "$OUTPUT_DIR"/nido-windows-amd64.zip \
      "$OUTPUT_DIR"/nido.mcpb \
      "$OUTPUT_DIR"/server.json \
      "$OUTPUT_DIR"/install.sh \
      "$OUTPUT_DIR"/install.ps1 \
      "$OUTPUT_DIR"/SHA256SUMS \
      "$OUTPUT_DIR"/nido-linux-amd64 \
      "$OUTPUT_DIR"/nido-linux-arm64 \
      "$OUTPUT_DIR"/nido-darwin-amd64 \
      "$OUTPUT_DIR"/nido-darwin-arm64 \
      "$OUTPUT_DIR"/nido.exe \
      "$OUTPUT_DIR"/nido-validator-*

# Linux (amd64)
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o "$OUTPUT_DIR/nido-linux-amd64" ./cmd/nido
GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o "$OUTPUT_DIR/nido-validator-linux-amd64" ./cmd/nido-validator

# Linux (arm64)
echo "Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o "$OUTPUT_DIR/nido-linux-arm64" ./cmd/nido
GOOS=linux GOARCH=arm64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o "$OUTPUT_DIR/nido-validator-linux-arm64" ./cmd/nido-validator

# macOS (Intel)
echo "Building for macOS (Intel)..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o "$OUTPUT_DIR/nido-darwin-amd64" ./cmd/nido
GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o "$OUTPUT_DIR/nido-validator-darwin-amd64" ./cmd/nido-validator

# macOS (Apple Silicon)
echo "Building for macOS (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o "$OUTPUT_DIR/nido-darwin-arm64" ./cmd/nido
GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o "$OUTPUT_DIR/nido-validator-darwin-arm64" ./cmd/nido-validator

# Windows (amd64)
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o "$OUTPUT_DIR/nido.exe" ./cmd/nido
GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o "$OUTPUT_DIR/nido-validator.exe" ./cmd/nido-validator

echo ""
echo "📦 Packaging releases..."

# Function to package
package() {
    OS=$1
    ARCH=$2
    BIN=$3
    EXT=$4
    
    NAME="nido-${OS}-${ARCH}"
    STAGING_DIR="${OUTPUT_DIR}/staging"
    PKG_DIR="${STAGING_DIR}/${NAME}"
    mkdir -p "$PKG_DIR"
    
    # Copy binaries
    cp "${OUTPUT_DIR}/${BIN}" "${PKG_DIR}/nido${EXT}"
    cp "${OUTPUT_DIR}/nido-validator-${OS}-${ARCH}" "${PKG_DIR}/nido-validator${EXT}"
    
    # Copy registry
    cp -r registry "${PKG_DIR}/"
    
    # Archive
    if [ "$OS" == "windows" ]; then
        (cd "$STAGING_DIR" && zip -qr "../${NAME}.zip" "${NAME}")
        echo "  Created ${NAME}.zip"
    else
        (cd "$STAGING_DIR" && tar -czf "../${NAME}.tar.gz" "${NAME}")
        echo "  Created ${NAME}.tar.gz"
    fi
    
    # Cleanup staging
    rm -rf "$PKG_DIR"
}

package "linux" "amd64" "nido-linux-amd64" ""
package "linux" "arm64" "nido-linux-arm64" ""
package "darwin" "amd64" "nido-darwin-amd64" ""
package "darwin" "arm64" "nido-darwin-arm64" ""

# Windows needs special handling for binary name in previous steps or just reuse
# I used "nido-windows-amd64.exe" but common convention is just nido.exe inside the zip
# Re-building or renaming...
# Let's just fix the packaging function to handle input name.

# Manual for Windows to ensure nido.exe naming
STAGING_DIR="${OUTPUT_DIR}/staging"
NAME="nido-windows-amd64"
PKG_DIR="${STAGING_DIR}/${NAME}"
mkdir -p "$PKG_DIR"
cp "${OUTPUT_DIR}/nido.exe" "${PKG_DIR}/nido.exe"
cp "${OUTPUT_DIR}/nido-validator.exe" "${PKG_DIR}/nido-validator.exe"
cp -r registry "${PKG_DIR}/"
(cd "$STAGING_DIR" && zip -qr "../${NAME}.zip" "${NAME}")
echo "  Created ${NAME}.zip"
rm -rf "$PKG_DIR"

echo ""
echo "🧠 Generating MCP Assets..."

# The MCP bundle is a flat tarball of runnable server binaries.
(cd "$OUTPUT_DIR" && tar -czf "nido.mcpb" nido-linux-amd64 nido-linux-arm64 nido-darwin-amd64 nido-darwin-arm64 nido.exe)
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
  '.version = $v | .packages[0].version = $v | .packages[0].fileSha256 = $s | .packages[0].identifier = $u' \
  server.json > "${OUTPUT_DIR}/server.json"
echo "  Created dist/server.json"

cp installers/quick-install.sh "${OUTPUT_DIR}/install.sh"
cp installers/quick-install.ps1 "${OUTPUT_DIR}/install.ps1"
chmod 0644 "${OUTPUT_DIR}/install.ps1"
chmod 0755 "${OUTPUT_DIR}/install.sh"
echo "  Created installer assets"

CHECKSUM_FILES=(nido-linux-amd64.tar.gz nido-linux-arm64.tar.gz nido-darwin-amd64.tar.gz nido-darwin-arm64.tar.gz nido-windows-amd64.zip nido.mcpb server.json install.sh install.ps1)

if command -v sha256sum &> /dev/null; then
    (cd "$OUTPUT_DIR" && sha256sum "${CHECKSUM_FILES[@]}" > SHA256SUMS)
    echo "  Created SHA256SUMS"
elif command -v shasum &> /dev/null; then
    (cd "$OUTPUT_DIR" && shasum -a 256 "${CHECKSUM_FILES[@]}" > SHA256SUMS)
    echo "  Created SHA256SUMS"
fi

# Cleanup raw binaries (Moved from previous location to be after MCP generation)
rm -rf "$STAGING_DIR"
rm -f "$OUTPUT_DIR"/nido-linux-amd64 "$OUTPUT_DIR"/nido-linux-arm64 "$OUTPUT_DIR"/nido-darwin-amd64 "$OUTPUT_DIR"/nido-darwin-arm64
rm -f "$OUTPUT_DIR"/nido-validator-* "$OUTPUT_DIR"/nido.exe "$OUTPUT_DIR"/nido-validator.exe

echo ""
echo "✅ Build & Package complete! Archives in $OUTPUT_DIR/"
ls -lh "$OUTPUT_DIR"/*.{tar.gz,zip,mcpb,json,sh,ps1} "$OUTPUT_DIR"/SHA256SUMS 2>/dev/null || true
