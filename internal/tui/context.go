package tui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/ui"
)

// Glamour renderers for bean bodies, one per wrap width
var (
	bodyRenderers   = map[int]*glamour.TermRenderer{}
	bodyRenderersMu sync.Mutex
)

func getBodyRenderer(wrap int) *glamour.TermRenderer {
	bodyRenderersMu.Lock()
	defer bodyRenderersMu.Unlock()
	if r, ok := bodyRenderers[wrap]; ok {
		return r
	}
	// markdownStyle is detected once at startup instead of using WithAutoStyle(),
	// which re-queries the terminal and can cause multi-second delays
	r, err := glamour.NewTermRenderer(glamour.WithStylePath(markdownStyle), glamour.WithWordWrap(wrap))
	if err != nil {
		r = nil
	}
	bodyRenderers[wrap] = r
	return r
}

// focusBackMsg asks the app to move focus back to the previously focused pane
type focusBackMsg struct{}

// focusContextMsg asks the app to move focus to the context pane
type focusContextMsg struct{}

// previewContext holds relationship info for the shown bean that isn't on the bean itself
type previewContext struct {
	parent     *bean.Bean // nil if no parent (or parent missing)
	childCount int
}

// contextModel is the context pane: the details of the bean selected in the
// list or the dependency graph. When focused, it scrolls and its action keys
// act on the bean.
type contextModel struct {
	config *config.Config
	bean   *bean.Bean
	ctx    previewContext
	scroll scroller
	width  int // content width, set by setSize

	// Rendered markdown body cache - glamour is too slow to run on every View()
	cachedBody      string
	cachedBodyWidth int
	cachedBodyOut   string
}

func newContextModel(cfg *config.Config) contextModel {
	return contextModel{config: cfg}
}

// setBean changes the shown bean. Scrolling restarts at the top for another bean.
func (m *contextModel) setBean(b *bean.Bean, ctx previewContext) {
	if b == nil || m.bean == nil || b.ID != m.bean.ID {
		m.scroll.offset = 0
	}
	m.bean = b
	m.ctx = ctx
}

func (m *contextModel) setSize(width, height int) {
	m.width, m.scroll.height = width, height
	m.scroll.clamp(len(m.lines()))
}

// title is the pane title.
func (m *contextModel) title() string {
	if m.bean == nil {
		return "Details"
	}
	return "Details " + m.bean.ID
}

func (m contextModel) Update(msg tea.Msg) (contextModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch {
	case key.Matches(keyMsg, keymap.Context.Back):
		return m, func() tea.Msg { return focusBackMsg{} }
	case m.scroll.scrollKey(keyMsg, len(m.lines())):
	default:
		if m.bean != nil {
			return m, m.beanAction(keyMsg)
		}
	}
	return m, nil
}

// beanAction returns the command for an action key on the shown bean, or nil.
func (m contextModel) beanAction(msg tea.KeyMsg) tea.Cmd {
	b := m.bean
	ids, title := []string{b.ID}, b.DisplayTitle()
	k := keymap.Context
	var out tea.Msg
	switch {
	case key.Matches(msg, k.ChangeParent):
		out = openParentPickerMsg{beanIDs: ids, beanTitle: title, beanTypes: []string{b.Type}, currentParent: b.Parent}
	case key.Matches(msg, k.ChangeStatus):
		out = openStatusPickerMsg{beanIDs: ids, beanTitle: title, currentStatus: b.Status}
	case key.Matches(msg, k.ChangeType):
		out = openTypePickerMsg{beanIDs: ids, beanTitle: title, currentType: b.Type}
	case key.Matches(msg, k.EditTags):
		out = openTagEditorMsg{beanIDs: ids, beanTitle: title}
	case key.Matches(msg, k.ChangePriority):
		out = openPriorityPickerMsg{beanIDs: ids, beanTitle: title, currentPriority: b.Priority}
	case key.Matches(msg, k.ChangeSize):
		out = openSizePickerMsg{beanIDs: ids, beanTitle: title, currentSize: b.Size}
	case key.Matches(msg, k.ManageBlocking):
		out = openBlockingPickerMsg{beanID: b.ID, beanTitle: title, currentIDs: b.Blocking}
	case key.Matches(msg, k.ManageBlockedBy):
		out = openBlockingPickerMsg{beanID: b.ID, beanTitle: title, blockedBy: true, currentIDs: b.BlockedBy}
	case key.Matches(msg, k.Edit):
		out = openEditModalMsg{bean: b}
	case key.Matches(msg, k.OpenEditor):
		out = openEditorMsg{beanID: b.ID, beanPath: b.Path}
	case key.Matches(msg, k.Launch):
		out = openLauncherPickerMsg{beanIDs: ids, beanTitle: title}
	case key.Matches(msg, k.CopyID):
		out = copyIDMsg{beanID: b.ID}
	case key.Matches(msg, k.Archive):
		out = archiveBeanMsg{beanID: b.ID}
	default:
		return nil
	}
	return func() tea.Msg { return out }
}

