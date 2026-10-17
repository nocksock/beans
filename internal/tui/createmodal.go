package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/ui"
)

// createModalDefaultStatus is the initial status for beans created in the TUI.
// New beans start as drafts so they can be refined after a quick capture.
const createModalDefaultStatus = "draft"

// createModalDefaultPriority is the initial priority for beans created in the TUI.
const createModalDefaultPriority = "normal"

// beanCreatedMsg is sent when a new bean is created
type beanCreatedMsg struct {
	title      string
	beanType   string // optional type (empty uses the configured default)
	status     string // optional status (empty uses draft)
	priority   string // optional priority (empty uses the default)
	body       string // optional markdown body
	parentID   string // optional parent ID for the new bean
	openEditor bool   // open the new bean in $EDITOR after creating it
}

// closeCreateModalMsg is sent when the create modal is cancelled
type closeCreateModalMsg struct{}

// openCreateModalMsg requests opening the create modal
type openCreateModalMsg struct {
	parentID    string // optional parent ID for the new bean
	parentTitle string // display title of the parent (for UI)
}

// openEditModalMsg requests opening the form to edit an existing bean
type openEditModalMsg struct {
	bean *bean.Bean
}

// beanEditedMsg is sent when the edit form is saved. input only carries the
// fields that changed, so an unchanged form results in an empty input.
type beanEditedMsg struct {
	beanID     string
	input      model.UpdateBeanInput
	openEditor bool // open the bean in $EDITOR after saving
}

// editModalChromeHeight is the number of rows the edit form needs besides the
// description field (modal border and padding, header, title, selectors, help)
const editModalChromeHeight = 24

// editModalMinBodyHeight is the minimum height of the description field when editing
const editModalMinBodyHeight = 5

// createField identifies a focusable field in the create form
type createField int

const (
	createFieldTitle createField = iota
	createFieldType
	createFieldStatus
	createFieldPriority
	createFieldBody
	createFieldCount
)

// createModalModel is the model for the create bean form
type createModalModel struct {
	textInput   textinput.Model
	body        textarea.Model
	types       []config.TypeConfig
	statuses    []config.StatusConfig
	priorities  []config.PriorityConfig
	typeIdx     int
	statusIdx   int
	priorityIdx int
	focus       createField
	width       int
	height      int
	parentID    string // optional parent ID for the new bean
	parentTitle string // display title of the parent (for UI)

	// Edit mode: set when the form edits an existing bean instead of creating one
	beanID       string
	origTitle    string
	origBody     string
	origTypeIdx  int
	origStatIdx  int
	origPrioIdx  int
	errorMessage string
}

// newEditModalModel returns the form prefilled with an existing bean's fields
func newEditModalModel(cfg *config.Config, width, height int, b *bean.Bean) createModalModel {
	m := newCreateModalModel(cfg, width, height, "", "")
	m.beanID = b.ID
	m.origTitle = strings.TrimSpace(b.Title)
	m.origBody = strings.TrimSpace(b.Body)
	m.textInput.SetValue(m.origTitle)
	m.body.SetValue(m.origBody)
	m.typeIdx = indexByName(m.types, func(t config.TypeConfig) string { return t.Name }, b.Type)
	m.statusIdx = indexByName(m.statuses, func(s config.StatusConfig) string { return s.Name }, b.Status)
	m.priorityIdx = indexByName(m.priorities, func(p config.PriorityConfig) string { return p.Name }, b.Priority)
	m.origTypeIdx, m.origStatIdx, m.origPrioIdx = m.typeIdx, m.statusIdx, m.priorityIdx
	m.resize()
	return m
}

func (m createModalModel) isEdit() bool {
	return m.beanID != ""
}

func newCreateModalModel(cfg *config.Config, width, height int, parentID, parentTitle string) createModalModel {
	ti := textinput.New()
	ti.Placeholder = "Enter bean title (optional)..."
	ti.CharLimit = 200
	ti.Focus()
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	ti.TextStyle = lipgloss.NewStyle()
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(ui.ColorMuted)
	ti.Prompt = ""

	ta := textarea.New()
	ta.Placeholder = "Description (markdown, optional)..."
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.CharLimit = 0
	ta.SetHeight(5)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(ui.ColorMuted)
	ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(ui.ColorMuted)
	ta.Blur()

	m := createModalModel{
		textInput:   ti,
		body:        ta,
		types:       config.DefaultTypes,
		statuses:    config.DefaultStatuses,
		priorities:  config.DefaultPriorities,
		width:       width,
		height:      height,
		parentID:    parentID,
		parentTitle: parentTitle,
	}
	m.typeIdx = indexByName(m.types, func(t config.TypeConfig) string { return t.Name }, cfg.GetDefaultType())
	m.statusIdx = indexByName(m.statuses, func(s config.StatusConfig) string { return s.Name }, createModalDefaultStatus)
	m.priorityIdx = indexByName(m.priorities, func(p config.PriorityConfig) string { return p.Name }, createModalDefaultPriority)
	m.resize()
	return m
}

