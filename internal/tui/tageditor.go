package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/ui"
)

// tagsConfirmedMsg is sent when tag changes are confirmed
type tagsConfirmedMsg struct {
	beanIDs  []string // the beans being modified
	toAdd    []string // tags to add to all beans
	toRemove []string // tags to remove from all beans
}

// closeTagEditorMsg is sent when the tag editor is cancelled
type closeTagEditorMsg struct{}

// openTagEditorMsg requests opening the tag editor for one or more beans
type openTagEditorMsg struct {
	beanIDs   []string
	beanTitle string // display title (single title or "N selected beans")
}

// tagState describes how a tag is applied across the beans being edited
type tagState int

const (
	tagOnNone tagState = iota
	tagOnSome
	tagOnAll
)

// tagEditorModel is a modal for adding and removing tags on one or more beans.
// The text input filters the tag list and allows creating new tags.
type tagEditorModel struct {
	input     textinput.Model
	beanIDs   []string
	beanTitle string
	tags      []string            // all candidate tags, sorted
	usage     map[string]int      // project-wide usage count per tag
	original  map[string]tagState // state when the editor was opened
	pending   map[string]tagState // state after user toggles
	cursor    int
	width     int
	height    int
}

func newTagEditorModel(beans []*bean.Bean, beanTitle string, allTags []tagWithCount, width, height int) tagEditorModel {
	ti := textinput.New()
	ti.Placeholder = "Filter or create tag..."
	ti.CharLimit = 50
	ti.Width = 40
	ti.Focus()
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(ui.ColorMuted)
	ti.Prompt = "# "

	usage := make(map[string]int)
	for _, t := range allTags {
		usage[t.tag] = t.count
	}

	// Count how many of the edited beans carry each tag
	onBeans := make(map[string]int)
	beanIDs := make([]string, 0, len(beans))
	for _, b := range beans {
		beanIDs = append(beanIDs, b.ID)
		for _, tag := range b.Tags {
			onBeans[tag]++
		}
	}

	original := make(map[string]tagState)
	pending := make(map[string]tagState)
	tagSet := make(map[string]bool)
	for tag := range usage {
		tagSet[tag] = true
	}
	for tag, n := range onBeans {
		tagSet[tag] = true
		state := tagOnSome
		if n == len(beans) {
			state = tagOnAll
		}
		original[tag] = state
		pending[tag] = state
	}

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	slices.Sort(tags)

	return tagEditorModel{
		input:     ti,
		beanIDs:   beanIDs,
		beanTitle: beanTitle,
		tags:      tags,
		usage:     usage,
		original:  original,
		pending:   pending,
		width:     width,
		height:    height,
	}
}

func (m tagEditorModel) Init() tea.Cmd {
	return textinput.Blink
}

// query returns the normalized input text
func (m tagEditorModel) query() string {
	return bean.NormalizeTag(m.input.Value())
}

// visibleTagNames returns the existing tags matching the current input
func (m tagEditorModel) visibleTagNames() []string {
	q := m.query()
	if q == "" {
		return m.tags
	}
	var result []string
	for _, tag := range m.tags {
		if strings.Contains(tag, q) {
			result = append(result, tag)
		}
	}
	return result
}

// hasCreateItem returns true if the input is a valid tag that doesn't exist yet
func (m tagEditorModel) hasCreateItem() bool {
	q := m.query()
	return q != "" && bean.ValidateTag(q) == nil && !slices.Contains(m.tags, q)
}

// validationError returns a message if the input can't be used as a new tag
func (m tagEditorModel) validationError() string {
	q := m.query()
	if q == "" || slices.Contains(m.tags, q) || bean.ValidateTag(q) == nil {
		return ""
	}
	return "Tags must be lowercase, start with a letter, and contain only letters, numbers, and hyphens"
}

// itemCount returns the number of selectable rows (matching tags + optional create item)
func (m tagEditorModel) itemCount() int {
	n := len(m.visibleTagNames())
	if m.hasCreateItem() {
		n++
	}
	return n
}

// toggle flips the tag under the cursor, or creates the new tag if the create item is selected
func (m *tagEditorModel) toggle() {
	visible := m.visibleTagNames()
	if m.cursor < len(visible) {
		tag := visible[m.cursor]
		if m.pending[tag] == tagOnAll {
			m.pending[tag] = tagOnNone
		} else {
			m.pending[tag] = tagOnAll
		}
		return
	}
	if m.hasCreateItem() {
		tag := m.query()
		m.tags = append(m.tags, tag)
		slices.Sort(m.tags)
		m.pending[tag] = tagOnAll
		m.input.SetValue("")
		m.cursor = slices.Index(m.tags, tag)
	}
}

// changes computes which tags to add to and remove from all edited beans
func (m tagEditorModel) changes() (toAdd, toRemove []string) {
	for _, tag := range m.tags {
		if m.pending[tag] == m.original[tag] {
			continue
		}
		switch m.pending[tag] {
		case tagOnAll:
			toAdd = append(toAdd, tag)
		case tagOnNone:
			toRemove = append(toRemove, tag)
		}
	}
	return toAdd, toRemove
}

