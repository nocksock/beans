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

// logPanelModel is the lazygit-style command log pane. Its cursor picks the
// run whose output the context pane shows.
type logPanelModel struct {
	entries  []logEntry
	nextID   int
	reveal   bool // the pane should be shown: a run started or failed
	selected int  // id of the entry under the cursor; 0 follows the latest
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

// selectedIndex is the index of the entry under the cursor, or -1 if there
// are no entries. A dropped entry moves the cursor to the latest.
func (m *logPanelModel) selectedIndex() int {
	if i := slices.IndexFunc(m.entries, func(e logEntry) bool { return e.id == m.selected }); i >= 0 {
		return i
	}
	return len(m.entries) - 1
}

// selectedEntry returns the entry under the cursor, or nil if nothing has run yet.
func (m *logPanelModel) selectedEntry() *logEntry {
	if i := m.selectedIndex(); i >= 0 {
		return &m.entries[i]
	}
	return nil
}

// moveCursor moves the cursor delta entries down (newer), stopping at the
// ends. On the latest entry, the cursor follows new runs.
func (m *logPanelModel) moveCursor(delta int) {
	if len(m.entries) == 0 {
		return
	}
	i := max(0, min(m.selectedIndex()+delta, len(m.entries)-1))
	m.selected = m.entries[i].id
	if i == len(m.entries)-1 {
		m.selected = 0
	}
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

// View renders the log at exactly width x height: the newest lines that fit
// along with the selected entry.
func (m logPanelModel) View(width, height int) string {
	var lines []string
	selected := m.selectedIndex()
	selStart, selEnd := 0, 0 // lines of the selected entry
	for i, e := range m.entries {
		cursor := " "
		if i == selected {
			cursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("▌")
			selStart = len(lines)
		}
		lines = append(lines, cursor+e.header())
		style := ui.Muted
		if e.state == runFailed {
			style = lipgloss.NewStyle().Foreground(ui.ColorDanger)
		}
		for _, l := range e.tail {
			lines = append(lines, cursor+style.Render("  "+l))
		}
		if i == selected {
			selEnd = len(lines)
		}
	}
	if len(lines) == 0 {
		lines = []string{ui.Muted.Render("No commands run yet")}
	}

	// Newest at the bottom, scrolled up as far as needed to show the
	// selected entry (its header, at least)
	rows := max(height, 0)
	end := len(lines)
	if end-rows > selStart {
		end = max(min(selEnd, selStart+rows), rows)
	}
	lines = lines[max(end-rows, 0):end]
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
// most its last tailReadBytes.
func tailLines(path string, n int) []string {
	lines, _ := readLogLines(path, tailReadBytes)
	lines = slices.DeleteFunc(lines, func(l string) bool { return strings.TrimSpace(l) == "" })
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

// readLogLines returns the lines of the run output at path, reading at most
// its last maxBytes; truncated reports whether earlier output was skipped.
// For lines rewritten with carriage returns (progress bars), only the final
// text is kept. Escape sequences are removed, since cursor movements would
// garble the panes.
func readLogLines(path string, maxBytes int64) (lines []string, truncated bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, false
	}
	offset := max(info.Size()-maxBytes, 0)
	data := make([]byte, info.Size()-offset)
	if _, err := f.ReadAt(data, offset); err != nil && err != io.EOF {
		return nil, false
	}
	if offset > 0 {
		// Drop the partial first line
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			data = data[i+1:]
		}
	}

	text := strings.TrimSuffix(string(data), "\n")
	if text == "" {
		return nil, offset > 0
	}
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimRight(l, "\r")
		if i := strings.LastIndex(l, "\r"); i >= 0 {
			l = l[i+1:]
		}
		lines = append(lines, strings.ReplaceAll(ansi.Strip(l), "\t", "    "))
	}
	return lines, offset > 0
}
