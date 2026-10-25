# Images, templates and blueprints

[Documentation](README.md) · [Everyday workflows](usage.md) · [Windows](windows.md)

Choose a starting point based on what you already have:

| Source | What it is | Typical use |
| --- | --- | --- |
| Cloud image | An upstream OS disk downloaded from the catalog | A clean Linux environment with cloud-init |
| Template | A prepared VM disk saved locally | Repeat a configured development/test environment |
| Blueprint | A recipe that builds a disk from media and automation files | Windows evaluation guests or custom image builds |
| Flavour | A published, preconfigured Nido image | A ready-made environment when its assets are available |

AMD64 and ARM64 host/guest selection, requirements and data-preserving upgrades:
[Architecture guide](architectures.md). Omitted architecture selects native; use `--arch`
on CLI, `arch` on MCP, or the TUI **Arch** selector for an explicit guest ISA.

## Browse and pull

```bash
nido images update
nido images list
nido images info ubuntu:lts --json
nido images pull ubuntu:lts
nido spawn test-lab --image ubuntu:lts
```

`images update` refreshes the local catalog, not an installed guest OS. A spawn
can download its requested image if the image is absent locally. Pulling first
makes that transfer explicit. Downloaded images normally live in `~/.nido/images`;
`IMAGE_DIR` can select another cache directory.

Use `nido images list` for the actual catalog rather than a copied list of OS
versions. `latest` selects the newest available release for that distribution;
Ubuntu `lts` selects the newest supported LTS. A numbered tag selects an OS
release, but the upstream disk build for that tag can still change. For exact
reproducibility retain and verify the resolved checksum and your selected base.

## Discovery and integrity

The daily registry workflow discovers supported Ubuntu cloud releases from
Canonical SimpleStreams, Debian stable/oldstable from Release metadata, supported
Alpine branches from the release feed, and the newest stable Kali/CirrOS release
from official indexes. openSUSE follows the newest Tumbleweed snapshot.
[`registry/sources.yaml`](../registry/sources.yaml) owns endpoints and selection
policies. Versions and codenames come from upstream metadata.

Newer image builds of the same OS release also trigger validation. Ubuntu and
Debian URLs and checksums refer to the same immutable build. Required-source
failures or invalid integrity metadata stop publication and leave the prior
catalog intact. Optional unpublished flavours are reported separately.

Discovery downloads metadata and probes sizes; it does not download guest disks
onto user machines. Scheduled VM validation and hosted CI have their own scope;
they do not certify every guest workload or host combination. Catalog removal
does not delete existing cached images, VM disks or guest data.

The transfer layer verifies advertised catalog checksums before publishing a
download and validates resumed ranges. Inspect the resolved checksum with
`nido images info <tag> --json`.

## Cache and linked clones

```bash
nido cache ls
nido cache info
```

With linked clones enabled, a VM has a writable QCOW2 overlay backed by a shared
base. The base stays in place while the overlay stores guest changes. Templates
can also participate in backing chains. Space usage includes those changes and
is not limited to the base image's size.

Use dependency-aware image/cache removal. Nido refuses destructive cleanup when
it finds a dependent backing chain or cannot inspect it safely. Do not manually
remove bases or move them away from existing VM disks.
See [template workflows](usage.md#reuse-a-prepared-environment).

## Build an image from a recipe

```bash
nido blueprint list
nido blueprint info windows-11-eval
nido blueprint build windows-11-eval
nido spawn win-lab --image windows-11-eval --gui
```

A blueprint can download installer media and allocate substantially more disk
space than a cloud-image pull. Inspect it and budget for media, staging and the
resulting disk first. Windows recipes prepare installer staging; first boot can
still run OOBE/post-install work. Use the VM display for that first boot.
The [Windows guide](windows.md) owns the edition list and host qualification notes.

## Guest initialization

Nido creates a NoCloud seed ISO for supported cloud images, including its SSH
configuration and optional custom user-data. Pass a file to CLI
`--user-data`; pass text to MCP `user_data`. Login users come from catalog/recipe
metadata or configuration; see `nido info` instead of assuming root credentials.

Guest cloud-init support varies. CirrOS accepts shell user-data rather than full
cloud-config YAML. A completed spawn means the VM process was started, not that
all guest provisioning has finished.

## MCP catalog access

Use `nido_image` actions `list`, `info`, `pull` and `refresh_catalog`, or read
`nido://catalog/images` and `nido://image/ubuntu:lts`. Blueprint inspection uses
`nido_blueprint`, `nido://catalog/blueprints` and `nido://blueprint/{name}`.
The [MCP guide](mcp.md) documents the current action catalog.
