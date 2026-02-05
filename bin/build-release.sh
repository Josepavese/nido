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

# Cleanup raw binaries
rm -f $OUTPUT_DIR/nido-* $OUTPUT_DIR/nido.exe

echo ""
echo "✅ Build & Package complete! Archives in $OUTPUT_DIR/"
ls -lh $OUTPUT_DIR/*.{tar.gz,zip}
