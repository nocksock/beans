package cmd

import (
	"context"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	listJSON       bool
	listSearch     string
	listStatus     []string
	listNoStatus   []string
	listType       []string
	listNoType     []string
	listPriority   []string
	listNoPriority []string
	listTag        []string
	listNoTag      []string
	listHasParent   bool
	listNoParent    bool
	listParentID    string
	listHasBlocking bool
	listNoBlocking  bool
	listIsBlocked   bool
	listUnblocked   bool
	listQuiet      bool
	listSort       string
	listFull       bool
	listInclude    []string
)

// hiddenByDefaultStatuses are statuses excluded from `beans list` unless
// explicitly requested via --include or --status.
var hiddenByDefaultStatuses = []string{"completed", "scrapped"}

// defaultExcludedStatuses returns the exclude-status list for `beans list`,
// adding the hidden-by-default statuses unless they were explicitly included
// or requested via --status. Excluding a builtin status also hides its custom
// substatuses, so requesting a substatus counts as requesting its builtin.
func defaultExcludedStatuses(cfg *config.Config, status, excludeStatus, include []string) ([]string, error) {
	requested := make(map[string]bool)
	for _, s := range status {
		requested[s] = true
		requested[cfg.StatusCategory(s)] = true
	}
	for _, s := range include {
		valid := false
		for _, h := range hiddenByDefaultStatuses {
			if s == h {
				valid = true
				break
			}
		}
		if !valid {
			return nil, fmt.Errorf("invalid --include value %q (allowed: %s)", s, strings.Join(hiddenByDefaultStatuses, ", "))
		}
		requested[s] = true
	}

	result := append([]string(nil), excludeStatus...)
	for _, h := range hiddenByDefaultStatuses {
		if !requested[h] && !slices.Contains(result, h) {
			result = append(result, h)
		}
	}
	return result, nil
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all beans",
	Long: `Lists all beans in the .beans directory.

Completed and scrapped beans are hidden by default. Use --include to show them:

  beans list --include completed,scrapped
  beans list --include completed --include scrapped

Explicitly filtering by status (e.g. --status completed) also shows them.

Search Syntax (--search/-S):
  The search flag supports Bleve query string syntax:

  login          Exact term match
  login~         Fuzzy match (1 edit distance, finds "loggin", "logins")
  login~2        Fuzzy match (2 edit distance)
  log*           Wildcard prefix match
  "user login"   Exact phrase match
  user AND login Both terms required
  user OR login  Either term matches
  slug:auth      Search only in slug field
  title:login    Search only in title field
  body:auth      Search only in body field`,
	RunE: func(cmd *cobra.Command, args []string) error {
		excludeStatus, err := defaultExcludedStatuses(cfg, listStatus, listNoStatus, listInclude)
		if err != nil {
			return err
		}

		// Build GraphQL filter from CLI flags
		filter := &model.BeanFilter{
			Status:          listStatus,
			ExcludeStatus:   excludeStatus,
			Type:            listType,
			ExcludeType:     listNoType,
			Priority:        listPriority,
			ExcludePriority: listNoPriority,
			Tags:            listTag,
			ExcludeTags:     listNoTag,
		}

		// Add search filter if provided
		if listSearch != "" {
			filter.Search = &listSearch
		}

		// Add parent/blocks filters
		if listHasParent {
			filter.HasParent = &listHasParent
		}
		if listNoParent {
			filter.NoParent = &listNoParent
		}
		if listParentID != "" {
			filter.ParentID = &listParentID
		}
		if listHasBlocking {
			filter.HasBlocking = &listHasBlocking
		}
		if listNoBlocking {
			filter.NoBlocking = &listNoBlocking
		}
		if listIsBlocked {
			filter.IsBlocked = &listIsBlocked
		}
		if listUnblocked {
			isBlocked := false
			filter.IsBlocked = &isBlocked
		}

		// Execute query via GraphQL resolver
		resolver := &graph.Resolver{Core: core}
		beans, err := resolver.Query().Beans(context.Background(), filter)
		if err != nil {
			return fmt.Errorf("querying beans: %w", err)
		}

		// Sort beans
		sortBeans(beans, listSort, cfg)

		// JSON output (flat list)
		if listJSON {
			if !listFull {
				for _, b := range beans {
					b.Body = ""
				}
			}
			return output.SuccessMultiple(beans)
		}

		// Quiet mode: just IDs (flat)
		if listQuiet {
			for _, b := range beans {
				fmt.Println(b.ID)
			}
			return nil
		}

		// Default: tree view
		// We need all beans to find ancestors for context
		allBeans, err := resolver.Query().Beans(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("querying all beans for tree: %w", err)
		}

		// Create sort function for tree building
		sortFn := func(b []*bean.Bean) {
			sortBeans(b, listSort, cfg)
		}

		// Build tree
		tree := ui.BuildTree(beans, allBeans, sortFn)

		if len(tree) == 0 {
			fmt.Println(ui.Muted.Render("No beans found. Create one with: beans new <title>"))
			return nil
		}

		// Calculate max ID width from all beans in tree
		maxIDWidth := 2
		for _, b := range allBeans {
			if len(b.ID) > maxIDWidth {
				maxIDWidth = len(b.ID)
			}
		}
		maxIDWidth += 2

		// Check if any beans have tags
		hasTags := false
		for _, b := range beans {
			if len(b.Tags) > 0 {
				hasTags = true
				break
			}
		}

		// Detect terminal width (default to 80 if not a terminal)
		termWidth := 80
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			termWidth = w
		}

		fmt.Print(ui.RenderTree(tree, cfg, maxIDWidth, hasTags, termWidth))
		return nil
	},
}

