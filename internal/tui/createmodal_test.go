package tui

import (
	"reflect"
	"regexp"
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
				m, _ = pressCreate(m, keyMsg("tab"))
				m = typeIntoCreate(m, "just a body")
				return pressCreate(m, ctrlS)
			},
			wantType: "task", wantStatus: "draft", wantPrio: "normal", wantBody: "just a body",
		},
		{
			name: "selectors cycle type, status and priority",
			build: func(m createModalModel) (createModalModel, tea.Cmd) {
				m = typeIntoCreate(m, "Bug")
				// tab past the description; type: task -> milestone (wraps), then back to task, then bug (left twice)
				return pressCreate(m,
					keyMsg("tab"), keyMsg("tab"), keyMsg("right"), keyMsg("left"), keyMsg("left"), keyMsg("left"),
					keyMsg("tab"), keyMsg("left"),
					keyMsg("tab"), keyMsg("up"),
					keyMsg("enter"))
			},
			wantTitle: "Bug", wantType: "bug", wantStatus: "todo", wantPrio: "high",
		},
		{
			name: "enter in description inserts newline, ctrl+s creates",
			build: func(m createModalModel) (createModalModel, tea.Cmd) {
				m = typeIntoCreate(m, "With body")
				m, _ = pressCreate(m, keyMsg("tab"))
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
				return pressCreate(typeIntoCreate(m, "Edit me"), keyMsg("tab"), keyMsg("tab"), keyMsg("e"))
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
			if !reflect.DeepEqual(got, want) {
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
	app.Update(keyMsg("tab"))
	app.Update(keyMsg("right"))
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
	m := typeIntoCreate(newTestCreateModal(), "My title")
	m.prefixes = []string{"beans"}
	view := m.View()
	for _, want := range []string{"Create New Bean", "My title", "Prefix", "beans", "Type", "task", "Status", "draft", "Priority", "normal"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestCreateModalPrefill(t *testing.T) {
	m := newTestCreateModal().withPrefill(beanPrefill{
		Type: "bug", Status: "todo", Priority: "high", Size: "sm", Tags: []string{"idea"},
	})
	m.textInput.SetValue("Prefilled")

	msg, ok := m.submit(false)().(beanCreatedMsg)
	if !ok {
		t.Fatal("submit should create a bean")
	}
	if msg.beanType != "bug" || msg.status != "todo" || msg.priority != "high" || msg.size != "sm" {
		t.Errorf("prefilled fields = %s/%s/%s/%s, want bug/todo/high/sm", msg.beanType, msg.status, msg.priority, msg.size)
	}
	if len(msg.tags) != 1 || msg.tags[0] != "idea" {
		t.Errorf("tags = %v, want [idea]", msg.tags)
	}
	if !strings.Contains(m.View(), "Tags: idea") {
		t.Error("form should show the prefilled tags")
	}
}

func TestCreateModalFocusOrder(t *testing.T) {
	tests := []struct {
		name     string
		prefixes []string
		want     []createField
	}{
		{
			name:     "tab walks the fields in visual order",
			prefixes: []string{"beans", "other"},
			want: []createField{createFieldBody, createFieldPrefix, createFieldType,
				createFieldStatus, createFieldPriority, createFieldSize, createFieldTitle},
		},
		{
			name:     "a single prefix is skipped",
			prefixes: []string{"beans"},
			want: []createField{createFieldBody, createFieldType,
				createFieldStatus, createFieldPriority, createFieldSize, createFieldTitle},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestCreateModal()
			m.prefixes = tt.prefixes
			var got []createField
			for range tt.want {
				m, _ = pressCreate(m, keyMsg("tab"))
				got = append(got, m.focus)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("focus order = %v, want %v", got, tt.want)
			}
			m, _ = pressCreate(m, shiftTab)
			if m.focus != createFieldSize {
				t.Errorf("shift+tab from title: focus = %v, want size", m.focus)
			}
		})
	}
}

// TestFormViewLayout checks that the text fields render without labels and
// that the selector row sits below the description, in both forms
func TestFormViewLayout(t *testing.T) {
	create := typeIntoCreate(newTestCreateModal(), "title")
	create, _ = pressCreate(create, keyMsg("tab"))
	create = typeIntoCreate(create, "body")
	create.prefixes = []string{"beans", "other"}
	tests := []struct {
		name   string
		view   string
		labels []string
	}{
		{"create", create.View(), []string{"Prefix", "Type", "Status", "Priority", "Size"}},
		{"edit", newTestEditModal(40).View(), []string{"Type", "Status", "Priority", "Size"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, label := range []string{"Title", "ID prefix"} {
				if strings.Contains(tt.view, label) {
					t.Errorf("view should not contain the %q label", label)
				}
			}
			if descriptionLabel.MatchString(tt.view) {
				t.Error("view should not contain the \"Description\" label")
			}
			if strings.Index(tt.view, "body") > strings.Index(tt.view, "Type") {
				t.Error("selector row should come after the description")
			}
			row := labelRow(tt.view, "Type")
			for _, label := range tt.labels {
				if !strings.Contains(row, label) {
					t.Errorf("selector row %q missing %q", row, label)
				}
			}
		})
	}
}

// descriptionLabel matches a "Description" label on a line of its own, as
// opposed to the description placeholder
var descriptionLabel = regexp.MustCompile(`Description\s+│`)

// labelRow returns the rendered line containing the given text
func labelRow(view, text string) string {
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, text) {
			return line
		}
	}
	return ""
}

func TestCreateModalColumnsFitMinimumWidth(t *testing.T) {
	m := newCreateModalModel(config.Default(), 40, 40, "", "")
	m.prefixes = []string{"project-x", "beans"}
	row := labelRow(m.View(), "Prefix")
	for _, label := range []string{"Type", "Status", "Priority", "Size"} {
		if !strings.Contains(row, label) {
			t.Errorf("at minimum width the selector row %q should still hold %q", row, label)
		}
	}
	if !strings.Contains(labelRow(m.View(), "project-x"), "md") {
		t.Error("prefix value should share its line with the size value")
	}
}
