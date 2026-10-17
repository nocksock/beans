package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/ui"
)

var (
	// List title style
	listTitleStyle = lipgloss.NewStyle().
			Foreground(ui.ColorPrimary).
			Bold(true)

	// Detail title style
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#fff")).
				Background(ui.ColorPrimary).
				Padding(0, 1)

	// Detail title style for untitled beans, whose body snippet is shown instead
	detailUntitledStyle = lipgloss.NewStyle().
				Italic(true).
				Foreground(ui.ColorMuted).
				Padding(0, 1)

	// Help text style
	helpStyle = lipgloss.NewStyle().
			Foreground(ui.ColorMuted)

	// Help key style
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(ui.ColorPrimary).
			Bold(true)
)

// renderDetailTitle renders a bean's title for detail and preview headers,
// falling back to a muted body snippet for untitled beans.
func renderDetailTitle(b *bean.Bean) string {
	if !b.HasTitle() {
		return detailUntitledStyle.Render(b.DisplayTitle())
	}
	return detailTitleStyle.Render(b.Title)
}
