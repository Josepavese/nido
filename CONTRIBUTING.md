# Contribute to Nido

Help make the nest useful: report reproducible bugs, improve a guide, or bring a
focused change with a clear user outcome.

## Report a problem

Open a [GitHub issue](https://github.com/Josepavese/nido/issues) with the command or
MCP action, expected/observed behavior, Nido version, host OS and QEMU version.
Include relevant `nido doctor` output and reproduction steps. Remove secrets,
private paths and customer data from logs before sharing them.

## Make a change

1. Fork or create a focused branch from current `main`.
2. Read the existing owner of the behavior: the
   [CLI manifest](internal/cli/commands.yaml), MCP catalog, provider contract or
   relevant backend. See [architecture](docs/ARCHITECTURE.md).
3. Update connected public surfaces and guides together. Keep CLI, TUI and MCP
   semantics consistent; preserve data and explicit ownership boundaries.
4. Run the checks appropriate to the change. [Testing](TESTING.md) describes
   isolated CLI/MCP validation and the limits of hardware qualification.
5. Open a PR describing the problem, resulting behavior and evidence. Required
   CI must pass before merge. Keep unrelated work separate.

Use the Go toolchain line in [`.go-version`](.go-version). Build outside the
checkout, using an existing shared module cache:

```bash
scratch=$(mktemp -d)
trap 'rm -rf "$scratch"' EXIT
go build -o "$scratch/nido" ./cmd/nido
go test ./...
go vet ./...
```

Check disk capacity before large builds or guest downloads. Preserve existing
VMs, disks, templates, cache dependencies, pending recovery records and other
contributors' work. Use private test storage; never run a destructive test against
your normal VM home.

## Writing and visuals

Lead with the supported behavior, then add a little arcade personality.
Avoid timing promises without measurements, invented CLI output and claims
that simulated tests qualify physical hardware. Prefer `-` for prose separators.
The [tone guide](docs/.tone_of_voice.md) and [visual assets](resources/README.md)
describe the current identity. Historical assets remain recoverable from Git;
superseded operational instructions should not remain in the active guides.
