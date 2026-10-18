package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/internal/bean"
)

func TestTitleScroll(t *testing.T) {
	app, _ := newTestApp(t)
	long := &bean.Bean{Title: "Start " + strings.Repeat("filler ", 30) + "the end", Status: "todo", Type: "task"}
	if err := app.core.Create(long); err != nil {
		t.Fatal(err)
	}
	app.Update(app.list.loadBeans())

	for i, it := range app.list.list.Items() {
		if it.(beanItem).bean.ID == long.ID {
			app.list.list.Select(i)
		}
	}
	if app.list.restartTitleScroll() == nil {
		t.Fatal("selecting a cut-off title should schedule its scroll")
	}
	s := app.list.titleScroll
	if row := selectedRow(app); !strings.Contains(row, "Start filler") {
		t.Errorf("the title should start at its beginning: %q", row)
	}

	app.Update(titleScrollMsg{seq: s.seq - 1})
	if s.shift != 0 {
		t.Errorf("a stale tick scrolled to %d", s.shift)
	}

	steps := 0
	for _, cmd := app.Update(titleScrollMsg{seq: s.seq}); cmd != nil && steps < 1000; steps++ {
		_, cmd = app.Update(titleScrollMsg{seq: s.seq})
	}
	if s.shift == 0 || s.shift != app.list.selectedTitleOverflow() {
		t.Errorf("shift = %d, want the overflow %d", s.shift, app.list.selectedTitleOverflow())
	}
	if row := selectedRow(app); !strings.Contains(row, "...iller") || !strings.Contains(row, "the end") {
		t.Errorf("the scrolled title should show its end behind \"...\": %q", row)
	}

	pump(app, keyMsg("k"))
	if s.shift != 0 || s.beanID == long.ID {
		t.Errorf("moving the cursor should reset the scroll (shift %d)", s.shift)
	}
}

// selectedRow returns the list row under the cursor, without styling.
func selectedRow(app *App) string {
	for _, l := range strings.Split(ansi.Strip(app.list.View()), "\n") {
		if strings.Contains(l, "▌") {
			return l
		}
	}
	return ""
}