// indexByName returns the index of the item with the given name, or 0 if not found
func indexByName[T any](items []T, nameOf func(T) string, name string) int {
	for i, item := range items {
		if nameOf(item) == name {
			return i
		}
	}
	return 0
}

func (m createModalModel) modalWidth() int {
	return max(50, min(72, m.width*60/100))
}

// resize fits the text fields to the modal width
func (m *createModalModel) resize() {
	inner := m.modalWidth() - 10 // modal border+padding, field border+padding
	m.textInput.Width = inner
	m.body.SetWidth(inner)
	if m.isEdit() {
		m.body.SetHeight(max(editModalMinBodyHeight, m.height-editModalChromeHeight))
	}
}

func (m createModalModel) Init() tea.Cmd {
	return textinput.Blink
}

// setFocus moves focus to the given field, (un)focusing the text inputs accordingly
func (m *createModalModel) setFocus(f createField) tea.Cmd {
	m.focus = (f + createFieldCount) % createFieldCount
	m.textInput.Blur()
	m.body.Blur()
	switch m.focus {
	case createFieldTitle:
		return m.textInput.Focus()
	case createFieldBody:
		return m.body.Focus()
	}
	return nil
}

// submit returns a command that creates the bean, or closes the modal if both
// title and description are empty (the title is optional when a description is given)
func (m createModalModel) submit(openEditor bool) tea.Cmd {
	if m.isEdit() {
		return m.submitEdit(openEditor)
	}
	title := strings.TrimSpace(m.textInput.Value())
	body := strings.TrimSpace(m.body.Value())
	if title == "" && body == "" {
		return func() tea.Msg { return closeCreateModalMsg{} }
	}
	msg := beanCreatedMsg{
		title:      title,
		beanType:   m.types[m.typeIdx].Name,
		status:     m.statuses[m.statusIdx].Name,
		priority:   m.priorities[m.priorityIdx].Name,
		body:       body,
		parentID:   m.parentID,
		openEditor: openEditor,
	}
	return func() tea.Msg { return msg }
}

// submitEdit returns a command that saves the changed fields of the edited bean.
// Selectors only count as changed when moved, so a bean whose type, status or
// priority is unknown to the config keeps its value.
func (m createModalModel) submitEdit(openEditor bool) tea.Cmd {
	title := strings.TrimSpace(m.textInput.Value())
	body := strings.TrimSpace(m.body.Value())
	msg := beanEditedMsg{beanID: m.beanID, openEditor: openEditor}
	if title != m.origTitle {
		msg.input.Title = &title
	}
	if body != m.origBody {
		msg.input.Body = &body
	}
	if m.typeIdx != m.origTypeIdx {
		msg.input.Type = &m.types[m.typeIdx].Name
	}
	if m.statusIdx != m.origStatIdx {
		msg.input.Status = &m.statuses[m.statusIdx].Name
	}
	if m.priorityIdx != m.origPrioIdx {
		msg.input.Priority = &m.priorities[m.priorityIdx].Name
	}
	return func() tea.Msg { return msg }
}

// validate reports why the form can't be submitted, or "" if it can
func (m createModalModel) validate() string {
	if m.isEdit() && strings.TrimSpace(m.textInput.Value()) == "" && strings.TrimSpace(m.body.Value()) == "" {
		return "A bean needs a title or a description"
	}
	return ""
}

// trySubmit submits the form, or shows a validation error instead
func (m createModalModel) trySubmit(openEditor bool) (createModalModel, tea.Cmd) {
	m.errorMessage = m.validate()
	if m.errorMessage != "" {
		return m, nil
	}
	return m, m.submit(openEditor)
}

// cycle moves the selection of the focused selector field by delta
func (m *createModalModel) cycle(delta int) {
	switch m.focus {
	case createFieldType:
		m.typeIdx = (m.typeIdx + delta + len(m.types)) % len(m.types)
	case createFieldStatus:
		m.statusIdx = (m.statusIdx + delta + len(m.statuses)) % len(m.statuses)
	case createFieldPriority:
		m.priorityIdx = (m.priorityIdx + delta + len(m.priorities)) % len(m.priorities)
	}
}

func (m createModalModel) isSelectorFocused() bool {
	return m.focus == createFieldType || m.focus == createFieldStatus || m.focus == createFieldPriority
}

func (m createModalModel) Update(msg tea.Msg) (createModalModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return closeCreateModalMsg{} }
		case "tab":
			return m, m.setFocus(m.focus + 1)
		case "shift+tab":
			return m, m.setFocus(m.focus - 1)
		case "ctrl+s":
			return m.trySubmit(false)
		case "ctrl+e":
			return m.trySubmit(true)
		case "enter":
			// Enter inserts newlines in the description; elsewhere it submits
			if m.focus != createFieldBody {
				return m.trySubmit(false)
			}
		}

		if m.isSelectorFocused() {
			switch msg.String() {
			case "left", "h", "up", "k":
				m.cycle(-1)
			case "right", "l", "down", "j", " ":
				m.cycle(1)
			case "e":
				return m.trySubmit(true)
			}
			return m, nil
		}
	}

	switch m.focus {
	case createFieldTitle:
		m.textInput, cmd = m.textInput.Update(msg)
	case createFieldBody:
		m.body, cmd = m.body.Update(msg)
	}
	return m, cmd
}

