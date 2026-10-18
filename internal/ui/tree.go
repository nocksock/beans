package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
)

// TreeNode represents a node in the bean tree hierarchy.
type TreeNode struct {
	Bean     *bean.Bean
	Children []*TreeNode
	Matched  bool // true if this bean matched the filter (vs. shown for context)
}

// TreeNodeJSON is the JSON-serializable version of TreeNode.
type TreeNodeJSON struct {
	ID       string          `json:"id"`
	Slug     string          `json:"slug,omitempty"`
	Path     string          `json:"path"`
	Title    string          `json:"title"`
	Status   string          `json:"status"`
	Type     string          `json:"type,omitempty"`
	Priority string          `json:"priority,omitempty"`
	Tags     []string        `json:"tags,omitempty"`
	Body     string          `json:"body,omitempty"`
	Matched  bool            `json:"matched"`
	Children []*TreeNodeJSON `json:"children,omitempty"`
}

// ToJSON converts a TreeNode to its JSON-serializable form.
func (n *TreeNode) ToJSON(includeFull bool) *TreeNodeJSON {
	json := &TreeNodeJSON{
		ID:       n.Bean.ID,
		Slug:     n.Bean.Slug,
		Path:     n.Bean.Path,
		Title:    n.Bean.Title,
		Status:   n.Bean.Status,
		Type:     n.Bean.Type,
		Priority: n.Bean.Priority,
		Tags:     n.Bean.Tags,
		Matched:  n.Matched,
	}
	if includeFull {
		json.Body = n.Bean.Body
	}
	if len(n.Children) > 0 {
		json.Children = make([]*TreeNodeJSON, len(n.Children))
		for i, child := range n.Children {
			json.Children[i] = child.ToJSON(includeFull)
		}
	}
	return json
}

// BuildTree builds a tree structure from filtered beans, including ancestors for context.
// matchedBeans: beans that matched the filter
// allBeans: all beans (needed to find ancestors)
// sortFn: function to sort beans at each level
func BuildTree(matchedBeans []*bean.Bean, allBeans []*bean.Bean, sortFn func([]*bean.Bean)) []*TreeNode {
	// Build index of all beans by ID
	beanByID := make(map[string]*bean.Bean)
	for _, b := range allBeans {
		beanByID[b.ID] = b
	}

	// Build set of matched bean IDs
	matchedSet := make(map[string]bool)
	for _, b := range matchedBeans {
		matchedSet[b.ID] = true
	}

	// Find all ancestors needed for context
	// Start with matched beans, then walk up parent links
	neededBeans := make(map[string]*bean.Bean)
	for _, b := range matchedBeans {
		neededBeans[b.ID] = b
	}

	// Add ancestors of matched beans
	for _, b := range matchedBeans {
		addAncestors(b, beanByID, neededBeans)
	}

	// Build children index (parent ID -> children)
	children := make(map[string][]*bean.Bean)
	for _, b := range neededBeans {
		if b.Parent != "" {
			// Only add as child if parent is in our needed set
			if _, ok := neededBeans[b.Parent]; ok {
				children[b.Parent] = append(children[b.Parent], b)
			}
		}
	}

	// Sort children at each level
	for parentID := range children {
		sortFn(children[parentID])
	}

	// Find root beans (no parent or parent not in needed set)
	var roots []*bean.Bean
	for _, b := range neededBeans {
		if b.Parent == "" {
			roots = append(roots, b)
		} else {
			// Check if parent is in the tree
			if _, ok := neededBeans[b.Parent]; !ok {
				roots = append(roots, b)
			}
		}
	}
	sortFn(roots)

	// Build tree nodes recursively
	return buildNodes(roots, children, matchedSet)
}

// addAncestors recursively adds all ancestors of a bean to the needed set.
func addAncestors(b *bean.Bean, beanByID map[string]*bean.Bean, needed map[string]*bean.Bean) {
	if b.Parent == "" {
		return
	}
	parent, ok := beanByID[b.Parent]
	if !ok {
		return // parent doesn't exist (broken link)
	}
	if _, alreadyNeeded := needed[b.Parent]; alreadyNeeded {
		return // already processed
	}
	needed[b.Parent] = parent
	addAncestors(parent, beanByID, needed)
}

