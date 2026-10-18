package tui

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
)

// depTreeLines renders tree rows as "prefix+id" with markers for tests.
func depTreeLines(rows []depRow) []string {
	var out []string
	for _, r := range rows {
		line := r.prefix + r.bean.ID
		switch {
		case r.cycle:
			line += " cycle"
		case r.repeat:
			line += " repeat"
		}
		out = append(out, line)
	}
	return out
}

func TestBuildDepTree(t *testing.T) {
	tests := []struct {
		name   string
		edges  map[string][]string
		hidden []string
		want   []string
	}{
		{
			name:  "no edges",
			edges: map[string][]string{},
			want:  nil,
		},
		{
			name:  "chain",
			edges: map[string][]string{"r": {"a"}, "a": {"b"}},
			want:  []string{"└─ a", "   └─ b"},
		},
		{
			name:  "siblings",
			edges: map[string][]string{"r": {"a", "b"}, "a": {"c"}},
			want:  []string{"├─ a", "│  └─ c", "└─ b"},
		},
		{
			name:  "diamond repeats without expanding",
			edges: map[string][]string{"r": {"a", "b"}, "a": {"c"}, "b": {"c"}, "c": {"d"}},
			want:  []string{"├─ a", "│  └─ c", "│     └─ d", "└─ b", "   └─ c repeat"},
		},
		{
			name:  "cycle back to root",
			edges: map[string][]string{"r": {"a"}, "a": {"r"}},
			want:  []string{"└─ a", "   └─ r cycle"},
		},
		{
			name:  "cycle below root",
			edges: map[string][]string{"r": {"a"}, "a": {"b"}, "b": {"a"}},
			want:  []string{"└─ a", "   └─ b", "      └─ a cycle"},
		},
		{
			name:   "hidden beans cut their subtree",
			edges:  map[string][]string{"r": {"a", "b"}, "b": {"c"}},
			hidden: []string{"b"},
			want:   []string{"└─ a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beans := map[string]*bean.Bean{}
			get := func(id string) *bean.Bean {
				if beans[id] == nil {
					beans[id] = &bean.Bean{ID: id}
				}
				return beans[id]
			}
			next := func(b *bean.Bean) []*bean.Bean {
				var out []*bean.Bean
				for _, id := range tt.edges[b.ID] {
					out = append(out, get(id))
				}
				return out
			}
			visible := func(b *bean.Bean) bool { return !slices.Contains(tt.hidden, b.ID) }

			got := depTreeLines(buildDepTree(get("r"), next, visible))
			if !slices.Equal(got, tt.want) {
				t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(tt.want, "\n"))
			}
		})
	}
}

// newDepNavTestApp creates beans where focus is blocked by blocker, blocker
// is blocked by the completed done bean, and focus blocks blocked.
func newDepNavTestApp(t *testing.T) (*App, map[string]*bean.Bean) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Default()
	cfg.SetConfigDir(dir)
	core := beancore.New(dir, cfg)
	core.SetWarnWriter(nil)

	beans := map[string]*bean.Bean{}
	for _, name := range []string{"focus", "blocker", "done", "blocked"} {
		b := &bean.Bean{Title: name, Status: "todo", Type: "task"}
		if name == "done" {
			b.Status = "completed"
		}
		if err := core.Create(b); err != nil {
			t.Fatal(err)
		}
		beans[name] = b
	}
	link := func(from, to string) {
		b := beans[from]
		b.AddBlocking(beans[to].ID)
		if err := core.Update(b); err != nil {
			t.Fatal(err)
		}
	}
	link("blocker", "focus")
	link("done", "blocker")
	link("focus", "blocked")

	app := New(core, cfg)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app.Update(app.list.loadBeans())
	return app, beans
}

// depNavIDs returns the bean IDs of the navigator's rows.
func depNavIDs(m depNavModel) []string {
	var ids []string
	for _, r := range m.rows {
		if r.bean != nil {
			ids = append(ids, r.bean.ID)
		}
	}
	return ids
}

