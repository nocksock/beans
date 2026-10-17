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

func TestDepNav(t *testing.T) {
	app, beans := newDepNavTestApp(t)
	id := func(name string) string { return beans[name].ID }

	t.Run("D in list opens the navigator", func(t *testing.T) {
		_, cmd := app.Update(keyMsg("D"))
		if cmd == nil {
			t.Fatal("expected command from D key")
		}
		if _, ok := cmd().(openDepNavMsg); !ok {
			t.Fatalf("got %#v, want openDepNavMsg", cmd())
		}
	})

	app.Update(openDepNavMsg{bean: beans["focus"]})

	t.Run("shows open dependencies with cursor on focus", func(t *testing.T) {
		if app.state != viewDepNav {
			t.Fatalf("state = %v, want viewDepNav", app.state)
		}
		want := []string{id("blocker"), id("focus"), id("blocked")}
		if got := depNavIDs(app.depNav); !slices.Equal(got, want) {
			t.Errorf("rows = %v, want %v", got, want)
		}
		if b := app.depNav.selectedBean(); b == nil || b.ID != id("focus") {
			t.Errorf("selected = %v, want focus", b)
		}
		view := app.View()
		for _, s := range []string{"blocked by", "blocks", id("blocker")} {
			if !strings.Contains(view, s) {
				t.Errorf("view lacks %q", s)
			}
		}
	})

	t.Run("A shows completed beans", func(t *testing.T) {
		app.Update(keyMsg("A"))
		want := []string{id("blocker"), id("done"), id("focus"), id("blocked")}
		if got := depNavIDs(app.depNav); !slices.Equal(got, want) {
			t.Errorf("rows = %v, want %v", got, want)
		}
		app.Update(keyMsg("A"))
	})

	t.Run("cursor skips headings", func(t *testing.T) {
		app.Update(keyMsg("k"))
		if b := app.depNav.selectedBean(); b == nil || b.ID != id("blocker") {
			t.Fatalf("selected = %v, want blocker", b)
		}
		app.Update(keyMsg("k"))
		if b := app.depNav.selectedBean(); b == nil || b.ID != id("blocker") {
			t.Errorf("selected = %v, want blocker to stay", b)
		}
	})

	t.Run("enter opens detail and back returns to navigator", func(t *testing.T) {
		_, cmd := app.Update(keyMsg("enter"))
		app.Update(cmd())
		if app.state != viewDetail || app.detail.bean.ID != id("blocker") {
			t.Fatalf("state = %v, detail = %s; want detail of blocker", app.state, app.detail.bean.ID)
		}
		app.Update(backToListMsg{})
		if app.state != viewDepNav {
			t.Fatalf("state = %v, want viewDepNav", app.state)
		}
		if b := app.depNav.selectedBean(); b == nil || b.ID != id("blocker") {
			t.Errorf("selected = %v, want blocker", b)
		}
	})

	t.Run("D refocuses on the selected bean", func(t *testing.T) {
		app.Update(keyMsg("D"))
		if app.depNav.root.ID != id("blocker") {
			t.Fatalf("root = %s, want blocker", app.depNav.root.ID)
		}
		want := []string{id("blocker"), id("focus"), id("blocked")}
		if got := depNavIDs(app.depNav); !slices.Equal(got, want) {
			t.Errorf("rows = %v, want %v", got, want)
		}
	})

	t.Run("esc returns to the list", func(t *testing.T) {
		_, cmd := app.Update(keyMsg("esc"))
		app.Update(cmd())
		if app.state != viewList {
			t.Errorf("state = %v, want viewList", app.state)
		}
	})

	t.Run("opened from detail, esc restores that detail", func(t *testing.T) {
		app.Update(selectBeanMsg{bean: beans["blocked"]})
		app.Update(openDepNavMsg{bean: beans["blocked"]})
		app.Update(keyMsg("k"))
		_, cmd := app.Update(keyMsg("enter"))
		app.Update(cmd())
		app.Update(backToListMsg{})
		_, cmd = app.Update(keyMsg("esc"))
		app.Update(cmd())
		if app.state != viewDetail || app.detail.bean.ID != id("blocked") {
			t.Errorf("state = %v, detail = %s; want detail of blocked", app.state, app.detail.bean.ID)
		}
	})
}
