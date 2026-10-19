#!/usr/bin/env bash
# Ensure a git worktree on branch task/<bean-id> and print its path: worktree [--base BRANCH] [--dir DIR]
set -euo pipefail

: "${BEANS_ID:?BEANS_ID not set}"
repo=${BEANS_ROOT:-$PWD}
base=""
dir=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --base) base=${2:?--base needs a value}; shift 2 ;;
        --dir) dir=${2:?--dir needs a value}; shift 2 ;;
        *) echo "usage: worktree [--base BRANCH] [--dir DIR]" >&2; exit 2 ;;
    esac
done
# A relative --dir is relative to the caller's cwd, not the repo
[[ -z $dir || $dir == /* ]] || dir=$PWD/$dir
dir=${dir:-$repo/.worktrees/$BEANS_ID}
branch=task/$BEANS_ID

# The worktree that has the branch checked out, if any
checked_out_in() {
    local line path=""
    while IFS= read -r line; do
        case $line in
            "worktree "*) path=${line#worktree } ;;
            "branch refs/heads/$1") echo "$path"; return ;;
        esac
    done < <(git -C "$repo" worktree list --porcelain)
}

if [[ ! -d $dir ]]; then
    git -C "$repo" worktree prune
    existing=$(checked_out_in "$branch")
    if [[ -n $existing ]]; then
        dir=$existing
    elif git -C "$repo" show-ref --verify --quiet "refs/heads/$branch"; then
        git -C "$repo" worktree add --quiet "$dir" "$branch" >&2
    else
        # Without --base, the branch starts at the repo's current HEAD
        git -C "$repo" worktree add --quiet -b "$branch" "$dir" ${base:+"$base"} >&2
    fi
fi

cd "$dir" && pwd -P