// buildNodes recursively builds TreeNodes from beans.
func buildNodes(beans []*bean.Bean, children map[string][]*bean.Bean, matchedSet map[string]bool) []*TreeNode {
	nodes := make([]*TreeNode, len(beans))
	for i, b := range beans {
		nodes[i] = &TreeNode{
			Bean:     b,
			Matched:  matchedSet[b.ID],
			Children: buildNodes(children[b.ID], children, matchedSet),
		}
	}
	return nodes
}

// Tree rendering constants
const (
	treeBranch     = "├─ "
	treeLastBranch = "└─ "
	treePipe       = "│  " // vertical line for ongoing branches
	treeSpace      = "   " // empty space for completed branches
)

// RenderTree renders the tree as an ASCII tree with styled columns.
// termWidth is used to calculate responsive column widths.
func RenderTree(nodes []*TreeNode, cfg *config.Config, maxIDWidth int, hasTags bool, termWidth int) string {
	var sb strings.Builder

	cols := CalculateResponsiveColumns(termWidth, hasTags)

	// The title column takes what's left after the other columns, the space
	// after it and a margin; tree prefixes are indented within it
	titleWidth := termWidth - ColWidthType - ColWidthStatus - ColWidthSize - maxIDWidth - 2
	if cols.ShowTags {
		titleWidth -= cols.Tags
	}
	titleWidth = max(titleWidth, 20)

	// Header with manual padding (lipgloss Width doesn't handle styled strings well)
	headerCol := lipgloss.NewStyle().Foreground(ColorMuted)
	pad := func(label string, width int) string {
		return headerCol.Render(label) + strings.Repeat(" ", max(1, width-len(label)))
	}
	padLeft := func(label string, width int) string {
		return strings.Repeat(" ", max(0, width-1-len(label))) + headerCol.Render(label) + " "
	}
	header := padLeft("STATUS", ColWidthStatus) + pad("TITLE", titleWidth+1)
	if cols.ShowTags {
		header += pad("TAGS", cols.Tags)
	}
	header += padLeft("TYPE", ColWidthType) + pad("SZ", ColWidthSize) + headerCol.Render("ID")
	dividerWidth := termWidth - 1 // -1 to avoid wrapping on exact terminal width
	sb.WriteString(header)
	sb.WriteString("\n")
	sb.WriteString(Muted.Render(strings.Repeat("─", dividerWidth)))
	sb.WriteString("\n")

	// Build render config from responsive columns
	renderCfg := treeRenderConfig{
		titleWidth: titleWidth,
		cols:       cols,
	}

	// Render nodes (depth 0 = root level, no ancestry yet)
	renderNodes(&sb, nodes, 0, nil, cfg, renderCfg)

	return sb.String()
}

// treeRenderConfig holds computed rendering configuration for tree output
type treeRenderConfig struct {
	titleWidth int
	cols       ResponsiveColumns
}

// renderNodes recursively renders tree nodes with proper indentation.
// depth 0 = root level (no connector), depth 1+ = nested (has connector)
// ancestry tracks whether each parent level was a last child (true = last, no continuation line needed)
func renderNodes(sb *strings.Builder, nodes []*TreeNode, depth int, ancestry []bool, cfg *config.Config, renderCfg treeRenderConfig) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1
		renderNode(sb, node, depth, isLast, ancestry, cfg, renderCfg)
		// Only add to ancestry when depth > 0 (roots have no connectors to continue)
		if len(node.Children) > 0 {
			var newAncestry []bool
			if depth > 0 {
				newAncestry = append(ancestry, isLast)
			}
			renderNodes(sb, node.Children, depth+1, newAncestry, cfg, renderCfg)
		}
	}
}

