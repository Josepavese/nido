# Test Nido

[Contributing](CONTRIBUTING.md) · [Architecture and qualification](docs/ARCHITECTURE.md)

Choose the smallest check that can establish the changed contract. Documentation
changes need link/example consistency and visual inspection. Runtime changes
need relevant regressions plus the required cross-platform CI. A device-less
test cannot qualify physical hardware.

## Go checks

Use the toolchain line in [`.go-version`](.go-version), existing module caches
and temporary build output outside the checkout:

```bash
go test ./...
go vet ./...
```

CI also checks formatting, staticcheck, govulncheck, shell scripts, Windows
installer parsing, Linux VM smoke behavior and release packaging. The workflow
in [`.github/workflows/cross-platform.yml`](.github/workflows/cross-platform.yml)
owns the current matrix.

## Isolated CLI and MCP contracts

On Linux, the existing Python harnesses use bubblewrap with a private home,
private VM state and controlled host fixtures. They do not expose existing VMs.
Build once, reuse that binary for both suites, then retire the scratch:

```bash
# Pick a filesystem with enough free space. Keep build output out of the repo.
scratch=$(mktemp -d /path/to/scratch/nido-tests.XXXXXX)
trap 'rm -rf "$scratch"' EXIT
go build -o "$scratch/nido" ./cmd/nido
NIDO_TEST_BINARY="$scratch/nido" NIDO_TEST_STORAGE="$scratch" python3 tests/catalog_surfaces.py
NIDO_TEST_BINARY="$scratch/nido" NIDO_TEST_STORAGE="$scratch" python3 tests/cli_mcp_safety.py
```

`catalog_surfaces.py` verifies catalog versions/aliases/integrity across CLI,
MCP tools and resources, plus fixture-based accelerator parity.
`cli_mcp_safety.py` verifies persistence, path/process boundaries, backing-chain
protection and configuration semantics, including raw QEMU replacement, omission
and explicit clearing. Inspect harness prerequisites before executing them.

## End-to-end VM validation

`nido-validator` exercises CLI and MCP using shared YAML workflows. It can pull
images, boot guests and create temporary disks; budget disk/RAM first and use a
disposable test environment. Ownership checks limit its cleanup, but they are
not a reason to expose valuable user state to a qualification run.

See the [validator guide](cmd/nido-validator/README.md) for flags, output,
timeouts and cleanup behavior. Build `nido` and `nido-validator` into the same
external scratch and reuse them throughout the run. Archive only useful bounded
evidence; remove owned artifacts after their final consumer exits.

## Platform and hardware evidence

- Cross-platform compilation/unit tests establish the tested software contracts.
- A real VM smoke test establishes the lifecycle on that host/guest combination.
- Native PCI qualification establishes attachment/restoration only for the
  explicitly selected device and scenario. A skipped test is not a pass.

The opt-in [native PCI fixture and cutover procedure](docs/ARCHITECTURE.md#opt-in-native-pci-qualification)
protect the primary display, require a suitable empty controller and retain
failed recovery state. Do not use existing VMs or a user's active peripherals
as disposable test fixtures.

## Shared events

`go test ./...` includes election/fan-out/handoff, late-subscriber snapshots,
bounded slow consumers, native filesystem/process waits, QMP identity/whitelist,
MCP opt-in/filter/mute/coalescing, stdout framing and modal/background TUI state.
Run the changed concurrent boundaries with the race detector:

```sh
go test -race ./internal/events ./internal/pkg/sysutil ./internal/provider ./internal/mcp ./internal/tui/app/...
NIDO_TEST_BINARY="$scratch/nido" NIDO_TEST_STORAGE="$scratch" python3 tests/events_integration.py
NIDO_EVENT_QEMU_TEST_STORAGE="$scratch" go test -v ./internal/provider -run '^TestRealQEMUEventLifecycle$' -count=1
```

The Python integration uses actual Unix PTYs with two TUI processes and separate
CLI/MCP processes, all in a private fixture home. It covers a hidden route,
CLI mutation, mute/unmute, external atomic publication/deletion and broker crash
handoff. The opt-in QEMU test creates a blank private 1 MiB disk, runs software
emulation with 128 MiB RAM and confirms pause/resume/termination. No guest OS
readiness or physical hardware qualification is claimed. Native macOS/Windows
contracts run in the existing cross-platform CI; cross-compilation alone is not
native runtime proof.

The validator's `event-stream` scenario exercises the CLI NDJSON envelope,
another process's config change and rejected arguments in its own fixture home.
Use the validator from a disposable environment as documented above; its general
cleanup scenario is not a permission to sweep an existing user's resources.
