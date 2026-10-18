package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
)

func newTestEditModal(height int) createModalModel {
	b := &bean.Bean{ID: "beans-abcd", Title: "Old title", Type: "bug", Status: "todo", Priority: "high", Body: "Old body\n"}
	return newEditModalModel(config.Default(), 120, height, b)
}

func submitted(t *testing.T, cmd tea.Cmd) beanEditedMsg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected submit command")
	}
	msg, ok := cmd().(beanEditedMsg)
	if !ok {
		t.Fatalf("got %T, want beanEditedMsg", cmd())
	}
	return msg
}

func TestEditModalPrefill(t *testing.T) {
	m := newTestEditModal(40)
	if got := m.textInput.Value(); got != "Old title" {
		t.Errorf("title = %q", got)
	}
	if got := m.body.Value(); got != "Old body" {
		t.Errorf("body = %q", got)
	}
	if m.types[m.typeIdx].Name != "bug" || m.statuses[m.statusIdx].Name != "todo" || m.priorities[m.priorityIdx].Name != "high" {
		t.Errorf("selectors = %s/%s/%s", m.types[m.typeIdx].Name, m.statuses[m.statusIdx].Name, m.priorities[m.priorityIdx].Name)
	}
}

func TestEditModalSubmitOnlyChangedFields(t *testing.T) {
	str := func(p *string) string {
		if p == nil {
			return "<nil>"
		}
		return *p
	}
	tests := []struct {
		name                           string
		build                          func(m createModalModel) (createModalModel, tea.Cmd)
		title, body, typ, status, prio string
		wantEditor                     bool
	}{
		{
			name:  "unchanged",
			build: func(m createModalModel) (createModalModel, tea.Cmd) { return pressCreate(m, keyMsg("enter")) },
			title: "<nil>", body: "<nil>", typ: "<nil>", status: "<nil>", prio: "<nil>",
		},
		{
			name: "title changed",
			build: func(m createModalModel) (createModalModel, tea.Cmd) {
				m = typeIntoCreate(m, "!")
				return pressCreate(m, keyMsg("enter"))
			},
			title: "Old title!", body: "<nil>", typ: "<nil>", status: "<nil>", prio: "<nil>",
		},
		{
			name: "status changed, save & edit",
			build: func(m createModalModel) (createModalModel, tea.Cmd) {
				return pressCreate(m, keyMsg("tab"), keyMsg("tab"), keyMsg("tab"), keyMsg("l"), keyMsg("e"))
			},
			title: "<nil>", body: "<nil>", typ: "<nil>", status: "draft", prio: "<nil>", wantEditor: true,
		},
		{
			name: "selector cycled back is unchanged",
			build: func(m createModalModel) (createModalModel, tea.Cmd) {
				return pressCreate(m, keyMsg("tab"), keyMsg("tab"), keyMsg("l"), keyMsg("h"), keyMsg("enter"))
			},
			title: "<nil>", body: "<nil>", typ: "<nil>", status: "<nil>", prio: "<nil>",
		},
		{
			name: "body changed",
			build: func(m createModalModel) (createModalModel, tea.Cmd) {
				m, _ = pressCreate(m, keyMsg("tab"))
				m = typeIntoCreate(m, "x")
				return pressCreate(m, ctrlS)
			},
			title: "<nil>", body: "Old bodyx", typ: "<nil>", status: "<nil>", prio: "<nil>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmd := tt.build(newTestEditModal(40))
			got := submitted(t, cmd)
			if got.beanID != "beans-abcd" {
				t.Errorf("beanID = %q", got.beanID)
			}
			in := got.input
			if str(in.Title) != tt.title || str(in.Body) != tt.body || str(in.Type) != tt.typ ||
				str(in.Status) != tt.status || str(in.Priority) != tt.prio {
				t.Errorf("input = title:%s body:%s type:%s status:%s prio:%s",
					str(in.Title), str(in.Body), str(in.Type), str(in.Status), str(in.Priority))
			}
			if got.openEditor != tt.wantEditor {
				t.Errorf("openEditor = %v, want %v", got.openEditor, tt.wantEditor)
			}
		})
	}
}

func TestEditModalUnknownStatusIsKept(t *testing.T) {
	b := &bean.Bean{ID: "beans-abcd", Title: "T", Type: "task", Status: "no-such-status", Priority: "normal"}
	m := newEditModalModel(config.Default(), 120, 40, b)
	_, cmd := pressCreate(m, keyMsg("enter"))
	if got := submitted(t, cmd); got.input.Status != nil {
		t.Errorf("status = %q, want unchanged", *got.input.Status)
	}
}

func TestEditModalRequiresTitleOrBody(t *testing.T) {
	b := &bean.Bean{ID: "beans-abcd", Title: "T", Type: "task", Status: "todo", Priority: "normal"}
	m := newEditModalModel(config.Default(), 120, 40, b)
	m, _ = pressCreate(m, keyMsg("backspace"))
	m, cmd := pressCreate(m, keyMsg("enter"))
	if cmd != nil {
		t.Fatal("expected no submit with empty title and body")
	}
	if !strings.Contains(m.View(), "needs a title or a description") {
		t.Error("expected validation message in view")
	}
}

