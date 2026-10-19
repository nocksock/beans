#!/usr/bin/env bash
# Open or switch to a tmux session for the bean (tmux 3.0+): tmux-session [--cwd DIR] [-- CMD...]
# --cwd and CMD only apply when the session is created; an existing session is just attached.
set -euo pipefail

: "${BEANS_ID:?BEANS_ID not set}"
cwd=$PWD
while [[ $# -gt 0 ]]; do
    case $1 in
        --cwd) cwd=${2:?--cwd needs a value}; shift 2 ;;
        --) shift; break ;;
        *) break ;;
    esac
done
[[ $cwd == /* ]] || cwd=$(cd "$cwd" && pwd)

session=${BEANS_ID//[^a-zA-Z0-9_-]/}

# The session's shell comes from the tmux server, so the bean's env is passed
# on with -e. That sets the session environment, which every window of the
# session inherits, so the launch-chain variables (which pin nested
# `beans launch` calls) only go to CMD itself.
env_args=()
for var in BEANS_ROOT BEANS_CONFIG_DIR BEANS_DIR BEANS_ID BEANS_TASK BEANS_BIN; do
    if [[ -n ${!var+x} ]]; then env_args+=(-e "$var=${!var}"); fi
done
chain_env=()
for var in BEANS_LAUNCH_CONFIG BEANS_LAUNCH_LAYER BEANS_LAUNCH_DEPTH; do
    if [[ -n ${!var+x} ]]; then chain_env+=("$var=${!var}"); fi
done
# A tmux server started below copies tmux's own environment into its global
# environment, which every session would inherit
unset BEANS_ROOT BEANS_CONFIG_DIR BEANS_DIR BEANS_ID BEANS_TASK BEANS_BIN \
    BEANS_LAUNCH_CONFIG BEANS_LAUNCH_LAYER BEANS_LAUNCH_DEPTH

if ! tmux has-session -t "=$session" 2>/dev/null; then
    if [[ $# -gt 0 ]]; then
        # A single argument is a shell command, as tmux itself would run it
        if [[ $# -eq 1 ]]; then set -- "${SHELL:-/bin/sh}" -c "$1"; fi
        # Given as several arguments, tmux 3.0+ runs the command directly
        # instead of through its default-shell, so no quoting is needed
        tmux new-session -d -s "$session" -c "$cwd" ${env_args[@]+"${env_args[@]}"} -- \
            env ${chain_env[@]+"${chain_env[@]}"} "$@"
    else
        tmux new-session -d -s "$session" -c "$cwd" ${env_args[@]+"${env_args[@]}"}
    fi
fi

if [[ -n ${TMUX:-} ]]; then
    tmux switch-client -t "=$session"
else
    tmux attach-session -t "=$session"
fi
