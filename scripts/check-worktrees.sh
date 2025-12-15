#!/usr/bin/env bash
# Check worktree status, merge status, and bean details

set -euo pipefail

# Parse arguments
FILTER_NO_MERGED=false
COMMAND=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --no-merged)
            FILTER_NO_MERGED=true
            shift
            ;;
        merge)
            COMMAND="merge"
            shift
            # Parse merge subcommand options
            while [[ $# -gt 0 ]]; do
                case $1 in
                    --completed)
                        MERGE_MODE="completed"
                        shift
                        ;;
                    *)
                        echo "Unknown merge option: $1"
                        echo "Usage: $0 merge --completed"
                        exit 1
                        ;;
                esac
            done
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS] [COMMAND]"
            echo ""
            echo "Check worktree status, merge status, and bean details."
            echo ""
            echo "Options:"
            echo "  --no-merged     Show only worktrees NOT merged into current branch"
            echo "  -h, --help      Show this help message"
            echo ""
            echo "Commands:"
            echo "  merge --completed    Merge all unmerged completed worktrees into current branch"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--no-merged]"
            echo "Try '$0 --help' for more information."
            exit 1
            ;;
    esac
done

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
GRAY='\033[0;90m'
NC='\033[0m' # No Color

# Get current branch or HEAD commit
CURRENT_BRANCH=$(git branch --show-current)
CURRENT_HEAD=$(git rev-parse HEAD)
CURRENT_REF=""

if [[ -z "$CURRENT_BRANCH" ]]; then
    # Detached HEAD - use the commit hash
    CURRENT_REF="$CURRENT_HEAD"
    CURRENT_DISPLAY="${CURRENT_HEAD:0:8} (detached HEAD)"
else
    CURRENT_REF="$CURRENT_BRANCH"
    CURRENT_DISPLAY="$CURRENT_BRANCH"
fi

