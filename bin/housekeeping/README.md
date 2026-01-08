# Housekeeping scripts

Quick cleaners for build and temporary artifacts.

- `clean-nido.sh` — wipes Go build outputs and release bundles (`dist/`, `nido`, `bin/nido`, `registry-builder`, `registry-validator`) and compacts git history.
- `clean-all.sh` — runs `clean-nido.sh` and the GUI cleaner to reclaim space in one shot.

Usage:
```bash
bin/housekeeping/clean-nido.sh [-n|--dry-run] [-y|--yes] [-g|--git-gc|--no-git-gc] [--shallow-current|--no-shallow-current] [--allow-tracked] [--no-dist] [--no-binaries] [--no-registry-tools]
bin/housekeeping/clean-all.sh   [-n|--dry-run] [-y|--yes] [...]
```

Defaults:
- Interactive UI (nerdy TUI) when a TTY is available; `--yes` or `--no-ui` skips it.
- Removes `dist/`, built binaries, and registry tool binaries.
- Runs `git gc --prune=now --aggressive` to reclaim space in `.git` (toggle with `--no-git-gc`).
- Runs a shallow reset to the origin default branch (depth=1) to drop local history. Disable with `--no-shallow-current`. If the working tree is dirty, the script stashes changes (including untracked), resets, then reapplies the stash; on conflict the stash is left for manual application.

Switches:
- `--no-git-gc` disables git compaction (enabled by default; `--git-gc` forces on).
- `--no-shallow-current` keeps existing history (enabled by default; `--shallow-current` forces on).
- `--allow-tracked` allows deletion of tracked targets (skipped by default to keep the working tree clean for tracked binaries).
- `--no-dist`, `--no-binaries`, `--no-registry-tools` skip those removal categories.
- `--dry-run` shows actions; `--yes` skips confirmation.
