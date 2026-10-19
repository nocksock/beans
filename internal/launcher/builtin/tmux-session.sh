#!/usr/bin/env bash
# Open or switch to a tmux session for the bean: tmux-session [--cwd DIR] [-- CMD...]
set -euo pipefail

: "${BEANS_ID:?BEANS_ID not set}"
cwd=$PWD
while [[ $# -gt 0 ]]; do
    case $1 in
        --cwd) cwd=$2; shift 2 ;;
        --) shift; break ;;
        *) break ;;
    esac
done

session=${BEANS_ID//[^a-zA-Z0-9_-]/}
if ! tmux has-session -t "=$session" 2>/dev/null; then
    # The session's shell comes from the tmux server, so the bean's env is passed on
    env_args=()
    for var in $(compgen -e | grep '^BEANS_'); do
        env_args+=(-e "$var=${!var}")
    done
    cmd=()
    [[ $# -gt 0 ]] && cmd=("$(printf '%q ' "$@")")
    tmux new-session -d -s "$session" -c "$cwd" "${env_args[@]}" "${cmd[@]}"
fi

if [[ -n ${TMUX:-} ]]; then
    tmux switch-client -t "=$session"
else
    tmux attach-session -t "=$session"
fi