func (m tagEditorModel) Update(msg tea.Msg) (tagEditorModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		k := keymap.TagEditor
		switch {
		case key.Matches(msg, k.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case key.Matches(msg, k.Down):
			if m.cursor < m.itemCount()-1 {
				m.cursor++
			}
			return m, nil
		case key.Matches(msg, k.Toggle):
			m.toggle()
			return m, nil
		case key.Matches(msg, k.Apply):
			toAdd, toRemove := m.changes()
			beanIDs := m.beanIDs
			return m, func() tea.Msg {
				return tagsConfirmedMsg{beanIDs: beanIDs, toAdd: toAdd, toRemove: toRemove}
			}
		case key.Matches(msg, k.Cancel):
			return m, func() tea.Msg {
				return closeTagEditorMsg{}
			}
		}
	}

	// Everything else goes to the input, which filters the list
	before := m.input.Value()
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != before {
		m.cursor = 0
	}
	return m, cmd
}

// modalSize returns the modal and list dimensions for the current screen size
func (m tagEditorModel) modalSize() (modalWidth, listHeight int) {
	modalWidth = max(40, min(80, m.width*60/100))
	modalHeight := max(12, min(24, m.height*60/100))
	// header(1) + subtitle(1) + blank(1) + title(1) + blank(1) + input(1) + blank(1)
	// + blank(1) + description(1) + blank(1) + help(1) + border(2)
	listHeight = max(3, modalHeight-13)
	return modalWidth, listHeight
}

func (m tagEditorModel) renderList(height int) string {
	visible := m.visibleTagNames()
	count := m.itemCount()
	if count == 0 {
		if len(m.tags) == 0 && m.query() == "" {
			return ui.Muted.Render("No tags yet. Type to create one.")
		}
		return ui.Muted.Render("No matching tags")
	}

	// Scroll window that keeps the cursor visible
	start := 0
	if m.cursor >= height {
		start = m.cursor - height + 1
	}
	end := min(count, start+height)

	cursorStyle := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true)
	var lines []string
	for i := start; i < end; i++ {
		cursor := "  "
		if i == m.cursor {
			cursor = cursorStyle.Render("▌") + " "
		}

		if i >= len(visible) {
			lines = append(lines, cursor+cursorStyle.Render("+ ")+"create "+ui.RenderTag(m.query()))
			continue
		}

		tag := visible[i]
		var indicator string
		switch m.pending[tag] {
		case tagOnAll:
			indicator = lipgloss.NewStyle().Foreground(ui.ColorSuccess).Bold(true).Render("● ")
		case tagOnSome:
			indicator = lipgloss.NewStyle().Foreground(ui.ColorWarning).Render("◐ ")
		default:
			indicator = ui.Muted.Render("○ ")
		}
		line := cursor + indicator + ui.RenderTag(tag)
		if n := m.usage[tag]; n > 0 {
			line += ui.Muted.Render(fmt.Sprintf(" (%d)", n))
		}
		lines = append(lines, line)
	}

	// Pad so the modal doesn't jump in size while filtering
	for len(lines) < min(height, len(m.tags)+1) {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m tagEditorModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	modalWidth, listHeight := m.modalSize()

	var beanID string
	if len(m.beanIDs) == 1 {
		beanID = m.beanIDs[0]
	}

	m.input.Width = modalWidth - 8 // border (2) + padding (2) + prompt (2) + cursor margin (2)
	content := listTitleStyle.Render("Edit Tags") + "\n\n" + m.input.View() + "\n\n" + m.renderList(listHeight)

	description := ""
	if errMsg := m.validationError(); errMsg != "" {
		description = lipgloss.NewStyle().Foreground(ui.ColorDanger).Width(modalWidth - 4).Render(errMsg)
	}

	k := keymap.TagEditor
	help := hints(hint(k.Toggle, ""), hint(k.Apply, ""), hint(k.Cancel, ""))

	return renderPickerModal(pickerModalConfig{
		Title:       "Edit Tags",
		BeanTitle:   m.beanTitle,
		BeanID:      beanID,
		ListContent: content,
		Description: description,
		Help:        help,
		Width:       m.width,
		WidthPct:    60,
		MaxWidth:    80,
	})
}

// ModalView returns the editor rendered as a centered modal overlay on top of the background
func (m tagEditorModel) ModalView(bgView string, fullWidth, fullHeight int) string {
	return overlayModal(bgView, m.View(), fullWidth, fullHeight)
}

// applyTagChanges returns current with toRemove removed and toAdd appended,
// preserving the order of existing tags.
func applyTagChanges(current, toAdd, toRemove []string) []string {
	result := make([]string, 0, len(current)+len(toAdd))
	for _, tag := range current {
		if !slices.Contains(toRemove, tag) {
			result = append(result, tag)
		}
	}
	for _, tag := range toAdd {
		if !slices.Contains(result, tag) {
			result = append(result, tag)
		}
	}
	return result
}
