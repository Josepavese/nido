---
description: "Deploy Nido: Build, Validate (Blackbox), Increment Version (Z-only), and Tag."
---

1. **Build Phase**
   - **Command**: `make build` (compiles `bin/nido`)
   - **Command**: `go build -o bin/nido-validator ./cmd/nido-validator` (compiles validator)
   - **Check**: Ensure exit code is 0.

2. **Lint Phase**
   - **Command**: `make lint`
   - **Check**: Ensure exit code is 0. Verify no "glitches in the matrix" (vet/staticcheck errors).

3. **Blackbox Validation Phase**
   - **Context**: Run the blackbox validation engine (`internal/validator/README.md`).
   - **Command**: `NIDO_BIN=bin/nido SKIP_GUI=true SKIP_UPDATE=true bin/nido-validator`
   - **Check**: Look for "Release validation FAILED" or non-zero exit code. Proceed ONLY if tests pass.

4. **Versioning Phase** (If validation passes)
   - **Source**: `internal/build/version.go`
   - **Logic**: 
     - Read the `Version` variable (e.g., `v4.5.2`).
     - **Increment Z only** (Patch level). Do NOT touch X or Y without explicit instruction.
     - Example: `v4.5.2` -> `v4.5.3`.
   - **Action**: Update the file with the new version.

### 5. Branch & Push Strategy (Matrix Security Protocol) 🛡️

**CRITICAL: Tone of Voice & Commit Hygiene**
- **MANDATORY**: All commit messages and PR titles MUST follow the **[Tone of Voice Guide](../../docs/.tone_of_voice.md)**.
  - Commits are "patches to the game code". Use emojis. Be "Cyber-Nerd".
  - ❌ "Fix bug" -> ✅ "fix(core): patch glitch in hyperspace jump 🌌"
- **ATOMIC GROUPS**: Group changes into coherent, contextual chunks.
  - Do NOT squash wildly different features into one commit.
  - Write beautiful, complete descriptions. Context is King.

**CRITICAL: .agent Directory Protocol**
1. **NEVER UPLOAD**: The `.agent` directory must **NEVER** be pushed to the remote repository.
2. **KILL SWITCH**: If `.agent` exists on the remote (check `git ls-files .agent`), you MUST remove it from the remote index immediately:
   ```bash
   git rm -r --cached .agent
   ```
3. **NO GITIGNORE**: Do **NOT** add `.agent` to `.gitignore`. It must remain visible to the local agent but untracked by the remote.
4. **SELECTIVE ADD**:
   - ❌ **NEVER** run `git add .` or `git add .agent`
   - ✅ **ALWAYS** use `git add <file>` to select specific files.

Main branch is protected. Direct pushes are rejected by the system.
1. Create a release branch:
   ```bash
   git checkout -b release/v$VERSION
   ```
2. Commit the version bump (Atomic & Toned):
   ```bash
   git add internal/build/version.go
   git commit -m "chore(release): level up to v$VERSION 🚀🕹️"
   ```
3. Push the branch to the Matrix:
   ```bash
   git push origin release/v$VERSION
   ```
4. Create a Pull Request via GH CLI:
   ```bash
   gh pr create --title "release: level up to v$VERSION 🚀" --body "## 🕹️ Release v$VERSION\n\nPromoting code to the mainline. System integrity checks required.\n\n### 📝 Changes\n(Insert beautiful description here)" --base main
   ```

### 6. Validation & Merge (The Checkpoint) 🚦
1. **Wait for CI (Critical)**:
   The system rejects unverified code. You MUST wait for GitHub Actions to complete.
   ```bash
   echo "📡 Scanning Matrix for check completion..."
   gh pr checks --watch --interval 10
   ```
2.  **Handle the Pull Request**:
   Once genetic integrity is verified (All Green), merge the PR and clean up.
   ```bash
   gh pr merge --squash --delete-branch
   ```

### 7. Tagging & Ascent (Finalizing) 🏷️
1. Switch back to main and sync:
   ```bash
   git checkout main
   git pull origin main
   ```
2. Apply the official release tag:
   ```bash
   git tag v$VERSION
   git push origin v$VERSION
   ```
3. The GitHub Actions "Create Release" workflow will automatically:
   - Build and package binaries (Linux, macOS, Windows).
   - **Generate MCP Assets**: `nido.mcpb` (Bundle) and `server.json` (Manifest/SSOT) are now automatically created and uploaded.
   - **Publish to Registry**: Automatically pushes `nido.mcpb` to the official MCP Registry using `mcp-publisher`.
   - Publish the release.

*Mission Complete. Insert Coin.* 🪙
