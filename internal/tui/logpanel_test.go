package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func writeLog(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "launch.log")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLogPanelLifecycle(t *testing.T) {
	now := time.Date(2026, 10, 17, 15, 4, 0, 0, time.UTC)

	tests := []struct {
		name      string
		output    string
		exitCode  int
		err       error
		wantState runState
		wantTail  []string
	}{
		{name: "success", output: "one\ntwo\nthree\nfour\n", wantState: runOK, wantTail: []string{"two", "three", "four"}},
		{name: "failure", output: "boom\n", exitCode: 3, wantState: runFailed, wantTail: []string{"boom"}},
		{name: "not startable", exitCode: -1, err: errors.New("chdir: no such dir"), wantState: runFailed, wantTail: []string{"chdir: no such dir"}},
		{name: "carriage returns keep last segment", output: "10%\r50%\r100%\n", wantState: runOK, wantTail: []string{"100%"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m logPanelModel
			id := m.start("tmux", "beans-x", writeLog(t, tt.output), now)
			if !m.visible {
				t.Error("starting a run should show the panel")
			}
			if got := m.entries[0].state; got != runRunning {
				t.Errorf("state after start = %v, want running", got)
			}

			m.visible = false
			m.finish(id, tt.exitCode, tt.err, now.Add(time.Second))

			e := m.entries[0]
			if e.state != tt.wantState {
				t.Errorf("state = %v, want %v", e.state, tt.wantState)
			}
			if fmt.Sprint(e.tail) != fmt.Sprint(tt.wantTail) {
				t.Errorf("tail = %q, want %q", e.tail, tt.wantTail)
			}
			if wantVisible := tt.wantState == runFailed; m.visible != wantVisible {
				t.Errorf("visible after finish = %v, want %v (failures show the panel)", m.visible, wantVisible)
			}
		})
	}
}

func TestLogPanelCapsEntries(t *testing.T) {
	var m logPanelModel
	var firstPath string
	for i := range maxLogEntries + 5 {
		path := writeLog(t, "x\n")
		if i == 0 {
			firstPath = path
		}
		id := m.start("l", fmt.Sprintf("beans-%d", i), path, time.Now())
		m.finish(id, 0, nil, time.Now())
	}

	if len(m.entries) != maxLogEntries {
		t.Fatalf("len(entries) = %d, want %d", len(m.entries), maxLogEntries)
	}
	if got := m.entries[0].beanID; got != "beans-5" {
		t.Errorf("oldest kept entry = %s, want beans-5", got)
	}
	if _, err := os.Stat(firstPath); !os.IsNotExist(err) {
		t.Errorf("evicted entry's log file should be removed, stat err = %v", err)
	}
}

func TestLogPanelCleanupKeepsRunningLogs(t *testing.T) {
	var m logPanelModel
	donePath, runningPath := writeLog(t, "x\n"), writeLog(t, "y\n")
	id := m.start("l", "beans-a", donePath, time.Now())
	m.finish(id, 0, nil, time.Now())
	m.start("l", "beans-b", runningPath, time.Now())

	m.cleanup()

	if _, err := os.Stat(donePath); !os.IsNotExist(err) {
		t.Errorf("finished run's log should be removed, stat err = %v", err)
	}
	if _, err := os.Stat(runningPath); err != nil {
		t.Errorf("running run's log should be kept, stat err = %v", err)
	}
}

