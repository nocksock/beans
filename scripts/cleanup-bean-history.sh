#!/usr/bin/env bash
# Clean up .beans modifications from feature branch history
# Strategy: Remove .beans from all commits, then merge main to get them back

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

TARGET_BRANCH="${1:-main}"

echo -e "${CYAN}=== Bean History Cleanup ===${NC}\n"

# Get current branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ -z "$CURRENT_BRANCH" ]]; then
    echo -e "${RED}Error: Cannot run on detached HEAD${NC}"
    exit 1
fi

if [[ "$CURRENT_BRANCH" == "$TARGET_BRANCH" ]]; then
    echo -e "${RED}Error: Already on $TARGET_BRANCH. This is for cleaning up feature branches.${NC}"
    exit 1
fi

echo -e "Current branch: ${YELLOW}${CURRENT_BRANCH}${NC}"
echo -e "Target branch:  ${YELLOW}${TARGET_BRANCH}${NC}\n"

# Create target branch if it doesn't exist locally
if ! git rev-parse --verify "$TARGET_BRANCH" >/dev/null 2>&1; then
    if git rev-parse --verify "origin/$TARGET_BRANCH" >/dev/null 2>&1; then
        echo -e "${YELLOW}Creating local $TARGET_BRANCH from origin/$TARGET_BRANCH${NC}"
        git branch --track "$TARGET_BRANCH" "origin/$TARGET_BRANCH"
    else
        echo -e "${RED}Error: $TARGET_BRANCH doesn't exist${NC}"
        exit 1
    fi
fi

# Step 1: Commit any uncommitted changes
if ! git diff-index --quiet HEAD --; then
    echo -e "${YELLOW}Committing uncommitted changes...${NC}"
    git add -A
    git commit -m "wip: auto-commit before cleanup"
fi

# Step 2: Create a patch with ALL .beans changes from this branch
echo -e "${YELLOW}Extracting .beans changes...${NC}"
PATCH_FILE=$(mktemp)
git diff "$TARGET_BRANCH"...HEAD --binary -- .beans/ ':!.beans/.worktrees' > "$PATCH_FILE"

if [[ ! -s "$PATCH_FILE" ]]; then
    echo -e "${YELLOW}No .beans changes to clean up${NC}"
    rm -f "$PATCH_FILE"
    exit 0
fi

# Step 3: Apply patch to target branch
echo -e "${YELLOW}Switching to $TARGET_BRANCH...${NC}"
git checkout "$TARGET_BRANCH"

echo -e "${YELLOW}Applying .beans changes to $TARGET_BRANCH...${NC}"
if git apply "$PATCH_FILE" 2>/dev/null; then
    git add .beans/
    git commit -m "chore: sync bean metadata from $CURRENT_BRANCH"
    echo -e "${GREEN}✓ .beans committed to $TARGET_BRANCH${NC}"
else
    echo -e "${RED}✗ Failed to apply patch (may already be applied)${NC}"
fi

rm -f "$PATCH_FILE"

# Step 4: Go back and rewrite history
echo -e "${YELLOW}Switching back to $CURRENT_BRANCH...${NC}"
git checkout "$CURRENT_BRANCH"

echo -e "${YELLOW}Rewriting history to remove .beans from all commits...${NC}"
FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch -f --index-filter \
    'git rm -r --cached --ignore-unmatch .beans/ >/dev/null 2>&1 || true; git reset HEAD -- .beans/.worktrees/ >/dev/null 2>&1 || true' \
    "$TARGET_BRANCH".."$CURRENT_BRANCH" 2>&1 | grep -v "^Rewrite" || true

# Clean up refs
git for-each-ref --format="%(refname)" refs/original/ | xargs -r git update-ref -d 2>/dev/null || true

echo -e "${GREEN}✓ History rewritten${NC}"

# Step 5: Merge target to get .beans back
echo -e "${YELLOW}Merging $TARGET_BRANCH to restore .beans...${NC}"
if git merge "$TARGET_BRANCH" --no-edit -m "chore: merge $TARGET_BRANCH to restore .beans metadata"; then
    echo -e "${GREEN}✓ Merged $TARGET_BRANCH${NC}"
else
    echo -e "${RED}✗ Merge failed - you may need to resolve conflicts${NC}"
    exit 1
fi

echo -e "\n${GREEN}=== Cleanup Complete ===${NC}\n"
echo -e "✓ .beans metadata is now on $TARGET_BRANCH"
echo -e "✓ Feature branch history is clean"
echo -e "\nNext steps:"
echo -e "  git push origin $TARGET_BRANCH"
echo -e "  git push --force-with-lease origin $CURRENT_BRANCH"
