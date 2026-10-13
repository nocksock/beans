package tui

import (
	"slices"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
)

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func typeText(m tagEditorModel, text string) tagEditorModel {
	for _, r := range text {
		m, _ = m.Update(keyMsg(string(r)))
	}
	return m
}

// confirm presses enter and returns the resulting tagsConfirmedMsg.
func confirm(t *testing.T, m tagEditorModel) tagsConfirmedMsg {
	t.Helper()
	_, cmd := m.Update(keyMsg("enter"))
	if cmd == nil {
		t.Fatal("expected a command on enter")
	}
	msg, ok := cmd().(tagsConfirmedMsg)
	if !ok {
		t.Fatalf("expected tagsConfirmedMsg, got %T", cmd())
	}
	slices.Sort(msg.toAdd)
	slices.Sort(msg.toRemove)
	return msg
}

func newTestTagEditor(beans []*bean.Bean, allTags []tagWithCount) tagEditorModel {
	return newTagEditorModel(beans, "title", allTags, 80, 40)
}

func TestTagEditorToggleExistingTags(t *testing.T) {
	b := &bean.Bean{ID: "b1", Tags: []string{"frontend"}}
	allTags := []tagWithCount{{"frontend", 2}, {"backend", 1}, {"idea", 3}}
	m := newTestTagEditor([]*bean.Bean{b}, allTags)

	// Items are sorted alphabetically: backend, frontend, idea
	if got := m.visibleTagNames(); !slices.Equal(got, []string{"backend", "frontend", "idea"}) {
		t.Fatalf("visible tags = %v", got)
	}

	// Toggle backend on (cursor starts at 0)
	m, _ = m.Update(keyMsg("space"))
	// Move to frontend and toggle it off
	m, _ = m.Update(keyMsg("down"))
	m, _ = m.Update(keyMsg("tab"))

	msg := confirm(t, m)
	if !slices.Equal(msg.beanIDs, []string{"b1"}) {
		t.Errorf("beanIDs = %v", msg.beanIDs)
	}
	if !slices.Equal(msg.toAdd, []string{"backend"}) {
		t.Errorf("toAdd = %v, want [backend]", msg.toAdd)
	}
	if !slices.Equal(msg.toRemove, []string{"frontend"}) {
		t.Errorf("toRemove = %v, want [frontend]", msg.toRemove)
	}
}

func TestTagEditorNoChanges(t *testing.T) {
	b := &bean.Bean{ID: "b1", Tags: []string{"frontend"}}
	m := newTestTagEditor([]*bean.Bean{b}, []tagWithCount{{"frontend", 1}})

	// Toggle twice = no change
	m, _ = m.Update(keyMsg("space"))
	m, _ = m.Update(keyMsg("space"))

	msg := confirm(t, m)
	if len(msg.toAdd) != 0 || len(msg.toRemove) != 0 {
		t.Errorf("expected no changes, got add=%v remove=%v", msg.toAdd, msg.toRemove)
	}
}

func TestTagEditorFilter(t *testing.T) {
	b := &bean.Bean{ID: "b1"}
	allTags := []tagWithCount{{"frontend", 1}, {"backend", 1}, {"idea", 1}}
	m := newTestTagEditor([]*bean.Bean{b}, allTags)

	m = typeText(m, "end")
	if got := m.visibleTagNames(); !slices.Equal(got, []string{"backend", "frontend"}) {
		t.Fatalf("filtered tags = %v, want [backend frontend]", got)
	}
	// "end" is a valid tag name that doesn't exist yet, so it can be created,
	// but the create item comes after existing matches
	if !m.hasCreateItem() {
		t.Error("expected create item for 'end'")
	}
	m, _ = m.Update(keyMsg("space"))
	msg := confirm(t, m)
	if !slices.Equal(msg.toAdd, []string{"backend"}) {
		t.Errorf("toAdd = %v, want [backend] (first match, not the create item)", msg.toAdd)
	}

	// Exact match: no create item
	m = typeText(newTestTagEditor([]*bean.Bean{b}, allTags), "idea")
	if m.hasCreateItem() {
		t.Error("should not offer to create a tag that already exists")
	}
}

func TestTagEditorCreateNewTag(t *testing.T) {
	b := &bean.Bean{ID: "b1"}
	m := newTestTagEditor([]*bean.Bean{b}, []tagWithCount{{"frontend", 1}})

	m = typeText(m, "Needs-Review")
	if !m.hasCreateItem() {
		t.Fatal("expected create item for new tag")
	}
	// No existing tag matches, so the create item is the only (selected) row
	m, _ = m.Update(keyMsg("space"))

	if m.input.Value() != "" {
		t.Errorf("input should be cleared after creating, got %q", m.input.Value())
	}
	if !slices.Contains(m.visibleTagNames(), "needs-review") {
		t.Errorf("new tag should appear in list, got %v", m.visibleTagNames())
	}

	msg := confirm(t, m)
	if !slices.Equal(msg.toAdd, []string{"needs-review"}) {
		t.Errorf("toAdd = %v, want [needs-review]", msg.toAdd)
	}
}

func TestTagEditorInvalidTag(t *testing.T) {
	b := &bean.Bean{ID: "b1"}
	m := newTestTagEditor([]*bean.Bean{b}, nil)

	m = typeText(m, "1bad")
	if m.hasCreateItem() {
		t.Error("should not offer to create an invalid tag")
	}
	if m.validationError() == "" {
		t.Error("expected a validation error for invalid tag")
	}

	// Toggling with nothing to toggle should not panic or create anything
	m, _ = m.Update(keyMsg("space"))
	msg := confirm(t, m)
	if len(msg.toAdd) != 0 {
		t.Errorf("toAdd = %v, want none", msg.toAdd)
	}
}

