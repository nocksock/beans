package layout

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func show(ids ...ID) func(ID) bool {
	return func(id ID) bool { return slices.Contains(ids, id) }
}

// demoTree mirrors the TUI's default layout.
var demoTree = VSplit(
	HSplit(
		Leaf("list").Min(40),
		VSplit(
			Leaf("deps").Min(5),
			Leaf("context").Weight(2).Min(5),
		).Min(80).Max(80).Drop(1),
	).Min(10),
	Leaf("log").Fixed(8).Drop(1),
)

func TestArrange(t *testing.T) {
	tests := []struct {
		name    string
		root    Node
		visible []ID
		focus   ID
		w, h    int
		want    []Rect
	}{
		{
			name:    "single leaf fills the screen",
			root:    Leaf("a"),
			visible: []ID{"a"},
			w:       30, h: 10,
			want: []Rect{{"a", 0, 0, 30, 10}},
		},
		{
			name:    "weights share the space",
			root:    HSplit(Leaf("a"), Leaf("b").Weight(3)),
			visible: []ID{"a", "b"},
			w:       40, h: 10,
			want: []Rect{{"a", 0, 0, 10, 10}, {"b", 10, 0, 30, 10}},
		},
		{
			name:    "rounding remainder goes to the last child",
			root:    HSplit(Leaf("a"), Leaf("b"), Leaf("c")),
			visible: []ID{"a", "b", "c"},
			w:       10, h: 5,
			want: []Rect{{"a", 0, 0, 3, 5}, {"b", 3, 0, 3, 5}, {"c", 6, 0, 4, 5}},
		},
		{
			name:    "fixed child keeps its size",
			root:    VSplit(Leaf("a"), Leaf("log").Fixed(8)),
			visible: []ID{"a", "log"},
			w:       20, h: 30,
			want: []Rect{{"a", 0, 0, 20, 22}, {"log", 0, 22, 20, 8}},
		},
		{
			name:    "max caps a child, its sibling takes the rest",
			root:    HSplit(Leaf("a"), Leaf("b").Max(80)),
			visible: []ID{"a", "b"},
			w:       200, h: 10,
			want: []Rect{{"a", 0, 0, 120, 10}, {"b", 120, 0, 80, 10}},
		},
		{
			name:    "min grows a child at the expense of its sibling",
			root:    HSplit(Leaf("a"), Leaf("b").Min(80)),
			visible: []ID{"a", "b"},
			w:       120, h: 10,
			want: []Rect{{"a", 0, 0, 40, 10}, {"b", 40, 0, 80, 10}},
		},
		{
			name:    "max does not apply to an only child",
			root:    HSplit(Leaf("a"), Leaf("b").Max(80)),
			visible: []ID{"b"},
			w:       200, h: 10,
			want: []Rect{{"b", 0, 0, 200, 10}},
		},
		{
			name:    "hidden leaf collapses",
			root:    VSplit(Leaf("a"), Leaf("log").Fixed(8)),
			visible: []ID{"a"},
			w:       20, h: 30,
			want: []Rect{{"a", 0, 0, 20, 30}},
		},
		{
			name:    "default layout, wide",
			root:    demoTree,
			visible: []ID{"list", "context", "log"},
			focus:   "list",
			w:       200, h: 40,
			want: []Rect{{"list", 0, 0, 120, 32}, {"context", 120, 0, 80, 32}, {"log", 0, 32, 200, 8}},
		},
		{
			name:    "default layout, narrow: unfocused context is dropped",
			root:    demoTree,
			visible: []ID{"list", "context", "log"},
			focus:   "list",
			w:       100, h: 40,
			want: []Rect{{"list", 0, 0, 100, 32}, {"log", 0, 32, 100, 8}},
		},
		{
			name:    "default layout, narrow: focused context drops the list",
			root:    demoTree,
			visible: []ID{"list", "context", "log"},
			focus:   "context",
			w:       100, h: 40,
			want: []Rect{{"context", 0, 0, 100, 32}, {"log", 0, 32, 100, 8}},
		},
		{
			name:    "default layout, short: log is dropped",
			root:    demoTree,
			visible: []ID{"list", "context", "log"},
			focus:   "list",
			w:       200, h: 17,
			want: []Rect{{"list", 0, 0, 120, 17}, {"context", 120, 0, 80, 17}},
		},
		{
			name:    "dependency graph above the context, list hidden",
			root:    demoTree,
			visible: []ID{"deps", "context", "log"},
			focus:   "context",
			w:       100, h: 38,
			want: []Rect{{"deps", 0, 0, 100, 10}, {"context", 0, 10, 100, 20}, {"log", 0, 30, 100, 8}},
		},
		{
			name:    "nothing visible",
			root:    demoTree,
			visible: nil,
			w:       100, h: 40,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Arrange(tt.root, show(tt.visible...), tt.focus, tt.w, tt.h).Rects()
			if !slices.Equal(got, tt.want) {
				t.Errorf("rects\n got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestRenderSize(t *testing.T) {
	for _, size := range [][2]int{{200, 40}, {100, 40}, {60, 12}} {
		l := Arrange(demoTree, show("list", "context", "log"), "list", size[0], size[1])
		out := l.Render(func(r Rect) string {
			return Frame(string(r.ID), "hello\nworld", r.W, r.H, lipgloss.NewStyle())
		})
		lines := strings.Split(out, "\n")
		if len(lines) != size[1] {
			t.Errorf("%v: %d lines, want %d", size, len(lines), size[1])
		}
		for i, line := range lines {
			if w := lipgloss.Width(line); w != size[0] {
				t.Errorf("%v: line %d is %d wide, want %d: %q", size, i, w, size[0], line)
				break
			}
		}
	}
}

func TestFrame(t *testing.T) {
	out := Frame("Title", "a long line that gets clipped\nb", 12, 4, lipgloss.NewStyle())
	want := strings.Join([]string{
		"╭─ Title ──╮",
		"│a long lin│",
		"│b         │",
		"╰──────────╯",
	}, "\n")
	if out != want {
		t.Errorf("frame\n%s\nwant\n%s", out, want)
	}
}

func TestRect(t *testing.T) {
	l := Arrange(demoTree, show("list", "log"), "list", 100, 40)
	if _, ok := l.Rect("context"); ok {
		t.Error("hidden pane should have no rect")
	}
	if r, ok := l.Rect("log"); !ok || r.H != 8 {
		t.Errorf("log rect = %v, %v", r, ok)
	}
}
