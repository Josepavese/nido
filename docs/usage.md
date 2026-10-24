# Everyday workflows

[Documentation](README.md) · [MCP](mcp.md) · [Images](images.md)

Use a distinct VM name for each disposable task. Commands here target that named
VM; run cleanup only after saving the outputs you need.

## Provision and connect

```bash
nido images update
nido images info ubuntu:lts
nido spawn test-lab --image ubuntu:lts --memory 2048 --cpus 2
nido info test-lab --json
nido ssh test-lab
```

Spawn starts QEMU; SSH becomes available after guest boot and cloud-init.
Nido injects its SSH key into supported cloud images. The login user comes from
image metadata or configuration, and is shown by `nido info`. Do not assume
`root`. Supported Linux cloud-init guests receive administrator access through
sudo (or doas where applicable).

For extra provisioning, save this as `provision.yaml`:

```yaml
#cloud-config
package_update: true
packages:
  - git
  - make
```

```bash
nido spawn build-lab --image ubuntu:lts --user-data provision.yaml
nido ssh build-lab cloud-init status --wait
```

CLI `--user-data` takes a **file path**; MCP `user_data` takes the **file content**.
Cloud-config YAML and shell-script support depend on the guest. CirrOS accepts
shell user-data, not the full cloud-config feature set.

## Forward a service port

```bash
nido spawn web-lab --image ubuntu:lts --port app:8080/tcp
nido info web-lab --json
nido info web-lab --json
```

CLI `config --port` replaces the saved service-forwarding list; include every
rule you still need. MCP also exposes individual `port_forward`, `port_unforward`
and `port_list` actions.

Automatic host ports are selected from the configured port range. Read back the
assigned endpoint instead of assuming a number. For a fixed mapping:

```bash
nido config web-lab --port app:8080:33000/tcp --json
nido info web-lab --json

```

QEMU user-mode forwarding binds to host loopback. Inside a guest, `127.0.0.1`
means that guest, not the host or another VM. Port changes are persistent
configuration; restart a running VM to apply them. Use a host-side broker when
coordinating multiple guests across platforms.

## Reuse a prepared environment

After provisioning a disposable source VM, shut down the guest cleanly before
saving its disk as a template:

```bash
nido ssh build-lab sudo poweroff
nido info build-lab
# After the guest has exited:
nido template create build-lab build-base
nido spawn test-next build-base
```

A template captures a disk environment, not live RAM or a running process.
Copied credentials, downloaded dependencies and project files remain in it;
prepare the guest accordingly before reuse. Linked clones reuse a base disk,
while every VM keeps its own writable changes.

## Change persistent VM settings

```bash
nido config test-lab --memory 4096 --cpus 4 --json
nido info test-lab --json
```

Changes take effect on the next start; they do not hot-plug resources.
Raw QEMU arguments are an ordered array:

```bash
nido config test-lab --qemu-arg=-fw_cfg --qemu-arg='name=opt/example,string=hello world' --json
nido config test-lab --cpus 2 --json       # preserve the saved raw argument list
nido config test-lab --clear-qemu-args --json
```

Providing arguments replaces the whole list. Omitting them preserves it.
`--clear-qemu-args` removes the list and cannot be combined with `--qemu-arg`.
`--qemu-arg=` is a literal empty argument, not a clear request. The MCP equivalent
is `nido_vm` / `config_update`: omit `raw_qemu_args` to preserve it, send `[]` to
clear it. `null` is rejected. See [MCP](mcp.md).

## Share files with a Linux guest using 9p

Prefer SSH transfer for simple uploads. For a deliberately shared host directory,
QEMU 9p can be supplied through raw arguments when your QEMU build and Linux guest
support it. Create a dedicated host directory, not a mount of your whole home:

```bash
mkdir -p /path/to/task-share
nido config test-lab \
  --qemu-arg=-virtfs \
  --qemu-arg='local,path=/path/to/task-share,mount_tag=taskshare,security_model=mapped-xattr,id=taskshare' \
  --json
```

Start the stopped VM, then mount inside the guest:

```bash
nido start test-lab
nido ssh test-lab sudo mkdir -p /mnt/taskshare
nido ssh test-lab sudo mount -t 9p -o trans=virtio,version=9p2000.L taskshare /mnt/taskshare
```

This replaces any existing raw arguments, so include arguments you still need.
The share exposes host files to guest writes; `readonly=on` on the QEMU share
option is available for a read-only export. To remove it, unmount in the guest,
shut down the guest, clear the saved arguments, and start again. This is a QEMU
Linux-guest workflow, not a portable Nido shared-folder API or a Windows share.

## Stop and clean up

Prefer guest shutdown for filesystem consistency:

```bash
nido ssh test-lab sudo poweroff
nido info test-lab
```

If you need to terminate the managed VM process, `nido stop test-lab` verifies its
identity and waits for exit. It does not send a guest ACPI shutdown request.
After preserving useful data, delete only the VM you own:

```bash
nido delete test-lab
```

`nido prune` deletes **all stopped VMs**. It is not a task-scoped cleanup command.
`nido uninstall` removes installation data and is not an upgrade procedure.
Use `nido cache ls` and dependency-aware cache removal to manage downloaded images.
Do not manually delete backing files used by VM disks or templates.

## Isolation and storage

The default state root is `~/.nido` (Windows: `%USERPROFILE%\.nido`). Inspect
`nido config --json` before changing paths. `BACKUP_DIR` stores compressed disk templates; active VM disks remain under
`~/.nido/vms`. Its name does not imply a second recovery copy of every VM.
`IMAGE_DIR` owns the downloaded image cache. Changing a path does not migrate
existing files. See [configuration](../config/README.md).

VM disks persist until explicitly deleted. With linked clones, plan capacity
for the shared bases **plus** all guest writes, templates, download staging and
logs. Downloaded images, VM disks and templates are different resources.

Each VM has a separate guest kernel. Forwarded services, host directories and
PCI passthrough intentionally connect guest and host resources. Treat those
connections as explicit grants. PCI support varies by host and device; primary
display protection and IOMMU checks do not qualify every GPU or peripheral.
See [hardware contracts and qualification limits](ARCHITECTURE.md#qualification-and-cutover-limits).
