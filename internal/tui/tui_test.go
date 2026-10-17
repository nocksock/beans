package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
)

func newTestApp(t *testing.T) (*App, *bean.Bean) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Default()
	cfg.SetConfigDir(dir)
	core := beancore.New(dir, cfg)
	core.SetWarnWriter(nil)
	b := &bean.Bean{Title: "Milestone", Status: "todo", Type: "milestone"}
	other := &bean.Bean{Title: "Task", Status: "todo", Type: "task"}
	for _, nb := range []*bean.Bean{b, other} {
		if err := core.Create(nb); err != nil {
			t.Fatal(err)
		}
	}
	app := New(core, cfg)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app.Update(app.list.loadBeans())
	return app, b
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestQuitKey(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(a *App, b *bean.Bean)
		wantQuit bool
	}{
		{
			name:     "list",
			setup:    func(a *App, b *bean.Bean) {},
			wantQuit: true,
		},
		{
			name: "list while filtering",
			setup: func(a *App, b *bean.Bean) {
				a.Update(keyMsg("/"))
			},
			wantQuit: false,
		},
		{
			name: "create modal",
			setup: func(a *App, b *bean.Bean) {
				a.Update(openCreateModalMsg{})
			},
			wantQuit: false,
		},
		{
			name: "tag editor",
			setup: func(a *App, b *bean.Bean) {
				a.Update(openTagEditorMsg{beanIDs: []string{b.ID}, beanTitle: b.Title})
			},
			wantQuit: false,
		},
		{
			name: "status picker",
			setup: func(a *App, b *bean.Bean) {
				a.Update(openStatusPickerMsg{beanIDs: []string{b.ID}, beanTitle: b.Title})
			},
			wantQuit: true,
		},
		{
			name: "status picker while filtering",
			setup: func(a *App, b *bean.Bean) {
				a.Update(openStatusPickerMsg{beanIDs: []string{b.ID}, beanTitle: b.Title})
				a.Update(keyMsg("/"))
			},
			wantQuit: false,
		},
		{
			name: "blocking picker while filtering",
			setup: func(a *App, b *bean.Bean) {
				a.Update(openBlockingPickerMsg{beanID: b.ID, beanTitle: b.Title})
				a.Update(keyMsg("/"))
			},
			wantQuit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, b := newTestApp(t)
			tt.setup(app, b)
			_, cmd := app.Update(keyMsg("q"))
			if got := isQuit(cmd); got != tt.wantQuit {
				t.Errorf("quit = %v, want %v (state %v)", got, tt.wantQuit, app.state)
			}
		})
	}
}

func TestCreateModalAcceptsQ(t *testing.T) {
	app, _ := newTestApp(t)
	app.Update(openCreateModalMsg{})
	for _, r := range "quick" {
		app.Update(keyMsg(string(r)))
	}
	if got := app.createModal.textInput.Value(); got != "quick" {
		t.Errorf("create modal input = %q, want %q", got, "quick")
	}
}

func TestBlockedByPicker(t *testing.T) {
	app, b := newTestApp(t)
	var blockerID string
	for _, o := range app.resolver.Core.All() {
		if o.ID != b.ID {
			blockerID = o.ID
		}
	}

	t.Run("B in list opens blocked-by picker", func(t *testing.T) {
		_, cmd := app.Update(keyMsg("B"))
		if cmd == nil {
			t.Fatal("expected command from B key")
		}
		msg, ok := cmd().(openBlockingPickerMsg)
		if !ok || !msg.blockedBy {
			t.Fatalf("got %#v, want openBlockingPickerMsg with blockedBy", msg)
		}
	})

	t.Run("picker is titled for blocked-by", func(t *testing.T) {
		app.Update(openBlockingPickerMsg{beanID: b.ID, beanTitle: b.Title, blockedBy: true})
		if app.state != viewBlockingPicker || !app.blockingPicker.blockedBy {
			t.Fatalf("state = %v, blockedBy = %v", app.state, app.blockingPicker.blockedBy)
		}
		if app.blockingPicker.list.Title != "Manage Blocked By" {
			t.Errorf("title = %q", app.blockingPicker.list.Title)
		}
	})

	t.Run("confirm applies blocked-by mutations", func(t *testing.T) {
		app.Update(blockingConfirmedMsg{beanID: b.ID, blockedBy: true, toAdd: []string{blockerID}})
		got, _ := app.resolver.Core.Get(b.ID)
		if !got.IsBlockedBy(blockerID) || len(got.Blocking) != 0 {
			t.Errorf("BlockedBy = %v, Blocking = %v; want blocked by %s only", got.BlockedBy, got.Blocking, blockerID)
		}

		app.Update(blockingConfirmedMsg{beanID: b.ID, blockedBy: true, toRemove: []string{blockerID}})
		got, _ = app.resolver.Core.Get(b.ID)
		if got.IsBlockedBy(blockerID) {
			t.Errorf("BlockedBy = %v, want empty", got.BlockedBy)
		}
	})
}

func selectedBeanID(a *App) string {
	if item, ok := a.list.list.SelectedItem().(beanItem); ok {
		return item.bean.ID
	}
	return ""
}

func TestCreateBeanMovesCursorToNewBean(t *testing.T) {
	app, _ := newTestApp(t)

	_, cmd := app.Update(beanCreatedMsg{title: "Zzz newly created"})
	if cmd == nil {
		t.Fatal("expected command after creating bean")
	}

	var created *bean.Bean
	for _, b := range app.core.All() {
		if b.Title == "Zzz newly created" {
			created = b
		}
	}
	if created == nil {
		t.Fatal("created bean not found")
	}

	// Simulate the list reload that runs alongside the editor
	app.Update(app.list.loadBeans())
	if got := selectedBeanID(app); got != created.ID {
		t.Errorf("selected = %q, want new bean %q", got, created.ID)
	}

	// Returning from the editor keeps the cursor on the new bean
	app.Update(openEditorMsg{beanID: created.ID, beanPath: created.Path})
	_, cmd = app.Update(editorFinishedMsg{})
	if cmd == nil {
		t.Fatal("expected list reload after editor finished")
	}
	app.Update(cmd())
	if got := selectedBeanID(app); got != created.ID {
		t.Errorf("after editor: selected = %q, want new bean %q", got, created.ID)
	}
}
