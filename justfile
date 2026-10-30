set shell := ["bash", "-cu"]

# Default: list available tasks
default:
    @just --list

# Add a worktree under ./<dir> on a new <branch> off dev; optional <paths> enable sparse-checkout
# Usage: just worktree <dir> <branch> [paths...]
#   e.g. just worktree apps-web feat/web apps/web packages/ui
worktree dir branch *paths:
    #!/usr/bin/env bash
    set -euo pipefail
    if [[ -z "{{paths}}" ]]; then
        git worktree add {{dir}} -b {{branch}} dev
    else
        git worktree add --no-checkout {{dir}} -b {{branch}} dev
        git -C {{dir}} sparse-checkout init --cone
        git -C {{dir}} sparse-checkout set {{paths}}
        git -C {{dir}} checkout
    fi
    echo "✓ worktree ready: {{dir}} ({{branch}}) {{ if paths == '' { '' } else { '→ ' + paths } }}"

# Remove a worktree created above
# Usage: just unworktree <dir> [--force]
unworktree dir *args:
    git worktree remove {{dir}} {{args}}

# Show current worktrees
worktrees:
    git worktree list

# Show git status and the last 5 commits for every checked-out worktree
status:
    #!/usr/bin/env bash
    set -euo pipefail
    git worktree list --porcelain | sed -n 's/^worktree //p' | while read -r wt; do
        branch=$(git -C "$wt" branch --show-current)
        printf '\n\033[1m== %s\033[0m (%s)\n' "$wt" "${branch:-detached}"
        git -C "$wt" status --short --branch
        git -C "$wt" log --oneline --no-decorate -5
    done

# Install beans from a worktree (default: dev)
install dir="dev":
    cd ./{{dir}} && go install .

# Run beans from a worktree (default: dev), e.g. just run feat-x list
run dir="dev" *args:
    cd ./{{dir}} && go run . {{args}}

# Start the TUI from a worktree (default: dev)
tui dir="dev" *args:
    cd ./{{dir}} && go run . tui {{args}}

# Start the docs dev server (watch + live reload) at http://localhost:8082
docs:
    cd ./docs && { [ -d node_modules ] || pnpm install --ignore-workspace; } && pnpm dev
