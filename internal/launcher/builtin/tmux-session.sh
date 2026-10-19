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
if ! tmux has-session -t "=$session" 2>/dev/null; then
    # The session's shell comes from the tmux server, so the bean's env is passed on
    env_args=()
    for var in BEANS_ROOT BEANS_CONFIG_DIR BEANS_DIR BEANS_ID BEANS_TASK BEANS_BIN \
        BEANS_LAUNCH_CONFIG BEANS_LAUNCH_LAYER BEANS_LAUNCH_DEPTH; do
        if [[ -n ${!var+x} ]]; then env_args+=(-e "$var=${!var}"); fi
    done
    if [[ $# -gt 0 ]]; then
        # Given as several arguments, tmux 3.0+ runs the command directly
        # instead of through its default-shell, so no quoting is needed
        tmux new-session -d -s "$session" -c "$cwd" ${env_args[@]+"${env_args[@]}"} -- "$@"
    else
        tmux new-session -d -s "$session" -c "$cwd" ${env_args[@]+"${env_args[@]}"}
    fi
fi

if [[ -n ${TMUX:-} ]]; then
    tmux switch-client -t "=$session"
else
    tmux attach-session -t "=$session"
fi
