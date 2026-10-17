package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestHelpDescStyleUsesTerminalForeground(t *testing.T) {
	if fg := helpDescStyle.GetForeground(); fg != (lipgloss.NoColor{}) {
		t.Errorf("helpDescStyle foreground = %v, want terminal default (NoColor)", fg)
	}
}
