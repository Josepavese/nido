# Nido Maintainer's Guide 🛠️

This document outlines the tools and processes for managing the Nido repository, including registry updates, flavour publishing, and automated testing.

---

## 📂 Repository Layout (Admin View)

- `/bin`: Administrative shell scripts.
- `/cmd`: Source code for registry tools.
- `/flavours`: Cloud-init configurations for custom Nido images.
- `/registry`: The official image catalog and source definitions.
- `.github/workflows`: CI/CD automation logic.

---

## 🌍 Registry Management

Nido uses an automated system to keep its image catalog fresh.

### 🍱 Registry Builder

- **Path**: `cmd/registry-builder/main.go`
- **Purpose**: Scans `registry/sources.yaml` for new image versions from upstream (Ubuntu, Debian, Alpine, etc.) and updates `registry/images.json`.
- **Usage**:

    ```bash
    go run cmd/registry-builder/main.go --sources registry/sources.yaml --output registry/images.json
    ```

- **Automation**: Runs daily via GitHub Actions (`update-registry.yml`).
- **Discovery policy**: Canonical released SimpleStreams (supported Ubuntu cloud releases), Debian stable/oldstable Release files, Alpine release branches still under main support, newest stable Kali/CirrOS directory, latest openSUSE snapshot. Endpoints and selection policy live in `registry/sources.yaml`.
- **Safe publication**: Required source failures, empty discoveries, invalid integrity metadata and ambiguous aliases fail before replacement. Output is written to a sibling temporary file and renamed after complete generation. Explicitly optional unpublished Nido flavours are warnings; prior published flavour records survive a temporary source failure.
- **Validation**: `VALIDATE:` includes new versions and changed URL/checksum identities under existing versions. Identical output preserves `updated_at`. Run `go test ./internal/registry/builder ./cmd/registry-builder`; the Linux CI smoke job also runs `tests/catalog_surfaces.py` with a disposable, networkless home.
- **Resource limits**: Discovery uses metadata GET and image HEAD requests only. On constrained hosts put `GOCACHE`, `GOMODCACHE`, `GOTMPDIR`, `TMPDIR` and `NIDO_TEST_STORAGE` on the larger scratch filesystem. Never run VM validation against an existing user home; `bin/validate-registry.sh` writes the home cache and is intended for disposable CI hosts.

### 🧪 Registry Validator (Testing)

- **Path**: `cmd/registry-validator/main.go` / `bin/validate-registry.sh`
- **Purpose**: Spawns temporary VMs for new images to verify SSH connectivity and cloud-init success before they are accepted into the catalog.
- **Usage**:

    ```bash
    ./bin/validate-registry.sh [registry_file] [filter]
    ```

- **Example**: `./bin/validate-registry.sh registry/images.json alpine`

---

## 📦 Flavour Publishing

Flavours are pre-configured environments optimized for Nido.

### 🍱 Build Scripts

- **Path**: `/flavours`
- **Format**: Cloud-init YAML files (e.g., `ubuntu-24.04-lubuntu.yaml`).

### 🧩 Publishing Tool

- **Path**: `bin/publish-flavour.sh`
- **Purpose**: Prepares a `.qcow2` image for distribution.
    1. Splits the image into 1GB chunks (to bypass GitHub Release limits).
    2. Calculates the global SHA256 checksum.
    3. Generates a JSON fragment for the registry.
- **Usage**:

    ```bash
    ./bin/publish-flavour.sh <path_to_image> <flavour_tag> <version>
    ```

- **Example**:

    ```bash
    ./bin/publish-flavour.sh my_image.qcow2 ubuntu-24.04-lubuntu-minimal v3.0.0
    ```

### 📡 Syncing Tool (Automated Upload)

- **Path**: `bin/upload-flavours.sh`
- **Purpose**: Sequentially uploads flavour chunks to a GitHub Release.
- **Key Features**:
  - **Idempotent**: Checks remote assets and skips files already present.
  - **Live Monitoring**: Shows a progress bar for each 1GB chunk using terminal UI.
  - **Dry Run**: Use `--dry-run` to see what would be uploaded without sending data.
- **Usage**:

    ```bash
    ./bin/upload-flavours.sh [release_tag] [--dry-run]
    ```

- **Example**: `./bin/upload-flavours.sh v4.0.0`

---

*“It’s not just code, it’s a living ecosystem.”* 🐣✨
