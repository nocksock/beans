package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/ui"
)

// openDepNavMsg requests opening the dependency navigator on a bean
type openDepNavMsg struct {
	bean *bean.Bean
}

// closeDepNavMsg is sent when the dependency navigator is closed
type closeDepNavMsg struct{}

// depRow is one line of the dependency navigator.
type depRow struct {
	bean   *bean.Bean // nil for headings and empty-section lines
	text   string     // heading or empty-section text
	prefix string     // tree prefix
	repeat bool       // already shown above in the same tree, not expanded again
	cycle  bool       // closes a cycle, not expanded
	root   bool       // the bean the navigator is focused on
	empty  bool       // placeholder for a section without beans
}

func (r depRow) selectable() bool { return r.bean != nil }

// buildDepTree returns the rows of the tree of beans reachable from root via
// next, root excluded. Beans for which visible returns false are left out,
// together with what lies behind them.
func buildDepTree(root *bean.Bean, next func(*bean.Bean) []*bean.Bean, visible func(*bean.Bean) bool) []depRow {
	var rows []depRow
	seen := map[string]bool{root.ID: true}
	path := map[string]bool{root.ID: true}

	var walk func(b *bean.Bean, ancestry string)
	walk = func(b *bean.Bean, ancestry string) {
		var children []*bean.Bean
		for _, c := range next(b) {
			if visible(c) {
				children = append(children, c)
			}
		}
		for i, c := range children {
			branch, cont := "├─ ", "│  "
			if i == len(children)-1 {
				branch, cont = "└─ ", "   "
			}
			row := depRow{bean: c, prefix: ancestry + branch}
			switch {
			case path[c.ID]:
				row.cycle = true
			case seen[c.ID]:
				row.repeat = true
			}
			rows = append(rows, row)
			if row.cycle || row.repeat {
				continue
			}
			seen[c.ID] = true
			path[c.ID] = true
			walk(c, ancestry+cont)
			delete(path, c.ID)
		}
	}
	walk(root, "")
	return rows
}

// depNavModel shows the beans blocking a bean and the beans it blocks, both
// transitively, as trees.
type depNavModel struct {
	root     *bean.Bean
	resolver *graph.Resolver
	config   *config.Config
	showDone bool
	rows     []depRow
	cursor   int // index into rows, always a selectable row
	offset   int // first visible row
	width    int
	height   int
}

func newDepNavModel(root *bean.Bean, resolver *graph.Resolver, cfg *config.Config, width, height int) depNavModel {
	m := depNavModel{resolver: resolver, config: cfg, width: width, height: height}
	m.focus(root)
	return m
}

// focus rebuilds the navigator around b, with the cursor on b.
func (m *depNavModel) focus(b *bean.Bean) {
	m.root = b
	m.rebuild()
	for i, r := range m.rows {
		if r.root {
			m.cursor = i
		}
	}
	m.offset = 0
	m.scrollToCursor()
}

// reload re-reads the beans, keeping the cursor on the same bean if possible.
// It returns false if the focused bean is gone.
func (m *depNavModel) reload() bool {
	root, err := m.resolver.Query().Bean(context.Background(), m.root.ID)
	if err != nil || root == nil {
		return false
	}
	selected := m.selectedBean()
	m.root = root
	m.rebuild()
	m.cursor = 0
	for i, r := range m.rows {
		if r.root || (selected != nil && r.bean != nil && r.bean.ID == selected.ID && !r.repeat && !r.cycle) {
			m.cursor = i
			if !r.root {
				break
			}
		}
	}
	m.scrollToCursor()
	return true
}

func (m *depNavModel) rebuild() {
	ctx := context.Background()
	br := m.resolver.Bean()
	blockers := func(b *bean.Bean) []*bean.Bean {
		bs, _ := br.BlockedBy(ctx, b, nil)
		return bs
	}
	blocked := func(b *bean.Bean) []*bean.Bean {
		bs, _ := br.Blocking(ctx, b, nil)
		return bs
	}
	visible := func(b *bean.Bean) bool {
		return m.showDone || !m.config.IsArchiveStatus(b.Status)
	}

	section := func(title string, rows []depRow) []depRow {
		out := []depRow{{text: title}}
		if len(rows) == 0 {
			return append(out, depRow{text: "none", empty: true})
		}
		return append(out, rows...)
	}
	m.rows = nil
	m.rows = append(m.rows, section("blocked by", buildDepTree(m.root, blockers, visible))...)
	m.rows = append(m.rows, depRow{bean: m.root, prefix: "● ", root: true})
	m.rows = append(m.rows, section("blocks", buildDepTree(m.root, blocked, visible))...)
}

func (m depNavModel) selectedBean() *bean.Bean {
	if m.cursor >= 0 && m.cursor < len(m.rows) {
		return m.rows[m.cursor].bean
	}
	return nil
}

