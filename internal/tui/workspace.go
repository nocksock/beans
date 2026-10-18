package tui

import (
	"slices"

	"github.com/hmans/beans/internal/tui/layout"
)

// maxFocusHistory caps how many previously focused panes esc can go back to
const maxFocusHistory = 16

// workspace is the state of the pane layout: which panes are shown, which
// one has focus, and where +/- are in their preset cycle.
type workspace struct {
	visible map[layout.ID]bool
	focus   layout.ID
	history []layout.ID // previously focused panes, most recent last

	cycle  layout.ID // the pane whose preset cycle +/- step through
	preset int       // index into that cycle; -1 if the layout matches none
}

func newWorkspace() workspace {
	w := workspace{focus: paneList, cycle: paneList}
	w.apply(presetCycles[paneList][0])
	return w
}

func (w *workspace) isVisible(id layout.ID) bool {
	return w.visible[id]
}

// visiblePanes returns the shown panes in tree order.
func (w *workspace) visiblePanes() []layout.ID {
	var out []layout.ID
	for _, id := range layoutTree.Leaves() {
		if w.visible[id] {
			out = append(out, id)
		}
	}
	return out
}

// arrange lays the visible panes out on a width x height screen.
func (w *workspace) arrange(width, height int) layout.Layout {
	return layout.Arrange(layoutTree, w.isVisible, w.focus, width, height)
}

// cyclePreset switches to the next (delta 1) or previous (-1) preset of the
// current cycle, dropping manual toggles.
func (w *workspace) cyclePreset(delta int) {
	presets := presetCycles[w.cycle]
	if len(presets) == 0 {
		return
	}
	next := w.preset + delta
	if w.preset < 0 && delta < 0 {
		next = len(presets) - 1
	}
	w.preset = (next%len(presets) + len(presets)) % len(presets)
	w.apply(presets[w.preset])
}

// apply shows exactly the panes of p. If the focused pane isn't among them,
// focus moves to the cycle's pane, or the first pane of p.
func (w *workspace) apply(p preset) {
	w.visible = map[layout.ID]bool{}
	for _, id := range p {
		w.visible[id] = true
	}
	if !w.visible[w.focus] {
		w.focus = p[0]
		if w.visible[w.cycle] {
			w.focus = w.cycle
		}
	}
}

// show makes a pane visible, keeping the rest of the layout.
func (w *workspace) show(id layout.ID) {
	w.visible[id] = true
}

// toggle shows or hides a pane. The last visible pane stays. Hiding the
// focused pane moves focus back to the previous visible one.
func (w *workspace) toggle(id layout.ID) {
	if !w.visible[id] {
		w.visible[id] = true
		return
	}
	if len(w.visiblePanes()) == 1 {
		return
	}
	delete(w.visible, id)
	if w.focus == id {
		w.focus = w.previousVisible()
		w.setCycle(w.focus)
	}
}

// previousVisible pops the history up to the most recent visible pane,
// falling back to the first visible pane.
func (w *workspace) previousVisible() layout.ID {
	for len(w.history) > 0 {
		id := w.history[len(w.history)-1]
		w.history = w.history[:len(w.history)-1]
		if w.visible[id] {
			return id
		}
	}
	return w.visiblePanes()[0]
}

// focusPane moves focus to id, remembering the current pane for back. A
// hidden pane is revealed with the first preset of its cycle that shows it.
func (w *workspace) focusPane(id layout.ID) {
	if id == w.focus {
		return
	}
	w.history = append(w.history, w.focus)
	if len(w.history) > maxFocusHistory {
		w.history = w.history[1:]
	}
	w.moveFocus(id)
}

// moveFocus focuses id, revealing it if hidden, without touching the history.
func (w *workspace) moveFocus(id layout.ID) {
	w.focus = id
	if !w.visible[id] {
		w.reveal(id)
	}
	w.setCycle(id)
}

// reveal shows id with the first preset of its cycle containing it, or by
// adding it to the current layout if there is none.
func (w *workspace) reveal(id layout.ID) {
	for _, p := range presetCycles[id] {
		if slices.Contains(p, id) {
			w.apply(p)
			return
		}
	}
	w.visible[id] = true
}

// setCycle makes +/- use id's cycle, if it has one, positioned at the preset
// matching the current layout.
func (w *workspace) setCycle(id layout.ID) {
	presets, ok := presetCycles[id]
	if !ok {
		return
	}
	w.cycle = id
	w.preset = slices.IndexFunc(presets, w.matches)
}

// matches reports whether exactly the panes of p are visible.
func (w *workspace) matches(p preset) bool {
	if len(p) != len(w.visible) {
		return false
	}
	for _, id := range p {
		if !w.visible[id] {
			return false
		}
	}
	return true
}

// cycleFocus focuses the next (delta 1) or previous (-1) visible pane that
// can take focus.
func (w *workspace) cycleFocus(delta int) {
	var panes []layout.ID
	for _, id := range w.visiblePanes() {
		if !unfocusable[id] || id == w.focus {
			panes = append(panes, id)
		}
	}
	i := slices.Index(panes, w.focus)
	n := len(panes)
	w.focusPane(panes[((i+delta)%n+n)%n])
}

// back returns focus to the previously focused pane, or the list if there is
// none, revealing it if it was hidden since.
func (w *workspace) back() {
	target := paneList
	if len(w.history) > 0 {
		target = w.history[len(w.history)-1]
		w.history = w.history[:len(w.history)-1]
	}
	if target != w.focus {
		w.moveFocus(target)
	}
}
