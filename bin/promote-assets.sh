#!/bin/bash
# 🛸 Nido Asset Promotor
# Copies flavour assets from a source release to a destination release.

set -e

# --- Colors ---
CYAN="$(tput setaf 6 2>/dev/null || echo '')"
GREEN="$(tput setaf 2 2>/dev/null || echo '')"
YELLOW="$(tput setaf 3 2>/dev/null || echo '')"
RED="$(tput setaf 1 2>/dev/null || echo '')"
BOLD="$(tput bold 2>/dev/null || echo '')"
RESET="$(tput sgr0 2>/dev/null || echo '')"

usage() {
    echo "Usage: $0 <source_tag> <dest_tag>"
    echo "Example: $0 v4.0.0 v4.0.1"
    exit 1
}

SRC_TAG=$1
DST_TAG=$2

if [ -z "$SRC_TAG" ] || [ -z "$DST_TAG" ]; then
    usage
fi

echo "${BOLD}${CYAN}🛸 Nido Asset Promotor${RESET}"
echo "Source: ${SRC_TAG}"
echo "Target: ${DST_TAG}"
echo ""

# 1. Fetch source assets
echo "🔍 Scanning ${SRC_TAG} for flavours..."
ASSETS=$(gh release view "$SRC_TAG" --json assets -q '.assets[].name' | grep "^flavour-" || true)

if [ -z "$ASSETS" ]; then
    echo "${YELLOW}ℹ️  No flavour assets found in ${SRC_TAG}. Nothing to promote.${RESET}"
    exit 0
fi

# 2. Fetch destination assets (to avoid duplicates)
echo "🔍 Checking ${DST_TAG} existing assets..."
DST_ASSETS=$(gh release view "$DST_TAG" --json assets -q '.assets[].name' 2>/dev/null || echo "")

TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

for ASSET in $ASSETS; do
    if echo "$DST_ASSETS" | grep -q "^${ASSET}$"; then
        echo "  - ${ASSET}: ${GREEN}Already present. Skipping. ✅${RESET}"
        continue
    fi

    echo "  - ${ASSET}: ${CYAN}Promoting... ☁️${RESET}"
    
    # Download from source
    gh release download "$SRC_TAG" -p "$ASSET" -D "$TEMP_DIR" > /dev/null
    
    # Upload to destination
    gh release upload "$DST_TAG" "$TEMP_DIR/$ASSET" --clobber > /dev/null
    
    # Cleanup file to save space
    rm "$TEMP_DIR/$ASSET"
    
    echo "  - ${ASSET}: ${GREEN}Promoted successfully! 🛸${RESET}"
done

echo ""
echo "${BOLD}${GREEN}🎉 Promotion complete!${RESET}"
echo "All flavour assets from ${SRC_TAG} are now available in ${DST_TAG}."
