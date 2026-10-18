package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/ui"
)

// scroller is the scroll position of a pane showing lines of text. Its keys
// are the context pane's scroll keys.
type scroller struct {
	offset int // first visible line
	height int // visible lines
}

// scrollKey scrolls for msg, given the number of lines, and reports whether
// msg was a scroll key.
func (s *scroller) scrollKey(msg tea.KeyMsg, total int) bool {
	k := keymap.Context
	switch {
	case key.Matches(msg, k.ScrollUp):
		s.offset--
	case key.Matches(msg, k.ScrollDown):
		s.offset++
	case key.Matches(msg, k.HalfPageUp):
		s.offset -= max(1, s.height/2)
	case key.Matches(msg, k.HalfPageDown):
		s.offset += max(1, s.height/2)
	case key.Matches(msg, k.PageUp):
		s.offset -= max(1, s.height)
	case key.Matches(msg, k.PageDown):
		s.offset += max(1, s.height)
	case key.Matches(msg, k.GoToStart):
		s.offset = 0
	case key.Matches(msg, k.GoToEnd):
		s.offset = total
	default:
		return false
	}
	s.clamp(total)
	return true
}

// clamp keeps the last page full when scrolled to the end.
func (s *scroller) clamp(total int) {
	s.offset = max(0, min(s.offset, total-s.height))
}

// atEnd reports whether the last line is visible.
func (s scroller) atEnd(total int) bool {
	return s.offset >= total-s.height
}

// window returns the visible lines, with a "…" on the last row if more follow.
func (s scroller) window(lines []string) []string {
	end := min(len(lines), s.offset+s.height)
	visible := append([]string(nil), lines[min(s.offset, end):end]...)
	if end < len(lines) && len(visible) > 0 {
		visible[len(visible)-1] = ui.Muted.Render("…")
	}
	return visible
}

// percentHint renders how far the text is scrolled, from 0% to 100%.
func (s scroller) percentHint(total int) string {
	p := 100
	if maxOffset := total - s.height; maxOffset > 0 {
		p = s.offset * 100 / maxOffset
	}
	return helpStyle.Render(fmt.Sprintf("%d%%", p))
}

// scrollTo scrolls the least amount needed to make line visible, keeping it
// off the last row, which window replaces with "…" while more lines follow.
func (s *scroller) scrollTo(line, total int) {
	last := s.offset + s.height - 1
	if line < s.offset {
		s.offset = line
	} else if line > last || (line == last && line < total-1) {
		s.offset = line - s.height + 2
	}
	s.clamp(total)
}
