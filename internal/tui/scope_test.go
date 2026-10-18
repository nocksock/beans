package tui

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
)

// newScopedApp builds a TUI for packages/api in a cascading monorepo, with
// beans of several scopes in the shared beans directory.
func newScopedApp(t *testing.T) *App {
	t.Helper()
	return newMonorepoApp(t, "packages/api")
}

// newMonorepoApp builds a TUI for dir in a cascading monorepo.
func newMonorepoApp(t *testing.T, dir string) *App {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	files := map[string]string{
		".beans.yml":              "cascade: true\nbeans:\n  path: .beans\n  prefix: repo-\n",
		"packages/api/.beans.yml": "beans:\n  prefix: api-\n",
	}
	for rel, content := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := config.LoadFromDirectory(filepath.Join(root, dir))
	if err != nil {
		t.Fatal(err)
	}

	core := beancore.New(cfg.ResolveBeansPath(), cfg)
	core.SetWarnWriter(nil)
	for _, b := range []*bean.Bean{
		{ID: "repo-a1b2", Title: "Milestone", Status: "todo", Type: "milestone"},
		{ID: "api-c3d4", Title: "API task", Status: "todo", Type: "task", Parent: "repo-a1b2"},
		{ID: "web-g7h8", Title: "Web task", Status: "todo", Type: "task"},
	} {
		if err := core.Create(b); err != nil {
			t.Fatal(err)
		}
	}

	app := New(core, cfg)
	app.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	app.Update(app.list.loadBeans())
	return app
}

// dimmedIDs returns the listed beans shown only as context for a match.
func dimmedIDs(app *App) []string {
	var ids []string
	for _, it := range app.list.list.Items() {
		if item := it.(beanItem); !item.matched {
			ids = append(ids, item.bean.ID)
		}
	}
	return ids
}

func TestListScope(t *testing.T) {
	app := newScopedApp(t)

	if got, want := listedIDs(app), []string{"repo-a1b2", "api-c3d4"}; !slices.Equal(got, want) {
		t.Errorf("scoped list = %v, want %v", got, want)
	}
	if got, want := dimmedIDs(app), []string{"repo-a1b2"}; !slices.Equal(got, want) {
		t.Errorf("dimmed = %v, want out-of-scope parent %v", got, want)
	}
	if title := app.list.breadcrumbTitle(80); !strings.Contains(title, "[api-*]") {
		t.Errorf("title %q should show the scope", title)
	}

	press(app, "g")
	press(app, "s")
	got := listedIDs(app)
	slices.Sort(got)
	if want := []string{"api-c3d4", "repo-a1b2", "web-g7h8"}; !slices.Equal(got, want) {
		t.Errorf("list with all scopes = %v, want %v", got, want)
	}
	if d := dimmedIDs(app); len(d) != 0 {
		t.Errorf("dimmed with all scopes = %v, want none", d)
	}
	if title := app.list.breadcrumbTitle(80); !strings.Contains(title, "[all scopes]") {
		t.Errorf("title %q should show all scopes", title)
	}
}

func TestListScopeUnscopedConfig(t *testing.T) {
	app, _ := newTestApp(t)
	before := len(listedIDs(app))

	press(app, "g")
	press(app, "s")
	if app.list.showAllScope {
		t.Error("toggle should do nothing without a scope")
	}
	if got := len(listedIDs(app)); got != before {
		t.Errorf("listed %d beans, want %d", got, before)
	}
	if title := app.list.breadcrumbTitle(80); title != "Beans" {
		t.Errorf("title = %q, want no scope indicator", title)
	}
}

func TestCreateModalPrefix(t *testing.T) {
	multi := func() createModalModel {
		m := newTestCreateModal()
		m.prefixes = []string{"repo-", "api-", "web-"}
		return m
	}

	tests := []struct {
		name      string
		modal     func() createModalModel
		keys      []tea.KeyMsg
		wantFocus createField
		want      string
	}{
		{"single prefix is skipped", newTestCreateModal, []tea.KeyMsg{keyMsg("tab"), keyMsg("tab")}, createFieldType, ""},
		{"several prefixes are focusable", multi, []tea.KeyMsg{keyMsg("tab"), keyMsg("tab")}, createFieldPrefix, "repo-"},
		{"cycle to next prefix", multi, []tea.KeyMsg{keyMsg("tab"), keyMsg("tab"), keyMsg("right")}, createFieldPrefix, "api-"},
		{"cycle wraps around", multi, []tea.KeyMsg{keyMsg("tab"), keyMsg("tab"), keyMsg("left")}, createFieldPrefix, "web-"},
		{"shift+tab from type skips single prefix", newTestCreateModal, []tea.KeyMsg{keyMsg("tab"), keyMsg("tab"), shiftTab, shiftTab}, createFieldTitle, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := pressCreate(typeIntoCreate(tt.modal(), "New"), tt.keys...)
			if m.focus != tt.wantFocus {
				t.Errorf("focus = %v, want %v", m.focus, tt.wantFocus)
			}
			_, cmd := pressCreate(m, keyMsg("enter"))
			got, ok := cmd().(beanCreatedMsg)
			if !ok {
				t.Fatal("enter should create the bean")
			}
			if got.prefix != tt.want {
				t.Errorf("prefix = %q, want %q", got.prefix, tt.want)
			}
		})
	}
}

func TestCreateModalPrefixView(t *testing.T) {
	single := newTestCreateModal()
	multi := newTestCreateModal()
	multi.prefixes = []string{"repo-", "api-"}
	multi.setFocus(createFieldPrefix)

	if v := single.View(); !strings.Contains(v, "Prefix") || !strings.Contains(v, "(none)") {
		t.Errorf("single prefix should render fixed:\n%s", v)
	}
	if v := multi.View(); !strings.Contains(v, "Prefix") || !strings.Contains(v, "‹ repo- ›") {
		t.Errorf("several prefixes should render a selector:\n%s", v)
	}
	if lipgloss.Height(single.View()) != lipgloss.Height(multi.View()) {
		t.Error("layout height should not depend on the number of prefixes")
	}
	edit := newEditModalModel(config.Default(), 120, 40, &bean.Bean{ID: "x-1", Title: "T"})
	if strings.Contains(edit.View(), "Prefix") {
		t.Error("edit form should not show the prefix")
	}
}

func TestCreateWithPrefix(t *testing.T) {
	app := newMonorepoApp(t, ".")
	app.Update(press(app, "c"))
	if got, want := app.createModal.prefixes, []string{"repo-", "api-"}; !slices.Equal(got, want) {
		t.Fatalf("prefixes = %v, want %v", got, want)
	}

	for _, r := range "From root" {
		app.Update(keyMsg(string(r)))
	}
	app.Update(keyMsg("tab"))
	app.Update(keyMsg("tab"))
	app.Update(keyMsg("right"))
	_, cmd := app.Update(keyMsg("enter"))
	app.Update(cmd())

	id := app.list.cursorTarget
	if !strings.HasPrefix(id, "api-") {
		t.Errorf("created bean %q, want api- prefix", id)
	}
}
