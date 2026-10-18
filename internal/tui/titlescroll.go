package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/ui"
)

const (
	titleScrollDelay = 2 * time.Second       // how long a row is selected before its title scrolls
	titleScrollStep  = 60 * time.Millisecond // time per scrolled column
)

// titleScroll scrolls the cut-off title of the selected list row, so its end
// shows: after titleScrollDelay it moves one column per titleScrollStep until
// the "..." is at the front.
type titleScroll struct {
	beanID string // the bean whose title is scrolled
	shift  int    // columns scrolled
	seq    int    // identifies the current scroll; ticks of older ones are ignored
}

// titleScrollMsg advances the scroll with the given seq by one column
type titleScrollMsg struct{ seq int }

func (s *titleScroll) tick(delay time.Duration) tea.Cmd {
	seq := s.seq
	return tea.Tick(delay, func(time.Time) tea.Msg { return titleScrollMsg{seq: seq} })
}

// restartTitleScroll starts over when the cursor moved to another bean.
func (m listModel) restartTitleScroll() tea.Cmd {
	s := m.titleScroll
	item, _ := m.list.SelectedItem().(beanItem)
	id := ""
	if item.bean != nil {
		id = item.bean.ID
	}
	if id == s.beanID {
		return nil
	}
	s.beanID, s.shift = id, 0
	s.seq++
	if m.selectedTitleOverflow() == 0 {
		return nil
	}
	return s.tick(titleScrollDelay)
}

// stepTitleScroll scrolls one column further, until the title's end shows.
func (m listModel) stepTitleScroll(msg titleScrollMsg) tea.Cmd {
	s := m.titleScroll
	if msg.seq != s.seq || s.shift >= m.selectedTitleOverflow() {
		return nil
	}
	s.shift++
	return s.tick(titleScrollStep)
}

// selectedTitleOverflow is how far the selected row's title can scroll.
func (m listModel) selectedTitleOverflow() int {
	item, ok := m.list.SelectedItem().(beanItem)
	if !ok {
		return 0
	}
	return ui.TitleOverflow(item.bean.DisplayTitle(), m.delegate().rowConfig(item, m.list.Width()))
}
