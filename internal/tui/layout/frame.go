package layout

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Frame draws body in a rounded border of exactly width x height cells, with
// title set into the top edge. The body is clipped and padded to fit inside.
func Frame(title, body string, width, height int, border lipgloss.Style) string {
	if width < 2 || height < 2 {
		return Fill(body, width, height)
	}
	iw, ih := width-2, height-2

	top := "─"
	if title != "" && iw > 4 {
		top += " " + ansi.Truncate(title, iw-4, "…") + " "
	}
	top += strings.Repeat("─", max(iw-lipgloss.Width(top), 0))

	var sb strings.Builder
	sb.WriteString(border.Render("╭" + top + "╮"))
	side := border.Render("│")
	for _, line := range strings.Split(Fill(body, iw, ih), "\n") {
		sb.WriteString("\n" + side + line + side)
	}
	sb.WriteString("\n" + border.Render("╰"+strings.Repeat("─", iw)+"╯"))
	return sb.String()
}

// Fill clips and pads s to exactly width x height cells.
func Fill(s string, width, height int) string {
	if height <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	for i, l := range lines {
		l = ansi.Truncate(l, max(width, 0), "")
		lines[i] = l + strings.Repeat(" ", max(width-lipgloss.Width(l), 0))
	}
	return strings.Join(lines, "\n")
}
