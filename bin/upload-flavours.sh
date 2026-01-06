#!/bin/bash
# 🪺 Nido Flavour Sync Tool
# Sequential, idempotent upload of flavour segments to GitHub Releases.

set -e

# --- Colors & UI ---
BOLD="$(tput bold 2>/dev/null || echo '')"
GREEN="$(tput setaf 2 2>/dev/null || echo '')"
CYAN="$(tput setaf 6 2>/dev/null || echo '')"
YELLOW="$(tput setaf 3 2>/dev/null || echo '')"
RED="$(tput setaf 1 2>/dev/null || echo '')"
DIM="$(tput dim 2>/dev/null || echo '')"
RESET="$(tput sgr0 2>/dev/null || echo '')"

# --- Configuration ---
FLAVOURS_DIR="dist/flavours"
DEFAULT_TAG=$(grep -oE 'Version = "v[0-9.]+"' cmd/nido/main.go | cut -d'"' -f2 || echo "latest")

usage() {
    echo "Usage: $0 [release_tag] [--dry-run]"
    echo "Example: $0 v4.0.0"
    exit 1
}

TAG=${1:-$DEFAULT_TAG}
DRY_RUN=false
if [[ "$*" == *"--dry-run"* ]]; then DRY_RUN=true; fi

echo "${BOLD}${CYAN}🪺 Nido Flavour Sync Tool${RESET}"
echo "Target Release: ${BOLD}${TAG}${RESET}"
if [ "$DRY_RUN" = true ]; then echo "${YELLOW}⚠️  DRY RUN MODE - No files will be uploaded.${RESET}"; fi
echo ""

# 1. Verification
if ! command -v gh &> /dev/null; then
    echo "${RED}❌ Error: 'gh' CLI not found. Please install it.${RESET}"
    exit 1
fi

if ! gh auth status &> /dev/null; then
    echo "${RED}❌ Error: Not authenticated with GitHub. Run 'gh auth login'.${RESET}"
    exit 1
fi

if [ ! -d "$FLAVOURS_DIR" ]; then
    echo "${RED}❌ Error: Flavours directory '$FLAVOURS_DIR' not found.${RESET}"
    exit 1
fi

# 2. Fetch remote assets list
echo "🔍 Fetching remote assets for ${TAG}..."
REMOTE_ASSETS=$(gh release view "$TAG" --json assets -q '.assets[].name' 2>/dev/null || echo "")

# 3. Find local files
LOCAL_FILES=$(find "$FLAVOURS_DIR" -type f \( -name "flavour-*" \) | sort)
if [ -z "$LOCAL_FILES" ]; then
    echo "${YELLOW}ℹ️  No local flavours found in $FLAVOURS_DIR.${RESET}"
    exit 0
fi

# 4. Processing
TOTAL=$(echo "$LOCAL_FILES" | wc -l)
CURRENT=0
UPLOADED=0
SKIPPED=0

for FILE in $LOCAL_FILES; do
    CURRENT=$((CURRENT + 1))
    BASENAME=$(basename "$FILE")
    FILE_SIZE=$(stat -c%s "$FILE")
    
    printf "  [%d/%d] %-60s " "$CURRENT" "$TOTAL" "$BASENAME"

    if echo "$REMOTE_ASSETS" | grep -q "^${BASENAME}$"; then
        echo "${DIM}Already present. Skipping. ✅${RESET}"
        SKIPPED=$((SKIPPED + 1))
        continue
    fi

    if [ "$DRY_RUN" = true ]; then
        echo "${YELLOW}Would upload. ☁️${RESET}"
        continue
    fi

    # Upload with progress
    echo "${CYAN}Uploading... ☁️${RESET}"
    
    # Start upload in background to monitor progress
    gh release upload "$TAG" "$FILE" --clobber > /dev/null 2>&1 &
    GH_PID=$!
    
    # Monitor progress using IO stats
    while kill -0 $GH_PID 2>/dev/null; do
        if [ -f "/proc/$GH_PID/io" ]; then
            READ=$(cat "/proc/$GH_PID/io" 2>/dev/null | grep rchar | awk '{print $2}')
            if [ -n "$READ" ]; then
                [ "$READ" -gt "$FILE_SIZE" ] && READ=$FILE_SIZE
                PERC=$((READ * 100 / FILE_SIZE))
                printf "\r  [%d/%d] %-60s ${CYAN}Progress: %3d%%${RESET}" "$CURRENT" "$TOTAL" "$BASENAME" "$PERC"
            fi
        fi
        sleep 1
    done
    wait $GH_PID
    
    printf "\r  [%d/%d] %-60s ${GREEN}Completed. ✅${RESET}\n" "$CURRENT" "$TOTAL" "$BASENAME"
    UPLOADED=$((UPLOADED + 1))
done

echo ""
echo "${BOLD}${GREEN}🎉 Sync Complete!${RESET}"
echo "   Uploaded: $UPLOADED"
echo "   Skipped:  $SKIPPED"
echo "   Total:    $TOTAL"
echo ""
echo "The nest is synced. 🪺✨"
