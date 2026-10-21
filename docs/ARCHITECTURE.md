# Runtime boundaries and audit

Nido supports Linux, macOS and Windows. The CLI, TUI and MCP express VM intent
through `provider.VMProvider`. They do not retain a concrete QEMU provider,
construct native processes, or discover host PCI devices themselves.

## Current contracts

- `internal/hardware.Host` owns virtualization capabilities, PCI inventory,
  per-VM exclusive leases, device preparation and restoration. Native backends
  are selected at build time. Missing support is an explicit error; an empty
  accelerator capability means software emulation, never hardware acceleration.
- `internal/pkg/sysutil` is the PAL for native commands, process cancellation,
  stream consumers, privileged writes, executable discovery and atomic
  publication without replacement. Its native backends contain platform APIs.
  `HostIO` and `HTTPClient` can be substituted without native device access.
- `internal/provider` implements the VM contract using the injected HAL.
  QEMU launch, PID identity and control-channel adapters are build-specific.
  CLI/MCP/TUI receive structured SSH argv; shell text is rendered separately.
  Stop means confirmed process termination, not an unimplemented guest ACPI
  promise. The ignored historical boolean is removed from the current contract.
- `image.Transfer` owns transfer, resume validation and decompression. Its
  concrete implementation is private; the factory accepts network/context and
  progress dependencies. Archives never extract arbitrary members into the
  existing image directory. Multipart assembly retires each consumed chunk.
  Download reservations exclude concurrent writers to the same partial file;
  catalog checksums are verified before publication of the final artifact.
- `lifecycle.Updater` owns verified release installation and bundled registry
  synchronization. A single metadata response selects the release identity and
  assets. Its scratch stays on the installation filesystem and is removed when
  the operation returns, including errors. Existing rollback binaries are not
  overwritten: mapped executables may still serve live sessions.

Data-storage adapters (`config`, `image`, `lifecycle`) own portable filesystem
persistence; native device paths and native process APIs stay below the above
boundaries. The CLI bootstrap composes native implementations. Validators are
qualification harnesses, not application consumers or alternative runtime owners.

`internal/architecture` checks the device owner, concrete-adapter leaks, native
process imports in consumers, runtime OS selection and direct console I/O in
core packages. These checks are part of `go test ./...` on all supported hosts.

## Confirmed audit findings and changes

| Finding | Consequence before correction | Current behavior |
| --- | --- | --- |
| Missing IOMMU group counted as isolated; boot display allowed by headless heuristic | Unsafe device could be selected automatically | Missing or unreadable isolation/display identity fails closed; primary display remains attached |
| PCI inventory and binding had different owners; unsupported stubs succeeded | False physical capability and untestable host coupling | One injected HAL; explicit unsupported errors and private filesystem fixtures |
| Driver record errors, restore failures and remove/rescan recovery were ignored | Lost recovery information or unintended physical device removal | Save before unbind; confirm driver changes; retain failed leases; report errors |
| PCI restoration record had no VM owner or exclusive reservation | A second VM could interfere with another VM's binding | One versioned, exclusive per-device lease, with VM ownership checked before restoration |
| Installer and VM startup independently guessed acceleration from OS/device existence | Permission failures and false HVF/WHPX claims | Both consume the same effective capability; Linux queries the accessible KVM API and native backends query support |
| Concrete QEMU/downloader types and native process APIs leaked into consumers | Backend replacement required consumer changes; SSH string splitting lost arguments | VM/transfer/updater contracts, PAL process API and structured SSH argv |
| Stop accepted an ignored graceful flag and help claimed ACPI delivery | Contract and backend behavior disagreed | Remove the unused internal flag; describe actual process termination without changing existing stop behavior |
| Metadata updates truncated the authoritative file before a successful write | Storage exhaustion could corrupt VM/config state | Same-filesystem staging and atomic replacement preserve previous bytes and permissions on error |
| Reused seed paths were deleted and temporary output was predictable | Concurrent builds or pre-existing work could be destroyed | Unique owned workspace and staging; no-replace publication; child reaped before cleanup |
| Partial files could alias existing disks; HTTP 416 recursed; resume ranges were not verified; multipart staging doubled the full artifact | Unbounded retry, corrupt resume or unnecessary SSD peak | Refuse symlink/hardlink aliases; bounded rejection preserving partial data; validate ranges; append and retire chunks sequentially |
| Tar extraction wrote into the image directory | Unrelated archive members could replace cached images | Extract only the requested image to stdout, stage privately and publish without replacement |
| CLI update exited through deferred cleanup, skipped absent checksums and staged on another filesystem | Residues, unverified installation or failed cutover | Updater returns errors, requires integrity metadata and stages beside the actual executable |
| ISO conversion ran on every build; theme/message compatibility exports remained | Permanent historical path and a cache request that missed the application router | Current cache naming only, obsolete exports removed, one application cache message |

