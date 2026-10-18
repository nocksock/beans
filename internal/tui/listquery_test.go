package tui

import (
	"slices"
	"strings"
	"testing"
)

// openQuery presses / without running the returned cursor blink command.
func openQuery(app *App) {
	app.Update(keyMsg("/"))
}

// typeQuery types text into the list's filter query and applies the reload.
func typeQuery(app *App, text string) {
	for _, r := range text {
		k := string(r)
		if r == ' ' {
			k = "space"
		}
		app.Update(keyMsg(k))
	}
	app.Update(app.list.loadBeans())
}

func TestListFilterQuery(t *testing.T) {
	app, _, h := newHierarchyApp(t, 160)

	openQuery(app)
	if !app.list.editingQuery() {
		t.Fatal("/ should focus the filter query")
	}
	if app.canQuit() {
		t.Error("q must not quit while typing a query")
	}

	typeQuery(app, "type:task")
	got := listedIDs(app)
	// The nested task keeps its ancestors as context
	for _, id := range []string{h.loose.ID, h.task.ID, h.epic.ID, h.milestone.ID} {
		if !slices.Contains(got, id) {
			t.Errorf("type:task should list %s (match or ancestor), got %v", id, got)
		}
	}
	matched := 0
	for _, it := range app.list.list.Items() {
		if it.(beanItem).matched {
			matched++
		}
	}
	if matched != 2 {
		t.Errorf("type:task should match 2 beans, got %d", matched)
	}

	// Free text narrows further
	typeQuery(app, " loose")
	if got := listedIDs(app); len(got) != 1 || got[0] != h.loose.ID {
		t.Errorf("type:task loose should list only %s, got %v", h.loose.ID, got)
	}

	// Enter applies and leaves typing mode; the title shows the query
	press(app, "enter")
	if app.list.editingQuery() {
		t.Error("enter should stop editing the query")
	}
	if !strings.Contains(app.View(), "[type:task loose]") {
		t.Error("title should show the applied query")
	}

	// Esc clears the applied query before anything else
	press(app, "esc")
	if app.list.hasActiveFilter() {
		t.Error("esc should clear the applied query")
	}
	if got := len(listedIDs(app)); got != 4 {
		t.Errorf("after clearing, should list all 4 beans, got %d", got)
	}
}

func TestListFilterQueryEscWhileTyping(t *testing.T) {
	app, _, _ := newHierarchyApp(t, 160)

	openQuery(app)
	typeQuery(app, "size:huge")
	if !strings.Contains(app.list.Footer(), `unknown size "huge"`) {
		t.Errorf("footer should warn about the unknown value, got %q", app.list.Footer())
	}

	press(app, "esc")
	if app.list.editingQuery() || app.list.hasActiveFilter() {
		t.Error("esc while typing should close and clear the query")
	}
}

func TestListFilterQuerySort(t *testing.T) {
	app, _, h := newHierarchyApp(t, 160)

	openQuery(app)
	typeQuery(app, "sort:title:desc")
	press(app, "enter")

	// Top-level siblings: Milestone and Loose, sorted by title descending
	var top []string
	for _, it := range app.list.list.Items() {
		if it.(beanItem).treePrefix == "" {
			top = append(top, it.(beanItem).bean.ID)
		}
	}
	if want := []string{h.milestone.ID, h.loose.ID}; !slices.Equal(top, want) {
		t.Errorf("top level = %v, want %v", top, want)
	}
}

func TestTagPickerSetsQueryToken(t *testing.T) {
	app, _, _ := newHierarchyApp(t, 160)

	app.list.setQuery("size:sm tag:old")
	app.Update(tagSelectedMsg{tag: "idea"})
	if got, want := app.list.queryInput.Value(), "size:sm tag:idea"; got != want {
		t.Errorf("query = %q, want %q", got, want)
	}
}
