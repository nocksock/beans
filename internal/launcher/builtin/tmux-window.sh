#!/usr/bin/env bash
# Run a command in a new tmux window (tmux 3.0+): tmux-window [--cwd DIR] [--name NAME] -- CMD...
set -euo pipefail

cwd=$PWD
name=${BEANS_ID:-beans}
while [[ $# -gt 0 ]]; do
    case $1 in
        --cwd) cwd=${2:?--cwd needs a value}; shift 2 ;;
        --name) name=${2:?--name needs a value}; shift 2 ;;
        --) shift; break ;;
        *) break ;;
    esac
done
[[ $# -gt 0 ]] || { echo "usage: tmux-window [--cwd DIR] [--name NAME] -- CMD..." >&2; exit 2; }
[[ $cwd == /* ]] || cwd=$(cd "$cwd" && pwd)

# The window's shell comes from the tmux server, so the bean's env is passed on
env_args=()
for var in BEANS_ROOT BEANS_CONFIG_DIR BEANS_DIR BEANS_ID BEANS_TASK BEANS_BIN \
    BEANS_LAUNCH_CONFIG BEANS_LAUNCH_LAYER BEANS_LAUNCH_DEPTH; do
    if [[ -n ${!var+x} ]]; then env_args+=(-e "$var=${!var}"); fi
done

# Given as several arguments, tmux 3.0+ runs the command directly instead of
# through its default-shell, so no quoting is needed. A single argument is
# still run as a shell command.
tmux new-window -c "$cwd" -n "$name" ${env_args[@]+"${env_args[@]}"} -- "$@"
