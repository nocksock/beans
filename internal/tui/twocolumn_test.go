package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
)

// hierarchy of beans used by the navigation tests:
//
//	Milestone
//	└─ Epic
//	   └─ Task
//	Loose
type testHierarchy struct {
	milestone, epic, task, loose *bean.Bean
}

func newHierarchyApp(t *testing.T, width int) (*App, *beancore.Core, testHierarchy) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Default()
	cfg.SetConfigDir(dir)
	core := beancore.New(dir, cfg)
	core.SetWarnWriter(nil)

	h := testHierarchy{
		milestone: &bean.Bean{Title: "Milestone", Status: "todo", Type: "milestone"},
		loose:     &bean.Bean{Title: "Loose", Status: "todo", Type: "task", Body: "Loose body"},
	}
	for _, b := range []*bean.Bean{h.milestone, h.loose} {
		if err := core.Create(b); err != nil {
			t.Fatal(err)
		}
	}
	h.epic = &bean.Bean{Title: "Epic", Status: "todo", Type: "epic", Parent: h.milestone.ID}
	if err := core.Create(h.epic); err != nil {
		t.Fatal(err)
	}
	h.task = &bean.Bean{Title: "Task", Status: "todo", Type: "task", Parent: h.epic.ID, Body: "Task body"}
	if err := core.Create(h.task); err != nil {
		t.Fatal(err)
	}

	app := New(core, cfg)
	app.Update(tea.WindowSizeMsg{Width: width, Height: 30})
	app.Update(app.list.loadBeans())
	return app, core, h
}

// press sends a key and feeds the resulting command's message back into the app
func press(app *App, key string) tea.Msg {
	_, cmd := app.Update(keyMsg(key))
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if _, ok := msg.(beansLoadedMsg); ok {
		app.Update(msg)
	}
	return msg
}

func listedIDs(app *App) []string {
	var ids []string
	for _, it := range app.list.list.Items() {
		ids = append(ids, it.(beanItem).bean.ID)
	}
	return ids
}

func selectID(t *testing.T, app *App, id string) {
	t.Helper()
	for i, it := range app.list.list.Items() {
		if it.(beanItem).bean.ID == id {
			app.list.list.Select(i)
			app.syncPanes()
			return
		}
	}
	t.Fatalf("bean %s not in list %v", id, listedIDs(app))
}

func selectedID(app *App) string {
	if item, ok := app.list.list.SelectedItem().(beanItem); ok {
		return item.bean.ID
	}
	return ""
}

func TestDrillDownNavigation(t *testing.T) {
	app, _, h := newHierarchyApp(t, 160)

	if got := len(listedIDs(app)); got != 4 {
		t.Fatalf("top level should list all 4 beans, got %v", listedIDs(app))
	}

	// Enter on a bean with children drills into it
	selectID(t, app, h.milestone.ID)
	press(app, "enter")
	if got, want := strings.Join(listedIDs(app), ","), h.epic.ID+","+h.task.ID; got != want {
		t.Errorf("after drilling into milestone, listed = %s, want %s", got, want)
	}
	if !strings.Contains(app.View(), "Beans › Milestone") {
		t.Error("breadcrumb should show the milestone")
	}

	// Drill one level deeper
	selectID(t, app, h.epic.ID)
	press(app, "enter")
	if got := strings.Join(listedIDs(app), ","); got != h.task.ID {
		t.Errorf("after drilling into epic, listed = %s, want %s", got, h.task.ID)
	}
	if !strings.Contains(app.View(), "Beans › Milestone › Epic") {
		t.Error("breadcrumb should show milestone and epic")
	}

	// Enter on a leaf shows it in the context pane instead
	if msg, ok := press(app, "enter").(selectBeanMsg); !ok || msg.bean.ID != h.task.ID {
		t.Errorf("enter on leaf should select it, got %#v", msg)
	}

	// esc goes back up, restoring the cursor to the bean we came from
	press(app, "esc")
	if got := selectedID(app); got != h.epic.ID {
		t.Errorf("after esc, cursor on %s, want %s", got, h.epic.ID)
	}
	press(app, "backspace")
	if got := len(listedIDs(app)); got != 4 {
		t.Errorf("after backing out to the top, listed %v", listedIDs(app))
	}
	if got := selectedID(app); got != h.milestone.ID {
		t.Errorf("after backspace, cursor on %s, want %s", got, h.milestone.ID)
	}
}

func TestTabFocusesContextOfBeanWithChildren(t *testing.T) {
	app, _, h := newHierarchyApp(t, 160)
	selectID(t, app, h.milestone.ID)
	press(app, "tab")
	if app.ws.focus != paneContext || app.ctx.bean.ID != h.milestone.ID {
		t.Errorf("tab should focus the context showing the milestone, focus = %v", app.ws.focus)
	}
}

func TestDrillDownEscClearsSelectionFirst(t *testing.T) {
	app, _, h := newHierarchyApp(t, 160)
	selectID(t, app, h.milestone.ID)
	press(app, "enter")
	press(app, "space")
	press(app, "esc")
	if len(app.list.selectedBeans) != 0 {
		t.Error("esc should clear the selection")
	}
	if len(app.list.rootStack) != 1 {
		t.Error("esc should not leave the level while beans were selected")
	}
}

func TestDrillDownRootDeleted(t *testing.T) {
	app, core, h := newHierarchyApp(t, 160)
	selectID(t, app, h.milestone.ID)
	press(app, "enter")
	selectID(t, app, h.epic.ID)
	press(app, "enter")

	if err := core.Delete(h.epic.ID); err != nil {
		t.Fatal(err)
	}
	app.Update(app.list.loadBeans())

	if got := len(app.list.rootStack); got != 1 {
		t.Errorf("root stack should fall back to the milestone, got depth %d", got)
	}
}

