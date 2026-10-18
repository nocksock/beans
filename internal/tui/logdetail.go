package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/internal/ui"
)

// logDetailBytes is how much of the end of a run's output the context pane reads
const logDetailBytes = 1 << 20

// openLogMsg asks the app to open a run's output in the user's editor
type openLogMsg struct {
	path string
}

// logDetailModel is the context pane showing the full output of the run
// selected in the log. It starts at the end of the output and stays there
// while a running command writes more.
type logDetailModel struct {
	entry     *logEntry // nil if nothing has run yet
	output    []string  // the output's lines, read on setEntry
	truncated bool      // earlier output wasn't read
	readSize  int64     // size of the output file when it was read
	scroll    scroller
	width     int

	// Wrapped lines cache, rebuilt when the output or width changes
	wrapped      []string
	wrappedWidth int
}

// setEntry shows e (or nothing, if nil), re-reading its output if it changed.
func (m *logDetailModel) setEntry(e *logEntry) {
	other := e == nil || m.entry == nil || e.id != m.entry.id
	size := logSize(e)
	if !other && e.state == m.entry.state && size == m.readSize {
		return
	}
	follow := other || m.scroll.atEnd(len(m.lines()))

	var c *logEntry
	if e != nil {
		copied := *e
		c = &copied
	}
	m.entry = c
	m.output, m.truncated = nil, false
	if e != nil && e.logPath != "" {
		m.output, m.truncated = readLogLines(e.logPath, logDetailBytes)
	}
	m.readSize = size
	m.wrapped = nil

	if follow {
		m.scroll.offset = len(m.lines())
	}
	m.scroll.clamp(len(m.lines()))
}

// logSize is the size of e's output file, or -1 if there is none.
func logSize(e *logEntry) int64 {
	if e == nil || e.logPath == "" {
		return -1
	}
	info, err := os.Stat(e.logPath)
	if err != nil {
		return -1
	}
	return info.Size()
}

func (m *logDetailModel) setSize(width, height int) {
	follow := m.scroll.atEnd(len(m.lines()))
	if width != m.width {
		m.wrapped = nil
	}
	m.width, m.scroll.height = width, height
	if follow {
		m.scroll.offset = len(m.lines())
	}
	m.scroll.clamp(len(m.lines()))
}

// title is the pane title.
func (m *logDetailModel) name() string { return "Output" }

func (m *logDetailModel) title() string {
	if m.entry == nil {
		return "Output"
	}
	return "Output " + m.entry.launcher + " " + m.entry.beanID
}

func (m *logDetailModel) update(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	k := keymap.Context
	switch {
	case key.Matches(keyMsg, k.Back):
		return func() tea.Msg { return focusBackMsg{} }
	case key.Matches(keyMsg, k.OpenEditor):
		if m.entry != nil && m.entry.logPath != "" {
			path := m.entry.logPath
			return func() tea.Msg { return openLogMsg{path: path} }
		}
	default:
		m.scroll.scrollKey(keyMsg, len(m.lines()))
	}
	return nil
}

// lines renders the run's header and output, wrapped to the pane width.
func (m *logDetailModel) lines() []string {
	if m.entry == nil || m.width <= 0 {
		return nil
	}
	if m.wrapped != nil && m.wrappedWidth == m.width {
		return m.wrapped
	}

	lines := []string{m.entry.header(), ""}
	output := m.output
	if len(output) == 0 {
		// Nothing captured; a run that couldn't start has its error here
		output = m.entry.tail
	}
	switch {
	case len(output) == 0 && m.entry.state == runRunning:
		lines = append(lines, ui.Muted.Render("No output yet"))
	case len(output) == 0:
		lines = append(lines, ui.Muted.Render("No output"))
	case m.truncated:
		lines = append(lines, ui.Muted.Render("… earlier output not shown"))
	}
	for _, l := range output {
		lines = append(lines, strings.Split(ansi.Wrap(l, m.width, ""), "\n")...)
	}

	m.wrapped, m.wrappedWidth = lines, m.width
	return lines
}

// View renders the visible part of the output at exactly the set size.
func (m *logDetailModel) View() string {
	if m.entry == nil {
		return centeredMuted("No commands run yet", m.width, m.scroll.height)
	}
	return strings.Join(m.scroll.window(m.lines()), "\n")
}

// footer renders the hints for the focused pane.
func (m *logDetailModel) footer() string {
	k := keymap.Context
	return m.scroll.percentHint(len(m.lines())) + "  " + hints(
		groupHint("scroll", k.ScrollDown, k.ScrollUp),
		hint(k.OpenEditor, ""),
		hint(k.Back, ""),
	)
}

func (m *logDetailModel) typing() bool { return false }

func (m *logDetailModel) bindings() []key.Binding {
	k := keymap.Context
	return []key.Binding{k.ScrollUp, k.ScrollDown, k.PageUp, k.PageDown, k.HalfPageUp, k.HalfPageDown, k.GoToStart, k.GoToEnd, k.Back, k.OpenEditor}
}
