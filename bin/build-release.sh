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

# macOS (Intel)
echo "Building for macOS (Intel)..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o $OUTPUT_DIR/nido-darwin-amd64 ./cmd/nido

# macOS (Apple Silicon)
echo "Building for macOS (Apple Silicon)..."
GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o $OUTPUT_DIR/nido-darwin-arm64 ./cmd/nido

# Windows (amd64)
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/Josepavese/nido/internal/build.Version=$VERSION" -o $OUTPUT_DIR/nido-windows-amd64.exe ./cmd/nido

echo ""
echo "✅ Build complete! Binaries in $OUTPUT_DIR/"
ls -lh $OUTPUT_DIR/

echo ""
echo "To create release archives:"
echo "  cd $OUTPUT_DIR"
echo "  tar -czf nido-linux-amd64.tar.gz nido-linux-amd64"
echo "  tar -czf nido-darwin-amd64.tar.gz nido-darwin-amd64"
echo "  tar -czf nido-darwin-arm64.tar.gz nido-darwin-arm64"
echo "  zip nido-windows-amd64.zip nido-windows-amd64.exe"
