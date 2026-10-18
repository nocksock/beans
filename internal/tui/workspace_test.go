package tui

import (
	"slices"
	"testing"

	"github.com/hmans/beans/internal/tui/layout"
)

// wsOp is one step applied to a workspace in a test.
type wsOp func(w *workspace)

func plus(w *workspace)  { w.cyclePreset(1) }
func minus(w *workspace) { w.cyclePreset(-1) }
func tab(w *workspace)   { w.cycleFocus(1) }
func esc(w *workspace)   { w.back() }
func focus(id layout.ID) wsOp {
	return func(w *workspace) { w.focusPane(id) }
}
func toggle(id layout.ID) wsOp {
	return func(w *workspace) { w.toggle(id) }
}

func TestWorkspace(t *testing.T) {
	tests := []struct {
		name        string
		ops         []wsOp
		wantVisible []layout.ID // in tree order
		wantFocus   layout.ID
	}{
		{
			name:        "starts with the first list preset",
			wantVisible: []layout.ID{paneList, paneContext, paneLog},
			wantFocus:   paneList,
		},
		{
			name:        "+ steps through the list presets",
			ops:         []wsOp{plus},
			wantVisible: []layout.ID{paneList, paneLog},
			wantFocus:   paneList,
		},
		{
			name:        "+ twice shows only the list",
			ops:         []wsOp{plus, plus},
			wantVisible: []layout.ID{paneList},
			wantFocus:   paneList,
		},
		{
			name:        "+ wraps around",
			ops:         []wsOp{plus, plus, plus},
			wantVisible: []layout.ID{paneList, paneContext, paneLog},
			wantFocus:   paneList,
		},
		{
			name:        "- walks backwards",
			ops:         []wsOp{minus},
			wantVisible: []layout.ID{paneList},
			wantFocus:   paneList,
		},
		{
			name:        "focusing the context keeps the layout",
			ops:         []wsOp{focus(paneContext)},
			wantVisible: []layout.ID{paneList, paneContext, paneLog},
			wantFocus:   paneContext,
		},
		{
			name:        "+ on the context shows the dependency graph",
			ops:         []wsOp{focus(paneContext), plus},
			wantVisible: []layout.ID{paneDepGraph, paneContext, paneLog},
			wantFocus:   paneContext,
		},
		{
			name:        "+ twice on the context shows only the context",
			ops:         []wsOp{focus(paneContext), plus, plus},
			wantVisible: []layout.ID{paneContext},
			wantFocus:   paneContext,
		},
		{
			name:        "focusing a hidden context switches to a preset showing it",
			ops:         []wsOp{plus, focus(paneContext)},
			wantVisible: []layout.ID{paneList, paneContext, paneLog},
			wantFocus:   paneContext,
		},
		{
			name:        "+ after a manual toggle jumps to the next preset",
			ops:         []wsOp{toggle(paneLog), plus},
			wantVisible: []layout.ID{paneList, paneLog},
			wantFocus:   paneList,
		},
		{
			name:        "toggle hides and shows a pane",
			ops:         []wsOp{toggle(paneLog)},
			wantVisible: []layout.ID{paneList, paneContext},
			wantFocus:   paneList,
		},
		{
			name:        "toggle shows a pane outside the preset",
			ops:         []wsOp{toggle(paneDepGraph)},
			wantVisible: []layout.ID{paneList, paneDepGraph, paneContext, paneLog},
			wantFocus:   paneList,
		},
		{
			name:        "the last visible pane can't be hidden",
			ops:         []wsOp{plus, plus, toggle(paneList)},
			wantVisible: []layout.ID{paneList},
			wantFocus:   paneList,
		},
		{
			name:        "hiding the focused pane focuses the previous one",
			ops:         []wsOp{focus(paneContext), toggle(paneContext)},
			wantVisible: []layout.ID{paneList, paneLog},
			wantFocus:   paneList,
		},
		{
			name:        "tab cycles through visible panes",
			ops:         []wsOp{tab},
			wantVisible: []layout.ID{paneList, paneContext, paneLog},
			wantFocus:   paneContext,
		},
		{
			name:        "tab wraps around",
			ops:         []wsOp{tab, tab, tab},
			wantVisible: []layout.ID{paneList, paneContext, paneLog},
			wantFocus:   paneList,
		},
		{
			name:        "esc goes back to the previous pane",
			ops:         []wsOp{focus(paneContext), esc},
			wantVisible: []layout.ID{paneList, paneContext, paneLog},
			wantFocus:   paneList,
		},
		{
			name:        "esc reveals a previous pane hidden since",
			ops:         []wsOp{focus(paneContext), plus, plus, esc},
			wantVisible: []layout.ID{paneList, paneContext, paneLog},
			wantFocus:   paneList,
		},
		{
			name:        "esc without history goes to the list",
			ops:         []wsOp{toggle(paneDepGraph), focus(paneDepGraph), esc, esc},
			wantVisible: []layout.ID{paneList, paneDepGraph, paneContext, paneLog},
			wantFocus:   paneList,
		},
		{
			name:        "a pane without a cycle uses the last cycle",
			ops:         []wsOp{focus(paneContext), plus, focus(paneDepGraph), plus},
			wantVisible: []layout.ID{paneContext},
			wantFocus:   paneContext,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorkspace()
			for _, op := range tt.ops {
				op(&w)
			}
			if got := w.visiblePanes(); !slices.Equal(got, tt.wantVisible) {
				t.Errorf("visible = %v, want %v", got, tt.wantVisible)
			}
			if w.focus != tt.wantFocus {
				t.Errorf("focus = %v, want %v", w.focus, tt.wantFocus)
			}
		})
	}
}
