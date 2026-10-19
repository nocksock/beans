package tui

import (
	"slices"
	"strings"
	"time"

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
	tags       []string
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
	prefill     beanPrefill
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

// editModalSlackHeight is the number of rows kept free below the edit form
// for the error line and a wrapped help line
const editModalSlackHeight = 4

// editModalChromeHeight is the number of rows the edit form needs besides the
// description field (modal border and padding, header, title, selectors, help)
// plus the slack
const editModalChromeHeight = 18 + editModalSlackHeight

// selectorColumnWidths are the widths of the type, status, priority and size
// columns: their longest values ("milestone", "in-progress", "deferred") plus
// the selector padding
var selectorColumnWidths = []int{13, 15, 12, 6}

// selectorPadding is the room a selector adds around its value for the arrows
const selectorPadding = 4

// editModalMinBodyHeight is the minimum height of the description field when editing
const editModalMinBodyHeight = 5

// typeAheadTimeout is how long a selector waits for the next letter of a
// type-ahead prefix before starting a new one
const typeAheadTimeout = time.Second

// createField identifies a focusable field in the create form
type createField int

// The order is the tab order, which follows the layout
const (
	createFieldTitle createField = iota
	createFieldBody
	createFieldPrefix
	createFieldType
	createFieldStatus
	createFieldPriority
	createFieldSize
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
	typeAhead   bool      // the focused selector selects by typed letters
	typed       string    // the type-ahead prefix typed so far
	typedAt     time.Time // when the last type-ahead letter was typed
	now         func() time.Time
	width       int
	height      int
	parentID    string // optional parent ID for the new bean
	parentTitle string // display title of the parent (for UI)
	tags        []string

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
		now:         time.Now,
	}
	m.typeIdx = indexByName(m.types, func(t config.TypeConfig) string { return t.Name }, cfg.GetDefaultType())
	m.statusIdx = indexByName(m.statuses, func(s config.StatusConfig) string { return s.Name }, createModalDefaultStatus)
	m.priorityIdx = indexByName(m.priorities, func(p config.PriorityConfig) string { return p.Name }, createModalDefaultPriority)
	m.sizeIdx = indexByName(m.sizes, func(s config.SizeConfig) string { return s.Name }, config.DefaultSize)
	m.resize()
	return m
}

// withPrefill preselects the given fields; empty or unknown values keep the defaults
func (m createModalModel) withPrefill(p beanPrefill) createModalModel {
	prefill := func(idx *int, names []string, name string) {
		if i := slices.Index(names, name); name != "" && i >= 0 {
			*idx = i
		}
	}
	prefill(&m.typeIdx, namesOf(m.types, func(t config.TypeConfig) string { return t.Name }), p.Type)
	prefill(&m.statusIdx, namesOf(m.statuses, func(s config.StatusConfig) string { return s.Name }), p.Status)
	prefill(&m.priorityIdx, namesOf(m.priorities, func(p config.PriorityConfig) string { return p.Name }), p.Priority)
	prefill(&m.sizeIdx, namesOf(m.sizes, func(s config.SizeConfig) string { return s.Name }), p.Size)
	m.tags = p.Tags
	return m
}

func namesOf[T any](items []T, nameOf func(T) string) []string {
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = nameOf(item)
	}
	return names
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

// modalWidth is 60% of the terminal, between 52 columns and 72, but always
// wide enough for the selector row
func (m createModalModel) modalWidth() int {
	minWidth := 6 + sum(m.selectorColumns()) // modal border and padding
	return max(minWidth, min(72, m.width*60/100))
}

func sum(ns []int) int {
	total := 0
	for _, n := range ns {
		total += n
	}
	return total
}

// selectorColumns returns the width of each selector column, with the prefix
// column first when creating
func (m createModalModel) selectorColumns() []int {
	if m.isEdit() {
		return selectorColumnWidths
	}
	return append([]int{m.prefixColumnWidth()}, selectorColumnWidths...)
}

