#!/usr/bin/env bash
# Start a coding agent on the bean in the current directory: agent <claude|opencode|crush> [PROMPT-ARGS...]
set -euo pipefail

agent=${1:?usage: agent <claude|opencode|crush> [PROMPT-ARGS...]}
shift

# Unqualified, so a project can override the prompt launch
prompt=$("${BEANS_BIN:-beans}" launch prompt -- "$@")

case $agent in
    claude) exec claude "$prompt" ;;
    opencode) exec opencode --prompt "$prompt" ;;
    # Interactive crush takes no prompt, so this runs it once, non-interactively
    crush) exec crush run "$prompt" ;;
    *) echo "agent: unknown agent '$agent', use claude, opencode or crush" >&2; exit 2 ;;
esac
