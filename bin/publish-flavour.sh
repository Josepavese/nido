#!/bin/bash
# Nido Flavour Publisher 🐣📦
# Usage: ./bin/publish-flavour.sh <path_to_image.qcow2> <flavour_name> <version>

IMAGE_PATH=$1
FLAVOUR=$2
VERSION=$3

if [ -z "$FLAVOUR" ]; then
    echo "Usage: $0 <path_to_image.qcow2> <full_flavour_tag>"
    echo "Example: $0 xfce.qcow2 ubuntu-24.04-xfce-dev"
    exit 1
fi

DIST_DIR="dist/flavours/$FLAVOUR/$VERSION"
mkdir -p "$DIST_DIR"

echo "🐣 Preparing $FLAVOUR:$VERSION for the world..."

# 1. Calculate SHA256 of the FULL image
echo "🔍 Calculating genetic fingerprint (SHA256)..."
FULL_HASH=$(sha256sum "$IMAGE_PATH" | awk '{print $1}')
FULL_SIZE=$(stat -c%s "$IMAGE_PATH")

# Create checksum file for automation
echo "$FULL_HASH  flavour-$FLAVOUR-$VERSION-amd64.qcow2" > "$DIST_DIR/flavour-$FLAVOUR-$VERSION-amd64.qcow2.sha256"

# 2. Split into 1GB chunks
echo "🧩 Partitioning into 1GB chunks..."
split -b 1000M --numeric-suffixes=1 --suffix-length=3 "$IMAGE_PATH" "$DIST_DIR/flavour-$FLAVOUR-$VERSION-amd64.qcow2."

# 3. Generate JSON Snippet
echo "----------------------------------------"
echo "✅ Done! All parts are in $DIST_DIR"
echo "Copy this entry to registry/images.json:"
echo ""

cat << EOF
{
  "version": "$VERSION",
  "arch": "amd64",
  "url": "https://github.com/Josepavese/nido/releases/download/v$VERSION/flavour-$FLAVOUR-$VERSION-amd64.qcow2.001",
  "checksum_type": "sha256",
  "checksum": "$FULL_HASH",
  "size_bytes": $FULL_SIZE,
  "format": "qcow2",
  "part_urls": [
$(ls "$DIST_DIR" | sort | xargs -I {} echo '    "https://github.com/Josepavese/nido/releases/download/TAG/'{}'",' | sed '$ s/,$//')
  ]
}
EOF
echo "----------------------------------------"
echo "⚠️  Remember to replace 'TAG' with the actual GitHub Release tag."
