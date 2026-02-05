#!/bin/bash
# Build release binaries for all platforms

set -e

VERSION=${1:-v3.0.0}
OUTPUT_DIR="dist"

echo "Building Nido $VERSION for all platforms..."

mkdir -p $OUTPUT_DIR

# Linux (amd64)
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o $OUTPUT_DIR/nido-linux-amd64 ./cmd/nido
GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o $OUTPUT_DIR/nido-validator-linux-amd64 ./cmd/nido-validator

# macOS (Intel)
echo "Building for macOS (Intel)..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o $OUTPUT_DIR/nido-darwin-amd64 ./cmd/nido
GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o $OUTPUT_DIR/nido-validator-darwin-amd64 ./cmd/nido-validator

# macOS (Apple Silicon)
echo "Building for macOS (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o $OUTPUT_DIR/nido-darwin-arm64 ./cmd/nido
GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o $OUTPUT_DIR/nido-validator-darwin-arm64 ./cmd/nido-validator

# Windows (amd64)
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o $OUTPUT_DIR/nido.exe ./cmd/nido
GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o $OUTPUT_DIR/nido-validator.exe ./cmd/nido-validator

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
        (cd "$STAGING_DIR" && zip -r "../${NAME}.zip" "${NAME}")
        echo "  Created ${NAME}.zip"
    else
        (cd "$STAGING_DIR" && tar -czf "../${NAME}.tar.gz" "${NAME}")
        echo "  Created ${NAME}.tar.gz"
    fi
    
    # Cleanup staging
    rm -rf "$PKG_DIR"
}

package "linux" "amd64" "nido-linux-amd64" ""
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
(cd "$STAGING_DIR" && zip -r "../${NAME}.zip" "${NAME}")
echo "  Created ${NAME}.zip"
rm -rf "$PKG_DIR"

echo ""
echo "🧠 Generating MCP Assets..."

# Create MCP Bundle
# Re-package binaries into a flat tarball for the MCP server
# We use the staging directory to collect them first
MCP_STAGING="${OUTPUT_DIR}/mcp_staging"
mkdir -p "$MCP_STAGING"

# Copy all built binaries to MCP staging
cp "${OUTPUT_DIR}/nido-linux-amd64" "$MCP_STAGING/"
cp "${OUTPUT_DIR}/nido-darwin-amd64" "$MCP_STAGING/"
cp "${OUTPUT_DIR}/nido-darwin-arm64" "$MCP_STAGING/"
cp "${OUTPUT_DIR}/nido.exe" "$MCP_STAGING/"
# Note: MCP bundle technically only needs the 'nido' binary to act as the server,
# but we include all for completeness or if the user wants to use them directly.
# However, the MCP config usually points to one executable.
# For simplicity and to match the previous logic, we'll tarball the bin directory content.
# But here we have binaries in dist root.

# Let's tarball the binaries we just built.
(cd "$OUTPUT_DIR" && tar -czf "nido.mcpb" nido-linux-amd64 nido-darwin-amd64 nido-darwin-arm64 nido.exe)
echo "  Created nido.mcpb"

# Generate server.json with SHA256 injection
if command -v jq &> /dev/null && command -v sha256sum &> /dev/null; then
    MCPB_PATH="${OUTPUT_DIR}/nido.mcpb"
    SHA=$(sha256sum "$MCPB_PATH" | awk '{print $1}')
    REPO=${GITHUB_REPOSITORY:-"Josepavese/nido"}
    URL="https://github.com/${REPO}/releases/download/${VERSION}/nido.mcpb"
    
    echo "  Injecting version and SHA into server.json..."
    jq --arg v "${VERSION}" --arg s "$SHA" --arg u "$URL" \
      '.version = $v | .packages[0].version = $v | .packages[0].fileSha256 = $s | .packages[0].identifier = $u' \
      server.json > "${OUTPUT_DIR}/server.json"
      
    echo "  Created dist/server.json"
else
    echo "⚠️  Skipping server.json generation: jq or sha256sum not found."
fi

# Cleanup raw binaries (Moved from previous location to be after MCP generation)
rm -rf "$STAGING_DIR"
rm -rf "$MCP_STAGING" # Cleanup our temp dir
rm -f $OUTPUT_DIR/nido-linux-amd64 $OUTPUT_DIR/nido-darwin-amd64 $OUTPUT_DIR/nido-darwin-arm64
rm -f $OUTPUT_DIR/nido-validator-* $OUTPUT_DIR/nido.exe $OUTPUT_DIR/nido-validator.exe

echo ""
echo "✅ Build & Package complete! Archives in $OUTPUT_DIR/"
ls -lh $OUTPUT_DIR/*.{tar.gz,zip,mcpb,json}
