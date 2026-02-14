---
description: "Manage Pull Requests: Scan, Review (Tone of Voice), Validate, and Merge."
---

// turbo-all

### 1. Matrix Scan (Discovery) 🔍
- **Action**: Check for open pull requests across the ecosystem.
- **Command**: `gh pr list --json number,title,author,state`
- **Goal**: Identify anomalies or pending patches requiring human/AI intervention.

### 2. Genetic Integrity Review (Tone of Voice) 🧬🕹️
- **Rule**: All interactions MUST adhere to **[Tone of Voice Guide](../../docs/.tone_of_voice.md)**.
- **Action**: Add a "System Scan" comment to the PR using Cyber-Nerd terminology.
- **Example**:
  ```markdown
  🕹️ **SYSTEM SCAN COMPLETE**
  Hatching check... 🐣
  ROM sequences validated... ✅
  Proceeding to patch the Matrix.
  ```

### 3. Poking the Matrix (CI Kickstart) ⚡🕹️
- **Context**: Bot-created PRs (like Registry Updates) often fail to trigger secondary flows (CI) due to token limitations.
- **Symptom**: PR status remains `pending` or "No checks reported" despite commits being present.
- **Action**: Push an empty commit from a local authorized environment to trigger the CI sequence.
- **Commands**:
  ```bash
  git fetch origin <branch-name>
  git checkout <branch-name>
  git commit --allow-empty -m "chore(matrix): kickstart CI sequence 🕹️⚡"
  git push origin <branch-name>
  ```

### 4. Simulation Validation (Local Testing) 🎮
- **Action**: Before merging, run the local validation battery to ensure no glitches are introduced.
- **Commands**:
  ```bash
  make lint
  go test ./...
  ```

### 5. Final Ascension (Merge Sequence) 🚀
- **Requirement**: All CI checks MUST be green (Status: Successful).
- **Merge Strategy**: Use **Squash and Merge** to maintain a clean history.
- **Commands**:
  ```bash
  gh pr merge <number> --squash --delete-branch --body "<Toned description of changes>"
  ```

### 6. Cleanup & Sync 🍳
- **Action**: Return to the mainline and synchronize the local ROM cache.
- **Commands**:
  ```bash
  git checkout main
  git pull origin main
  ```

*Mission Complete. Keep the Nest Clean.* 🪺✨
