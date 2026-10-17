package cmd

import (
	"reflect"
	"testing"

	"github.com/hmans/beans/internal/bean"
)

func graphTestBeans() []*bean.Bean {
	return []*bean.Bean{
		{ID: "a", Title: "Epic A", Status: "todo"},
		{ID: "b", Title: "Task B", Status: "in-progress", Parent: "a", Blocking: []string{"c", "e"}},
		{ID: "c", Title: "Task C", Status: "todo", Parent: "a"},
		{ID: "d", Title: "Lonely", Status: "draft", Blocking: []string{"missing"}},
		// e is blocked by c (declared on e) and by b (declared both ways, deduplicated)
		{ID: "e", Title: "Task E", Status: "todo", BlockedBy: []string{"c", "b", "missing"}},
	}
}

func edgeKeys(edges []graphEdge) []string {
	var keys []string
	for _, e := range edges {
		keys = append(keys, e.From+" "+e.Type+" "+e.To)
	}
	return keys
}

func nodeIDs(nodes []*bean.Bean) []string {
	var ids []string
	for _, n := range nodes {
		ids = append(ids, n.ID)
	}
	return ids
}

func TestBuildGraph(t *testing.T) {
	tests := []struct {
		name      string
		types     []string
		focus     []string
		wantEdges []string
		wantNodes []string
	}{
		{
			name:      "all relationships",
			wantEdges: []string{"b blocking c", "b blocking e", "b parent a", "c blocking e", "c parent a"},
			wantNodes: []string{"a", "b", "c", "e"},
		},
		{
			name:      "only blocking",
			types:     []string{relBlocking},
			wantEdges: []string{"b blocking c", "b blocking e", "c blocking e"},
			wantNodes: []string{"b", "c", "e"},
		},
		{
			name:      "only parent",
			types:     []string{relParent},
			wantEdges: []string{"b parent a", "c parent a"},
			wantNodes: []string{"a", "b", "c"},
		},
		{
			name:      "focus on bean",
			focus:     []string{"c"},
			wantEdges: []string{"b blocking c", "c blocking e", "c parent a"},
			wantNodes: []string{"a", "b", "c", "e"},
		},
		{
			name:      "focus on isolated bean keeps node",
			focus:     []string{"d"},
			wantEdges: nil,
			wantNodes: []string{"d"},
		},
		{
			name:      "focus and type combined",
			types:     []string{relParent},
			focus:     []string{"b"},
			wantEdges: []string{"b parent a"},
			wantNodes: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := buildGraph(graphTestBeans(), tt.types, tt.focus)
			if got := edgeKeys(g.Edges); !reflect.DeepEqual(got, tt.wantEdges) {
				t.Errorf("edges = %v, want %v", got, tt.wantEdges)
			}
			if got := nodeIDs(g.Nodes); !reflect.DeepEqual(got, tt.wantNodes) {
				t.Errorf("nodes = %v, want %v", got, tt.wantNodes)
			}
		})
	}
}

func TestRenderGraphDOT(t *testing.T) {
	g := buildGraph(graphTestBeans()[:3], []string{relBlocking}, nil)
	colors := map[string]string{"in-progress": "yellow"}
	got := renderGraphDOT(g, func(s string) string { return colors[s] })

	want := `digraph beans {
  node [shape=box];
  "b" [label="b\nTask B", color="yellow"];
  "c" [label="c\nTask C"];
  "b" -> "c" [label="blocking"];
}
`
	if got != want {
		t.Errorf("renderGraphDOT() =\n%s\nwant\n%s", got, want)
	}
}

func TestRenderGraphDOTEscapesTitles(t *testing.T) {
	g := beanGraph{Nodes: []*bean.Bean{{ID: "x", Title: `Say "hi"`}}}
	got := renderGraphDOT(g, func(string) string { return "" })
	want := "digraph beans {\n  node [shape=box];\n  \"x\" [label=\"x\\nSay \\\"hi\\\"\"];\n}\n"
	if got != want {
		t.Errorf("renderGraphDOT() = %q, want %q", got, want)
	}
}

func TestRenderGraphASCII(t *testing.T) {
	g := buildGraph(graphTestBeans(), nil, nil)
	got := renderGraphASCII(g)
	want := "b ──blocking──> c\nb ──blocking──> e\nb ──parent──> a\nc ──blocking──> e\nc ──parent──> a\n"
	if got != want {
		t.Errorf("renderGraphASCII() =\n%s\nwant\n%s", got, want)
	}
}
