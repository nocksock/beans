#!/usr/bin/env bash
# Run a command in a new tmux window: tmux-window [--cwd DIR] [--name NAME] -- CMD...
set -euo pipefail

cwd=$PWD
name=${BEANS_ID:-beans}
while [[ $# -gt 0 ]]; do
    case $1 in
        --cwd) cwd=$2; shift 2 ;;
        --name) name=$2; shift 2 ;;
        --) shift; break ;;
        *) break ;;
    esac
done
[[ $# -gt 0 ]] || { echo "usage: tmux-window [--cwd DIR] [--name NAME] -- CMD..." >&2; exit 2; }

# The window's shell comes from the tmux server, so the bean's env is passed on
env_args=()
for var in $(compgen -e | grep '^BEANS_'); do
    env_args+=(-e "$var=${!var}")
done

tmux new-window -c "$cwd" -n "$name" "${env_args[@]}" "$(printf '%q ' "$@")"
