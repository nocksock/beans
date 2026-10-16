package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
)

// pump sends msg to the app and recursively feeds back resulting messages.
func pump(a *App, msg tea.Msg) {
	_, cmd := a.Update(msg)
	runCmd(a, cmd, 0)
}

func runCmd(a *App, cmd tea.Cmd, depth int) {
	if cmd == nil || depth > 10 {
		return
	}
	switch msg := cmd().(type) {
	case nil:
	case tea.BatchMsg:
		for _, c := range msg {
			runCmd(a, c, depth+1)
		}
	default:
		_, next := a.Update(msg)
		runCmd(a, next, depth+1)
	}
}

func newArchiveTestApp(t *testing.T) (*App, map[string]*bean.Bean) {
	t.Helper()
	dir := t.TempDir()
	cfg := config.Default()
	cfg.SetConfigDir(dir)
	core := beancore.New(dir, cfg)
	core.SetWarnWriter(nil)
	beans := map[string]*bean.Bean{
		"todo":      {Title: "Todo", Status: "todo", Type: "task"},
		"completed": {Title: "Done", Status: "completed", Type: "task"},
		"scrapped":  {Title: "Scrapped", Status: "scrapped", Type: "task"},
	}
	for _, b := range beans {
		if err := core.Create(b); err != nil {
			t.Fatal(err)
		}
	}
	app := New(core, cfg)
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app.Update(app.list.loadBeans())
	return app, beans
}

func remaining(a *App) map[string]bool {
	ids := map[string]bool{}
	for _, b := range a.core.All() {
		ids[b.ID] = true
	}
	return ids
}

func TestArchive(t *testing.T) {
	tests := []struct {
		name      string
		openBean  string   // bean to open in detail view ("" = list view)
		keys      []string // keys pressed
		wantKept  []string
		wantState viewState
	}{
		{
			name:      "list y prompts",
			keys:      []string{"y"},
			wantKept:  []string{"todo", "completed", "scrapped"},
			wantState: viewArchiveConfirm,
		},
		{
			name:      "list yy archives all completed",
			keys:      []string{"y", "y"},
			wantKept:  []string{"todo"},
			wantState: viewList,
		},
		{
			name:      "list y n cancels",
			keys:      []string{"y", "n"},
			wantKept:  []string{"todo", "completed", "scrapped"},
			wantState: viewList,
		},
		{
			name:      "detail y archives completed bean directly",
			openBean:  "completed",
			keys:      []string{"y"},
			wantKept:  []string{"todo", "scrapped"},
			wantState: viewList,
		},
		{
			name:      "detail y prompts for unfinished bean",
			openBean:  "todo",
			keys:      []string{"y"},
			wantKept:  []string{"todo", "completed", "scrapped"},
			wantState: viewArchiveConfirm,
		},
		{
			name:      "detail yy archives unfinished bean",
			openBean:  "todo",
			keys:      []string{"y", "y"},
			wantKept:  []string{"completed", "scrapped"},
			wantState: viewList,
		},
		{
			name:      "detail y esc cancels",
			openBean:  "todo",
			keys:      []string{"y", "esc"},
			wantKept:  []string{"todo", "completed", "scrapped"},
			wantState: viewDetail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, beans := newArchiveTestApp(t)
			if tt.openBean != "" {
				pump(app, selectBeanMsg{bean: beans[tt.openBean]})
			}
			for _, k := range tt.keys {
				pump(app, keyMsg(k))
			}

			got := remaining(app)
			if len(got) != len(tt.wantKept) {
				t.Errorf("remaining beans = %v, want %v", got, tt.wantKept)
			}
			for _, name := range tt.wantKept {
				if !got[beans[name].ID] {
					t.Errorf("bean %q was archived, want kept", name)
				}
			}
			if app.state != tt.wantState {
				t.Errorf("state = %v, want %v", app.state, tt.wantState)
			}
		})
	}
}
