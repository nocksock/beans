#!/usr/bin/env bash
# Ensure a git worktree on branch task/<bean-id> and print its path: worktree [--base BRANCH] [--dir DIR]
set -euo pipefail

: "${BEANS_ID:?BEANS_ID not set}"
repo=${BEANS_ROOT:-$PWD}
base=""
dir=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --base) base=$2; shift 2 ;;
        --dir) dir=$2; shift 2 ;;
        *) echo "usage: worktree [--base BRANCH] [--dir DIR]" >&2; exit 2 ;;
    esac
done
dir=${dir:-$repo/.worktrees/$BEANS_ID}
branch=task/$BEANS_ID

if [[ ! -d $dir ]]; then
    git -C "$repo" worktree prune
    if git -C "$repo" show-ref --verify --quiet "refs/heads/$branch"; then
        git -C "$repo" worktree add --quiet "$dir" "$branch" >&2
    else
        # Without --base, the branch starts at the repo's current HEAD
        git -C "$repo" worktree add --quiet -b "$branch" "$dir" ${base:+"$base"} >&2
    fi
fi

cd "$dir" && pwd -P