func TestTwoColumnView(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		wantPreview bool
	}{
		{"narrow terminal is single column", 100, false},
		{"just too narrow", 119, false},
		{"threshold", 120, true},
		{"wide terminal", 200, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _, h := newHierarchyApp(t, tt.width)
			selectID(t, app, h.loose.ID)
			view := app.View()

			if got := strings.Contains(ansi.Strip(view), "Loose body"); got != tt.wantPreview {
				t.Errorf("preview shown = %v, want %v", got, tt.wantPreview)
			}

			lines := strings.Split(view, "\n")
			if got, want := len(lines), app.height; got != want {
				t.Errorf("view has %d lines, want %d", got, want)
			}
			for i, line := range lines {
				if w := lipgloss.Width(line); w > tt.width {
					t.Errorf("line %d is %d wide, terminal is %d", i, w, tt.width)
				}
			}
		})
	}
}

func TestTwoColumnPreviewFollowsCursor(t *testing.T) {
	app, _, h := newHierarchyApp(t, 160)
	selectID(t, app, h.loose.ID)
	if !strings.Contains(ansi.Strip(app.View()), "Loose body") {
		t.Fatal("preview should show the highlighted bean")
	}
	selectID(t, app, h.task.ID)
	view := ansi.Strip(app.View())
	if !strings.Contains(view, "Task body") || strings.Contains(view, "Loose body") {
		t.Error("preview should follow the cursor")
	}
}

func TestDescendantsOf(t *testing.T) {
	beans := []*bean.Bean{
		{ID: "root"},
		{ID: "child", Parent: "root"},
		{ID: "grandchild", Parent: "child"},
		{ID: "other"},
		{ID: "orphan", Parent: "missing"},
		{ID: "cycle-a", Parent: "cycle-b"},
		{ID: "cycle-b", Parent: "cycle-a"},
	}
	byID := map[string]*bean.Bean{}
	for _, b := range beans {
		byID[b.ID] = b
	}

	tests := []struct {
		root string
		want string
	}{
		{"root", "child,grandchild"},
		{"child", "grandchild"},
		{"grandchild", ""},
		{"other", ""},
		{"cycle-a", "cycle-b"},
	}
	for _, tt := range tests {
		t.Run(tt.root, func(t *testing.T) {
			var ids []string
			for _, b := range descendantsOf(tt.root, beans, byID) {
				ids = append(ids, b.ID)
			}
			if got := strings.Join(ids, ","); got != tt.want {
				t.Errorf("descendantsOf(%s) = %s, want %s", tt.root, got, tt.want)
			}
		})
	}
}

func TestBreadcrumbTitle(t *testing.T) {
	roots := []*bean.Bean{{Title: "Milestone 0.4.0"}, {Title: "Two-column layout"}, {Title: "Preview pane"}}
	tests := []struct {
		name     string
		roots    []*bean.Bean
		tag      string
		maxWidth int
		want     string
	}{
		{"top level", nil, "", 80, "Beans"},
		{"top level with tag", nil, "idea", 80, "Beans [tag: idea]"},
		{"nested fits", roots, "", 80, "Beans › Milestone 0.4.0 › Two-column layout › Preview pane"},
		{"outer levels elided", roots, "", 45, "Beans › … › Two-column layout › Preview pane"},
		{"only innermost fits", roots, "", 25, "Beans › … › Preview pane"},
		{"hard truncated", roots, "", 15, "Beans › … › Pr…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := listModel{rootBeans: tt.roots, tagFilter: tt.tag}
			got := m.breadcrumbTitle(tt.maxWidth)
			if got != tt.want {
				t.Errorf("breadcrumbTitle(%d) = %q, want %q", tt.maxWidth, got, tt.want)
			}
			if lipgloss.Width(got) > tt.maxWidth {
				t.Errorf("title %q exceeds max width %d", got, tt.maxWidth)
			}
		})
	}
}

func TestEnterFocusesContext(t *testing.T) {
	app, _, h := newHierarchyApp(t, 160)
	selectID(t, app, h.loose.ID)
	pump(app, keyMsg("enter"))
	if app.ws.focus != paneContext || app.ctx.bean.ID != h.loose.ID {
		t.Fatalf("focus = %v, context = %v; want the context showing the leaf", app.ws.focus, app.ctx.bean)
	}

	// Keys scroll the context instead of moving the list cursor
	pump(app, keyMsg("j"))
	if got := selectedID(app); got != h.loose.ID {
		t.Errorf("list cursor moved to %s while the context had focus", got)
	}

	pump(app, keyMsg("esc"))
	if app.ws.focus != paneList {
		t.Errorf("esc should return focus to the list, focus = %v", app.ws.focus)
	}
}

func TestNarrowContextFocusShowsItFullWidth(t *testing.T) {
	app, _, h := newHierarchyApp(t, 100)
	selectID(t, app, h.loose.ID)
	if strings.Contains(ansi.Strip(app.View()), "Loose body") {
		t.Fatal("narrow list view should not show the context")
	}
	pump(app, keyMsg("enter"))
	view := ansi.Strip(app.View())
	if !strings.Contains(view, "Loose body") {
		t.Errorf("focused context should show in a narrow terminal:\n%s", view)
	}
}

func TestBreadcrumbTitleShowArchived(t *testing.T) {
	m := listModel{tagFilter: "idea", showArchived: true}
	if got, want := m.breadcrumbTitle(80), "Beans [tag: idea] [+archived]"; got != want {
		t.Errorf("breadcrumbTitle = %q, want %q", got, want)
	}
}