// renderNode renders a single tree node with tree connectors.
// depth 0 = root (no connector), depth 1+ = nested (has connector)
// ancestry tracks whether each parent level was a last child (true = last, no continuation line needed)
func renderNode(sb *strings.Builder, node *TreeNode, depth int, isLast bool, ancestry []bool, cfg *config.Config, renderCfg treeRenderConfig) {
	b := node.Bean

	prefix := treePrefix(depth, ancestry, isLast)

	// Get colors from config
	colors := cfg.GetBeanColors(b.Status, b.Type, b.Priority)

	// Use shared RenderBeanRow function with responsive columns
	row := RenderBeanRow(b.ID, b.Status, b.Type, b.DisplayTitle(), BeanRowConfig{
		StatusColor:   colors.StatusColor,
		TypeColor:     colors.TypeColor,
		PriorityColor: colors.PriorityColor,
		Priority:      b.Priority,
		Size:          b.Size,
		SizeColor:     SizeColor(cfg, b.Size),
		ShowSize:      true,
		IsArchive:     colors.IsArchive,
		MaxTitleWidth: renderCfg.titleWidth,
		ShowCursor:    false,
		Tags:          b.Tags,
		ShowTags:      renderCfg.cols.ShowTags,
		TagsColWidth:  renderCfg.cols.Tags,
		MaxTags:       renderCfg.cols.MaxTags,
		TreePrefix:    prefix,
		Dimmed:        !node.Matched,
		TitleMuted:    !b.HasTitle(),
	})

	sb.WriteString(row)
	sb.WriteString("\n")
}

// treePrefix builds the connector prefix for a node (e.g. "│ └─").
// Roots (depth 0) get no prefix; ancestry holds whether each ancestor level was a last child.
func treePrefix(depth int, ancestry []bool, isLast bool) string {
	if depth == 0 {
		return ""
	}
	var prefix string
	for _, wasLast := range ancestry {
		if wasLast {
			prefix += treeSpace // parent was last child, no continuation line
		} else {
			prefix += treePipe // parent has more siblings, show continuation line
		}
	}
	if isLast {
		return prefix + treeLastBranch
	}
	return prefix + treeBranch
}

// FlatItem represents a flattened tree node with rendering context.
// Used by TUI to render tree structure in a flat list.
type FlatItem struct {
	Bean        *bean.Bean
	Depth       int    // 0 = root, 1+ = nested
	IsLast      bool   // last child at this level
	Matched     bool   // true if bean matched filter (vs. shown for context)
	TreePrefix  string // pre-computed tree prefix (e.g., "  └─")
	HasChildren bool   // true if this bean has children
}

// FlattenTreeWithCollapsed converts a tree into a flat slice, hiding children of collapsed beans.
// collapsedBeans is a set of bean IDs whose children should be hidden.
func FlattenTreeWithCollapsed(nodes []*TreeNode, collapsedBeans map[string]bool) []FlatItem {
	var items []FlatItem
	flattenNodes(nodes, 0, nil, collapsedBeans, &items)
	return items
}

// flattenNodes recursively flattens tree nodes.
// ancestry tracks whether each parent level was a last child (true = last, no continuation line needed)
// collapsedBeans (if not nil) is a set of bean IDs whose children should be hidden
func flattenNodes(nodes []*TreeNode, depth int, ancestry []bool, collapsedBeans map[string]bool, items *[]FlatItem) {
	for i, node := range nodes {
		isLast := i == len(nodes)-1

		prefix := treePrefix(depth, ancestry, isLast)
		hasChildren := len(node.Children) > 0

		*items = append(*items, FlatItem{
			Bean:        node.Bean,
			Depth:       depth,
			IsLast:      isLast,
			Matched:     node.Matched,
			TreePrefix:  prefix,
			HasChildren: hasChildren,
		})

		// Recurse into children, passing updated ancestry
		// Only add to ancestry when depth > 0 (roots have no connectors to continue)
		// Skip children if this bean is collapsed
		isCollapsed := collapsedBeans != nil && collapsedBeans[node.Bean.ID]
		if hasChildren && !isCollapsed {
			var newAncestry []bool
			if depth > 0 {
				newAncestry = append(ancestry, isLast)
			}
			flattenNodes(node.Children, depth+1, newAncestry, collapsedBeans, items)
		}
	}
}
