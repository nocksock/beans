package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/internal/ui"
)

// closeHelpMsg is sent when the keybinds dialog is closed.
type closeHelpMsg struct{}

// runKeybindMsg is sent when a keybind is chosen in the dialog. The app
// replays key as if it had been pressed in the focused pane.
type runKeybindMsg struct {
	key string
}

// helpDescStyle renders shortcut descriptions in the terminal's default
// foreground, so they stay readable on both light and dark backgrounds.
var helpDescStyle = lipgloss.NewStyle()

// keybindSection is a titled group of bindings shown in the dialog.
type keybindSection struct {
	title    string
	bindings []key.Binding
}

// keybindRow is one line of the dialog: a section header, or a binding with
// its key label, description and the key to replay.
type keybindRow struct {
	section string // header rows only
	keys    string // all keys, for display ("esc/bksp")
	desc    string
	key     string // the first key, replayed when run
}

func (r keybindRow) header() bool { return r.section != "" }

// keybindRows flattens the sections into rows, skipping unbound actions and
// sections without any.
func keybindRows(sections []keybindSection) []keybindRow {
	var rows []keybindRow
	for _, s := range sections {
		start := len(rows)
		rows = append(rows, keybindRow{section: s.title})
		for _, b := range s.bindings {
			if len(b.Keys()) == 0 {
				continue
			}
			rows = append(rows, keybindRow{keys: keyLabel(b.Keys()), desc: b.Help().Desc, key: b.Keys()[0]})
		}
		if len(rows) == start+1 {
			rows = rows[:start]
		}
	}
	return rows
}

// filterRows keeps the rows whose key label or description contains query
// (case-insensitive), and the headers of sections that still have rows.
func filterRows(rows []keybindRow, query string) []keybindRow {
	query = strings.ToLower(query)
	var out []keybindRow
	var header *keybindRow
	for _, r := range rows {
		if r.header() {
			header = &r
			continue
		}
		if !strings.Contains(strings.ToLower(r.keys), query) && !strings.Contains(strings.ToLower(r.desc), query) {
			continue
		}
		if header != nil {
			out = append(out, *header)
			header = nil
		}
		out = append(out, r)
	}
	return out
}

// keybindsModel is the dialog listing the keys available in the focused pane,
// filterable, with a cursor to run one.
type keybindsModel struct {
	rows    []keybindRow // all rows
	visible []keybindRow // rows matching the filter
	cursor  int          // index into visible; never on a header
	filter  textinput.Model
	scroll  scroller
	width   int
	height  int
}

func newKeybindsModel(sections []keybindSection, width, height int) keybindsModel {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = "filter"
	ti.CharLimit = 50
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(ui.ColorMuted)

	m := keybindsModel{rows: keybindRows(sections), filter: ti}
	m.setSize(width, height)
	m.applyFilter()
	return m
}

func (m keybindsModel) Init() tea.Cmd {
	return nil
}

// typing reports whether keys go to the filter input.
func (m keybindsModel) typing() bool {
	return m.filter.Focused()
}

// selected returns the row under the cursor, if any.
func (m keybindsModel) selected() (keybindRow, bool) {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return keybindRow{}, false
	}
	return m.visible[m.cursor], true
}

// applyFilter recomputes the visible rows and puts the cursor on the first.
func (m *keybindsModel) applyFilter() {
	m.visible = filterRows(m.rows, m.filter.Value())
	m.cursor = -1
	m.moveCursor(1)
	m.scroll.offset = 0
}

// moveCursor moves the cursor by delta rows, skipping headers and stopping
// at the first or last binding.
func (m *keybindsModel) moveCursor(delta int) {
	step := 1
	if delta < 0 {
		step, delta = -1, -delta
	}
	for ; delta > 0; delta-- {
		next := m.cursor + step
		for next >= 0 && next < len(m.visible) && m.visible[next].header() {
			next += step
		}
		if next < 0 || next >= len(m.visible) {
			return
		}
		m.cursor = next
	}
}

// setSize sizes the dialog for the screen.
func (m *keybindsModel) setSize(width, height int) {
	m.width, m.height = width, height
	m.filter.Width = m.modalWidth() - 8
	m.scroll.height = max(3, min(len(m.rows), height-12))
}

func (m keybindsModel) modalWidth() int {
	return calculateModalWidth(m.width, 60, 50, 70)
}

func (m keybindsModel) Update(msg tea.Msg) (keybindsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.setSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		return m.updateKey(msg)
	}
	return m, nil
}

func (m keybindsModel) updateKey(msg tea.KeyMsg) (keybindsModel, tea.Cmd) {
	k := keymap.Help
	closeCmd := func() tea.Msg { return closeHelpMsg{} }

	if m.typing() {
		switch {
		case key.Matches(msg, k.Run):
			return m, m.run()
		case msg.Type == tea.KeyEsc:
			m.filter.SetValue("")
			m.filter.Blur()
			m.applyFilter()
			return m, nil
		case msg.Type == tea.KeyUp:
			m.moveCursor(-1)
			return m, nil
		case msg.Type == tea.KeyDown:
			m.moveCursor(1)
			return m, nil
		}
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.applyFilter()
		return m, cmd
	}

	switch {
	case key.Matches(msg, k.Close):
		return m, closeCmd
	case key.Matches(msg, k.Filter):
		return m, m.filter.Focus()
	case key.Matches(msg, k.Run):
		return m, m.run()
	case key.Matches(msg, k.Up):
		m.moveCursor(-1)
	case key.Matches(msg, k.Down):
		m.moveCursor(1)
	case key.Matches(msg, k.GoToStart):
		m.moveCursor(-len(m.visible))
	case key.Matches(msg, k.GoToEnd):
		m.moveCursor(len(m.visible))
	}
	return m, nil
}

// run returns the command running the selected row, or nil if there is none.
func (m keybindsModel) run() tea.Cmd {
	r, ok := m.selected()
	if !ok {
		return nil
	}
	return func() tea.Msg { return runKeybindMsg{key: r.key} }
}

func (m keybindsModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	modalWidth := m.modalWidth()
	innerWidth := modalWidth - 4 // padding

	title := listTitleStyle.Render("Keybinds")
	keyStyle := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Width(12).MarginRight(1)
	cursorStyle := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorMuted)

	lines := make([]string, 0, len(m.visible))
	for i, r := range m.visible {
		var line string
		switch {
		case r.header():
			line = headerStyle.Render(r.section)
		case i == m.cursor:
			line = cursorStyle.Render("▌ ") + keyStyle.Render(r.keys) + helpDescStyle.Render(r.desc)
		default:
			line = "  " + keyStyle.Render(r.keys) + helpDescStyle.Render(r.desc)
		}
		lines = append(lines, ansi.Truncate(line, innerWidth, "…"))
	}
	if len(lines) == 0 {
		lines = []string{ui.Muted.Render("no matching keybinds")}
	}

	scroll := m.scroll
	scroll.scrollTo(m.cursor, len(lines))
	body := strings.Join(scroll.window(lines), "\n")

	k := keymap.Help
	footer := hints(hint(k.Run, ""), hint(k.Filter, ""), hint(k.Close, ""))

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary).
		Padding(1, 2).
		Width(modalWidth)

	return border.Render(title + "\n" + m.filter.View() + "\n\n" + body + "\n\n" + footer)
}

// ModalView returns the dialog as a centered modal on top of the background.
func (m keybindsModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	return overlayModal(bgView, m.View(), fullWidth, fullHeight)
}