// prefixColumnWidth fits the longest prefix with its selector padding, and at
// least the "Prefix" label with a gap
func (m createModalModel) prefixColumnWidth() int {
	longest := len("(none)")
	for _, p := range m.prefixes {
		longest = max(longest, len(p))
	}
	return max(len("Prefix")+2, longest+selectorPadding)
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
	m.typeAhead = false
	m.typed = ""
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
		tags:       m.tags,
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

// selectorValues returns the value names of a selector field
func (m createModalModel) selectorValues(f createField) []string {
	switch f {
	case createFieldPrefix:
		return m.prefixes
	case createFieldType:
		return namesOf(m.types, func(t config.TypeConfig) string { return t.Name })
	case createFieldStatus:
		return namesOf(m.statuses, func(s config.StatusConfig) string { return s.Name })
	case createFieldPriority:
		return namesOf(m.priorities, func(p config.PriorityConfig) string { return p.Name })
	case createFieldSize:
		return namesOf(m.sizes, func(s config.SizeConfig) string { return s.Name })
	}
	return nil
}

// selectorIndex returns the selected index of a selector field
func (m *createModalModel) selectorIndex(f createField) *int {
	switch f {
	case createFieldPrefix:
		return &m.prefixIdx
	case createFieldType:
		return &m.typeIdx
	case createFieldStatus:
		return &m.statusIdx
	case createFieldPriority:
		return &m.priorityIdx
	case createFieldSize:
		return &m.sizeIdx
	}
	return nil
}

// selectedName returns the selected value of a selector field
func (m createModalModel) selectedName(f createField) string {
	return m.selectorValues(f)[*m.selectorIndex(f)]
}

// cycle moves the selection of the focused selector field by delta
func (m *createModalModel) cycle(delta int) {
	n := len(m.selectorValues(m.focus))
	if idx := m.selectorIndex(m.focus); idx != nil {
		*idx = (*idx + delta + n) % n
	}
}

// typeAheadKey selects the first value of the focused selector that starts
// with the letters typed so far. Typing the only letter again moves to the
// next value starting with it. A letter that matches nothing is ignored.
func (m *createModalModel) typeAheadKey(r rune) {
	now := m.now()
	if now.Sub(m.typedAt) > typeAheadTimeout {
		m.typed = ""
	}
	m.typedAt = now

	letter := strings.ToLower(string(r))
	values := m.selectorValues(m.focus)
	idx := m.selectorIndex(m.focus)
	from := 0
	prefix := m.typed + letter
	if m.typed == letter {
		from, prefix = *idx+1, letter
	}
	for i := range values {
		j := (from + i) % len(values)
		if strings.HasPrefix(strings.ToLower(values[j]), prefix) {
			*idx = j
			m.typed = prefix
			return
		}
	}
}

// selectorFields returns the selector fields that can be focused, left to right
func (m createModalModel) selectorFields() []createField {
	fields := []createField{createFieldType, createFieldStatus, createFieldPriority, createFieldSize}
	if m.prefixSelectable() {
		fields = append([]createField{createFieldPrefix}, fields...)
	}
	return fields
}

// moveSelector moves focus to the selector delta columns away, stopping at
// the ends of the row
func (m *createModalModel) moveSelector(delta int) {
	fields := m.selectorFields()
	i := slices.Index(fields, m.focus) + delta
	if i >= 0 && i < len(fields) {
		m.setFocus(fields[i])
	}
}

// focusField moves focus to a field picked by a jump chord: selectors start
// in type-ahead mode, a prefix that can't be changed keeps the focus
func (m *createModalModel) focusField(f createField) tea.Cmd {
	if f == createFieldPrefix && !m.prefixSelectable() {
		return nil
	}
	cmd := m.setFocus(f)
	m.typeAhead = m.isSelectorFocused()
	return cmd
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
		jumps := []struct {
			binding key.Binding
			field   createField
		}{
			{k.FocusTitle, createFieldTitle},
			{k.FocusBody, createFieldBody},
			{k.FocusPrefix, createFieldPrefix},
			{k.FocusType, createFieldType},
			{k.FocusStatus, createFieldStatus},
			{k.FocusPriority, createFieldPriority},
			{k.FocusSize, createFieldSize},
		}
		for _, j := range jumps {
			if key.Matches(msg, j.binding) {
				return m, m.focusField(j.field)
			}
		}

		switch {
		case key.Matches(msg, k.Cancel):
			if m.typeAhead {
				m.typeAhead = false
				return m, nil
			}
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
			return m.updateSelector(msg)
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

// updateSelector handles a key on a focused selector field
func (m createModalModel) updateSelector(msg tea.KeyMsg) (createModalModel, tea.Cmd) {
	k := keymap.CreateModal
	switch {
	case key.Matches(msg, k.CyclePrev):
		m.cycle(-1)
		return m, nil
	case key.Matches(msg, k.CycleNext):
		m.cycle(1)
		return m, nil
	}

	if !m.typeAhead {
		switch {
		case key.Matches(msg, k.NavLeft):
			m.moveSelector(-1)
			return m, nil
		case key.Matches(msg, k.NavRight):
			m.moveSelector(1)
			return m, nil
		case key.Matches(msg, k.NavUp):
			return m, m.setFocus(createFieldBody)
		case key.Matches(msg, k.NavDown):
			// The selectors are the bottom row
			return m, nil
		case key.Matches(msg, k.TypeAhead):
			m.typeAhead = true
			return m, nil
		case key.Matches(msg, k.QuickSubmitAndEdit):
			return m.trySubmit(true)
		}
	}

	// Any other letter types ahead, entering type-ahead mode
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && !msg.Alt {
		m.typeAhead = true
		m.typeAheadKey(msg.Runes[0])
	}
	return m, nil
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
	if len(m.tags) > 0 {
		header += "\n" + lipgloss.NewStyle().Foreground(ui.ColorMuted).Render("Tags: "+strings.Join(m.tags, ", "))
	}

	label := func(f createField, text string) string {
		style := lipgloss.NewStyle().Foreground(ui.ColorMuted)
		if m.focus == f {
			style = lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Underline(m.typeAhead)
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
	p := m.priorities[m.priorityIdx]
	sz := m.sizes[m.sizeIdx]
	fields := []string{
		label(createFieldType, "Type") + "\n" +
			selector(createFieldType, ui.RenderTypeText(t.Name, t.Color)),
		label(createFieldStatus, "Status") + "\n" +
			selector(createFieldStatus, ui.RenderStatusTextWithColor(s.Name, s.Color, s.Archive)),
		label(createFieldPriority, "Priority") + "\n" +
			selector(createFieldPriority, ui.RenderPriorityText(p.Name, p.Color)),
		label(createFieldSize, "Size") + "\n" +
			selector(createFieldSize, ui.RenderSizeText(sz.Name, sz.Color)),
	}
	// The prefix always renders when creating, so the layout doesn't depend on
	// how many prefixes there are; with only one it can't be changed
	if !m.isEdit() {
		prefix := m.prefixes[m.prefixIdx]
		if prefix == "" {
			prefix = "(none)"
		}
		value := lipgloss.NewStyle().Foreground(ui.ColorMuted).Render("  " + prefix)
		if m.prefixSelectable() {
			value = selector(createFieldPrefix, prefix)
		}
		fields = append([]string{label(createFieldPrefix, "Prefix") + "\n" + value}, fields...)
	}
	// Columns fit their longest values; extra room is shared between them,
	// and the last column takes the rest
	widths := m.selectorColumns()
	extra := max(0, fieldWidth-sum(widths)) / len(widths)
	for i := range fields[:len(fields)-1] {
		fields[i] = lipgloss.NewStyle().Width(widths[i] + extra).Render(fields[i])
	}
	selectors := lipgloss.JoinHorizontal(lipgloss.Top, fields...)

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
		if m.typeAhead {
			help = helpStyle.Render("type to select") + "  " +
				groupHint("change", k.CyclePrev, k.CycleNext) +
				hint(k.QuickSubmit, verb)
		} else {
			// The quick submit & edit key works too, but doesn't fit
			help = groupHint("move", k.NavLeft, k.NavRight) +
				groupHint("change", k.CyclePrev, k.CycleNext) +
				hint(k.TypeAhead, "type") +
				hint(k.QuickSubmit, verb)
		}
	}
	if !m.isSelectorFocused() {
		help += hint(k.SubmitAndEdit, verb+" & edit")
	}
	cancel := hint(k.Cancel, "")
	if m.typeAhead {
		cancel = hint(k.Cancel, "done")
	}
	help = hints(help, hint(k.NextField, ""), cancel)

	// Assemble content: the placeholders and the focused border identify the
	// text fields, so they have no labels
	content := header + "\n\n" +
		box(createFieldTitle, m.textInput.View()) + "\n\n" +
		box(createFieldBody, m.body.View()) + "\n\n" +
		selectors + "\n\n"
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
