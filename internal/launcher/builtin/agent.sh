#!/usr/bin/env bash
# Start a coding agent on the bean in the current directory: agent <claude|opencode|crush> [PROMPT-ARGS...]
set -euo pipefail

agent=${1:?usage: agent <claude|opencode|crush> [PROMPT-ARGS...]}
shift

case $agent in
    claude | opencode | crush) ;;
    *) echo "agent: unknown agent '$agent', use claude, opencode or crush" >&2; exit 2 ;;
esac

# Unqualified, so a project can override the prompt launch
prompt=$("${BEANS_BIN:-beans}" launch prompt -- "$@")

# The agent's shell is no longer part of the launch chain, so beans launch
# calls made from it must not be pinned to this launch's config
unset BEANS_LAUNCH_CONFIG BEANS_LAUNCH_LAYER BEANS_LAUNCH_DEPTH

case $agent in
    claude) exec claude "$prompt" ;;
    opencode) exec opencode --prompt "$prompt" ;;
    # Interactive crush takes no prompt, so this runs it once, non-interactively
    crush) exec crush run "$prompt" ;;
esac