func (m createModalModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	modalWidth := m.modalWidth()
	fieldWidth := modalWidth - 6

	// Header - show parent context if creating a child
	var header string
	if m.isEdit() {
		header = lipgloss.NewStyle().Bold(true).Render("Edit Bean") + "\n" +
			lipgloss.NewStyle().Foreground(ui.ColorMuted).Render(m.beanID)
	} else if m.parentTitle != "" {
		header = lipgloss.NewStyle().Bold(true).Render("Create Child Bean")
		// Truncate parent title if too long
		parentDisplay := m.parentTitle
		maxParentLen := modalWidth - 16 // account for "Parent: " prefix and padding
		if len(parentDisplay) > maxParentLen {
			parentDisplay = parentDisplay[:maxParentLen-3] + "..."
		}
		parentLine := lipgloss.NewStyle().Foreground(ui.ColorMuted).Render("Parent: " + parentDisplay)
		header = header + "\n" + parentLine
	} else {
		header = lipgloss.NewStyle().Bold(true).Render("Create New Bean")
	}

	label := func(f createField, text string) string {
		style := lipgloss.NewStyle().Foreground(ui.ColorMuted)
		if m.focus == f {
			style = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
		}
		return style.Render(text)
	}
	box := func(f createField, content string) string {
		borderColor := ui.ColorMuted
		if m.focus == f {
			borderColor = ui.ColorPrimary
		}
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1).
			Width(fieldWidth).
			Render(content)
	}
	selector := func(f createField, value string) string {
		if m.focus == f {
			arrow := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
			return arrow.Render("‹ ") + value + arrow.Render(" ›")
		}
		return "  " + value + "  "
	}

	t := m.types[m.typeIdx]
	s := m.statuses[m.statusIdx]
	typeField := label(createFieldType, "Type") + "\n" +
		selector(createFieldType, ui.RenderTypeText(t.Name, t.Color))
	statusField := label(createFieldStatus, "Status") + "\n" +
		selector(createFieldStatus, ui.RenderStatusTextWithColor(s.Name, s.Color, s.Archive))
	p := m.priorities[m.priorityIdx]
	priorityField := label(createFieldPriority, "Priority") + "\n" +
		selector(createFieldPriority, ui.RenderPriorityText(p.Name, p.Color))
	column := lipgloss.NewStyle().Width(fieldWidth / 3)
	selectors := lipgloss.JoinHorizontal(lipgloss.Top,
		column.Render(typeField),
		column.Render(statusField),
		priorityField,
	)

	// Help text depends on the focused field
	verb := "create"
	if m.isEdit() {
		verb = "save"
	}
	var help string
	switch m.focus {
	case createFieldBody:
		help = helpKeyStyle.Render("ctrl+s") + " " + helpStyle.Render(verb) + "  "
	case createFieldTitle:
		help = helpKeyStyle.Render("enter") + " " + helpStyle.Render(verb) + "  "
	default:
		help = helpKeyStyle.Render("←/→") + " " + helpStyle.Render("change") + "  " +
			helpKeyStyle.Render("enter") + " " + helpStyle.Render(verb) + "  " +
			helpKeyStyle.Render("e") + " " + helpStyle.Render(verb+" & edit") + "  "
	}
	if !m.isSelectorFocused() {
		help += helpKeyStyle.Render("ctrl+e") + " " + helpStyle.Render(verb+" & edit") + "  "
	}
	help += helpKeyStyle.Render("tab") + " " + helpStyle.Render("next") + "  " +
		helpKeyStyle.Render("esc") + " " + helpStyle.Render("cancel")

	// Assemble content
	content := header + "\n\n" +
		label(createFieldTitle, "Title") + "\n" + box(createFieldTitle, m.textInput.View()) + "\n\n" +
		selectors + "\n\n" +
		label(createFieldBody, "Description") + "\n" + box(createFieldBody, m.body.View()) + "\n\n"
	if m.errorMessage != "" {
		content += lipgloss.NewStyle().Foreground(ui.ColorDanger).Render(m.errorMessage) + "\n"
	}
	content += lipgloss.NewStyle().Width(fieldWidth).Render(help)

	// Border style
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary).
		Padding(1, 2).
		Width(modalWidth)

	return border.Render(content)
}

// ModalView returns the modal rendered as a centered overlay
func (m createModalModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	modal := m.View()
	return overlayModal(bgView, modal, fullWidth, fullHeight)
}
