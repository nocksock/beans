#!/usr/bin/env bash
# Run a command in a new kitty window: kitty-window [--cwd DIR] -- CMD...
set -euo pipefail

cwd=$PWD
while [[ $# -gt 0 ]]; do
    case $1 in
        --cwd) cwd=$2; shift 2 ;;
        --) shift; break ;;
        *) break ;;
    esac
done
[[ $# -gt 0 ]] || { echo "usage: kitty-window [--cwd DIR] -- CMD..." >&2; exit 2; }

# The window's shell comes from kitty, so the bean's env is passed on
env_args=()
for var in $(compgen -e | grep '^BEANS_'); do
    env_args+=(--env "$var=${!var}")
done

kitty @ launch --type=window --cwd="$cwd" "${env_args[@]}" "$@"
