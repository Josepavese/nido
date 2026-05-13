# Nido Image Registry 🐣

Starting from v3.1.0, Nido includes a built-in Image Registry to easily find, download, and spawn VMs from official cloud images.

## Concepts

- **Catalog**: A JSON manifest of verified images from official sources (Ubuntu, Debian, Alpine).
- **Image**: A read-only OS disk image (qcow2 format).
- **Blueprint**: A local recipe that builds a qcow2 image from installer media and automation files. Windows evaluation images use this path because Microsoft ships installers, not cloud qcow2 images.
- **Backing File (Linked Clones)**: When you spawn a VM from an image, Nido creates a "Clone" (overlay) using QCOW2 backing files. The original image remains untouched and read-only, while your VM only stores the differences. 🧬

## Commands

### List Images

Browse available sources in the catalog. Nido flavours are shown first, buildable blueprints second, and official cloud images last.

```bash
nido image list
```

**Output:**

```
OFFICIAL:
  ubuntu:24.04 (noble)     600 MB
  debian:12    (bookworm)  331 MB
  alpine:3.20  (latest)    52 MB
```

### Pull Image

Download an image to your local cache (`~/.nido/images/`). Partial downloads are automatically resumed.

```bash
nido image pull ubuntu:24.04
```

### Spawn from Image

Create a new VM using a downloaded image. If the image isn't downloaded, Nido will offer to pull it automatically.

```bash
nido spawn my-vm --image ubuntu:24.04
```

### Build from Blueprint

Blueprints are listed and inspected like other image sources, then built into the normal image cache.

```bash
nido blueprint list
nido blueprint info windows-11-eval
nido blueprint build windows-11-eval
nido spawn my-win-vm --image windows-11-eval --gui
```

`nido build <blueprint>` remains a compatibility alias for `nido blueprint build <blueprint>`.

Windows blueprints finish installer staging during the build, then finish OOBE/post-install work on first boot. Use `--gui` for the first boot so you can see setup progress.

Available Windows blueprint entries use the official Microsoft product names:

| Blueprint | Product |
| --- | --- |
| `windows-11-eval` | Windows 11 Enterprise Evaluation |
| `windows-11-iot-ltsc-eval` | Windows 11 IoT Enterprise LTSC 2024 Evaluation |
| `windows-server-2022-core-eval` | Windows Server 2022 Evaluation (Server Core) |

Windows host support has passed smoke testing on a real Windows VM for installer parsing, diagnostics, catalog/blueprint discovery, and basic spawn/SSH/stop/delete lifecycle. It is usable for core workflows, but still needs heavier long-running and workload-specific validation.

### Update Catalog

Refresh the local catalog from the GitHub repository.

```bash
nido image update
```

## Known Limitations

### Cloud-Init

Most cloud images (Ubuntu Cloud, Debian Cloud, Alpine) require **Cloud-Init** to set the default user password or inject SSH keys.

Nido fully supports **Zero-Touch Cloud-Init** injection:

1. Pass your user-data file via `--user-data <file>`.
2. Nido automatically creates a transient ISO, attaches it to the VM, and handles the handshake.
3. The default user is typically `vmuser` (configurable in `nido config`).
**Workaround:**

4. Use images that allow empty passwords (rare).
5. Or use `guestfish` or `virt-customize` to modify the image password before spawning.
6. Or use a custom template (the "old way") which is pre-configured.

### Alpine Linux

Alpine cloud images usually allow login as `root` with no password connected via serial console or display, but behavior varies by image build.

## Directory Structure

Images are stored in:

- **Linux/macOS**: `~/.nido/images/`
- **Windows**: `%USERPROFILE%\.nido\images\`