func TestDepGraphPane(t *testing.T) {
	app, beans := newDepNavTestApp(t)
	id := func(name string) string { return beans[name].ID }

	pump(app, selectBeanMsg{bean: beans["focus"]})
	pump(app, keyMsg("+"))

	t.Run("+ on the context shows the graph rooted at its bean", func(t *testing.T) {
		if !app.ws.isVisible(paneDepGraph) {
			t.Fatalf("visible = %v, want the dependency graph", app.ws.visiblePanes())
		}
		want := []string{id("blocker"), id("focus"), id("blocked")}
		if got := depNavIDs(app.deps); !slices.Equal(got, want) {
			t.Errorf("rows = %v, want %v", got, want)
		}
		if b := app.deps.selectedBean(); b == nil || b.ID != id("focus") {
			t.Errorf("selected = %v, want focus", b)
		}
		view := app.View()
		for _, s := range []string{"blocked by", "blocks", id("blocker"), "Dependencies of " + id("focus")} {
			if !strings.Contains(view, s) {
				t.Errorf("view lacks %q", s)
			}
		}
	})

	pump(app, keyMsg("tab"))
	if app.ws.focus != paneDepGraph {
		t.Fatalf("focus = %v, want the dependency graph", app.ws.focus)
	}

	t.Run("A shows completed beans", func(t *testing.T) {
		pump(app, keyMsg("A"))
		want := []string{id("blocker"), id("done"), id("focus"), id("blocked")}
		if got := depNavIDs(app.deps); !slices.Equal(got, want) {
			t.Errorf("rows = %v, want %v", got, want)
		}
		pump(app, keyMsg("A"))
	})

	t.Run("cursor skips headings and drives the context", func(t *testing.T) {
		pump(app, keyMsg("k"))
		if b := app.deps.selectedBean(); b == nil || b.ID != id("blocker") {
			t.Fatalf("selected = %v, want blocker", b)
		}
		if app.ctx.bean.ID != id("blocker") {
			t.Errorf("context = %s, want blocker", app.ctx.bean.ID)
		}
		pump(app, keyMsg("k"))
		if b := app.deps.selectedBean(); b == nil || b.ID != id("blocker") {
			t.Errorf("selected = %v, want blocker to stay", b)
		}
	})

	t.Run("enter refocuses on the selected bean", func(t *testing.T) {
		pump(app, keyMsg("enter"))
		if app.deps.root.ID != id("blocker") {
			t.Fatalf("root = %s, want blocker", app.deps.root.ID)
		}
		want := []string{id("blocker"), id("focus"), id("blocked")}
		if got := depNavIDs(app.deps); !slices.Equal(got, want) {
			t.Errorf("rows = %v, want %v", got, want)
		}
	})

	t.Run("esc returns to the context, which keeps the bean", func(t *testing.T) {
		pump(app, keyMsg("esc"))
		if app.ws.focus != paneContext {
			t.Errorf("focus = %v, want the context", app.ws.focus)
		}
		if app.ctx.bean.ID != id("blocker") {
			t.Errorf("context = %s, want blocker", app.ctx.bean.ID)
		}
	})

	t.Run("D hides the graph", func(t *testing.T) {
		pump(app, keyMsg("D"))
		if app.ws.isVisible(paneDepGraph) {
			t.Errorf("visible = %v, want no dependency graph", app.ws.visiblePanes())
		}
	})

	t.Run("D in the list roots the graph at the list's bean", func(t *testing.T) {
		pump(app, keyMsg("esc"))
		if app.ws.focus != paneList {
			t.Fatalf("focus = %v, want the list", app.ws.focus)
		}
		pump(app, keyMsg("D"))
		selected, _ := app.list.previewData()
		if app.deps.root == nil || app.deps.root.ID != selected.ID {
			t.Errorf("root = %v, want the list's bean %s", app.deps.root, selected.ID)
		}
	})
}
