package tui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/internal/ui"
)

const (
	maxLogEntries = 50       // runs kept in memory; older ones are dropped
	logTailLines  = 3        // output lines shown per run
	tailReadBytes = 16 << 10 // only this much of the end of a run's output is read for its tail
)

// runState is the lifecycle state of a launcher run
type runState int

const (
	runRunning runState = iota
	runOK
	runFailed
)

// logEntry is one launcher run shown in the log panel
type logEntry struct {
	id       int
	launcher string
	beanID   string
	started  time.Time
	finished time.Time
	state    runState
	exitCode int
	logPath  string   // captured output (stdout+stderr in the background, stderr in the foreground)
	tail     []string // last lines of output, read when the run finishes
}

// launchFinishedMsg reports that the run with the given log entry id exited
type launchFinishedMsg struct {
	id       int
	exitCode int
	err      error // set when the command couldn't be run at all
}

// logPanelModel is the lazygit-style command log pane
type logPanelModel struct {
	entries []logEntry
	nextID  int
	reveal  bool // the pane should be shown: a run started or failed
}

// start records a new running entry, asks for the pane to be shown and
// returns the entry's id.
func (m *logPanelModel) start(launcher, beanID, logPath string, now time.Time) int {
	m.nextID++
	m.entries = append(m.entries, logEntry{
		id:       m.nextID,
		launcher: launcher,
		beanID:   beanID,
		started:  now,
		state:    runRunning,
		logPath:  logPath,
	})
	// Drop the oldest finished runs; running ones stay until they finish
	for over := len(m.entries) - maxLogEntries; over > 0; over-- {
		i := slices.IndexFunc(m.entries, func(e logEntry) bool { return e.state != runRunning })
		if i < 0 {
			break
		}
		m.entries[i].removeLog()
		m.entries = slices.Delete(m.entries, i, i+1)
	}
	m.reveal = true
	return m.nextID
}

// finish marks the entry as done, reads its output tail and asks for the pane
// to be shown on failure.
func (m *logPanelModel) finish(id, exitCode int, err error, now time.Time) {
	e := m.entry(id)
	if e == nil {
		return
	}
	e.finished = now
	e.exitCode = exitCode
	e.state = runOK
	if exitCode != 0 {
		e.state = runFailed
		m.reveal = true
	}
	e.tail = tailLines(e.logPath, logTailLines)
	if exitCode == -1 && err != nil {
		e.tail = append(e.tail, err.Error())
	}
}

func (m *logPanelModel) entry(id int) *logEntry {
	for i := range m.entries {
		if m.entries[i].id == id {
			return &m.entries[i]
		}
	}
	return nil
}

// latest returns the most recent entry, or nil if nothing has run yet.
func (m *logPanelModel) latest() *logEntry {
	if len(m.entries) == 0 {
		return nil
	}
	return &m.entries[len(m.entries)-1]
}

// cleanup removes the log files of finished runs. Running ones are left alone,
// since their processes outlive the TUI and still write to them.
func (m *logPanelModel) cleanup() {
	for _, e := range m.entries {
		e.removeLog()
	}
}

func (e logEntry) removeLog() {
	if e.state != runRunning && e.logPath != "" {
		os.Remove(e.logPath)
	}
}

// takeReveal reports whether the pane should be shown, and clears the request.
func (m *logPanelModel) takeReveal() bool {
	r := m.reveal
	m.reveal = false
	return r
}

// View renders the newest log lines at exactly width x height.
func (m logPanelModel) View(width, height int) string {
	var lines []string
	for _, e := range m.entries {
		lines = append(lines, e.header())
		style := ui.Muted
		if e.state == runFailed {
			style = lipgloss.NewStyle().Foreground(ui.ColorDanger)
		}
		for _, l := range e.tail {
			lines = append(lines, style.Render("  "+l))
		}
	}
	if len(lines) == 0 {
		lines = []string{ui.Muted.Render("No commands run yet")}
	}

	// Newest at the bottom: keep only the lines that fit
	rows := max(height, 0)
	if len(lines) > rows {
		lines = lines[len(lines)-rows:]
	}
	for len(lines) < rows {
		lines = append(lines, "")
	}
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, width, "…")
	}

	return strings.Join(lines, "\n")
}

// header renders the entry's status line, e.g. "15:04 ✗ tmux beans-x (exit 3)".
func (e logEntry) header() string {
	icon, style := "⟳", lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	switch e.state {
	case runOK:
		icon, style = "✓", lipgloss.NewStyle().Foreground(ui.ColorSuccess)
	case runFailed:
		icon, style = "✗", lipgloss.NewStyle().Foreground(ui.ColorDanger)
	}
	text := fmt.Sprintf("%s %s %s", icon, e.launcher, e.beanID)
	if e.state == runFailed {
		text += fmt.Sprintf(" (exit %d)", e.exitCode)
	}
	return ui.Muted.Render(e.started.Format("15:04")) + " " + style.Render(text)
}

// tailLines returns the last n non-empty lines of the file at path, reading at
// most its last tailReadBytes. For lines rewritten with carriage returns
// (progress bars), only the final text is kept.
func tailLines(path string, n int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil
	}
	offset := max(info.Size()-tailReadBytes, 0)
	data := make([]byte, info.Size()-offset)
	if _, err := f.ReadAt(data, offset); err != nil && err != io.EOF {
		return nil
	}
	if offset > 0 {
		// Drop the partial first line
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			data = data[i+1:]
		}
	}

	var lines []string
	for _, l := range strings.Split(string(data), "\n") {
		l = strings.TrimRight(l, "\r")
		if i := strings.LastIndex(l, "\r"); i >= 0 {
			l = l[i+1:]
		}
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}
