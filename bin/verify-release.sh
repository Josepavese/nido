#!/usr/bin/env bash
# Nido release validator.
# Default mode checks a published GitHub Release. Use --local to validate dist/
# before upload in CI.

set -euo pipefail

REQUIRED_ASSETS=(
  "nido-linux-amd64.tar.gz"
  "nido-linux-arm64.tar.gz"
  "nido-darwin-amd64.tar.gz"
  "nido-darwin-arm64.tar.gz"
  "nido-windows-amd64.zip"
  "nido.mcpb"
  "server.json"
  "install.sh"
  "install.ps1"
  "SHA256SUMS"
)

default_version() {
  grep -oE 'Version = "v[0-9.]+"' internal/build/version.go | cut -d'"' -f2
}

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    echo "No SHA-256 tool found." >&2
    return 1
  fi
}

checksum_entry() {
  local checksum_file=$1
  local asset=$2
  awk -v asset="$asset" '
    { name=$2; sub(/^\*/, "", name); if (name == asset) { print $1; found=1; exit } }
    END { if (!found) exit 1 }
  ' "$checksum_file"
}

validate_local() {
  local dist_dir=${1:-dist}
  local missing=0

  echo "Checking local release artifacts in ${dist_dir}..."
  for asset in "${REQUIRED_ASSETS[@]}"; do
    if [ -f "${dist_dir}/${asset}" ]; then
      echo "Found: ${asset}"
    else
      echo "Missing: ${asset}"
      missing=$((missing + 1))
    fi
  done

  if [ "$missing" -ne 0 ]; then
    echo "Local release validation failed: missing ${missing} required asset(s)."
    return 1
  fi

  for asset in "${REQUIRED_ASSETS[@]}"; do
    if [ "$asset" = "SHA256SUMS" ]; then
      continue
    fi
    expected=$(checksum_entry "${dist_dir}/SHA256SUMS" "$asset") || {
      echo "Missing checksum entry: ${asset}"
      missing=$((missing + 1))
      continue
    }
    actual=$(sha256_file "${dist_dir}/${asset}")
    if [ "$actual" != "$expected" ]; then
      echo "Checksum mismatch: ${asset}"
      missing=$((missing + 1))
    else
      echo "Checksum OK: ${asset}"
    fi
  done

  if [ "$missing" -ne 0 ]; then
    echo "Local release validation failed."
    return 1
  fi
  echo "Local release artifacts look valid."
}

validate_remote() {
  local version=${1:-$(default_version)}
  local release_data
  local is_draft
  local assets
  local missing=0

  command -v gh >/dev/null 2>&1 || { echo "gh is required for remote release validation." >&2; return 1; }
  command -v jq >/dev/null 2>&1 || { echo "jq is required for remote release validation." >&2; return 1; }

  echo "Checking GitHub release health for ${version}..."
  if ! release_data=$(gh release view "$version" --json isDraft,isPrerelease,assets,tagName,url 2>/dev/null); then
    echo "Release ${version} not found on GitHub."
    return 1
  fi

  is_draft=$(echo "$release_data" | jq -r '.isDraft')
  assets=$(echo "$release_data" | jq -r '.assets[].name')

  echo "Tag: $(echo "$release_data" | jq -r '.tagName')"
  echo "URL: $(echo "$release_data" | jq -r '.url')"

  if [ "$is_draft" = "true" ]; then
    echo "Release is still a draft."
    missing=$((missing + 1))
  else
    echo "Release is published."
  fi

  for asset in "${REQUIRED_ASSETS[@]}"; do
    if echo "$assets" | grep -Fxq "$asset"; then
      echo "Found: ${asset}"
    else
      echo "Missing: ${asset}"
      missing=$((missing + 1))
    fi
  done

  if ! echo "$assets" | grep -q '^flavour-'; then
    echo "Note: no flavour image assets found in this release."
  fi

  if [ "$missing" -ne 0 ]; then
    echo "Release validation failed."
    return 1
  fi
  echo "Release looks structurally sound."
}

case "${1:-}" in
  --local)
    validate_local "${2:-dist}"
    ;;
  -h|--help)
    echo "Usage: $0 [--local [dist-dir]] [version]"
    ;;
  *)
    validate_remote "${1:-$(default_version)}"
    ;;
esac
