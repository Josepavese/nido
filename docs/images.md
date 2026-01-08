# Nido Image Registry 🐣

Starting from v3.1.0, Nido includes a built-in Image Registry to easily find, download, and spawn VMs from official cloud images.

## Concepts

- **Catalog**: A JSON manifest of verified images from official sources (Ubuntu, Debian, Alpine).
- **Image**: A read-only OS disk image (qcow2 format).
- **Backing File (Linked Clones)**: When you spawn a VM from an image, Nido creates a "Clone" (overlay) using QCOW2 backing files. The original image remains untouched and read-only, while your VM only stores the differences. 🧬

## Commands

### List Images

Browse available images in the catalog.

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
