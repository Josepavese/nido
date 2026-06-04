#!/usr/bin/env bash
# Dispatch every scheduled workflow and verify the workflow plus its bot PR checks.

set -euo pipefail

REPO="${GITHUB_REPOSITORY:-Josepavese/nido}"
REF="${SCHEDULED_WORKFLOW_REF:-main}"
TIMEOUT_SECONDS="${SCHEDULED_WORKFLOW_TIMEOUT_SECONDS:-3600}"
PR_DISCOVERY_SECONDS="${SCHEDULED_WORKFLOW_PR_DISCOVERY_SECONDS:-120}"
POLL_SECONDS="${SCHEDULED_WORKFLOW_POLL_SECONDS:-10}"

require_tool() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing required tool: $1" >&2
    exit 1
  fi
}

iso_now() {
  date -u +"%Y-%m-%dT%H:%M:%SZ"
}

epoch_now() {
  date -u +%s
}

require_tool gh
require_tool jq
require_tool python3

if [ -z "${GH_TOKEN:-${GITHUB_TOKEN:-}}" ]; then
  echo "GH_TOKEN or GITHUB_TOKEN is required." >&2
  exit 1
fi

discover_workflows() {
  python3 - <<'PY'
import re
import sys
from pathlib import Path

workflow_dir = Path(".github/workflows")
errors = 0
for path in sorted(list(workflow_dir.glob("*.yml")) + list(workflow_dir.glob("*.yaml"))):
    text = path.read_text(encoding="utf-8")
    if not re.search(r"(?m)^\s*schedule\s*:", text):
        continue
    if not re.search(r"(?m)^\s*workflow_dispatch\s*:", text):
        print(f"::error file={path}::Scheduled workflow must expose workflow_dispatch so releases can verify it.", file=sys.stderr)
        errors += 1
        continue
    pr_branches = sorted(set(re.findall(r"(?m)^\s*branch:\s*[\"']?([^\"'#\s]+)", text)))
    print(f"{path.name}\t{','.join(pr_branches)}")
if errors:
    sys.exit(1)
PY
}

wait_for_workflow_run() {
  local workflow=$1
  local started_at=$2
  local deadline=$(( $(epoch_now) + TIMEOUT_SECONDS ))
  local run_id=""

  echo "Waiting for workflow_dispatch run for ${workflow} on ${REF}..."
  while [ "$(epoch_now)" -lt "$deadline" ]; do
    run_id=$(gh run list \
      --repo "$REPO" \
      --workflow "$workflow" \
      --event workflow_dispatch \
      --branch "$REF" \
      --limit 20 \
      --json databaseId,createdAt,status | jq -r --arg started "$started_at" '
        [.[] | select(.createdAt >= $started)]
        | sort_by(.createdAt)
        | reverse
        | .[0].databaseId // empty
      ')
    if [ -n "$run_id" ]; then
      echo "Found ${workflow} run: ${run_id}"
      gh run watch "$run_id" --repo "$REPO" --interval "$POLL_SECONDS" --exit-status
      return 0
    fi
    sleep "$POLL_SECONDS"
  done

  echo "Timed out waiting for ${workflow} workflow_dispatch run." >&2
  return 1
}

wait_for_pr_checks() {
  local branch=$1
  local discovery_deadline=$(( $(epoch_now) + PR_DISCOVERY_SECONDS ))
  local deadline=$(( $(epoch_now) + TIMEOUT_SECONDS ))
  local pr_number=""

  echo "Checking for bot PR from branch ${branch}..."
  while [ "$(epoch_now)" -lt "$discovery_deadline" ]; do
    pr_number=$(gh pr list \
      --repo "$REPO" \
      --head "$branch" \
      --state open \
      --json number \
      --jq '.[0].number // empty')
    if [ -n "$pr_number" ]; then
      break
    fi
    sleep "$POLL_SECONDS"
  done

  if [ -z "$pr_number" ]; then
    echo "No open PR found for ${branch}; scheduled workflow may have had no changes."
    return 0
  fi

  echo "Waiting for checks on PR #${pr_number} (${branch})..."
  while [ "$(epoch_now)" -lt "$deadline" ]; do
    local rollup
    rollup=$(gh pr view "$pr_number" \
      --repo "$REPO" \
      --json statusCheckRollup,url \
      --jq '.statusCheckRollup')

    local total pending failing
    total=$(jq 'length' <<<"$rollup")
    pending=$(jq '[.[] | select(.status != "COMPLETED")] | length' <<<"$rollup")
    failing=$(jq '[.[] | select(.status == "COMPLETED") | select((.conclusion // "SUCCESS") != "SUCCESS" and (.conclusion // "SUCCESS") != "SKIPPED" and (.conclusion // "SUCCESS") != "NEUTRAL")] | length' <<<"$rollup")

    if [ "$total" -gt 0 ] && [ "$pending" -eq 0 ] && [ "$failing" -eq 0 ]; then
      echo "PR #${pr_number} checks passed."
      return 0
    fi
    if [ "$failing" -gt 0 ]; then
      echo "PR #${pr_number} has failing checks:" >&2
      jq -r '.[] | select(.status == "COMPLETED") | select((.conclusion // "SUCCESS") != "SUCCESS" and (.conclusion // "SUCCESS") != "SKIPPED" and (.conclusion // "SUCCESS") != "NEUTRAL") | " - \(.name): \(.conclusion) \(.detailsUrl // "")"' <<<"$rollup" >&2
      return 1
    fi

    echo "PR #${pr_number}: total=${total} pending=${pending}; waiting..."
    sleep "$POLL_SECONDS"
  done

  echo "Timed out waiting for PR #${pr_number} checks." >&2
  return 1
}

mapfile -t workflows < <(discover_workflows)
if [ "${#workflows[@]}" -eq 0 ]; then
  echo "No scheduled workflows found."
  exit 0
fi

for entry in "${workflows[@]}"; do
  workflow="${entry%%$'\t'*}"
  branches="${entry#*$'\t'}"
  started_at="$(iso_now)"

  echo "Dispatching scheduled workflow ${workflow} on ${REF}..."
  gh workflow run "$workflow" --repo "$REPO" --ref "$REF"
  wait_for_workflow_run "$workflow" "$started_at"

  if [ -n "$branches" ] && [ "$branches" != "$workflow" ]; then
    IFS=',' read -r -a branch_list <<<"$branches"
    for branch in "${branch_list[@]}"; do
      [ -n "$branch" ] || continue
      wait_for_pr_checks "$branch"
    done
  fi
done

echo "All scheduled workflow checks passed."