// lines renders the bean, wrapped to the pane width.
func (m *contextModel) lines() []string {
	if m.bean == nil || m.width <= 0 {
		return nil
	}
	return strings.Split(lipgloss.NewStyle().Width(m.width).Render(m.renderBean(m.width)), "\n")
}

// View renders the visible part of the bean at exactly the size set by setSize.
func (m *contextModel) View() string {
	if m.bean == nil {
		return centeredMuted("No bean selected", m.width, m.scroll.height)
	}
	return strings.Join(m.scroll.window(m.lines()), "\n")
}

// footer renders the hints for the focused pane.
func (m *contextModel) footer() string {
	k := keymap.Context
	return m.scroll.percentHint(len(m.lines())) + "  " + hints(
		groupHint("scroll", k.ScrollDown, k.ScrollUp),
		hint(k.Launch, ""),
		hint(k.Edit, ""),
		hint(k.OpenEditor, ""),
		hint(k.ChangeStatus, ""),
		hint(k.ChangeType, ""),
		hint(k.ChangePriority, ""),
		hint(k.ChangeSize, ""),
		hint(k.EditTags, ""),
		hint(k.ChangeParent, ""),
		hint(k.ManageBlocking, ""),
		hint(k.ManageBlockedBy, ""),
		hint(k.CopyID, ""),
		hint(k.Archive, ""),
		hint(k.Back, ""),
	)
}

func (m *contextModel) renderBean(width int) string {
	b := m.bean
	label := lipgloss.NewStyle().Foreground(ui.ColorMuted)

	var parts []string
	parts = append(parts, ui.ID.Render(b.ID))
	parts = append(parts, renderDetailTitle(b))
	parts = append(parts, "")

	// Status, type, priority, size
	colors := m.config.GetBeanColors(b.Status, b.Type, b.Priority)
	meta := ui.RenderStatusWithColor(b.Status, colors.StatusColor, colors.IsArchive) + "  " +
		ui.RenderTypeText(b.Type, colors.TypeColor)
	if b.Priority != "" && b.Priority != "normal" {
		meta += "  " + ui.RenderPriorityText(b.Priority, colors.PriorityColor)
	}
	meta += "  " + ui.RenderSizeWithColor(b.Size, ui.SizeColor(m.config, b.Size))
	parts = append(parts, meta)

	if len(b.Tags) > 0 {
		parts = append(parts, ui.RenderTags(b.Tags))
	}

	// Relationships
	var rels []string
	if m.ctx.parent != nil {
		rels = append(rels, label.Render("Parent:   ")+ui.ID.Render(m.ctx.parent.ID)+" "+m.ctx.parent.DisplayTitle())
	} else if b.Parent != "" {
		rels = append(rels, label.Render("Parent:   ")+ui.ID.Render(b.Parent))
	}
	if m.ctx.childCount > 0 {
		rels = append(rels, label.Render("Children: ")+fmt.Sprintf("%d", m.ctx.childCount))
	}
	if len(b.Blocking) > 0 {
		rels = append(rels, label.Render("Blocking: ")+ui.ID.Render(strings.Join(b.Blocking, ", ")))
	}
	if len(b.BlockedBy) > 0 {
		rels = append(rels, label.Render("Blocked by: ")+ui.ID.Render(strings.Join(b.BlockedBy, ", ")))
	}
	if len(rels) > 0 {
		parts = append(parts, "")
		parts = append(parts, rels...)
	}

	parts = append(parts, "")
	parts = append(parts, m.renderBody(width))

	return strings.Join(parts, "\n")
}

func (m *contextModel) renderBody(width int) string {
	if m.bean.Body == "" {
		return lipgloss.NewStyle().Foreground(ui.ColorMuted).Render("No description")
	}

	if m.cachedBody == m.bean.Body && m.cachedBodyWidth == width {
		return m.cachedBodyOut
	}

	out := m.bean.Body
	// Glamour adds a 2-char margin on each side, so wrap 4 chars narrower
	if renderer := getBodyRenderer(max(10, width-4)); renderer != nil {
		if rendered, err := renderer.Render(m.bean.Body); err == nil {
			out = strings.Trim(rendered, "\n")
		}
	}

	m.cachedBody = m.bean.Body
	m.cachedBodyWidth = width
	m.cachedBodyOut = out
	return out
}

// centeredMuted renders a muted note in the middle of a width x height pane.
func centeredMuted(text string, width, height int) string {
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(ui.ColorMuted).
		Render(text)
}
