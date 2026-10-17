package cmd

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/graph"
	"github.com/spf13/cobra"
)

// Relationship types that can be visualized.
const (
	relParent   = "parent"
	relBlocking = "blocking"
)

var validRelTypes = []string{relParent, relBlocking}

var (
	graphASCII bool
	graphTypes []string
)

// graphEdge is a directed relationship between two beans.
type graphEdge struct {
	From string
	To   string
	Type string
}

// beanGraph holds the edges and the beans they reference.
type beanGraph struct {
	Nodes []*bean.Bean
	Edges []graphEdge
}

var graphCmd = &cobra.Command{
	Use:   "graph [id...]",
	Short: "Visualize bean relationships as a graph",
	Long: `Outputs bean relationships (parent, blocking) as a graph.

By default the output is in DOT format, which can be rendered with Graphviz:

  beans graph | dot -Tpng -o beans.png

Pass bean IDs to only show relationships involving those beans.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, t := range graphTypes {
			if !slices.Contains(validRelTypes, t) {
				return fmt.Errorf("invalid relationship type %q (must be one of: %s)", t, strings.Join(validRelTypes, ", "))
			}
		}

		resolver := &graph.Resolver{Core: core}
		ctx := context.Background()

		var focus []string
		for _, id := range args {
			b, err := resolver.Query().Bean(ctx, id)
			if err != nil {
				return handleBeanQueryError(err, false, id)
			}
			if b == nil {
				return fmt.Errorf("bean not found: %s", id)
			}
			focus = append(focus, b.ID)
		}

		allBeans, err := resolver.Query().Beans(ctx, nil)
		if err != nil {
			return fmt.Errorf("querying beans: %w", err)
		}

		g := buildGraph(allBeans, graphTypes, focus)
		if graphASCII {
			fmt.Fprint(cmd.OutOrStdout(), renderGraphASCII(g))
		} else {
			fmt.Fprint(cmd.OutOrStdout(), renderGraphDOT(g, statusColor))
		}
		return nil
	},
}

// statusColor returns the configured color for a status.
func statusColor(status string) string {
	if s := cfg.GetStatus(status); s != nil {
		return s.Color
	}
	return ""
}

// buildGraph collects relationship edges between beans. Edges are limited to the
// given relationship types (all if empty) and, if focus is non-empty, to edges
// touching one of the focused bean IDs. Edges pointing to unknown beans are skipped.
func buildGraph(allBeans []*bean.Bean, types []string, focus []string) beanGraph {
	byID := make(map[string]*bean.Bean, len(allBeans))
	for _, b := range allBeans {
		byID[b.ID] = b
	}

	include := func(t string) bool {
		return len(types) == 0 || slices.Contains(types, t)
	}

	var edges []graphEdge
	seenEdges := make(map[graphEdge]bool)
	addEdge := func(from, to, typ string) {
		if _, ok := byID[from]; !ok {
			return
		}
		if _, ok := byID[to]; !ok {
			return
		}
		if len(focus) > 0 && !slices.Contains(focus, from) && !slices.Contains(focus, to) {
			return
		}
		e := graphEdge{From: from, To: to, Type: typ}
		if !seenEdges[e] {
			seenEdges[e] = true
			edges = append(edges, e)
		}
	}

	for _, b := range allBeans {
		if b.Parent != "" && include(relParent) {
			addEdge(b.ID, b.Parent, relParent)
		}
		if include(relBlocking) {
			for _, target := range b.Blocking {
				addEdge(b.ID, target, relBlocking)
			}
			// blocked_by is the inverse declaration: blocker -> b
			for _, blocker := range b.BlockedBy {
				addEdge(blocker, b.ID, relBlocking)
			}
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].Type != edges[j].Type {
			return edges[i].Type < edges[j].Type
		}
		return edges[i].To < edges[j].To
	})

	// Nodes: every bean referenced by an edge, plus focused beans (even if isolated)
	seen := make(map[string]bool)
	var nodes []*bean.Bean
	addNode := func(id string) {
		if !seen[id] {
			seen[id] = true
			nodes = append(nodes, byID[id])
		}
	}
	for _, id := range focus {
		addNode(id)
	}
	for _, e := range edges {
		addNode(e.From)
		addNode(e.To)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

	return beanGraph{Nodes: nodes, Edges: edges}
}

// renderGraphDOT renders the graph in Graphviz DOT format. Node borders are
// colored using colorFor(status) when it returns a non-empty color.
func renderGraphDOT(g beanGraph, colorFor func(status string) string) string {
	var sb strings.Builder
	sb.WriteString("digraph beans {\n")
	sb.WriteString("  node [shape=box];\n")
	for _, n := range g.Nodes {
		attrs := []string{"label=" + strconv.Quote(n.ID+"\n"+n.DisplayTitle())}
		if c := colorFor(n.Status); c != "" {
			attrs = append(attrs, "color="+strconv.Quote(c))
		}
		fmt.Fprintf(&sb, "  %s [%s];\n", strconv.Quote(n.ID), strings.Join(attrs, ", "))
	}
	for _, e := range g.Edges {
		fmt.Fprintf(&sb, "  %s -> %s [label=%s];\n", strconv.Quote(e.From), strconv.Quote(e.To), strconv.Quote(e.Type))
	}
	sb.WriteString("}\n")
	return sb.String()
}

// renderGraphASCII renders one line per edge, e.g. "beans-abc ──blocking──> beans-def".
func renderGraphASCII(g beanGraph) string {
	var sb strings.Builder
	for _, e := range g.Edges {
		fmt.Fprintf(&sb, "%s ──%s──> %s\n", e.From, e.Type, e.To)
	}
	return sb.String()
}

func init() {
	graphCmd.Flags().BoolVar(&graphASCII, "ascii", false, "Output as plain text instead of DOT")
	graphCmd.Flags().StringArrayVar(&graphTypes, "type", nil, "Only show relationships of this type: parent, blocking (can be repeated)")
	rootCmd.AddCommand(graphCmd)
}