Multiple current backends are intentional: built-in/system ISO writers, current
terminal palettes and explicit software emulation are not historical schemas.
Useful caches, VM disks, backups and pending recovery records are data, not legacy
code to delete by filename or age.

## Qualification and cutover limits

Run `go test ./...` and `go vet ./...`, then the mandatory cross-platform CI.
`tests/catalog_surfaces.py` verifies CLI, MCP tools and resources against the same
catalog and a private read-only sysfs fixture. `tests/cli_mcp_safety.py` exercises
private VM fixtures with the real user home hidden. Put build/test caches and
`NIDO_TEST_STORAGE` on the designated scratch filesystem.

PCI fixtures prove the ownership, rollback and error contracts. They do not
qualify physical GPU detachment or restoration. A dedicated device qualification
is required before distributing that backend change.

The current PCI lease schema is a JSON record with `version`, `owner` and `driver`.
Old plain-driver records are preserved and explicitly rejected, never silently
converted by normal runtime operations. Before installing this change into an
existing host, reconcile its driver records and VM ownership in a single offline
conversion with old writers drained. Preserve the original record and verify the
current owner before cutover. The conversion is a maintenance operation, never an automatic startup path.

### Opt-in native PCI qualification

`internal/hardware/pci_physical_linux_test.go` provides the physical backend
qualification. Its default CI result is an explicit skip, not hardware proof.
It accepts only an explicitly selected, isolated USB controller with no attached
peripherals and the expected native `xhci_hcd` binding. It uses the public VM
provider with a private 1 MiB disk, a paused QEMU instance and 128 MiB of RAM.
The fixture checks real VFIO attachment, the PCI identity exposed through QMP,
refusal of a foreign lease owner, and restoration of the original host driver.
It does not detach a primary display or qualify GPU-specific guest functions.

Build the test binary with scratch/cache paths on the designated filesystem,
then authenticate locally to execute it:

```sh
go test -c -o "$scratch/pci-qualification.test" ./internal/hardware
sudo env NIDO_PCI_TEST_DEVICE="$dedicated_controller" \
  NIDO_PCI_TEST_STORAGE="$scratch" TMPDIR="$scratch" \
  "$scratch/pci-qualification.test" \
  -test.run '^TestPhysicalPCIRoundTrip$' -test.count=1 -test.v
```

The fixture deletes only its own VM and workspace after confirming restoration.
If restoration fails, it keeps the recovery lease and reports its path. Do not
remove that workspace or treat a failed/skipped test as qualification.

An old journal whose original driver is already restored and which has no VM
owner is not converted into an invented active lease. During an offline cutover,
verify the current binding and empty override, drain old writers, preserve the
original record and its metadata outside the runtime, verify the copy and retire
the stale runtime record. The resulting current state has no active lease.
Unknown ownership, an active binding or changed input requires reconciliation;
normal startup must not infer an owner or replay this conversion.

The 2026-10-21 native qualification passed on an isolated Intel `8086:9a13`
USB controller, including real QEMU/VFIO attachment, guest PCI identity and host
`xhci_hcd` restoration. An initial fixture acceleration conflict also exercised
real preparation rollback successfully; the corrected fixture uses the HAL's
effective accelerator. The historical `i915` record was verified as already
restored and unclaimed, preserved outside the runtime, and retired with old
writers drained. Existing VM metadata remained unchanged. This proves the
shared PCI lifecycle on that device, not GPU-specific guest functionality.
