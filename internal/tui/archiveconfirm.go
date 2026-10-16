package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/ui"
)

// archiveCompletedMsg requests archiving all beans with an archive status
type archiveCompletedMsg struct{}

// archiveBeanMsg requests archiving a single bean
type archiveBeanMsg struct {
	beanID string
}

// archiveConfirmedMsg is sent when the user confirms archiving
type archiveConfirmedMsg struct {
	beanIDs []string
}

// closeArchiveConfirmMsg is sent when the user cancels archiving
type closeArchiveConfirmMsg struct{}

// archiveConfirmModel asks the user to confirm archiving bean(s).
// Confirming with "y" means "yy" always archives from the detail view.
type archiveConfirmModel struct {
	beanIDs []string
	title   string
	message string
	width   int
	height  int
}

func newArchiveConfirmModel(beanIDs []string, title, message string, width, height int) archiveConfirmModel {
	return archiveConfirmModel{
		beanIDs: beanIDs,
		title:   title,
		message: message,
		width:   width,
		height:  height,
	}
}

func (m archiveConfirmModel) Update(msg tea.Msg) (archiveConfirmModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y", "Y":
			ids := m.beanIDs
			return m, func() tea.Msg { return archiveConfirmedMsg{beanIDs: ids} }
		case "n", "N", "esc", "backspace":
			return m, func() tea.Msg { return closeArchiveConfirmMsg{} }
		}
	}
	return m, nil
}

func (m archiveConfirmModel) View() string {
	modalWidth := max(40, min(60, m.width*50/100))

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.ColorWarning).
		Render(m.title)

	help := helpKeyStyle.Render("y") + " " + helpStyle.Render("archive") + "  " +
		helpKeyStyle.Render("n/esc") + " " + helpStyle.Render("cancel")

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorWarning).
		Padding(1, 2).
		Width(modalWidth)

	return border.Render(title + "\n\n" + m.message + "\n\n" + help)
}

// ModalView returns the confirmation as a centered modal on top of the background
func (m archiveConfirmModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	return overlayModal(bgView, m.View(), fullWidth, fullHeight)
}
