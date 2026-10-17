package tui

import (
	"os"

	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
)

// markdownStyle is the glamour style used to render bean bodies. It is resolved
// once by detectMarkdownStyle before the TUI starts, since querying the terminal
// background while Bubble Tea owns stdin is unreliable (and glamour's
// WithAutoStyle re-queries on every renderer creation, which can be slow).
var markdownStyle = styles.DarkStyle

// detectMarkdownStyle picks the glamour style for the current terminal.
// GLAMOUR_STYLE takes precedence, otherwise the terminal background decides.
func detectMarkdownStyle() {
	markdownStyle = resolveMarkdownStyle(os.Getenv("GLAMOUR_STYLE"), lipgloss.HasDarkBackground)
}

func resolveMarkdownStyle(env string, hasDarkBackground func() bool) string {
	if env != "" {
		return env
	}
	if hasDarkBackground() {
		return styles.DarkStyle
	}
	return styles.LightStyle
}
