#!/usr/bin/env bash
# Clean up bean modifications from feature branch history
# Moves .beans changes to main branch, then merges main back

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Parse arguments
DRY_RUN=false
TARGET_BRANCH="main"
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --target)
            TARGET_BRANCH="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Clean up .beans modifications from feature branch history."
            echo "Moves .beans changes to target branch, then merges target back."
            echo ""
            echo "Options:"
            echo "  --dry-run         Show what would be done without making changes"
            echo "  --target BRANCH   Target branch to move .beans changes to (default: main)"
            echo "  -h, --help        Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--dry-run] [--target BRANCH]"
            echo "Try '$0 --help' for more information."
            exit 1
            ;;
    esac
done

echo -e "${CYAN}=== Bean History Cleanup ===${NC}\n"

# Get current branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ -z "$CURRENT_BRANCH" ]]; then
    echo -e "${RED}Error: Cannot run on detached HEAD${NC}"
    exit 1
fi

if [[ "$CURRENT_BRANCH" == "$TARGET_BRANCH" ]]; then
    echo -e "${RED}Error: Already on $TARGET_BRANCH branch. This script is for cleaning up feature branches.${NC}"
    exit 1
fi

echo -e "Current branch: ${YELLOW}${CURRENT_BRANCH}${NC}"
echo -e "Target branch:  ${YELLOW}${TARGET_BRANCH}${NC}\n"

# Check if target branch exists locally, if not try to create it from remote
if ! git rev-parse --verify "$TARGET_BRANCH" >/dev/null 2>&1; then
    echo -e "${YELLOW}Local $TARGET_BRANCH branch does not exist.${NC}"
    
    # Try to create from remote
    if git rev-parse --verify "origin/$TARGET_BRANCH" >/dev/null 2>&1; then
        echo -e "${YELLOW}Creating local $TARGET_BRANCH from origin/$TARGET_BRANCH${NC}"
        if git branch --track "$TARGET_BRANCH" "origin/$TARGET_BRANCH"; then
            echo -e "${GREEN}✓ Local $TARGET_BRANCH branch created${NC}\n"
        else
            echo -e "${RED}✗ Failed to create local $TARGET_BRANCH branch${NC}"
            exit 1
        fi
    else
        echo -e "${RED}Error: $TARGET_BRANCH branch does not exist locally or on remote${NC}"
        exit 1
    fi
fi

# Check for uncommitted changes and handle them
HAS_UNCOMMITTED=false
if ! git diff-index --quiet HEAD --; then
    HAS_UNCOMMITTED=true
    echo -e "${YELLOW}Uncommitted changes detected.${NC}"
    echo -e "${YELLOW}These will be committed to the current branch before cleanup.${NC}\n"
    git status --short
    echo ""
fi

if [[ "$DRY_RUN" == "true" ]]; then
    echo -e "${CYAN}=== DRY RUN MODE ===${NC}\n"
    echo "Would perform the following steps:"
    if [[ "$HAS_UNCOMMITTED" == "true" ]]; then
        echo "1. Commit uncommitted changes to $CURRENT_BRANCH"
        echo "2. Extract .beans changes from current branch"
        echo "3. Switch to $TARGET_BRANCH branch"
        echo "4. Apply .beans changes to $TARGET_BRANCH (excluding .beans/.worktrees)"
        echo "5. Commit changes to $TARGET_BRANCH"
        echo "6. Switch back to $CURRENT_BRANCH"
        echo "7. Rewrite commit history to remove all .beans modifications"
        echo "8. Merge $TARGET_BRANCH to restore .beans metadata"
    else
        echo "1. Extract .beans changes from current branch"
        echo "2. Switch to $TARGET_BRANCH branch"
        echo "3. Apply .beans changes to $TARGET_BRANCH (excluding .beans/.worktrees)"
        echo "4. Commit changes to $TARGET_BRANCH"
        echo "5. Switch back to $CURRENT_BRANCH"
        echo "6. Rewrite commit history to remove all .beans modifications"
        echo "7. Merge $TARGET_BRANCH to restore .beans metadata"
    fi
    echo ""
    echo -e "${CYAN}Changes that would be committed to $TARGET_BRANCH:${NC}"
    git diff "$TARGET_BRANCH"...HEAD -- .beans/ ':!.beans/.worktrees' --stat
    if [[ "$HAS_UNCOMMITTED" == "true" ]]; then
        echo ""
        echo -e "${CYAN}Uncommitted changes (will be committed first):${NC}"
        git diff HEAD --stat
        git diff --staged --stat 2>/dev/null
    fi
    exit 0
fi

