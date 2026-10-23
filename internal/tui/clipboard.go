package tui

import (
	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

// copyIDMsg requests copying a bean ID to the system clipboard.
type copyIDMsg struct {
	beanID string
}

// copiedIDMsg reports the result of copying a bean ID.
type copiedIDMsg struct {
	beanID string
	err    error
}

// writeClipboard puts text on the system clipboard. Tests replace it.
var writeClipboard = clipboard.WriteAll

// copyID returns a command that copies id to the clipboard. It runs off the
// update loop since the clipboard may shell out (xclip, wl-copy, pbcopy).
func copyID(id string) tea.Cmd {
	return func() tea.Msg {
		return copiedIDMsg{beanID: id, err: writeClipboard(id)}
	}
}