func sortBeans(beans []*bean.Bean, sortBy string, cfg *config.Config) {
	statusNames := cfg.StatusNames()
	priorityNames := cfg.PriorityNames()
	typeNames := cfg.TypeNames()

	switch sortBy {
	case "created":
		sort.Slice(beans, func(i, j int) bool {
			if beans[i].CreatedAt == nil && beans[j].CreatedAt == nil {
				return beans[i].ID < beans[j].ID
			}
			if beans[i].CreatedAt == nil {
				return false
			}
			if beans[j].CreatedAt == nil {
				return true
			}
			return beans[i].CreatedAt.After(*beans[j].CreatedAt)
		})
	case "updated":
		sort.Slice(beans, func(i, j int) bool {
			if beans[i].UpdatedAt == nil && beans[j].UpdatedAt == nil {
				return beans[i].ID < beans[j].ID
			}
			if beans[i].UpdatedAt == nil {
				return false
			}
			if beans[j].UpdatedAt == nil {
				return true
			}
			return beans[i].UpdatedAt.After(*beans[j].UpdatedAt)
		})
	case "status":
		// Build status order from configured statuses
		statusOrder := make(map[string]int)
		for i, s := range statusNames {
			statusOrder[s] = i
		}
		sort.Slice(beans, func(i, j int) bool {
			oi, oj := statusOrder[beans[i].Status], statusOrder[beans[j].Status]
			if oi != oj {
				return oi < oj
			}
			return beans[i].ID < beans[j].ID
		})
	case "priority":
		// Build priority order from configured priorities
		priorityOrder := make(map[string]int)
		for i, p := range priorityNames {
			priorityOrder[p] = i
		}
		// Find normal priority index for beans without priority
		normalIdx := len(priorityNames)
		for i, p := range priorityNames {
			if p == "normal" {
				normalIdx = i
				break
			}
		}
		sort.Slice(beans, func(i, j int) bool {
			pi := normalIdx
			if beans[i].Priority != "" {
				if order, ok := priorityOrder[beans[i].Priority]; ok {
					pi = order
				}
			}
			pj := normalIdx
			if beans[j].Priority != "" {
				if order, ok := priorityOrder[beans[j].Priority]; ok {
					pj = order
				}
			}
			if pi != pj {
				return pi < pj
			}
			return beans[i].ID < beans[j].ID
		})
	case "id":
		sort.Slice(beans, func(i, j int) bool {
			return beans[i].ID < beans[j].ID
		})
	default:
		// Default: sort by status order, then priority, then type order, then title (same as TUI)
		bean.SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	listCmd.Flags().StringVarP(&listSearch, "search", "S", "", "Full-text search in title and body")
	listCmd.Flags().StringArrayVarP(&listStatus, "status", "s", nil, "Filter by status (can be repeated)")
	listCmd.Flags().StringSliceVar(&listInclude, "include", nil, "Include statuses hidden by default: completed, scrapped (comma-separated or repeated)")
	listCmd.Flags().StringArrayVar(&listNoStatus, "no-status", nil, "Exclude by status (can be repeated)")
	listCmd.Flags().StringArrayVarP(&listType, "type", "t", nil, "Filter by type (can be repeated)")
	listCmd.Flags().StringArrayVar(&listNoType, "no-type", nil, "Exclude by type (can be repeated)")
	listCmd.Flags().StringArrayVarP(&listPriority, "priority", "p", nil, "Filter by priority (can be repeated)")
	listCmd.Flags().StringArrayVar(&listNoPriority, "no-priority", nil, "Exclude by priority (can be repeated)")
	listCmd.Flags().StringArrayVar(&listTag, "tag", nil, "Filter by tag (can be repeated, OR logic)")
	listCmd.Flags().StringArrayVar(&listNoTag, "no-tag", nil, "Exclude beans with tag (can be repeated)")
	listCmd.Flags().BoolVar(&listHasParent, "has-parent", false, "Filter beans with a parent")
	listCmd.Flags().BoolVar(&listNoParent, "no-parent", false, "Filter beans without a parent")
	listCmd.Flags().StringVar(&listParentID, "parent", "", "Filter by parent ID")
	listCmd.Flags().BoolVar(&listHasBlocking, "has-blocking", false, "Filter beans that are blocking others")
	listCmd.Flags().BoolVar(&listNoBlocking, "no-blocking", false, "Filter beans that aren't blocking others")
	listCmd.Flags().BoolVar(&listIsBlocked, "is-blocked", false, "Filter beans that are blocked by an active (not completed/scrapped) bean")
	listCmd.Flags().BoolVar(&listUnblocked, "unblocked", false, "Filter beans not blocked by any active bean (ready to start)")
	listCmd.MarkFlagsMutuallyExclusive("is-blocked", "unblocked")
	listCmd.Flags().BoolVarP(&listQuiet, "quiet", "q", false, "Only output IDs (one per line)")
	listCmd.Flags().StringVar(&listSort, "sort", "", "Sort by: created, updated, status, priority, id (default: status, priority, type, title)")
	listCmd.Flags().BoolVar(&listFull, "full", false, "Include bean body in JSON output")
	rootCmd.AddCommand(listCmd)
}
