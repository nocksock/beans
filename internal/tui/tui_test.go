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