# Confirm action
echo -e "${CYAN}This will:${NC}"
if [[ "$HAS_UNCOMMITTED" == "true" ]]; then
    echo "1. Commit uncommitted changes to $CURRENT_BRANCH"
    echo "2. Extract .beans changes from $CURRENT_BRANCH"
    echo "3. Commit them to $TARGET_BRANCH (excluding .beans/.worktrees)"
    echo "4. Rewrite $CURRENT_BRANCH history to remove all .beans modifications"
    echo "5. Merge $TARGET_BRANCH to restore .beans metadata"
else
    echo "1. Extract .beans changes from $CURRENT_BRANCH"
    echo "2. Commit them to $TARGET_BRANCH (excluding .beans/.worktrees)"
    echo "3. Rewrite $CURRENT_BRANCH history to remove all .beans modifications"
    echo "4. Merge $TARGET_BRANCH to restore .beans metadata"
fi
echo ""
echo -e "${YELLOW}Preview of changes to move to $TARGET_BRANCH:${NC}"
git diff "$TARGET_BRANCH"...HEAD -- .beans/ ':!.beans/.worktrees' --stat
if [[ "$HAS_UNCOMMITTED" == "true" ]]; then
    echo ""
    echo -e "${YELLOW}Uncommitted changes (will be committed first):${NC}"
    git status --short
fi
echo ""
read -p "Continue? [y/N] " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled."
    exit 0
fi

# Create temporary directory for storing changes
TEMP_DIR=$(mktemp -d)
PATCH_FILE="$TEMP_DIR/beans-changes.patch"

# Commit uncommitted changes if any (do this before checking for .beans changes)
if [[ "$HAS_UNCOMMITTED" == "true" ]]; then
    if [[ "$DRY_RUN" != "true" ]]; then
        echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${CYAN}Step 1: Commit uncommitted changes${NC}\n"
        
        # Add all changes
        git add -A
        
        # Create commit message
        COMMIT_MSG="wip: uncommitted changes before bean history cleanup

Auto-committed by cleanup-bean-history.sh"
        
        if git commit -m "$COMMIT_MSG"; then
            echo -e "${GREEN}✓ Changes committed to $CURRENT_BRANCH${NC}"
        else
            echo -e "${RED}✗ Failed to commit changes${NC}"
            rm -rf "$TEMP_DIR"
            exit 1
        fi
    fi
    
    STEP_NUM=2
else
    STEP_NUM=1
fi

# Now check if there are any .beans changes in the current branch vs target
# (after potentially committing, so uncommitted .beans changes are included)
if ! git diff "$TARGET_BRANCH"...HEAD -- .beans/ ':!.beans/.worktrees' | grep -q '^'; then
    echo -e "${YELLOW}No .beans changes found in current branch vs $TARGET_BRANCH.${NC}"
    echo -e "${YELLOW}Nothing to clean up.${NC}"
    exit 0
fi

echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}Step $STEP_NUM: Extract .beans changes${NC}\n"

# Create patch of .beans changes (excluding .worktrees)
if git diff "$TARGET_BRANCH"...HEAD -- .beans/ ':!.beans/.worktrees' > "$PATCH_FILE"; then
    echo -e "${GREEN}✓ Changes extracted to patch file${NC}"
else
    echo -e "${RED}✗ Failed to create patch${NC}"
    rm -rf "$TEMP_DIR"
    exit 1
fi

# Check if patch is empty
if [[ ! -s "$PATCH_FILE" ]]; then
    echo -e "${YELLOW}No changes to move (excluding .worktrees)${NC}"
    rm -rf "$TEMP_DIR"
    exit 0
fi

((STEP_NUM++))
echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}Step $STEP_NUM: Switch to $TARGET_BRANCH branch${NC}\n"

if git checkout "$TARGET_BRANCH"; then
    echo -e "${GREEN}✓ Switched to $TARGET_BRANCH${NC}"
else
    echo -e "${RED}✗ Failed to switch to $TARGET_BRANCH${NC}"
    rm -rf "$TEMP_DIR"
    exit 1
fi

((STEP_NUM++))
echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}Step $STEP_NUM: Apply .beans changes to $TARGET_BRANCH${NC}\n"

if git apply "$PATCH_FILE"; then
    echo -e "${GREEN}✓ Changes applied${NC}"
else
    echo -e "${RED}✗ Failed to apply patch${NC}"
    echo -e "${YELLOW}Attempting to switch back to $CURRENT_BRANCH${NC}"
    git checkout "$CURRENT_BRANCH"
    rm -rf "$TEMP_DIR"
    exit 1
fi

# Check if there are actually changes to commit
if git diff-index --quiet HEAD --; then
    echo -e "${YELLOW}No changes to commit after applying patch${NC}"
    git checkout "$CURRENT_BRANCH"
    rm -rf "$TEMP_DIR"
    exit 0
fi

((STEP_NUM++))
echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}Step $STEP_NUM: Commit changes to $TARGET_BRANCH${NC}\n"