# Handle merge command
if [[ "$COMMAND" == "merge" ]]; then
    if [[ -z "$CURRENT_BRANCH" ]]; then
        echo -e "${RED}Error: Cannot merge into detached HEAD. Please checkout a branch first.${NC}"
        exit 1
    fi
    
    if [[ "$MERGE_MODE" != "completed" ]]; then
        echo -e "${RED}Error: merge command requires --completed flag${NC}"
        exit 1
    fi
    
    echo -e "${CYAN}=== Merging Completed Worktrees ===${NC}\n"
    echo -e "Current branch: ${YELLOW}${CURRENT_BRANCH}${NC}\n"
    
    # Collect worktrees to merge
    WORKTREES_TO_MERGE=()
    WORKTREE_BRANCHES=()
    WORKTREE_BEANS=()
    WORKTREE_TITLES=()
    
    # Parse worktree list
    WORKTREE_PATH=""
    HEAD_COMMIT=""
    BRANCH=""
    IS_DETACHED=false
    
    while IFS= read -r line; do
        if [[ $line == worktree* ]]; then
            WORKTREE_PATH="${line#worktree }"
            HEAD_COMMIT=""
            BRANCH=""
            IS_DETACHED=false
        elif [[ $line == HEAD* ]]; then
            HEAD_COMMIT="${line#HEAD }"
        elif [[ $line == "detached" ]]; then
            IS_DETACHED=true
        elif [[ $line == branch* ]]; then
            BRANCH="${line#branch refs/heads/}"
        elif [[ -z "$line" && -n "$WORKTREE_PATH" ]]; then
            # Skip if this is the current worktree
            if [[ "$HEAD_COMMIT" == "$CURRENT_HEAD" ]]; then
                continue
            fi
            
            # Skip if already merged
            if git merge-base --is-ancestor "$HEAD_COMMIT" "$CURRENT_HEAD" 2>/dev/null; then
                continue
            fi
            
            # Extract bean ID from branch name
            BEAN_ID=""
            if [[ $BRANCH == task/beans-* ]]; then
                BEAN_ID="${BRANCH#task/}"
            elif [[ $BRANCH == beans-* ]]; then
                BEAN_ID="$BRANCH"
            fi
            
            # Check if bean is completed
            if [[ -n "$BEAN_ID" ]]; then
                BEAN_FILE=$(find .beans -maxdepth 1 -name "${BEAN_ID}--*.md" 2>/dev/null | head -1)
                
                if [[ -n "$BEAN_FILE" && -f "$BEAN_FILE" ]]; then
                    BEAN_STATUS=$(grep "^status:" "$BEAN_FILE" | sed 's/^status: *//' | tr -d "'\"")
                    BEAN_TITLE=$(grep "^title:" "$BEAN_FILE" | sed 's/^title: *//' | tr -d "'\"")
                    
                    if [[ "$BEAN_STATUS" == "completed" ]]; then
                        WORKTREES_TO_MERGE+=("$WORKTREE_PATH")
                        WORKTREE_BRANCHES+=("$BRANCH")
                        WORKTREE_BEANS+=("$BEAN_ID")
                        WORKTREE_TITLES+=("$BEAN_TITLE")
                    fi
                fi
            fi
        fi
    done < <(git worktree list --porcelain)
    
    # Check if there are any worktrees to merge
    if [[ ${#WORKTREES_TO_MERGE[@]} -eq 0 ]]; then
        echo -e "${YELLOW}No unmerged completed worktrees found.${NC}"
        exit 0
    fi
    
    # Display worktrees to be merged
    echo -e "Found ${GREEN}${#WORKTREES_TO_MERGE[@]}${NC} completed worktree(s) to merge:\n"
    for i in "${!WORKTREES_TO_MERGE[@]}"; do
        echo -e "  ${CYAN}${WORKTREE_BEANS[$i]}${NC} (${YELLOW}${WORKTREE_BRANCHES[$i]}${NC})"
        echo -e "    ${GRAY}${WORKTREE_TITLES[$i]}${NC}"
    done
    echo ""
    
    # Confirm merge
    read -p "Proceed with merge? [y/N] " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Merge cancelled."
        exit 0
    fi
    
    # Add all changes (tracked and untracked) and stash
    STASHED=false
    echo -e "${YELLOW}Stashing all changes (tracked and untracked) before merge...${NC}"
    git add -A
    if git stash push --include-untracked -m "Auto-stash before merge --completed"; then
        STASHED=true
        echo -e "${GREEN}✓ All changes stashed${NC}\n"
    else
        echo -e "${RED}✗ Failed to stash changes${NC}"
        exit 1
    fi
    
    # Temporarily move .worktrees directories to avoid merge conflicts
    WORKTREES_MOVED=false
    if [[ -d ".beans/.worktrees" ]]; then
        TEMP_WORKTREES_DIR=$(mktemp -d)
        echo -e "${YELLOW}Temporarily moving .worktrees directory to avoid conflicts...${NC}"
        if mv .beans/.worktrees "$TEMP_WORKTREES_DIR/"; then
            WORKTREES_MOVED=true
            echo -e "${GREEN}✓ Worktrees moved to $TEMP_WORKTREES_DIR${NC}\n"
        else
            echo -e "${RED}✗ Failed to move worktrees${NC}"
            exit 1
        fi
    fi
    
    # Perform merges
    MERGE_SUCCESS=0
    MERGE_FAILED=0
    
    for i in "${!WORKTREES_TO_MERGE[@]}"; do
        BRANCH="${WORKTREE_BRANCHES[$i]}"
        BEAN="${WORKTREE_BEANS[$i]}"
        
        echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "Merging: ${CYAN}${BEAN}${NC} (${YELLOW}${BRANCH}${NC})"
        
        if git merge --no-edit "$BRANCH"; then
            echo -e "${GREEN}✓ Merge successful${NC}"
            MERGE_SUCCESS=$((MERGE_SUCCESS + 1))
        else
            echo -e "${RED}✗ Merge failed${NC}"
            MERGE_FAILED=$((MERGE_FAILED + 1))
            echo -e "${YELLOW}Aborting this merge. Fix conflicts and merge manually.${NC}"
            git merge --abort 2>/dev/null || true
        fi
    done
    
    # Restore .worktrees directory if moved
    if [[ "$WORKTREES_MOVED" == "true" ]]; then
        echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "\n${YELLOW}Restoring .worktrees directory...${NC}"
        mkdir -p .beans
        if mv "$TEMP_WORKTREES_DIR/.worktrees" .beans/; then
            echo -e "${GREEN}✓ Worktrees restored${NC}"
            rm -rf "$TEMP_WORKTREES_DIR"
        else
            echo -e "${RED}✗ Failed to restore worktrees${NC}"
            echo -e "${YELLOW}Worktrees are in: $TEMP_WORKTREES_DIR/.worktrees${NC}"
        fi
    fi
    
    # Restore stashed changes if any
    if [[ "$STASHED" == "true" ]]; then
        echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "\n${YELLOW}Restoring stashed changes...${NC}"
        if git stash pop; then
            echo -e "${GREEN}✓ Changes restored${NC}"
        else
            echo -e "${RED}✗ Failed to restore stashed changes${NC}"
            echo -e "${YELLOW}Your changes are still in the stash. Use 'git stash pop' to restore them.${NC}"
        fi
    fi
    
    # Summary
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "\n${CYAN}=== Merge Summary ===${NC}\n"
    echo -e "Successful: ${GREEN}${MERGE_SUCCESS}${NC}"
    echo -e "Failed:     ${RED}${MERGE_FAILED}${NC}"
    
    if [[ $MERGE_FAILED -gt 0 ]]; then
        echo -e "\n${YELLOW}Some merges failed. Please resolve conflicts manually.${NC}"
        exit 1
    fi
    
    exit 0
fi

# Target branches to check (check both local and remote refs)
TARGETS=()
for branch in "main" "origin/main" "dev" "origin/dev"; do
    if git rev-parse --verify "$branch" >/dev/null 2>&1; then
        TARGETS+=("$branch")
    fi
done
if [[ -n "$CURRENT_BRANCH" ]]; then
    TARGETS+=("$CURRENT_BRANCH")
fi

echo -e "${CYAN}=== Worktree Status ===${NC}\n"
echo -e "Current: ${YELLOW}${CURRENT_DISPLAY}${NC}"
if [[ "$FILTER_NO_MERGED" == "true" ]]; then
    echo -e "${GRAY}Filter: Only showing worktrees NOT merged into current branch${NC}"
fi
echo ""

# Parse worktree list
WORKTREE_PATH=""
HEAD_COMMIT=""
BRANCH=""
IS_DETACHED=false

git worktree list --porcelain | while IFS= read -r line; do
    if [[ $line == worktree* ]]; then
        WORKTREE_PATH="${line#worktree }"
        HEAD_COMMIT=""
        BRANCH=""
        IS_DETACHED=false
    elif [[ $line == HEAD* ]]; then
        HEAD_COMMIT="${line#HEAD }"
    elif [[ $line == "detached" ]]; then
        IS_DETACHED=true
        # For detached HEAD, use the commit as the "branch"
        BRANCH="${HEAD_COMMIT:0:8}"
    elif [[ $line == branch* ]]; then
        BRANCH="${line#branch refs/heads/}"
        
    # Empty line signifies end of a worktree entry - process it
    elif [[ -z "$line" && -n "$WORKTREE_PATH" ]]; then
        # Extract bean ID from branch name (task/beans-XXXX -> beans-XXXX)
        BEAN_ID=""
        if [[ $BRANCH == task/beans-* ]]; then
            BEAN_ID="${BRANCH#task/}"
        elif [[ $BRANCH == beans-* ]]; then
            BEAN_ID="$BRANCH"
        fi
        
        # Find the bean file
        BEAN_FILE=""
        BEAN_TITLE=""
        BEAN_STATUS=""
        
        if [[ -n "$BEAN_ID" ]]; then
            # Look for bean file matching the ID
            BEAN_FILE=$(find .beans -maxdepth 1 -name "${BEAN_ID}--*.md" 2>/dev/null | head -1)
            
            if [[ -n "$BEAN_FILE" && -f "$BEAN_FILE" ]]; then
                # Extract title and status from frontmatter
                BEAN_TITLE=$(grep "^title:" "$BEAN_FILE" | sed 's/^title: *//' | tr -d "'\"")
                BEAN_STATUS=$(grep "^status:" "$BEAN_FILE" | sed 's/^status: *//' | tr -d "'\"")
            fi
        fi
        
        # Check if we should skip this branch based on --no-merged filter
        SKIP_BRANCH=false
        if [[ "$FILTER_NO_MERGED" == "true" ]]; then
            # Skip if this worktree is the current one (same HEAD commit)
            if [[ "$HEAD_COMMIT" == "$CURRENT_HEAD" ]]; then
                SKIP_BRANCH=true
            fi
            
            # Skip if branch is merged into current (use HEAD commit for comparison)
            if [[ "$SKIP_BRANCH" == "false" ]]; then
                if git merge-base --is-ancestor "$HEAD_COMMIT" "$CURRENT_HEAD" 2>/dev/null; then
                    SKIP_BRANCH=true
                fi
            fi
        fi
        
        # Skip this worktree if filtered
        if [[ "$SKIP_BRANCH" == "true" ]]; then
            continue
        fi
        
        # Display header
        echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        if [[ "$IS_DETACHED" == "true" ]]; then
            echo -e "HEAD:   ${YELLOW}${BRANCH}${NC} ${GRAY}(detached)${NC}"
        else
            echo -e "Branch: ${YELLOW}${BRANCH}${NC}"
        fi
        echo -e "Path:   ${GRAY}${WORKTREE_PATH}${NC}"
        
        if [[ -n "$BEAN_TITLE" ]]; then
            # Colorize status
            STATUS_COLOR=$NC
            case "$BEAN_STATUS" in
                completed) STATUS_COLOR=$GREEN ;;
                in-progress) STATUS_COLOR=$YELLOW ;;
                blocked) STATUS_COLOR=$RED ;;
                *) STATUS_COLOR=$GRAY ;;
            esac
            
            echo -e "Bean:   ${CYAN}${BEAN_ID}${NC} [${STATUS_COLOR}${BEAN_STATUS}${NC}]"
            echo -e "Title:  ${BEAN_TITLE}"
        elif [[ -n "$BEAN_ID" ]]; then
            echo -e "Bean:   ${RED}${BEAN_ID} (not found)${NC}"
        fi
        
        # Check merge status for each target
        echo -e "\nMerge status:"
        for target in "${TARGETS[@]}"; do
            # Check if target branch exists
            if ! git rev-parse --verify "$target" >/dev/null 2>&1; then
                echo -e "  ${target}: ${GRAY}(branch doesn't exist)${NC}"
                continue
            fi
            
            TARGET_COMMIT=$(git rev-parse "$target")
            
            # Check if they're the same commit
            if [[ "$HEAD_COMMIT" == "$TARGET_COMMIT" ]]; then
                echo -e "  ${target}: ${GREEN}✓ same commit${NC}"
                continue
            fi
            
            # Check if this worktree's HEAD is merged into target
            if git merge-base --is-ancestor "$HEAD_COMMIT" "$target" 2>/dev/null; then
                echo -e "  ${target}: ${GREEN}✓ merged${NC}"
            else
                echo -e "  ${target}: ${RED}✗ not merged${NC}"
            fi
        done
        
        # Also check against current HEAD if it's detached
        if [[ -z "$CURRENT_BRANCH" && "$HEAD_COMMIT" != "$CURRENT_HEAD" ]]; then
            if [[ "$HEAD_COMMIT" == "$CURRENT_HEAD" ]]; then
                echo -e "  current: ${GREEN}✓ same commit${NC}"
            elif git merge-base --is-ancestor "$HEAD_COMMIT" "$CURRENT_HEAD" 2>/dev/null; then
                echo -e "  current: ${GREEN}✓ merged${NC}"
            else
                echo -e "  current: ${RED}✗ not merged${NC}"
            fi
        fi
        
        echo ""
    fi