func TestLogPanelView(t *testing.T) {
	now := time.Date(2026, 10, 17, 15, 4, 0, 0, time.UTC)
	long := strings.Repeat("x", 200)

	var m logPanelModel
	if view := m.View(60); !strings.Contains(view, "No commands run yet") {
		t.Errorf("empty panel should say so:\n%s", view)
	}

	ok := m.start("tmux", "beans-ok", writeLog(t, "done\n"), now)
	m.finish(ok, 0, nil, now)
	failed := m.start("tmux", "beans-bad", writeLog(t, "boom\n"+long+"\n"), now)
	m.finish(failed, 3, nil, now)
	m.start("claude", "beans-run", writeLog(t, ""), now)

	view := m.View(60)
	lines := strings.Split(view, "\n")
	if len(lines) != logPanelHeight {
		t.Errorf("panel is %d lines, want %d:\n%s", len(lines), logPanelHeight, view)
	}
	for _, s := range []string{"Log", "✗ tmux beans-bad (exit 3)", "boom", "⟳ claude beans-run"} {
		if !strings.Contains(view, s) {
			t.Errorf("view missing %q:\n%s", s, view)
		}
	}
	// Only the newest lines fit; the oldest entry scrolls out
	if strings.Contains(view, "beans-ok") {
		t.Errorf("oldest entry should have scrolled out:\n%s", view)
	}
	for _, line := range lines {
		if w := lipgloss.Width(line); w > 60 {
			t.Errorf("line exceeds width (%d > 60): %q", w, line)
		}
	}
}

// runBatch executes cmd, flattening tea.Batch, and returns the produced messages.
func runBatch(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			msgs = append(msgs, runBatch(c)...)
		}
		return msgs
	}
	return []tea.Msg{msg}
}

func TestBackgroundLaunchLogsEachBean(t *testing.T) {
	app, b := newTestApp(t)
	other := listedIDs(app)[0]
	if other == b.ID {
		other = listedIDs(app)[1]
	}
	t.Cleanup(app.log.cleanup)

	_, cmd := app.Update(launcherSelectedMsg{
		launcher: launcher{name: "bg", exec: `echo "hi $BEANS_ID"; [ "$BEANS_ID" = "` + b.ID + `" ]`, background: true},
		beanIDs:  []string{b.ID, other},
	})

	if got := len(app.log.entries); got != 2 {
		t.Fatalf("entries = %d, want one per bean", got)
	}
	if !app.log.visible {
		t.Error("panel should show when a run starts")
	}
	if app.quitting {
		t.Error("background runs must not hand over the terminal")
	}

	for _, msg := range runBatch(cmd) {
		app.Update(msg)
	}

	byBean := map[string]logEntry{}
	for _, e := range app.log.entries {
		byBean[e.beanID] = e
	}
	if e := byBean[b.ID]; e.state != runOK || fmt.Sprint(e.tail) != "[hi "+b.ID+"]" {
		t.Errorf("%s: state %v tail %q, want ok with output", b.ID, e.state, e.tail)
	}
	if e := byBean[other]; e.state != runFailed || e.exitCode != 1 {
		t.Errorf("%s: state %v exit %d, want failed with exit 1", other, e.state, e.exitCode)
	}
	if view := app.View(); !strings.Contains(view, "✗ bg "+other+" (exit 1)") {
		t.Errorf("overview should show the log panel:\n%s", view)
	}
}

func TestLogPanelToggle(t *testing.T) {
	app, _ := newTestApp(t)
	fullHeight := app.list.height

	app.Update(keyMsg("L"))
	if !app.log.visible {
		t.Fatal("L should show the panel")
	}
	if got, want := app.list.height, fullHeight-logPanelHeight; got != want {
		t.Errorf("list height with panel = %d, want %d", got, want)
	}
	view := app.View()
	if got := lipgloss.Height(view); got > app.height {
		t.Errorf("view is %d lines, taller than terminal (%d)", got, app.height)
	}
	if !strings.Contains(view, "No commands run yet") {
		t.Errorf("view should contain the panel:\n%s", view)
	}

	app.Update(keyMsg("L"))
	if app.log.visible || app.list.height != fullHeight {
		t.Errorf("second L should hide the panel and restore height (visible=%v, height=%d)", app.log.visible, app.list.height)
	}

	// Too short a terminal keeps the panel out of the way
	app.Update(keyMsg("L"))
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 10})
	if app.list.height != 10 || strings.Contains(app.View(), "No commands run yet") {
		t.Errorf("panel should not render in a 10-row terminal (list height %d)", app.list.height)
	}
}
