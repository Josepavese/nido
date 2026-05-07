# Release Notes Template

Use this template when creating GitHub releases. Keep the Nido personality!

---

## 🐣 Nido vX.Y.Z - [Witty Release Name]

**Release Date:** YYYY-MM-DD

[Opening paragraph with personality - explain what's new in a fun way]

Example:
> The bird has learned new tricks! This release brings cross-platform support,
> meaning your VMs can now hatch on Linux, macOS, and Windows. We've also taught
> the MCP server to speak 12 different dialects of VM management. 🪺

---

### ✨ What's New

**[Major Feature Category]**

- Feature description with emoji
- Another feature
- Use bullet points for clarity

**[Another Category]**

- More features
- Keep it scannable

---

### 🐛 Bug Fixes

- Fixed: [Description with a touch of humor if appropriate]
- Fixed: [Another fix]

---

### 💥 Breaking Changes

> **⚠️ Important:** If you're upgrading from vX.Y.Z, read this!

- **[Breaking Change 1]:** What changed and why
  - **Migration:** How to adapt your setup
  
- **[Breaking Change 2]:** Description
  - **Migration:** Steps to follow

---

### 📦 Installation

#### Quick install

\`\`\`bash
curl -fsSL <https://github.com/Josepavese/nido/releases/download/vX.Y.Z/install.sh> | bash
\`\`\`

\`\`\`powershell
irm <https://github.com/Josepavese/nido/releases/download/vX.Y.Z/install.ps1> | iex
\`\`\`

#### Linux

\`\`\`bash
curl -L <https://github.com/Josepavese/nido/releases/download/vX.Y.Z/nido-linux-amd64.tar.gz> -o nido.tar.gz
tar -xzf nido.tar.gz
cd nido-linux-amd64
chmod +x nido
sudo mv nido /usr/local/bin/
\`\`\`

#### macOS

\`\`\`bash
curl -L <https://github.com/Josepavese/nido/releases/download/vX.Y.Z/nido-darwin-amd64.tar.gz> -o nido.tar.gz
tar -xzf nido.tar.gz
cd nido-darwin-amd64
chmod +x nido
sudo mv nido /usr/local/bin/
\`\`\`

#### Linux ARM64

\`\`\`bash
curl -L <https://github.com/Josepavese/nido/releases/download/vX.Y.Z/nido-linux-arm64.tar.gz> -o nido.tar.gz
tar -xzf nido.tar.gz
cd nido-linux-arm64
chmod +x nido
sudo mv nido /usr/local/bin/
\`\`\`

#### Windows

Download `nido-windows-amd64.zip` from the assets below, extract it, and add the extracted folder to PATH.

#### Integrity

All release assets, including `install.sh` and `install.ps1`, are listed in `SHA256SUMS`. Installers verify this file automatically when it is available.

---

### 🔗 Full Changelog

**All Changes:** <https://github.com/Josepavese/nido/compare/vX.Y.Z...vX.Y.Z>

---

### 🙏 Contributors

Thanks to everyone who contributed to this release! [List contributors or use GitHub's auto-generation]

---

<p align="center">
  <i>Made with ❤️ for the Agentic future.</i><br>
  <i>"It's not a VM, it's a lifestyle."</i> 🪺
</p>
