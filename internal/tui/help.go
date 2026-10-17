package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/ui"
)

// openHelpMsg requests opening the help overlay
type openHelpMsg struct{}

// closeHelpMsg is sent when the help overlay is closed
type closeHelpMsg struct{}

// helpDescStyle renders shortcut descriptions in the terminal's default
// foreground, so they stay readable on both light and dark backgrounds.
var helpDescStyle = lipgloss.NewStyle()

// helpOverlayModel displays keyboard shortcuts organized by context
type helpOverlayModel struct {
	width  int
	height int
}

func newHelpOverlayModel(width, height int) helpOverlayModel {
	return helpOverlayModel{
		width:  width,
		height: height,
	}
}

func (m helpOverlayModel) Init() tea.Cmd {
	return nil
}

func (m helpOverlayModel) Update(msg tea.Msg) (helpOverlayModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if key.Matches(msg, keymap.Help.Close) {
			return m, func() tea.Msg {
				return closeHelpMsg{}
			}
		}
	}

	return m, nil
}

func (m helpOverlayModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Calculate modal dimensions
	modalWidth := max(50, min(60, m.width*60/100))

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.ColorPrimary).
		Render("Keyboard Shortcuts")

	// Helper to create a shortcut line
	shortcut := func(key, desc string) string {
		keyStyle := lipgloss.NewStyle().
			Foreground(ui.ColorPrimary).
			Bold(true).
			Width(12).
			Render(key)
		return keyStyle + helpDescStyle.Render(desc)
	}

	var content strings.Builder
	content.WriteString(title + "\n\n")

	g, l, d := keymap.Global, keymap.List, keymap.Detail
	for _, sc := range []struct {
		keys []string
		desc string
	}{
		{l.Open.Keys(), "Open children (or details)"},
		{l.View.Keys(), "View details"},
		{l.Back.Keys(), "Back up a level"},
		{l.Create.Keys(), "Create new bean"},
		{l.CreateChild.Keys(), "Create child bean"},
		{l.Launch.Keys(), "Launch with tool"},
		{l.ToggleLog.Keys(), "Toggle log panel"},
		{l.OpenLog.Keys(), "Open latest log output"},
		{l.Edit.Keys(), "Edit bean"},
		{l.OpenEditor.Keys(), "Edit in $EDITOR"},
		{l.ChangeStatus.Keys(), "Change status"},
		{l.ChangeType.Keys(), "Change type"},
		{l.ChangePriority.Keys(), "Change priority"},
		{append(l.RaisePriority.Keys(), l.LowerPriority.Keys()...), "Raise / lower priority"},
		{l.ChangeSize.Keys(), "Change size"},
		{l.EditTags.Keys(), "Edit tags"},
		{l.ChangeParent.Keys(), "Set parent"},
		{l.ManageBlocking.Keys(), "Manage blocking"},
		{l.ManageBlockedBy.Keys(), "Manage blocked by"},
		{l.Archive.Keys(), "Archive (all completed in list)"},
		{l.ToggleArchived.Keys(), "Show / hide archived"},
		{l.Filter.Keys(), "Filter"},
		{l.FilterByTag.Keys(), "Filter by tag"},
		{l.Collapse.Keys(), "Collapse item"},
		{l.Expand.Keys(), "Expand item"},
		{append(d.PrevBean.Keys(), d.NextBean.Keys()...), "Prev / next bean (details)"},
		{l.ToggleCollapse.Keys(), "Toggle collapse"},
		{l.CollapseAll.Keys(), "Collapse all"},
		{g.ReloadConfig.Keys(), "Reload config"},
		{g.Quit.Keys(), "Quit"},
	} {
		if len(sc.keys) > 0 {
			content.WriteString(shortcut(keyLabel(sc.keys), sc.desc) + "\n")
		}
	}
	content.WriteString("\n")

	// Footer
	footer := helpKeyStyle.Render(keyLabel(keymap.Help.Close.Keys())) + " " + helpStyle.Render("close")

	// Border style
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary).
		Padding(1, 2).
		Width(modalWidth)

	return border.Render(content.String() + footer)
}

// ModalView returns the help overlay as a centered modal on top of the background
func (m helpOverlayModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	modal := m.View()
	return overlayModal(bgView, modal, fullWidth, fullHeight)
}
