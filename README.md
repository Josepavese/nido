> **"Hatch your agents. Let them fly."** 🐣💾

[![Release](https://img.shields.io/github/v/release/Josepavese/nido?style=flat-square&color=ff00ff)](https://github.com/Josepavese/nido/releases)
[![License](https://img.shields.io/github/license/Josepavese/nido?style=flat-square&color=00ffff)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/Josepavese/nido?style=flat-square)](https://goreportcard.com/report/github.com/Josepavese/nido)

**Nido** is the retro-futuristic nest for your AI agents. It spawns **real Virtual Machines** in milliseconds, giving your autonomous code a safe place to hatch, grow, and execute.

Containers are cages. Nido is a habitat. 🪺

Built on QEMU and fueled by 80s nostalgia, Nido feels like a game console for DevOps. It provides a full, unconstrained OS for your agents to explore, break, and rebuild.

---

## 🕹️ The Game Loop: Spawn -> Execute -> Destroy

![The Lifecycle](resources/nido_diagram_hatch.png)

1. **INSERT COIN (Spawn)**: An agent needs a body. Nido "hatches" a VM from a cached genetic sequence (Image) instantly using **Linked Clones** technology.
2. **PLAY (Execute)**: The VM is alive. The agent connects via SSH (Neural Link) and has full `root` access. No shared kernels. No rules.
3. **GAME OVER (Destroy)**: The mission is complete. The VM is vaporized. The nest remains pristine for the next player.

---

## ⚡ Loading... (Installation)

> **SYSTEM REQUIREMENTS:** Linux, macOS, or Windows (WSL2/PowerShell). QEMU must be installed.

### 💾 Quick Install (Web)

Run this command in your terminal. Do not turn off the console while saving.

**Linux & macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/Josepavese/nido/main/installers/quick-install.sh | bash
source ~/.nido/env   # Power up the path
nido version         # Check checksum
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/Josepavese/nido/main/installers/quick-install.ps1 | iex
# Restart your terminal to initialize the matrix
nido version
```

### 🪄 Cheat Codes (Auto-Complete)

Tab-completion is enabled by default. If it's missing, equip it manually:

```bash
# Add to ~/.bashrc or ~/.zshrc
source <(nido completion bash)  # or zsh / fish / powershell
```

---

## 🦄 Select Your Flavour

![Select Flavour](resources/nido_flavours_select.png)

Nido supports a wide roster of "fighters" (Cloud Images). Ubuntu, Debian, Alpine, Arch, choose your pixelated champion.

### 🎮 Real World Example (Combo Move)

Let's hatch a **Ubuntu 24.04** bird named `agent-01`, attach a graphical interface (GUI), and log in immediately.

```bash
# 1. DOWNLOAD ROM (Pull Image)
nido images pull ubuntu:24.04

# 2. START GAME (Spawn VM with GUI)
nido spawn agent-01 --image ubuntu:24.04 --gui

# 3. LINK CABLE (SSH Connection)
nido ssh agent-01
```

*Result:* A fresh VM boots in <2 seconds. A VNC window opens. You are root.

---

## 🦾 Neural Interface (For AI Agents)

Nido is designed to be driven by **Large Language Models** (Claude, GPT-4, Gemini).

### 🤖 Model Context Protocol (MCP)

Nido speaks native MCP. It exposes a suite of tools (`vm_spawn`, `vm_exec`, `vm_prune`, etc.) that allow your AI to manage its own infrastructure.

```bash
nido register  # Generates the config for your AI client
```

### 🧠 JSON Mode

Every command supports `--json`. Perfect for scripts and robot eyes.

```bash
nido ls --json
# Output: {"vms": [{"name": "agent-01", "state": "running", "ip": "10.0.2.15"}]}
```

---

## 🕹️ Control Deck (Command List)

Here is the full move list for the Nido console.

### 🐣 Life Cycle (The Game Loop)

| Command                               | Action                  | Arcade Analog               |
| :------------------------------------ | :---------------------- | :-------------------------- |
| `nido spawn <name> [--image <img>]` | Create & start a new VM | **START GAME**        |
| `nido start <name> [--gui]`         | Revive a stopped VM     | **CONTINUE? 10..9..** |
| `nido stop <name>`                  | ACPI Shutdown signal    | **PAUSE**             |
| `nido delete <name>`                | Destroy VM permanently  | **GAME OVER**         |
| `nido prune`                        | Delete ALL stopped VMs  | **CLEAR HIGH SCORES** |

### 🔍 Observability (HUD)

| Command              | Action                    | Arcade Analog           |
| :------------------- | :------------------------ | :---------------------- |
| `nido ls`          | List all VMs              | **PLAYER SELECT** |
| `nido info <name>` | Show IP, Ports, PID       | **STATS SCREEN**  |
| `nido gui`         | Interactive TUI Dashboard | **ARCADE MODE**   |
| `nido doctor`      | Diagnose system health    | **TEST MENU**     |

### 🔌 Connectivity (Link Cable)

| Command             | Action      | Arcade Analog        |
| :------------------ | :---------- | :------------------- |
| `nido ssh <name>` | SSH into VM | **LINK CABLE** |

### 🧬 Genetic Engineering (Images & Templates)

| Command                              | Action                    | Arcade Analog              |
| :----------------------------------- | :------------------------ | :------------------------- |
| `nido images list`                 | Browse cloud images       | **CHARACTER ROSTER** |
| `nido images pull <tag>`           | Download image            | **LOAD ROM**         |
| `nido cache ls`                    | View local cache          | **MEMORY CARD**      |
| `nido cache prune`                 | Clear unused images       | **DELETE SAVE**      |
| `nido template list`               | List custom templates     | **USER SKINS**       |
| `nido template create <vm> <name>` | Save VM state as template | **SAVE STATE**       |
| `nido template delete <name>`      | Delete template           | **ERASE**            |

### ⚙️ System (Options Menu)

| Command            | Action                  | Arcade Analog               |
| :----------------- | :---------------------- | :-------------------------- |
| `nido config`    | View/Edit settings      | **OPTIONS**           |
| `nido register`  | Setup MCP integration   | **CONTROLLER CONFIG** |
| `nido update`    | Self-update from GitHub | **OTA PATCH**         |
| `nido uninstall` | Nuclear cleanup         | **SELF DESTRUCT**     |
| `nido version`   | Show version info       | **CREDITS**           |
| `nido help`      | Show usage guide        | **TUTORIAL**          |

---

## 🧬 Killer Feature: Linked Clones

Why download 2GB every time? Nido downloads the generic "Common ROM" (Base Image) **once**.

Every VM you spawn is just a **diff layer** (mutation) on top of that ROM.

- **Base Image:** Read-Only. Safe.
- **VM Disk:** Read-Write. Ephemeral.

**Result:** Spawn 100 VMs, use disk space for 1. 🚀

---

## 🤝 Contributing

**Insert Coin to Join.**

1. Fork the repo.
2. `go run ./cmd/nido gui` to test the Arcade UI.
3. Submit a PR.
4. High Scores are recorded in `AUTHORS`.

## 📜 License

MIT License. Free as in "Free Play".

---

<p align="center">
  <i>Made with 💜 and ☕ by digital artisans.</i><br>
  <i>EST. 2025 • "The Grid. A digital frontier."</i>
</p>
