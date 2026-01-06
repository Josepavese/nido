# Nido Roadmap 🪺

Nido is **AI-centric and automation-first**: a focused execution surface for agentic workflows. It is *not* a generic VM manager. This roadmap keeps us aligned, intentional, and fast.

## North Star

- **Agent-ready by default**: zero-friction automation through MCP + CLI parity.
- **Reproducible environments**: fast, deterministic VM spawning from templates and images.
- **Local-first, cloud-friendly**: works on developer machines and CI without heavyweight stacks.
- **Small and sharp**: minimal moving parts, maximum leverage.

## Current State (Now)

### Core VM Engine

- Cross-platform QEMU lifecycle: spawn/start/stop/delete/list/info/prune.
- Compressed templates (`.compact.qcow2`) and fast cloning.
- SSH access + VNC toggle for GUI sessions.
- Built-in diagnostics (`nido doctor`).

### Agentic Interface (MCP)

- MCP server with full VM lifecycle coverage.
- Image tools: list/pull/update and cache inspection/pruning.
- Template tools + config lookup.
- AI-friendly defaults and consistent responses.

### Image Registry

- Catalog-based image system with cache + checksum verification.
- Registry sources already structured (`registry/sources.yaml`).
- Registry builder logic implemented (strategy-based fetching).

### Flavours

- Community-friendly flavours concept (compressed, split distribution).
- Early support for desktop flavours via images and VNC.

## Next Up

### 1) Image Registry: Finish the Loop

- CLI parity for image management: `image info`, `image remove`, cache commands.
- Registry automation: ship `registry-builder` CLI + GitHub Action updates.
- Expand catalog coverage (Tier 1 official + Tier 2 conversion).

### 2) Cloud-Init & First Boot UX

- Standardize cloud-init behavior for image-based spawn.
- Make user-data injection first-class in CLI + MCP.
- Tighten docs + defaults so agents can log in without guessing.

### 3) Flavours Pipeline

- Define minimal “agent-ready” flavours (headless GUI, lightweight desktop).
- cloud-init bootstrap for zero-touch GUI sessions.
- Add flavour entries to registry with split distribution support.

### 4) MCP Polish

- HTTP/SSE transport for remote agents.
- Tool annotations for safety and confirmations.
- Better structured outputs (typed JSON over raw text).

## Later

### Automation & Orchestration

- REST API + webhooks.
- CI/CD integrations (GitHub Actions, GitLab).
- Fleet operations (batch spawn/stop/prune).

### VM Superpowers

- Snapshots (create/restore/list).
- Advanced networking (custom NAT, port rules).
- Template marketplace and sharing flow.

### DX & Reliability

- Interactive TUI.
- Self-healing + auto-recovery.
- Hardening and long-run stability testing.

## How to Use This Roadmap

- This is the **single source of truth** for planned work.
- `.improvements/` is the nest’s sketchbook: keep ideas there, but graduate only what makes the cut.
- If a task isn’t here, it’s not on the flight plan (yet).

---

If you want to move a feature forward, open a PR that adds it here first. Let’s keep the flock in formation. 🐦
