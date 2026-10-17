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

// unarchived returns the IDs of beans not in the archived status.
func unarchived(a *App) map[string]bool {
	ids := map[string]bool{}
	for _, b := range a.core.All() {
		if b.Status != config.StatusArchived {
			ids[b.ID] = true
		}
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
			name:      "list Y prompts",
			keys:      []string{"Y"},
			wantKept:  []string{"todo", "completed", "scrapped"},
			wantState: viewArchiveConfirm,
		},
		{
			name:      "list YY archives all completed",
			keys:      []string{"Y", "Y"},
			wantKept:  []string{"todo"},
			wantState: viewList,
		},
		{
			name:      "list Y n cancels",
			keys:      []string{"Y", "n"},
			wantKept:  []string{"todo", "completed", "scrapped"},
			wantState: viewList,
		},
		{
			name:      "detail Y archives completed bean directly",
			openBean:  "completed",
			keys:      []string{"Y"},
			wantKept:  []string{"todo", "scrapped"},
			wantState: viewList,
		},
		{
			name:      "detail Y prompts for unfinished bean",
			openBean:  "todo",
			keys:      []string{"Y"},
			wantKept:  []string{"todo", "completed", "scrapped"},
			wantState: viewArchiveConfirm,
		},
		{
			name:      "detail YY archives unfinished bean",
			openBean:  "todo",
			keys:      []string{"Y", "Y"},
			wantKept:  []string{"completed", "scrapped"},
			wantState: viewList,
		},
		{
			name:      "detail Y esc cancels",
			openBean:  "todo",
			keys:      []string{"Y", "esc"},
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

			if n := len(app.core.All()); n != len(beans) {
				t.Errorf("bean count = %d, want %d (archiving must not delete)", n, len(beans))
			}
			got := unarchived(app)
			if len(got) != len(tt.wantKept) {
				t.Errorf("unarchived beans = %v, want %v", got, tt.wantKept)
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

func TestArchiveLowercaseYIgnored(t *testing.T) {
	app, _ := newArchiveTestApp(t)
	pump(app, keyMsg("y"))
	if app.state != viewList {
		t.Errorf("state = %v, want list (lowercase y must not archive)", app.state)
	}
}

func TestArchivedHiddenInList(t *testing.T) {
	app, beans := newArchiveTestApp(t)
	pump(app, keyMsg("Y"))
	pump(app, keyMsg("Y"))

	listed := func() map[string]bool {
		ids := map[string]bool{}
		for _, id := range listedIDs(app) {
			ids[id] = true
		}
		return ids
	}

	got := listed()
	if !got[beans["todo"].ID] || got[beans["completed"].ID] || got[beans["scrapped"].ID] {
		t.Errorf("listed = %v, want only todo bean (archived hidden)", got)
	}

	// A shows archived beans
	pump(app, keyMsg("A"))
	got = listed()
	if !got[beans["completed"].ID] || !got[beans["scrapped"].ID] {
		t.Errorf("after A: listed = %v, want archived beans shown", got)
	}

	// A again hides them
	pump(app, keyMsg("A"))
	if got = listed(); got[beans["completed"].ID] {
		t.Errorf("after AA: listed = %v, want archived beans hidden", got)
	}
}

func TestArchiveSkipsArchivedBeans(t *testing.T) {
	app, beans := newArchiveTestApp(t)
	pump(app, keyMsg("Y"))
	pump(app, keyMsg("Y"))

	// Nothing left to archive: Y does not prompt
	pump(app, keyMsg("Y"))
	if app.state != viewList {
		t.Errorf("state = %v, want list (no archivable beans)", app.state)
	}

	// Y on an archived bean in the detail view is a no-op
	pump(app, keyMsg("A"))
	b, _ := app.core.Get(beans["completed"].ID)
	pump(app, selectBeanMsg{bean: b})
	pump(app, keyMsg("Y"))
	if app.state != viewDetail {
		t.Errorf("state = %v, want detail (already archived)", app.state)
	}
}
