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
Main branch is protected. Direct pushes are rejected by the system.
1. Create a release branch:
   ```bash
   git checkout -b release/v$VERSION
   ```
2. Commit the version bump:
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
   gh pr create --title "Release v$VERSION 🚀" --body "Promoting version $VERSION to main. CI checks required." --base main
   ```

### 6. Validation & Merge (The Checkpoint) 🚦
1. **Wait for CI (Critical)**:
   The system rejects unverified code. You MUST wait for GitHub Actions to complete.
   ```bash
   echo "📡 Scanning Matrix for check completion..."
   gh pr checks --watch --interval 10
   ```
2. **Handle the Pull Request**:
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
3. The GitHub Actions "Create Release" workflow will automatically build and publish the binaries.

*Mission Complete. Insert Coin.* 🪙
