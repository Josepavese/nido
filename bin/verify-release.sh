#!/bin/bash
# Nido Release Validator
# Verifies that a release is public, has all binaries, and isn't a draft.

VERSION=$(grep -oE 'Version = "v[0-9.]+"' cmd/nido/main.go | cut -d'"' -f2)
echo "🔍 Checking release health for $VERSION... 🐣"

RELEASE_DATA=$(gh release view "$VERSION" --json isDraft,isPrerelease,assets,tagName,url 2>/dev/null)

if [ $? -ne 0 ]; then
  echo "❌ Error: Release $VERSION not found on GitHub!"
  exit 1
fi

IS_DRAFT=$(echo "$RELEASE_DATA" | jq -r '.isDraft')
IS_PRE=$(echo "$RELEASE_DATA" | jq -r '.isPrerelease')
ASSETS=$(echo "$RELEASE_DATA" | jq -r '.assets[].name')

echo "  Tag: $(echo "$RELEASE_DATA" | jq -r '.tagName')"
echo "  URL: $(echo "$RELEASE_DATA" | jq -r '.url')"

# Check Draft Status
if [ "$IS_DRAFT" == "true" ]; then
  echo "⚠️  WARNING: Release is still a DRAFT. Users won't see it as 'latest'."
else
  echo "✅ Release is PUBLISHED."
fi

# Check Binary Assets
REQUIRED_BINARIES=(
  "nido-linux-amd64"
  "nido-darwin-amd64"
  "nido-darwin-arm64"
  "nido-windows-amd64.exe"
)

MISSING=0
for BIN in "${REQUIRED_BINARIES[@]}"; do
  if ! echo "$ASSETS" | grep -q "$BIN"; then
    echo "❌ Missing Binary: $BIN"
    MISSING=$((MISSING + 1))
  else
    echo "✅ Found Binary: $BIN"
  fi
done

# Check Flavour Assets (optional but good)
if ! echo "$ASSETS" | grep -q "flavour-"; then
  echo "⚠️  Note: No flavour images found in this release."
fi

if [ $MISSING -eq 0 ] && [ "$IS_DRAFT" == "false" ]; then
  echo ""
  echo "🎉 Release $VERSION looks structurally sound and ready for hatchings! 🪺✨"
  exit 0
else
  echo ""
  echo "❌ Release validation FAILED. Fix the issues above before announcing."
  exit 1
fi
