package tui

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/ui"
)

// Glamour renderers for the preview pane, one per wrap width
var (
	previewRenderers   = map[int]*glamour.TermRenderer{}
	previewRenderersMu sync.Mutex
)

func getPreviewRenderer(wrap int) *glamour.TermRenderer {
	previewRenderersMu.Lock()
	defer previewRenderersMu.Unlock()
	if r, ok := previewRenderers[wrap]; ok {
		return r
	}
	// Same startup-detected style as the detail view
	r, err := glamour.NewTermRenderer(glamour.WithStylePath(markdownStyle), glamour.WithWordWrap(wrap))
	if err != nil {
		r = nil
	}
	previewRenderers[wrap] = r
	return r
}

// previewContext holds relationship info for the previewed bean that isn't on the bean itself
type previewContext struct {
	parent     *bean.Bean // nil if no parent (or parent missing)
	childCount int
}

// previewModel is a read-only detail preview for the two-column layout.
// It has no focus and no interaction - it just renders the highlighted bean.
type previewModel struct {
	config *config.Config
	bean   *bean.Bean
	ctx    previewContext

	// Rendered markdown body cache - glamour is too slow to run on every View()
	cachedBody      string
	cachedBodyWidth int
	cachedBodyOut   string
}

func newPreviewModel(cfg *config.Config) previewModel {
	return previewModel{config: cfg}
}

// setBean updates the previewed bean
func (m *previewModel) setBean(b *bean.Bean, ctx previewContext) {
	m.bean = b
	m.ctx = ctx
}

// View renders the preview pane at exactly the given size (including border).
func (m *previewModel) View(width, height int) string {
	innerWidth := max(1, width-4) // border (2) + horizontal padding (2)
	innerHeight := max(1, height-2)

	var content string
	if m.bean == nil {
		content = lipgloss.NewStyle().
			Width(innerWidth).
			Height(innerHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(ui.ColorMuted).
			Render("No bean selected")
	} else {
		content = m.renderBean(innerWidth)
	}

	// Wrap to the inner width, then clip to the inner height
	lines := strings.Split(lipgloss.NewStyle().Width(innerWidth).Render(content), "\n")
	if len(lines) > innerHeight {
		lines = lines[:innerHeight]
		lines[innerHeight-1] = ui.Muted.Render("…")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorMuted).
		Padding(0, 1).
		Width(width - 2).
		Height(innerHeight).
		Render(strings.Join(lines, "\n"))
}

func (m *previewModel) renderBean(width int) string {
	b := m.bean
	label := lipgloss.NewStyle().Foreground(ui.ColorMuted)

	var parts []string
	parts = append(parts, ui.ID.Render(b.ID))
	parts = append(parts, renderDetailTitle(b))
	parts = append(parts, "")

	// Status, type, priority
	colors := m.config.GetBeanColors(b.Status, b.Type, b.Priority)
	meta := ui.RenderStatusWithColor(b.Status, colors.StatusColor, colors.IsArchive) + "  " +
		ui.RenderTypeText(b.Type, colors.TypeColor)
	if b.Priority != "" && b.Priority != "normal" {
		meta += "  " + ui.RenderPriorityText(b.Priority, colors.PriorityColor)
	}
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
		rels = append(rels, label.Render("Children: ")+fmt.Sprintf("%d", m.ctx.childCount)+
			label.Render("  (enter to open)"))
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

func (m *previewModel) renderBody(width int) string {
	if m.bean.Body == "" {
		return lipgloss.NewStyle().Foreground(ui.ColorMuted).Render("No description")
	}

	if m.cachedBody == m.bean.Body && m.cachedBodyWidth == width {
		return m.cachedBodyOut
	}

	out := m.bean.Body
	// Glamour adds a 2-char margin on each side, so wrap 4 chars narrower
	if renderer := getPreviewRenderer(max(10, width-4)); renderer != nil {
		if rendered, err := renderer.Render(m.bean.Body); err == nil {
			out = strings.Trim(rendered, "\n")
		}
	}

	m.cachedBody = m.bean.Body
	m.cachedBodyWidth = width
	m.cachedBodyOut = out
	return out
}
