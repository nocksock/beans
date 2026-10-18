package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/hmans/beans/internal/tui/layout"
)

// Panes of the main screen
const (
	paneList     layout.ID = "list"
	paneContext  layout.ID = "context" // details of what's selected in the list, dependency graph or log
	paneDepGraph layout.ID = "deps"
	paneLog      layout.ID = "log"
)

// paneKeys are the bindings focusing each pane, shown in the pane titles.
func paneKeys() map[layout.ID]key.Binding {
	k := keymap.Layout
	return map[layout.ID]key.Binding{
		paneList:     k.FocusList,
		paneContext:  k.FocusContext,
		paneDepGraph: k.FocusDepGraph,
		paneLog:      k.FocusLog,
	}
}

// layoutTree arranges every pane. Which ones are shown is up to the presets
// and toggles; hidden panes collapse and their siblings take the space.
//
// Below 120 columns the list and the context don't fit side by side, so the
// unfocused one is dropped. The log needs 18 rows.
var layoutTree = layout.VSplit(
	layout.HSplit(
		layout.Leaf(paneList).Min(60),
		layout.VSplit(
			layout.Leaf(paneDepGraph).Min(6),
			layout.Leaf(paneContext).Weight(2).Min(6),
		).Min(60).Drop(1),
	).Min(10),
	layout.Leaf(paneLog).Fixed(8).Drop(1),
)

// preset is a set of visible panes.
type preset []layout.ID

// presetCycles are the layouts +/- step through, by focused pane. Panes
// without a cycle use the one of the last focused pane that has one. The
// first list preset is the layout at startup.
var presetCycles = map[layout.ID][]preset{
	paneList: {
		{paneList, paneContext, paneLog},
		{paneList, paneLog},
		{paneList},
	},
	paneContext: {
		{paneList, paneContext, paneLog},
		{paneDepGraph, paneContext, paneLog},
		{paneContext},
	},
	paneLog: {
		{paneList, paneContext, paneLog},
		{paneContext, paneLog}, // run output at full width
	},
}