# Stage all .beans changes (excluding .worktrees which should be in .gitignore anyway)
git add .beans/

# Create commit message
COMMIT_MSG="chore: sync bean metadata from $CURRENT_BRANCH

Moved .beans changes from $CURRENT_BRANCH to $TARGET_BRANCH to keep
bean metadata on the $TARGET_BRANCH branch and clean up feature branch history."

if git commit -m "$COMMIT_MSG"; then
    echo -e "${GREEN}✓ Changes committed to $TARGET_BRANCH${NC}"
    TARGET_COMMIT=$(git rev-parse HEAD)
    echo -e "Commit: ${YELLOW}${TARGET_COMMIT:0:8}${NC}"
else
    echo -e "${RED}✗ Failed to commit${NC}"
    git reset --hard
    git checkout "$CURRENT_BRANCH"
    rm -rf "$TEMP_DIR"
    exit 1
fi

((STEP_NUM++))
echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}Step $STEP_NUM: Switch back to $CURRENT_BRANCH${NC}\n"

if git checkout "$CURRENT_BRANCH"; then
    echo -e "${GREEN}✓ Switched back to $CURRENT_BRANCH${NC}"
else
    echo -e "${RED}✗ Failed to switch back${NC}"
    rm -rf "$TEMP_DIR"
    exit 1
fi

((STEP_NUM++))
echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}Step $STEP_NUM: Rewrite history to remove .beans changes${NC}\n"

echo -e "${YELLOW}Rewriting $CURRENT_BRANCH history to remove all .beans modifications...${NC}"
echo -e "${YELLOW}This removes .beans changes from all commits in the feature branch.${NC}\n"

# Use git filter-branch to remove .beans changes from all commits (except .worktrees)
# This rewrites history to exclude .beans modifications while keeping all other changes

# First, ensure we're on the current branch
git checkout "$CURRENT_BRANCH" >/dev/null 2>&1

# Check if there are any commits to rewrite
COMMITS_TO_REWRITE=$(git rev-list "$TARGET_BRANCH".."$CURRENT_BRANCH" 2>/dev/null)

if [[ -z "$COMMITS_TO_REWRITE" ]]; then
    echo -e "${YELLOW}No commits to rewrite${NC}"
else
    # Count commits for user feedback
    COMMIT_COUNT=$(echo "$COMMITS_TO_REWRITE" | wc -l)
    echo -e "${YELLOW}Rewriting $COMMIT_COUNT commit(s)...${NC}\n"
    
    # Use filter-branch with index-filter to remove .beans from the index in each commit
    # --index-filter is faster than --tree-filter as it doesn't check out files
    # We remove all .beans files except .worktrees
    
    FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch -f --index-filter '
        # Remove .beans files from the index (except .worktrees)
        git ls-files -s | 
        grep "^[0-9]* [0-9a-f]* [0-9]\t\.beans/" | 
        grep -v "\.beans/\.worktrees/" | 
        cut -f2 | 
        xargs -r git rm --cached --ignore-unmatch -q 2>/dev/null || true
    ' "$TARGET_BRANCH".."$CURRENT_BRANCH" 2>&1 | grep -v "^Rewrite" | grep -v "^WARNING" || true
    
    # Clean up filter-branch refs
    git for-each-ref --format="%(refname)" refs/original/ | xargs -r git update-ref -d 2>/dev/null || true
    
    echo -e "${GREEN}✓ History rewritten - .beans changes removed from all commits${NC}"
    
    # Now merge main to get the .beans changes back (from main)
    echo -e "${YELLOW}Merging $TARGET_BRANCH to restore .beans metadata...${NC}"
    if git merge "$TARGET_BRANCH" --no-edit -m "Merge $TARGET_BRANCH (restore .beans metadata)"; then
        echo -e "${GREEN}✓ Merged $TARGET_BRANCH${NC}"
    else
        echo -e "${RED}✗ Merge failed${NC}"
        echo -e "${YELLOW}You may need to resolve conflicts manually.${NC}"
        rm -rf "$TEMP_DIR"
        exit 1
    fi
fi

# Clean up
rm -rf "$TEMP_DIR"

echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "\n${CYAN}=== Cleanup Complete ===${NC}\n"

echo -e "${GREEN}✓ .beans changes moved to $TARGET_BRANCH${NC}"
echo -e "${GREEN}✓ $CURRENT_BRANCH rebased onto $TARGET_BRANCH${NC}"
echo ""
echo -e "Next steps:"
echo -e "  - Review the changes with: ${YELLOW}git log --oneline${NC}"
echo -e "  - Push to remote if needed: ${YELLOW}git push --force-with-lease${NC}"
echo -e "  - Push $TARGET_BRANCH: ${YELLOW}git push origin $TARGET_BRANCH${NC}"
echo ""
