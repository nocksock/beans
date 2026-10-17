package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/config"
)

func newTestCreateModal() createModalModel {
	return newCreateModalModel(config.Default(), 120, 40, "", "")
}

func typeIntoCreate(m createModalModel, text string) createModalModel {
	for _, r := range text {
		m, _ = m.Update(keyMsg(string(r)))
	}
	return m
}

func pressCreate(m createModalModel, keys ...tea.KeyMsg) (createModalModel, tea.Cmd) {
	var cmd tea.Cmd
	for _, k := range keys {
		m, cmd = m.Update(k)
	}
	return m, cmd
}

var (
	ctrlS    = tea.KeyMsg{Type: tea.KeyCtrlS}
	ctrlE    = tea.KeyMsg{Type: tea.KeyCtrlE}
	shiftTab = tea.KeyMsg{Type: tea.KeyShiftTab}
)

func TestCreateModalDefaults(t *testing.T) {
	m := newTestCreateModal()
	if m.focus != createFieldTitle {
		t.Errorf("focus = %v, want title", m.focus)
	}
	if got := m.types[m.typeIdx].Name; got != config.Default().GetDefaultType() {
		t.Errorf("type = %q, want configured default", got)
	}
	if got := m.statuses[m.statusIdx].Name; got != "draft" {
		t.Errorf("status = %q, want draft", got)
	}
	if got := m.priorities[m.priorityIdx].Name; got != "normal" {
		t.Errorf("priority = %q, want normal", got)
	}
}

func TestCreateModalSubmit(t *testing.T) {
	tests := []struct {
		name       string
		build      func(m createModalModel) (createModalModel, tea.Cmd)
		wantClose  bool
		wantTitle  string
		wantType   string
		wantStatus string
		wantPrio   string
		wantBody   string
		wantEditor bool
	}{
		{
			name: "enter in title creates with defaults",
			build: func(m createModalModel) (createModalModel, tea.Cmd) {
				return pressCreate(typeIntoCreate(m, "Hello"), keyMsg("enter"))
			},
			wantTitle: "Hello", wantType: "task", wantStatus: "draft", wantPrio: "normal",
		},
		{
			name: "empty title and description closes",
			build: func(m createModalModel) (createModalModel, tea.Cmd) {
				return pressCreate(m, keyMsg("enter"))
			},
			wantClose: true,
		},
		{
			name: "empty title with description creates untitled bean",
			build: func(m createModalModel) (createModalModel, tea.Cmd) {
				m, _ = pressCreate(m, shiftTab)
				m = typeIntoCreate(m, "just a body")
				return pressCreate(m, ctrlS)
			},
			wantType: "task", wantStatus: "draft", wantPrio: "normal", wantBody: "just a body",
		},
		{
			name: "selectors cycle type, status and priority",
			build: func(m createModalModel) (createModalModel, tea.Cmd) {
				m = typeIntoCreate(m, "Bug")
				// type: task -> milestone (wraps), then back to task, then bug (left twice)
				return pressCreate(m,
					keyMsg("tab"), keyMsg("l"), keyMsg("h"), keyMsg("h"), keyMsg("h"),
					keyMsg("tab"), keyMsg("h"),
					keyMsg("tab"), keyMsg("k"),
					keyMsg("enter"))
			},
			wantTitle: "Bug", wantType: "bug", wantStatus: "todo", wantPrio: "high",
		},
		{
			name: "enter in description inserts newline, ctrl+s creates",
			build: func(m createModalModel) (createModalModel, tea.Cmd) {
				m = typeIntoCreate(m, "With body")
				m, _ = pressCreate(m, shiftTab)
				m = typeIntoCreate(m, "line1")
				m, _ = pressCreate(m, keyMsg("enter"))
				m = typeIntoCreate(m, "line2")
				return pressCreate(m, ctrlS)
			},
			wantTitle: "With body", wantType: "task", wantStatus: "draft", wantPrio: "normal", wantBody: "line1\nline2",
		},
		{
			name: "ctrl+e creates and opens editor",
			build: func(m createModalModel) (createModalModel, tea.Cmd) {
				return pressCreate(typeIntoCreate(m, "Edit me"), ctrlE)
			},
			wantTitle: "Edit me", wantType: "task", wantStatus: "draft", wantPrio: "normal", wantEditor: true,
		},
		{
			name: "e on selector creates and opens editor",
			build: func(m createModalModel) (createModalModel, tea.Cmd) {
				return pressCreate(typeIntoCreate(m, "Edit me"), keyMsg("tab"), keyMsg("e"))
			},
			wantTitle: "Edit me", wantType: "task", wantStatus: "draft", wantPrio: "normal", wantEditor: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmd := tt.build(newTestCreateModal())
			if cmd == nil {
				t.Fatal("expected command")
			}
			msg := cmd()
			if tt.wantClose {
				if _, ok := msg.(closeCreateModalMsg); !ok {
					t.Fatalf("got %#v, want closeCreateModalMsg", msg)
				}
				return
			}
			got, ok := msg.(beanCreatedMsg)
			if !ok {
				t.Fatalf("got %#v, want beanCreatedMsg", msg)
			}
			want := beanCreatedMsg{title: tt.wantTitle, beanType: tt.wantType, status: tt.wantStatus, priority: tt.wantPrio, body: tt.wantBody, openEditor: tt.wantEditor}
			if got != want {
				t.Errorf("got %+v, want %+v", got, want)
			}
		})
	}
}

func TestCreateModalTypingInTitleIgnoresSelectorKeys(t *testing.T) {
	m := typeIntoCreate(newTestCreateModal(), "hjkl e")
	if got := m.textInput.Value(); got != "hjkl e" {
		t.Errorf("title = %q, want %q", got, "hjkl e")
	}
}

func TestCreateBeanFromFormAppliesFields(t *testing.T) {
	app, _ := newTestApp(t)
	_, cmd := app.Update(beanCreatedMsg{title: "Form bean", beanType: "bug", status: "todo", priority: "high", body: "Some body"})
	if cmd == nil {
		t.Fatal("expected command")
	}
	if _, ok := cmd().(openEditorMsg); ok {
		t.Error("editor should not open without openEditor")
	}
	for _, b := range app.core.All() {
		if b.Title == "Form bean" {
			if b.Type != "bug" || b.Status != "todo" || b.Priority != "high" || b.Body != "Some body" {
				t.Errorf("got type=%q status=%q priority=%q body=%q", b.Type, b.Status, b.Priority, b.Body)
			}
			return
		}
	}
	t.Fatal("created bean not found")
}

func TestCreateModalKeysReachFormThroughApp(t *testing.T) {
	app, _ := newTestApp(t)
	app.Update(openCreateModalMsg{})
	for _, r := range "Via app" {
		app.Update(keyMsg(string(r)))
	}
	app.Update(keyMsg("tab"))
	app.Update(keyMsg("l"))
	if app.createModal.focus != createFieldType {
		t.Fatalf("focus = %v, want type", app.createModal.focus)
	}
	_, cmd := app.Update(keyMsg("e"))
	if cmd == nil {
		t.Fatal("expected command")
	}
	got, ok := cmd().(beanCreatedMsg)
	if !ok || got.title != "Via app" || got.beanType != "milestone" || !got.openEditor {
		t.Errorf("got %+v", got)
	}
}

func TestCreateModalView(t *testing.T) {
	view := typeIntoCreate(newTestCreateModal(), "My title").View()
	for _, want := range []string{"Create New Bean", "My title", "Type", "task", "Status", "draft", "Priority", "normal", "Description"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
}
