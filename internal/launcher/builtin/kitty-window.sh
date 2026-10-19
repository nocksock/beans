#!/usr/bin/env bash
# Run a command in a new kitty window: kitty-window [--cwd DIR] -- CMD...
set -euo pipefail

cwd=$PWD
while [[ $# -gt 0 ]]; do
    case $1 in
        --cwd) cwd=${2:?--cwd needs a value}; shift 2 ;;
        --) shift; break ;;
        *) break ;;
    esac
done
[[ $# -gt 0 ]] || { echo "usage: kitty-window [--cwd DIR] -- CMD..." >&2; exit 2; }
[[ $cwd == /* ]] || cwd=$(cd "$cwd" && pwd)

# The window's shell comes from kitty, so the bean's env is passed on
env_args=()
for var in BEANS_ROOT BEANS_CONFIG_DIR BEANS_DIR BEANS_ID BEANS_TASK BEANS_BIN \
    BEANS_LAUNCH_CONFIG BEANS_LAUNCH_LAYER BEANS_LAUNCH_DEPTH; do
    if [[ -n ${!var+x} ]]; then env_args+=(--env "$var=${!var}"); fi
done

kitty @ launch --type=window --cwd="$cwd" ${env_args[@]+"${env_args[@]}"} "$@"
