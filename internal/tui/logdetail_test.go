package tui

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLogDetail(t *testing.T) {
	now := time.Now()
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = "line " + strings.Repeat("x", i%3)
	}
	lines[49] = "\x1b[31mlast\x1b[0m"

	var log logPanelModel
	id := log.start("tmux", "beans-x", writeLog(t, strings.Join(lines, "\n")+"\n"), now)
	log.finish(id, 2, nil, now)

	var m logDetailModel
	m.setSize(40, 10)
	m.setEntry(log.entry(id))

	if got := m.title(); got != "Output tmux beans-x" {
		t.Errorf("title = %q", got)
	}
	view := m.View()
	if got := strings.Count(view, "\n") + 1; got != 10 {
		t.Errorf("view is %d lines, want 10:\n%s", got, view)
	}
	if !strings.HasSuffix(view, "last") {
		t.Errorf("view should start at the end, with escapes removed:\n%q", view)
	}

	m.update(keyMsg("g"))
	view = m.View()
	if !strings.Contains(view, "✗ tmux beans-x (exit 2)") || !strings.Contains(view, "line") {
		t.Errorf("g should scroll to the header:\n%s", view)
	}
	if !strings.HasSuffix(view, "…") {
		t.Errorf("more output below should be marked:\n%s", view)
	}

	// Scrolled up, it stays put while the output grows
	m.setEntry(log.entry(id))
	if !strings.Contains(m.View(), "exit 2") {
		t.Error("re-reading the same run should keep the scroll position")
	}
}

func TestLogDetailFollowsRunningOutput(t *testing.T) {
	path := writeLog(t, "one\n")
	var log logPanelModel
	id := log.start("bg", "beans-x", path, time.Now())

	var m logDetailModel
	m.setSize(40, 3)
	m.setEntry(log.entry(id))

	if err := os.WriteFile(path, []byte("one\ntwo\nthree\nfour\n"), 0644); err != nil {
		t.Fatal(err)
	}
	m.setEntry(log.entry(id))
	if view := m.View(); !strings.HasSuffix(view, "four") {
		t.Errorf("at the end, new output should scroll in:\n%s", view)
	}
}

func TestLogDetailWithoutOutput(t *testing.T) {
	var log logPanelModel
	id := log.start("bad", "beans-x", "", time.Now())
	log.finish(id, -1, os.ErrNotExist, time.Now())

	var m logDetailModel
	m.setSize(40, 5)
	if view := m.View(); !strings.Contains(view, "No commands run yet") {
		t.Errorf("no run should say so:\n%s", view)
	}
	m.setEntry(log.entry(id))
	if view := m.View(); !strings.Contains(view, os.ErrNotExist.Error()) {
		t.Errorf("a run that couldn't start should show its error:\n%s", view)
	}
}

func TestReadLogLinesTruncates(t *testing.T) {
	path := writeLog(t, strings.Repeat("filler\n", 100)+"end\n")
	lines, truncated := readLogLines(path, 20)
	if !truncated || lines[len(lines)-1] != "end" || len(lines) > 3 {
		t.Errorf("lines = %q, truncated = %v", lines, truncated)
	}
	lines, truncated = readLogLines(path, 1<<20)
	if truncated || len(lines) != 101 {
		t.Errorf("whole file: %d lines, truncated = %v", len(lines), truncated)
	}
}

func TestLogPaneDrivesContext(t *testing.T) {
	app, b := newTestApp(t)
	t.Cleanup(app.log.cleanup)
	_, cmd := app.Update(launcherSelectedMsg{
		launcher: launcher{name: "bg", exec: `echo "out of $BEANS_ID"`, background: true},
		beanIDs:  []string{b.ID},
	})
	for _, msg := range runBatch(cmd) {
		app.Update(msg)
	}

	pump(app, keyMsg("tab"))
	pump(app, keyMsg("tab"))
	if app.ws.focus != paneLog {
		t.Fatalf("focus = %v, want the log", app.ws.focus)
	}
	view := app.View()
	if !strings.Contains(view, "Output bg "+b.ID) || !strings.Contains(view, "out of "+b.ID) {
		t.Errorf("the context should show the run's output:\n%s", view)
	}

	pump(app, keyMsg("enter"))
	if app.ws.focus != paneContext || !app.ctxPane.showRun {
		t.Errorf("enter should focus the output (focus %v)", app.ws.focus)
	}
	pump(app, keyMsg("esc"))
	if app.ws.focus != paneLog {
		t.Errorf("esc should go back to the log, focus = %v", app.ws.focus)
	}

	pump(app, keyMsg("+"))
	if got := app.ws.visiblePanes(); len(got) != 2 || app.ws.isVisible(paneList) {
		t.Errorf("+ on the log should show the output wide, visible = %v", got)
	}

	pump(app, keyMsg("-"))
	pump(app, keyMsg("tab"))
	if app.ws.focus != paneList {
		t.Fatalf("focus = %v, want the list", app.ws.focus)
	}
	if app.ctxPane.showRun {
		t.Error("focusing the list should show its bean again")
	}
}
