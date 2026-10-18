package tui

import "github.com/hmans/beans/internal/tui/layout"

// Panes of the main screen
const (
	paneList     layout.ID = "list"
	paneContext  layout.ID = "context" // details of what's selected in the list or dependency graph
	paneDepGraph layout.ID = "deps"
	paneLog      layout.ID = "log"
)

// layoutTree arranges every pane. Which ones are shown is up to the presets
// and toggles; hidden panes collapse and their siblings take the space.
//
// Below 120 columns the list and the context don't fit side by side, so the
// unfocused one is dropped. The log needs 18 rows.
var layoutTree = layout.VSplit(
	layout.HSplit(
		layout.Leaf(paneList).Min(40),
		layout.VSplit(
			layout.Leaf(paneDepGraph).Min(6),
			layout.Leaf(paneContext).Weight(2).Min(6),
		).Min(80).Max(80).Drop(1), // bean bodies are written for ~80 columns
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
}
