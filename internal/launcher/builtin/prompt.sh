#!/usr/bin/env bash
# Print the agent prompt for the bean: prompt [--tmux] [--worktree DIR]
set -euo pipefail

: "${BEANS_ID:?BEANS_ID not set}"
tmux_window=false
worktree=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --tmux) tmux_window=true; shift ;;
        --worktree) worktree=${2:?--worktree needs a value}; shift 2 ;;
        *) echo "usage: prompt [--tmux] [--worktree DIR]" >&2; exit 2 ;;
    esac
done

# Query the same beans the launch was started for, wherever this runs
beans=("${BEANS_BIN:-beans}")
[[ -n ${BEANS_LAUNCH_CONFIG:-} ]] && beans+=(--config "$BEANS_LAUNCH_CONFIG")
[[ -n ${BEANS_DIR:-} ]] && beans+=(--beans-path "$BEANS_DIR")

"${beans[@]}" show --prompt "$BEANS_ID"

if $tmux_window; then
    cat <<'EOT'

## tmux window

You are running in a tmux window. Before starting, rename it to `<bean-id>: <label>`, where <label> is a short summary of this bean of at most 20 characters:
`tmux rename-window -t "$TMUX_PANE" '<bean-id>: <label>'`
EOT
fi

if [[ -n $worktree ]]; then
    cat <<EOT

## Worktree

You are working in the git worktree $worktree on branch task/$BEANS_ID. Commit your work on that branch. Leave merging the branch and removing the worktree to the user unless they ask you to do it.
EOT
fi

# Captured rather than piped into grep -q, which can exit early and kill a
# large output with SIGPIPE, failing the pipeline under pipefail
json=$("${beans[@]}" show --json "$BEANS_ID")
draft_re='"status": *"draft"'
if [[ $json =~ $draft_re ]]; then
    cat <<EOT

## Draft: plan first

This bean is still a draft. Do not implement anything yet. Plan the feature first: work out the requirements and open questions with the user, then write the agreed plan into the bean body. Once the plan is done, set the status to in-progress with \`beans update $BEANS_ID --status in-progress\`.
EOT
fi