func TestTagEditorMultipleBeans(t *testing.T) {
	beans := []*bean.Bean{
		{ID: "b1", Tags: []string{"shared", "partial"}},
		{ID: "b2", Tags: []string{"shared"}},
	}
	allTags := []tagWithCount{{"partial", 1}, {"shared", 2}, {"unused", 4}}
	m := newTestTagEditor(beans, allTags)

	if got := m.pending["shared"]; got != tagOnAll {
		t.Errorf("shared state = %v, want tagOnAll", got)
	}
	if got := m.pending["partial"]; got != tagOnSome {
		t.Errorf("partial state = %v, want tagOnSome", got)
	}
	if got := m.pending["unused"]; got != tagOnNone {
		t.Errorf("unused state = %v, want tagOnNone", got)
	}

	// Untouched partial tags must not be changed
	msg := confirm(t, m)
	if len(msg.toAdd) != 0 || len(msg.toRemove) != 0 {
		t.Fatalf("expected no changes, got add=%v remove=%v", msg.toAdd, msg.toRemove)
	}

	// Toggling a partial tag adds it to all beans
	m, _ = m.Update(keyMsg("space")) // cursor on "partial"
	msg = confirm(t, m)
	if !slices.Equal(msg.toAdd, []string{"partial"}) {
		t.Errorf("toAdd = %v, want [partial]", msg.toAdd)
	}

	// Toggling it again removes it from all beans
	m, _ = m.Update(keyMsg("space"))
	msg = confirm(t, m)
	if !slices.Equal(msg.toRemove, []string{"partial"}) {
		t.Errorf("toRemove = %v, want [partial]", msg.toRemove)
	}
}

func TestTagEditorCursorBounds(t *testing.T) {
	b := &bean.Bean{ID: "b1"}
	m := newTestTagEditor([]*bean.Bean{b}, []tagWithCount{{"a", 1}, {"b", 1}})

	m, _ = m.Update(keyMsg("up"))
	if m.cursor != 0 {
		t.Errorf("cursor = %d after up at top, want 0", m.cursor)
	}
	for range 5 {
		m, _ = m.Update(keyMsg("down"))
	}
	if m.cursor != 1 {
		t.Errorf("cursor = %d after many downs, want 1", m.cursor)
	}

	// Filtering resets cursor into range
	m = typeText(m, "a")
	if m.cursor != 0 {
		t.Errorf("cursor = %d after filtering, want 0", m.cursor)
	}
}

func TestTagEditorCancel(t *testing.T) {
	m := newTestTagEditor([]*bean.Bean{{ID: "b1"}}, nil)
	_, cmd := m.Update(keyMsg("esc"))
	if cmd == nil {
		t.Fatal("expected command on esc")
	}
	if _, ok := cmd().(closeTagEditorMsg); !ok {
		t.Errorf("expected closeTagEditorMsg, got %T", cmd())
	}
}

func TestApplyTagChanges(t *testing.T) {
	tests := []struct {
		name     string
		current  []string
		toAdd    []string
		toRemove []string
		want     []string
	}{
		{"add to empty", nil, []string{"a"}, nil, []string{"a"}},
		{"preserves order and appends", []string{"b", "a"}, []string{"c"}, nil, []string{"b", "a", "c"}},
		{"remove", []string{"a", "b", "c"}, nil, []string{"b"}, []string{"a", "c"}},
		{"add existing is no-op", []string{"a"}, []string{"a"}, nil, []string{"a"}},
		{"remove missing is no-op", []string{"a"}, nil, []string{"x"}, []string{"a"}},
		{"remove all", []string{"a"}, nil, []string{"a"}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyTagChanges(tt.current, tt.toAdd, tt.toRemove)
			if !slices.Equal(got, tt.want) {
				t.Errorf("applyTagChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppTagEditorPersistsChanges(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.SetConfigDir(dir)
	core := beancore.New(dir, cfg)
	core.SetWarnWriter(nil)

	b1 := &bean.Bean{Title: "One", Status: "todo", Type: "task", Tags: []string{"keep", "drop"}}
	b2 := &bean.Bean{Title: "Two", Status: "todo", Type: "task", Tags: []string{"drop"}}
	for _, b := range []*bean.Bean{b1, b2} {
		if err := core.Create(b); err != nil {
			t.Fatal(err)
		}
	}

	app := New(core, cfg)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Open the editor for both beans
	app.Update(openTagEditorMsg{beanIDs: []string{b1.ID, b2.ID}, beanTitle: "2 selected beans"})
	if app.state != viewTagEditor {
		t.Fatalf("state = %v, want viewTagEditor", app.state)
	}
	if app.View() == "" {
		t.Error("expected tag editor to render")
	}

	// "drop" is on both beans: toggle it off. Then create "new".
	// Tags sorted: drop, keep
	app.Update(keyMsg("space"))
	for _, r := range "new" {
		app.Update(keyMsg(string(r)))
	}
	app.Update(keyMsg("space"))

	// Confirm and deliver the resulting message back to the app
	_, cmd := app.Update(keyMsg("enter"))
	app.Update(cmd())

	if app.state != viewList {
		t.Errorf("state = %v after confirm, want viewList", app.state)
	}

	for _, tc := range []struct {
		id   string
		want []string
	}{
		{b1.ID, []string{"keep", "new"}},
		{b2.ID, []string{"new"}},
	} {
		got, err := core.Get(tc.id)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(got.Tags, tc.want) {
			t.Errorf("bean %s tags = %v, want %v", tc.id, got.Tags, tc.want)
		}
	}
}