func TestEditModalBodyGrowsWithHeight(t *testing.T) {
	if got := newTestEditModal(20).body.Height(); got != editModalMinBodyHeight {
		t.Errorf("small terminal: body height = %d, want %d", got, editModalMinBodyHeight)
	}
	if got := newTestEditModal(60).body.Height(); got != 60-editModalChromeHeight {
		t.Errorf("tall terminal: body height = %d, want %d", got, 60-editModalChromeHeight)
	}
	m, _ := newTestEditModal(20).Update(tea.WindowSizeMsg{Width: 120, Height: 60})
	if got := m.body.Height(); got != 60-editModalChromeHeight {
		t.Errorf("after resize: body height = %d", got)
	}
	if got := newTestCreateModal().body.Height(); got != 5 {
		t.Errorf("create modal body height = %d, want 5", got)
	}
}

// TestEditModalFillsHeight checks that editModalChromeHeight matches the
// rendered chrome: the form never overflows the terminal, even with an error
// shown at the narrowest width, and uses the height at full width
func TestEditModalFillsHeight(t *testing.T) {
	m := newTestEditModal(60)
	if got := lipgloss.Height(m.View()); got < 60-editModalSlackHeight || got > 60 {
		t.Errorf("edit modal height = %d, want %d..60", got, 60-editModalSlackHeight)
	}
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 60})
	m.errorMessage = "A bean needs a title or a description"
	if got := lipgloss.Height(m.View()); got > 60 {
		t.Errorf("narrow edit modal with error: height = %d, want at most 60", got)
	}
}

func TestEditModalView(t *testing.T) {
	v := newTestEditModal(40).View()
	for _, want := range []string{"Edit Bean", "beans-abcd", "save"} {
		if !strings.Contains(v, want) {
			t.Errorf("view missing %q", want)
		}
	}
	if strings.Contains(v, "create") {
		t.Error("edit view should not mention create")
	}
}

func TestEditKeysOpenPopupOrEditor(t *testing.T) {
	tests := []struct {
		name    string
		context bool
		key     string
		want    string
	}{
		{"list e", false, "e", "popup"},
		{"list E", false, "E", "editor"},
		{"context e", true, "e", "popup"},
		{"context E", true, "E", "editor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := newTestApp(t)
			if tt.context {
				app.Update(keyMsg("tab"))
				if app.ws.focus != paneContext {
					t.Fatalf("focus = %v, want context", app.ws.focus)
				}
			}
			_, cmd := app.Update(keyMsg(tt.key))
			if cmd == nil {
				t.Fatal("expected command")
			}
			switch msg := cmd().(type) {
			case openEditModalMsg:
				if tt.want != "popup" {
					t.Errorf("got popup, want %s", tt.want)
				}
				app.Update(msg)
				if app.state != viewCreateModal || !app.createModal.isEdit() {
					t.Errorf("edit popup not open (state %v)", app.state)
				}
			case openEditorMsg:
				if tt.want != "editor" {
					t.Errorf("got editor, want %s", tt.want)
				}
			default:
				t.Errorf("got %T", msg)
			}
		})
	}
}

func TestBeanEditedSavesAndReturns(t *testing.T) {
	app, _ := newTestApp(t)
	target := selectedBeanID(app)
	b, _ := app.core.Get(target)
	app.Update(openEditModalMsg{bean: b})

	title := "Renamed"
	status := "in-progress"
	msg := beanEditedMsg{beanID: target, openEditor: true}
	msg.input.Title = &title
	msg.input.Status = &status
	_, cmd := app.Update(msg)
	if app.state != viewList {
		t.Errorf("state = %v, want list", app.state)
	}
	got, _ := app.core.Get(target)
	if got.Title != "Renamed" || got.Status != "in-progress" || got.Type != b.Type {
		t.Errorf("bean = %q/%q/%q", got.Title, got.Status, got.Type)
	}
	if cmd == nil {
		t.Fatal("expected reload + editor command")
	}
	app.Update(app.list.loadBeans())
	if sel := selectedBeanID(app); sel != target {
		t.Errorf("selected = %q, want %q", sel, target)
	}
}

func TestBeanEditedUnchangedDoesNotTouchBean(t *testing.T) {
	app, _ := newTestApp(t)
	target := selectedBeanID(app)
	before, _ := app.core.Get(target)
	updatedAt := *before.UpdatedAt
	app.Update(openEditModalMsg{bean: before})
	app.Update(beanEditedMsg{beanID: target})
	after, _ := app.core.Get(target)
	if !after.UpdatedAt.Equal(updatedAt) {
		t.Errorf("updated_at changed: %v -> %v", updatedAt, after.UpdatedAt)
	}
}

func TestBeanEditedFromContextRefreshesContext(t *testing.T) {
	app, _ := newTestApp(t)
	app.Update(keyMsg("tab"))
	target := app.ctx.bean.ID
	_, cmd := app.Update(keyMsg("e"))
	app.Update(cmd())

	title := "Renamed in context"
	msg := beanEditedMsg{beanID: target}
	msg.input.Title = &title
	pump(app, msg)
	if app.state != viewList || app.ws.focus != paneContext {
		t.Fatalf("state = %v, focus = %v, want context pane", app.state, app.ws.focus)
	}
	if app.ctx.bean.Title != title {
		t.Errorf("context title = %q, want %q", app.ctx.bean.Title, title)
	}
}
