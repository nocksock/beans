package tui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/ui"
)

// sizeSelectedMsg is sent when a size is selected from the picker
type sizeSelectedMsg struct {
	beanIDs []string
	size    string
}

// closeSizePickerMsg is sent when the size picker is cancelled
type closeSizePickerMsg struct{}

// openSizePickerMsg requests opening the size picker for bean(s)
type openSizePickerMsg struct {
	beanIDs     []string // IDs of beans to update
	beanTitle   string   // Display title (single title or "N beans")
	currentSize string   // Only meaningful for single bean
}

// sizeItem wraps a size to implement list.Item
type sizeItem struct {
	name        string
	description string
	color       string
	isCurrent   bool
}

func (i sizeItem) Title() string       { return i.name }
func (i sizeItem) Description() string { return i.description }
func (i sizeItem) FilterValue() string { return i.name + " " + i.description }

// sizeItemDelegate handles rendering of size picker items
type sizeItemDelegate struct{}

func (d sizeItemDelegate) Height() int                             { return 1 }
func (d sizeItemDelegate) Spacing() int                            { return 0 }
func (d sizeItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d sizeItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(sizeItem)
	if !ok {
		return
	}

	var cursor string
	if index == m.Index() {
		cursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render("▌") + " "
	} else {
		cursor = "  "
	}

	var currentIndicator string
	if item.isCurrent {
		currentIndicator = ui.Muted.Render(" (current)")
	}

	fmt.Fprint(w, cursor+ui.RenderSizeText(item.name, item.color)+currentIndicator)
}

// sizePickerModel is the model for the size picker view
type sizePickerModel struct {
	list      list.Model
	beanIDs   []string
	beanTitle string
	width     int
	height    int
}

// sizePickerListSize returns the list dimensions for the picker modal.
func sizePickerListSize(width, height int) (int, int) {
	modalWidth := max(40, min(60, width*50/100))
	modalHeight := max(10, min(16, height*50/100))
	return modalWidth - 6, modalHeight - 7
}

func newSizePickerModel(beanIDs []string, beanTitle, currentSize string, width, height int) sizePickerModel {
	// An unset size counts as the default size
	currentSize = config.EffectiveSize(currentSize)

	items := make([]list.Item, 0, len(config.DefaultSizes))
	selectedIndex := 0
	for i, s := range config.DefaultSizes {
		isCurrent := s.Name == currentSize
		if isCurrent {
			selectedIndex = i
		}
		items = append(items, sizeItem{
			name:        s.Name,
			description: s.Description,
			color:       s.Color,
			isCurrent:   isCurrent,
		})
	}

	listWidth, listHeight := sizePickerListSize(width, height)
	l := list.New(items, sizeItemDelegate{}, listWidth, listHeight)
	l.Title = "Select Size"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.KeyMap = pickerListKeyMap()
	l.SetShowPagination(false)
	l.Styles.Title = listTitleStyle
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 0, 0)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	l.Select(selectedIndex)

	return sizePickerModel{
		list:      l,
		beanIDs:   beanIDs,
		beanTitle: beanTitle,
		width:     width,
		height:    height,
	}
}

func (m sizePickerModel) Init() tea.Cmd {
	return nil
}

func (m sizePickerModel) Update(msg tea.Msg) (sizePickerModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(sizePickerListSize(msg.Width, msg.Height))

	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering {
			switch {
			case key.Matches(msg, keymap.Picker.Confirm):
				if item, ok := m.list.SelectedItem().(sizeItem); ok {
					return m, func() tea.Msg {
						return sizeSelectedMsg{beanIDs: m.beanIDs, size: item.name}
					}
				}
			case key.Matches(msg, keymap.Picker.Cancel):
				return m, func() tea.Msg {
					return closeSizePickerMsg{}
				}
			}
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m sizePickerModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var description string
	if item, ok := m.list.SelectedItem().(sizeItem); ok {
		description = item.description
	}

	// For multi-select, don't show individual bean ID
	var beanID string
	if len(m.beanIDs) == 1 {
		beanID = m.beanIDs[0]
	}

	return renderPickerModal(pickerModalConfig{
		Title:       "Select Size",
		BeanTitle:   m.beanTitle,
		BeanID:      beanID,
		ListContent: m.list.View(),
		Description: description,
		Width:       m.width,
	})
}

// ModalView returns the picker rendered as a centered modal overlay on top of the background
func (m sizePickerModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	return overlayModal(bgView, m.View(), fullWidth, fullHeight)
}
