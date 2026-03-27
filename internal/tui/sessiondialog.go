package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/sessionpoller"
	"github.com/hmans/beans/internal/ui"
)

var (
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#555"))
	runningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

type sessionItem struct {
	beanID    string
	beanTitle string
	status    sessionpoller.SessionStatus
}

func (s sessionItem) Title() string { return s.beanTitle }
func (s sessionItem) Description() string {
	if s.status.Error != "" {
		return s.status.Error
	}
	return s.beanID
}
func (s sessionItem) FilterValue() string { return s.beanTitle + " " + s.beanID }

type sessionDelegate struct{}

func (d sessionDelegate) Height() int                             { return 1 }
func (d sessionDelegate) Spacing() int                            { return 0 }
func (d sessionDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d sessionDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(sessionItem)
	if !ok {
		return
	}

	cursor := "  "
	if index == m.Index() {
		cursor = ui.Primary.Render("▸ ")
	}

	statusStr := "no session"
	statusStyle := dimStyle
	if item.status.HasSession {
		statusStr = "running"
		statusStyle = runningStyle
	} else if item.status.Error != "" {
		statusStr = "error: " + item.status.Error
		statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	}

	status := statusStyle.Render(statusStr)
	id := ui.ID.Render(item.beanID)

	row := cursor + id + "  " + status + "  " + item.beanTitle
	fmt.Fprint(w, row)
}

type openSessionDialogMsg struct {
	beanID string // optional - filter to this bean
}

type closeSessionDialogMsg struct{}

type attachSessionMsg struct {
	beanID    string
	beanTitle string
}

type launchSessionMsg struct {
	beanID    string
	beanTitle string
}

type sessionDialogModel struct {
	list   list.Model
	poller *sessionpoller.Poller
	beans  []*bean.Bean
	width  int
	height int
}

func newSessionDialogModel(poller *sessionpoller.Poller, beans []*bean.Bean, width, height int) sessionDialogModel {
	delegate := sessionDelegate{}

	l := list.New([]list.Item{}, delegate, width-4, height-8)
	l.Title = "Active Sessions"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.Styles.Title = listTitleStyle
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 1)

	m := sessionDialogModel{
		list:   l,
		poller: poller,
		beans:  beans,
		width:  width,
		height: height,
	}

	m.refreshSessions()
	return m
}

func (m *sessionDialogModel) refreshSessions() {
	var items []list.Item
	for _, b := range m.beans {
		status := m.poller.GetStatus(b.ID)
		if status.HasSession || status.Error != "" {
			items = append(items, sessionItem{
				beanID:    b.ID,
				beanTitle: b.Title,
				status:    status,
			})
		}
	}
	m.list.SetItems(items)

	if len(items) == 0 {
		m.list.Title = "Active Sessions (none)"
	} else {
		m.list.Title = "Active Sessions"
	}
}

func (m *sessionDialogModel) Init() tea.Cmd {
	return nil
}

func (m *sessionDialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-8)

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if item, ok := m.list.SelectedItem().(sessionItem); ok {
				return m, func() tea.Msg {
					return attachSessionMsg{beanID: item.beanID, beanTitle: item.beanTitle}
				}
			}
		case "esc", "q":
			return m, func() tea.Msg {
				return closeSessionDialogMsg{}
			}
		}
	}

	m.list, _ = m.list.Update(msg)
	return m, nil
}

func (m *sessionDialogModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if len(m.list.Items()) == 0 {
		return m.list.Styles.Title.Render("Active Sessions") + "\n\n  No active sessions.\n\n  Press Esc to close."
	}

	help := "\n  Enter: attach    Esc: close"
	return m.list.View() + help
}
