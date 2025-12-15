#!/usr/bin/env bash
# Lists bookmarks that are not ancestors or descendants of the current change
# and shows their corresponding bean status

set -euo pipefail

# Parse arguments
filter_completed=false
if [[ "${1:-}" == "--completed" ]]; then
    filter_completed=true
fi

# Get current change ID
current_change=$(jj log -r @ --no-graph -T 'change_id.short()' 2>/dev/null | head -1)

if [[ "$filter_completed" == true ]]; then
    echo "Completed bookmarks not included in current change (@):"
else
    echo "Bookmarks not included in current change (@):"
fi
echo ""
printf "%-40s %-12s %s\n" "BOOKMARK" "STATUS" "TITLE"
printf "%-40s %-12s %s\n" "────────────────────────────────────────" "────────────" "────────────────────────────────────"

# List all bookmarks and filter out those that include the current change
jj bookmark list | while read -r line; do
    # Extract bookmark name (first field before colon)
    bookmark=$(echo "$line" | awk '{print $1}' | sed 's/:$//')
    
    # Skip empty lines
    [[ -z "$bookmark" ]] && continue
    
    # Check if current change is an ancestor or descendant of this bookmark
    # If the bookmark includes @, skip it (it's related to current change)
    if ! jj log -r "$bookmark" --no-graph -T 'change_id.short()' 2>/dev/null | grep -q "^$current_change$"; then
        # Extract bean ID from bookmark (handles task/beans-XXXX, feat/beans-XXXX, or just beans)
        bean_id=""
        if [[ "$bookmark" =~ beans-([a-z0-9]{4}) ]]; then
            bean_id="beans-${BASH_REMATCH[1]}"
        elif [[ "$bookmark" == "beans" ]]; then
            bean_id="beans"
        fi
        
        # Look up bean status if we found a bean ID
        status="─"
        title="(not a bean)"
        if [[ -n "$bean_id" ]]; then
            # Use GraphQL to query bean
            result=$(go run . graphql --json "query { bean(id: \"$bean_id\") { status title } }" 2>/dev/null || echo "")
            
            if [[ -n "$result" ]] && echo "$result" | grep -q '"status"'; then
                status=$(echo "$result" | grep -o '"status":"[^"]*"' | cut -d':' -f2 | tr -d '"' || echo "─")
                title=$(echo "$result" | grep -o '"title":"[^"]*"' | sed 's/"title":"//' | sed 's/"$//' | head -c 40 || echo "")
            fi
        fi
        
        # Skip if filtering for completed and this isn't completed
        if [[ "$filter_completed" == true ]] && [[ "$status" != "completed" ]]; then
            continue
        fi
        
        printf "%-40s %-12s %s\n" "$bookmark" "$status" "$title"
    fi
done