done

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Summary
echo -e "\n${CYAN}=== Summary ===${NC}\n"

TOTAL_WORKTREES=$(git worktree list | wc -l)
MERGED_TO_CURRENT=0
NOT_MERGED_TO_CURRENT=0

# Count merge status to current HEAD
TEMP_HEAD=""
while IFS= read -r line; do
    if [[ $line == HEAD* ]]; then
        TEMP_HEAD="${line#HEAD }"
    elif [[ -z "$line" && -n "$TEMP_HEAD" ]]; then
        # End of worktree entry, check if merged
        if [[ "$TEMP_HEAD" != "$CURRENT_HEAD" ]]; then
            if git merge-base --is-ancestor "$TEMP_HEAD" "$CURRENT_HEAD" 2>/dev/null; then
                MERGED_TO_CURRENT=$((MERGED_TO_CURRENT + 1))
            else
                NOT_MERGED_TO_CURRENT=$((NOT_MERGED_TO_CURRENT + 1))
            fi
        fi
        TEMP_HEAD=""
    fi
done < <(git worktree list --porcelain)

echo -e "Total worktrees:                ${YELLOW}${TOTAL_WORKTREES}${NC}"
echo -e "Merged to ${YELLOW}${CURRENT_DISPLAY}${NC}:  ${GREEN}${MERGED_TO_CURRENT}${NC}"
echo -e "Not merged to ${YELLOW}${CURRENT_DISPLAY}${NC}: ${RED}${NOT_MERGED_TO_CURRENT}${NC}"
echo ""
