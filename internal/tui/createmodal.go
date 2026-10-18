package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
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
	size       string // optional size (empty uses the default)
	body       string // optional markdown body
	parentID   string // optional parent ID for the new bean
	prefix     string // ID prefix for the new bean
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

// selectorColumnsMinWidth is the width the type, status, priority and size
// selectors need side by side
const selectorColumnsMinWidth = 13 + 15 + 12 + 6

// editModalMinBodyHeight is the minimum height of the description field when editing
const editModalMinBodyHeight = 5

// createField identifies a focusable field in the create form
type createField int

const (
	createFieldTitle createField = iota
	createFieldPrefix
	createFieldType
	createFieldStatus
	createFieldPriority
	createFieldSize
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
	sizes       []config.SizeConfig
	prefixes    []string // ID prefixes to choose from; the first is the default
	prefixIdx   int
	typeIdx     int
	statusIdx   int
	priorityIdx int
	sizeIdx     int
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
	origSizeIdx  int
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
	m.sizeIdx = indexByName(m.sizes, func(s config.SizeConfig) string { return s.Name }, config.EffectiveSize(b.Size))
	m.origTypeIdx, m.origStatIdx, m.origPrioIdx, m.origSizeIdx = m.typeIdx, m.statusIdx, m.priorityIdx, m.sizeIdx
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
		statuses:    cfg.Statuses(),
		priorities:  config.DefaultPriorities,
		sizes:       config.DefaultSizes,
		prefixes:    []string{cfg.Beans.Prefix},
		width:       width,
		height:      height,
		parentID:    parentID,
		parentTitle: parentTitle,
	}
	m.typeIdx = indexByName(m.types, func(t config.TypeConfig) string { return t.Name }, cfg.GetDefaultType())
	m.statusIdx = indexByName(m.statuses, func(s config.StatusConfig) string { return s.Name }, createModalDefaultStatus)
	m.priorityIdx = indexByName(m.priorities, func(p config.PriorityConfig) string { return p.Name }, createModalDefaultPriority)
	m.sizeIdx = indexByName(m.sizes, func(s config.SizeConfig) string { return s.Name }, config.DefaultSize)
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
	return max(52, min(72, m.width*60/100))
}

// selectedSize returns the chosen size, or "" for the default size so that
// beans without an explicit size don't get one written.
func (m createModalModel) selectedSize() string {
	if name := m.sizes[m.sizeIdx].Name; name != config.DefaultSize {
		return name
	}
	return ""
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

// moveFocus moves focus by delta fields, skipping fields that can't be changed
func (m *createModalModel) moveFocus(delta createField) tea.Cmd {
	f := (m.focus + delta + createFieldCount) % createFieldCount
	if f == createFieldPrefix && !m.prefixSelectable() {
		f = (f + delta + createFieldCount) % createFieldCount
	}
	return m.setFocus(f)
}

// prefixSelectable reports whether the prefix field can be changed: only when
// creating, with several prefixes to choose from.
func (m createModalModel) prefixSelectable() bool {
	return !m.isEdit() && len(m.prefixes) > 1
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
		size:       m.selectedSize(),
		body:       body,
		parentID:   m.parentID,
		prefix:     m.prefixes[m.prefixIdx],
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
	if m.sizeIdx != m.origSizeIdx {
		size := m.selectedSize()
		msg.input.Size = &size
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
	case createFieldPrefix:
		m.prefixIdx = (m.prefixIdx + delta + len(m.prefixes)) % len(m.prefixes)
	case createFieldType:
		m.typeIdx = (m.typeIdx + delta + len(m.types)) % len(m.types)
	case createFieldStatus:
		m.statusIdx = (m.statusIdx + delta + len(m.statuses)) % len(m.statuses)
	case createFieldPriority:
		m.priorityIdx = (m.priorityIdx + delta + len(m.priorities)) % len(m.priorities)
	case createFieldSize:
		m.sizeIdx = (m.sizeIdx + delta + len(m.sizes)) % len(m.sizes)
	}
}

func (m createModalModel) isSelectorFocused() bool {
	switch m.focus {
	case createFieldPrefix, createFieldType, createFieldStatus, createFieldPriority, createFieldSize:
		return true
	}
	return false
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
		k := keymap.CreateModal
		switch {
		case key.Matches(msg, k.Cancel):
			return m, func() tea.Msg { return closeCreateModalMsg{} }
		case key.Matches(msg, k.NextField):
			return m, m.moveFocus(1)
		case key.Matches(msg, k.PrevField):
			return m, m.moveFocus(-1)
		case key.Matches(msg, k.Submit):
			return m.trySubmit(false)
		case key.Matches(msg, k.SubmitAndEdit):
			return m.trySubmit(true)
		case key.Matches(msg, k.QuickSubmit):
			// Enter inserts newlines in the description; elsewhere it submits
			if m.focus != createFieldBody {
				return m.trySubmit(false)
			}
		}

		if m.isSelectorFocused() {
			switch {
			case key.Matches(msg, k.CyclePrev):
				m.cycle(-1)
			case key.Matches(msg, k.CycleNext):
				m.cycle(1)
			case key.Matches(msg, k.QuickSubmitAndEdit):
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
	sz := m.sizes[m.sizeIdx]
	sizeField := label(createFieldSize, "Size") + "\n" +
		selector(createFieldSize, ui.RenderSizeText(sz.Name, sz.Color))
	// Columns fit their longest values ("milestone", "in-progress", "deferred");
	// extra room is shared between them, and size takes the rest.
	extra := max(0, fieldWidth-selectorColumnsMinWidth) / 3
	selectors := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(13+extra).Render(typeField),
		lipgloss.NewStyle().Width(15+extra).Render(statusField),
		lipgloss.NewStyle().Width(12+extra).Render(priorityField),
		sizeField,
	)

	// The prefix always renders when creating, so the layout doesn't depend on
	// how many prefixes there are; with only one it can't be changed
	var prefixRow string
	if !m.isEdit() {
		prefix := m.prefixes[m.prefixIdx]
		if prefix == "" {
			prefix = "(none)"
		}
		value := lipgloss.NewStyle().Foreground(ui.ColorMuted).Render("  " + prefix)
		if m.prefixSelectable() {
			value = selector(createFieldPrefix, prefix)
		}
		prefixRow = label(createFieldPrefix, "ID prefix") + " " + value + "\n\n"
	}

	// Help text depends on the focused field
	k := keymap.CreateModal
	verb := "create"
	if m.isEdit() {
		verb = "save"
	}
	var help string
	switch m.focus {
	case createFieldBody:
		help = hint(k.Submit, verb)
	case createFieldTitle:
		help = hint(k.QuickSubmit, verb)
	default:
		help = groupHint("change", k.CyclePrev, k.CycleNext) +
			hint(k.QuickSubmit, verb) +
			hint(k.QuickSubmitAndEdit, verb+" & edit")
	}
	if !m.isSelectorFocused() {
		help += hint(k.SubmitAndEdit, verb+" & edit")
	}
	help = hints(help, hint(k.NextField, ""), hint(k.Cancel, ""))

	// Assemble content
	content := header + "\n\n" +
		label(createFieldTitle, "Title") + "\n" + box(createFieldTitle, m.textInput.View()) + "\n\n" +
		prefixRow +
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
