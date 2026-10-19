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
			if !m.takeReveal() {
				t.Error("starting a run should show the pane")
			}
			if got := m.entries[0].state; got != runRunning {
				t.Errorf("state after start = %v, want running", got)
			}

			m.finish(id, tt.exitCode, tt.err, now.Add(time.Second))

			e := m.entries[0]
			if e.state != tt.wantState {
				t.Errorf("state = %v, want %v", e.state, tt.wantState)
			}
			if fmt.Sprint(e.tail) != fmt.Sprint(tt.wantTail) {
				t.Errorf("tail = %q, want %q", e.tail, tt.wantTail)
			}
			if wantReveal := tt.wantState == runFailed; m.takeReveal() != wantReveal {
				t.Errorf("reveal after finish = %v, want %v (failures show the pane)", !wantReveal, wantReveal)
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

func TestLogPanelCapKeepsRunningEntries(t *testing.T) {
	var m logPanelModel
	runningPath := writeLog(t, "")
	m.start("l", "beans-running", runningPath, time.Now())
	for i := range maxLogEntries {
		id := m.start("l", fmt.Sprintf("beans-%d", i), writeLog(t, "x\n"), time.Now())
		m.finish(id, 0, nil, time.Now())
	}

	if m.entries[0].beanID != "beans-running" {
		t.Errorf("oldest entry = %s, the running one should be kept", m.entries[0].beanID)
	}
	if len(m.entries) != maxLogEntries {
		t.Errorf("len(entries) = %d, want %d", len(m.entries), maxLogEntries)
	}
	if _, err := os.Stat(runningPath); err != nil {
		t.Errorf("running entry's log should be kept, stat err = %v", err)
	}
}

func TestTailLinesReadsOnlyTheEnd(t *testing.T) {
	path := writeLog(t, strings.Repeat("filler line\n", 10000)+"last one\nlast two\n")
	if got := tailLines(path, 2); fmt.Sprint(got) != "[last one last two]" {
		t.Errorf("tailLines = %q", got)
	}
	if got := tailLines(writeLog(t, strings.Repeat("y", tailReadBytes+10)+"\nend\n"), 3); fmt.Sprint(got) != "[end]" {
		t.Errorf("partial first line should be dropped, got %q", got)
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
	if view := m.View(60, 5); !strings.Contains(view, "No commands run yet") {
		t.Errorf("empty panel should say so:\n%s", view)
	}

	ok := m.start("tmux", "beans-ok", writeLog(t, "done\n"), now)
	m.finish(ok, 0, nil, now)
	failed := m.start("tmux", "beans-bad", writeLog(t, "boom\n"+long+"\n"), now)
	m.finish(failed, 3, nil, now)
	m.start("claude", "beans-run", writeLog(t, ""), now)

	view := m.View(60, 5)
	lines := strings.Split(view, "\n")
	if len(lines) != 5 {
		t.Errorf("panel is %d lines, want 5:\n%s", len(lines), view)
	}
	for _, s := range []string{"✗ tmux beans-bad (exit 3)", "boom", "⟳ claude beans-run"} {
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
	app.Update(keyMsg("L")) // hide the log

	_, cmd := app.Update(launcherSelectedMsg{
		launcher: launcher{name: "bg", exec: `echo "hi $BEANS_ID"; [ "$BEANS_ID" = "` + b.ID + `" ]`, background: true},
		beanIDs:  []string{b.ID, other},
	})

	if got := len(app.log.entries); got != 2 {
		t.Fatalf("entries = %d, want one per bean", got)
	}
	if !app.ws.isVisible(paneLog) {
		t.Error("the log should show when a run starts")
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
	withLog := app.list.height
	if !strings.Contains(app.View(), "No commands run yet") {
		t.Fatalf("the log should show by default:\n%s", app.View())
	}

	app.Update(keyMsg("L"))
	if app.ws.isVisible(paneLog) {
		t.Fatal("L should hide the log")
	}
	if got, want := app.list.height, withLog+8; got != want {
		t.Errorf("list height without log = %d, want %d", got, want)
	}

	app.Update(keyMsg("L"))
	view := app.View()
	if !app.ws.isVisible(paneLog) || app.list.height != withLog {
		t.Errorf("second L should show the log and restore height (height=%d)", app.list.height)
	}
	if got := lipgloss.Height(view); got != app.height {
		t.Errorf("view is %d lines, want the terminal's %d", got, app.height)
	}

	// Too short a terminal keeps the log out of the way
	app.Update(tea.WindowSizeMsg{Width: 120, Height: 12})
	if strings.Contains(app.View(), "No commands run yet") {
		t.Errorf("log should not render in a 12-row terminal:\n%s", app.View())
	}
}

func TestLogPanelCursor(t *testing.T) {
	now := time.Now()
	var m logPanelModel
	m.moveCursor(-1) // no entries: nothing to move
	if m.selectedEntry() != nil {
		t.Fatal("empty log should have no selection")
	}

	var ids []int
	for i := range 3 {
		ids = append(ids, m.start("l", fmt.Sprintf("beans-%d", i), "", now))
	}
	if got := m.selectedEntry().id; got != ids[2] {
		t.Errorf("selected = %d, want the latest", got)
	}

	m.moveCursor(-1)
	if got := m.selectedEntry().id; got != ids[1] {
		t.Errorf("after up: selected = %d, want %d", got, ids[1])
	}
	m.moveCursor(-10)
	if got := m.selectedEntry().id; got != ids[0] {
		t.Errorf("up stops at the oldest: selected = %d, want %d", got, ids[0])
	}

	// A new run doesn't move a cursor off the latest...
	newer := m.start("l", "beans-new", "", now)
	if got := m.selectedEntry().id; got != ids[0] {
		t.Errorf("selected = %d, want it to stay on %d", got, ids[0])
	}
	// ...but on the latest, the cursor follows new runs
	m.moveCursor(10)
	if got := m.selectedEntry().id; got != newer {
		t.Errorf("down stops at the latest: selected = %d, want %d", got, newer)
	}
	newest := m.start("l", "beans-newest", "", now)
	if got := m.selectedEntry().id; got != newest {
		t.Errorf("selected = %d, want to follow to %d", got, newest)
	}
}

func TestLogPanelViewScrollsToSelection(t *testing.T) {
	now := time.Now()
	var m logPanelModel
	for i := range 6 {
		id := m.start("l", fmt.Sprintf("beans-%d", i), writeLog(t, "a\nb\n"), now)
		m.finish(id, 0, nil, now)
	}

	m.moveCursor(-5) // the oldest
	view := m.View(60, 4)
	if !strings.Contains(view, "▌") || !strings.Contains(view, "beans-0") {
		t.Errorf("selected oldest entry should be shown with the cursor:\n%s", view)
	}
	if strings.Contains(view, "beans-5") {
		t.Errorf("the newest entry should have scrolled out:\n%s", view)
	}

	m.moveCursor(10)
	if view := m.View(60, 4); !strings.Contains(view, "beans-5") {
		t.Errorf("selected latest entry should be shown:\n%s", view)
	}
}

func TestLaunchClearsSelection(t *testing.T) {
	app, _ := newTestApp(t)
	t.Cleanup(app.log.cleanup)
	ids := listedIDs(app)
	app.list.selectedBeans[ids[0]] = true
	app.list.selectedBeans[ids[1]] = true

	app.Update(closeLauncherPickerMsg{})
	if got := len(app.list.selectedBeans); got != 2 {
		t.Fatalf("cancelling the picker cleared the selection, %d marked", got)
	}

	_, cmd := app.Update(launcherSelectedMsg{
		launcher: launcher{name: "bg", exec: "true", background: true},
		beanIDs:  ids[:2],
	})
	for _, msg := range runBatch(cmd) {
		app.Update(msg)
	}

	if got := len(app.list.selectedBeans); got != 0 {
		t.Errorf("selection after launch = %d beans, want none so the next launch targets the cursor", got)
	}
}