// moveCursor moves to the next selectable row in direction delta.
func (m *depNavModel) moveCursor(delta int) {
	for i := m.cursor + delta; i >= 0 && i < len(m.rows); i += delta {
		if m.rows[i].selectable() {
			m.cursor = i
			break
		}
	}
	m.scrollToCursor()
}

// listHeight is the number of rows that fit between header and footer.
func (m depNavModel) listHeight() int {
	return max(1, m.height-4)
}

func (m *depNavModel) scrollToCursor() {
	h := m.listHeight()
	// Keep the heading above the first rows visible.
	top := m.cursor
	if top == 1 {
		top = 0
	}
	if top < m.offset {
		m.offset = top
	}
	if m.cursor >= m.offset+h {
		m.offset = m.cursor - h + 1
	}
}

func (m depNavModel) Init() tea.Cmd {
	return nil
}

func (m depNavModel) Update(msg tea.Msg) (depNavModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.scrollToCursor()

	case tea.KeyMsg:
		k := keymap.DepNav
		switch {
		case key.Matches(msg, k.Up):
			m.moveCursor(-1)
		case key.Matches(msg, k.Down):
			m.moveCursor(1)
		case key.Matches(msg, k.Open):
			if b := m.selectedBean(); b != nil {
				return m, func() tea.Msg { return selectBeanMsg{bean: b} }
			}
		case key.Matches(msg, k.Refocus):
			if b := m.selectedBean(); b != nil && b.ID != m.root.ID {
				m.focus(b)
			}
		case key.Matches(msg, k.ToggleDone):
			m.showDone = !m.showDone
			m.reload()
		case key.Matches(msg, k.Back):
			return m, func() tea.Msg { return closeDepNavMsg{} }
		}
	}
	return m, nil
}

func (m depNavModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var sb strings.Builder
	sb.WriteString(" " + ui.Bold.Render("Dependencies of ") + ui.ID.Render(m.root.ID) + "  " + m.root.DisplayTitle())
	sb.WriteString("\n\n")

	cols := ui.CalculateResponsiveColumns(m.width, false)
	idColWidth := cols.ID
	for _, r := range m.rows {
		if r.bean != nil {
			// rows are indented by 2 (see renderRow), plus a space before the type
			idColWidth = max(idColWidth, 2+len([]rune(r.prefix))+len(r.bean.ID)+1)
		}
	}
	maxTitleWidth := max(1, m.width-idColWidth-cols.Status-cols.Type-4)

	end := min(len(m.rows), m.offset+m.listHeight())
	for i := m.offset; i < end; i++ {
		sb.WriteString(m.renderRow(m.rows[i], i == m.cursor, idColWidth, maxTitleWidth, cols.Compact))
		sb.WriteString("\n")
	}
	for i := end - m.offset; i < m.listHeight(); i++ {
		sb.WriteString("\n")
	}

	sb.WriteString("\n" + m.footer())
	return sb.String()
}

func (m depNavModel) renderRow(r depRow, selected bool, idColWidth, maxTitleWidth int, compact bool) string {
	if r.bean == nil {
		if r.empty {
			return "     " + ui.Muted.Render(r.text)
		}
		return " " + ui.Muted.Render("─ "+r.text+" "+strings.Repeat("─", max(0, min(40, m.width-6-len(r.text)))))
	}

	b := r.bean
	colors := m.config.GetBeanColors(b.Status, b.Type, b.Priority)
	title := b.DisplayTitle()
	switch {
	case r.cycle:
		title = "↺ " + title
	case r.repeat:
		title += " (see above)"
	}
	prefix := "  " + r.prefix
	if r.root {
		prefix = r.prefix
	}
	return ui.RenderBeanRow(b.ID, b.Status, b.Type, title, ui.BeanRowConfig{
		TitleMuted:    !b.HasTitle(),
		StatusColor:   colors.StatusColor,
		TypeColor:     colors.TypeColor,
		PriorityColor: colors.PriorityColor,
		Priority:      b.Priority,
		IsArchive:     colors.IsArchive,
		MaxTitleWidth: maxTitleWidth,
		ShowCursor:    true,
		IsSelected:    selected,
		IsMarked:      r.root,
		TreePrefix:    prefix,
		Dimmed:        r.repeat || r.cycle,
		IDColWidth:    idColWidth,
		Compact:       compact,
	})
}

func (m depNavModel) footer() string {
	g, k := keymap.Global, keymap.DepNav
	doneHelp := "show done"
	if m.showDone {
		doneHelp = "hide done"
	}
	legend := helpStyle.Render("↺ cycle") + "  "
	return legend + hints(
		hint(k.Open, ""),
		hint(k.Refocus, ""),
		hint(k.ToggleDone, doneHelp),
		hint(k.Back, ""),
		hint(g.Quit, ""),
	)
}
