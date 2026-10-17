package tui

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/hmans/beans/internal/ui"
)

// beanItem wraps a Bean to implement list.Item, with tree context
type beanItem struct {
	bean        *bean.Bean
	cfg         *config.Config
	treePrefix  string // tree prefix for rendering (e.g., "├─" or "  └─")
	matched     bool   // true if bean matched filter (vs. ancestor shown for context)
	hasChildren bool   // true if this bean has children in the tree
}

func (i beanItem) Title() string       { return i.bean.DisplayTitle() }
func (i beanItem) Description() string { return i.bean.ID + " · " + i.bean.Status }
func (i beanItem) FilterValue() string { return i.bean.DisplayTitle() + " " + i.bean.ID }

// itemDelegate handles rendering of list items
type itemDelegate struct {
	cfg            *config.Config
	hasTags        bool
	width          int
	cols           ui.ResponsiveColumns // cached responsive columns
	idColWidth     int                  // ID column width (accounts for tree prefix)
	selectedBeans  *map[string]bool     // pointer to marked beans for multi-select
	collapsedBeans *map[string]bool     // pointer to collapsed beans
}

func newItemDelegate(cfg *config.Config) itemDelegate {
	return itemDelegate{cfg: cfg, hasTags: false, width: 0}
}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(beanItem)
	if !ok {
		return
	}

	// Get colors from config
	colors := d.cfg.GetBeanColors(item.bean.Status, item.bean.Type, item.bean.Priority)

	// Calculate max title width using responsive columns
	idWidth := d.cols.ID
	if d.idColWidth > 0 {
		idWidth = d.idColWidth
	}
	baseWidth := idWidth + d.cols.Status + d.cols.Type + 4 // 4 for cursor + padding
	if d.cols.ShowTags {
		baseWidth += d.cols.Tags
	}
	maxTitleWidth := max(1, m.Width()-baseWidth) // 0 would disable truncation

	// Check if bean is marked for multi-select
	var isMarked bool
	if d.selectedBeans != nil {
		isMarked = (*d.selectedBeans)[item.bean.ID]
	}

	// Check if bean is collapsed
	var isCollapsed bool
	if d.collapsedBeans != nil {
		isCollapsed = (*d.collapsedBeans)[item.bean.ID]
	}

	str := ui.RenderBeanRow(
		item.bean.ID,
		item.bean.Status,
		item.bean.Type,
		item.bean.DisplayTitle(),
		ui.BeanRowConfig{
			TitleMuted:    !item.bean.HasTitle(),
			StatusColor:   colors.StatusColor,
			TypeColor:     colors.TypeColor,
			PriorityColor: colors.PriorityColor,
			Priority:      item.bean.Priority,
			IsArchive:     colors.IsArchive,
			MaxTitleWidth: maxTitleWidth,
			ShowCursor:    true,
			IsSelected:    index == m.Index(),
			IsMarked:      isMarked,
			Tags:          item.bean.Tags,
			ShowTags:      d.cols.ShowTags,
			TagsColWidth:  d.cols.Tags,
			MaxTags:       d.cols.MaxTags,
			TreePrefix:    item.treePrefix,
			Dimmed:        !item.matched,
			IDColWidth:    d.idColWidth,
			HasChildren:   item.hasChildren,
			IsCollapsed:   isCollapsed,
			Compact:       d.cols.Compact,
		},
	)

	fmt.Fprint(w, str)
}

// listModel is the model for the bean list view
type listModel struct {
	list     list.Model
	resolver *graph.Resolver
	config   *config.Config
	width    int
	height   int
	err      error

	// Responsive column state
	hasTags    bool                 // whether any beans have tags
	cols       ui.ResponsiveColumns // calculated responsive columns
	idColWidth int                  // ID column width (accounts for tree depth)

	// Active filters
	tagFilter string // if set, only show beans with this tag

	// Multi-select state
	selectedBeans map[string]bool // IDs of beans marked for multi-edit

	// Collapse/fold state
	collapsedBeans map[string]bool // IDs of beans whose children are hidden

	// Hierarchical navigation: IDs of beans drilled into, outermost first.
	// When non-empty, only descendants of the last entry are listed.
	rootStack []string
	rootBeans []*bean.Bean // resolved beans for rootStack (for the breadcrumb)

	// cursorTarget is a bean ID to move the cursor to after the next load
	cursorTarget string

	// notice is a transient message shown at the start of the footer
	notice      string
	noticeIsErr bool

	// Relationship info across all beans (for the preview pane)
	beansByID   map[string]*bean.Bean
	childCounts map[string]int
}

func newListModel(resolver *graph.Resolver, cfg *config.Config) listModel {
	selectedBeans := make(map[string]bool)
	collapsedBeans := make(map[string]bool)
	delegate := itemDelegate{
		cfg:            cfg,
		selectedBeans:  &selectedBeans,
		collapsedBeans: &collapsedBeans,
	}

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Beans"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = listTitleStyle
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 1)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ui.ColorPrimary)

	return listModel{
		list:           l,
		resolver:       resolver,
		config:         cfg,
		selectedBeans:  selectedBeans,
		collapsedBeans: collapsedBeans,
	}
}

// beansLoadedMsg is sent when beans are loaded
type beansLoadedMsg struct {
	items       []ui.FlatItem         // flattened tree items
	idColWidth  int                   // calculated ID column width for tree
	rootBeans   []*bean.Bean          // resolved drill-down roots (may be shorter than rootStack if some vanished)
	beansByID   map[string]*bean.Bean // all beans by ID
	childCounts map[string]int        // number of children per bean ID
}

// errMsg is sent when an error occurs
type errMsg struct {
	err error
}

// selectBeanMsg is sent when a bean is selected
type selectBeanMsg struct {
	bean *bean.Bean
}

func (m listModel) Init() tea.Cmd {
	return m.loadBeans
}

func (m listModel) loadBeans() tea.Msg {
	// Build filter if tag filter is set
	var filter *model.BeanFilter
	if m.tagFilter != "" {
		filter = &model.BeanFilter{Tags: []string{m.tagFilter}}
	}

	// Query filtered beans
	filteredBeans, err := m.resolver.Query().Beans(context.Background(), filter)
	if err != nil {
		return errMsg{err}
	}

	// Query all beans for tree context (ancestors)
	allBeans, err := m.resolver.Query().Beans(context.Background(), nil)
	if err != nil {
		return errMsg{err}
	}

	beansByID := make(map[string]*bean.Bean, len(allBeans))
	childCounts := make(map[string]int)
	for _, b := range allBeans {
		beansByID[b.ID] = b
		if b.Parent != "" {
			childCounts[b.Parent]++
		}
	}

	// Resolve drill-down roots; stop at the first one that no longer exists
	var rootBeans []*bean.Bean
	for _, id := range m.rootStack {
		b, ok := beansByID[id]
		if !ok {
			break
		}
		rootBeans = append(rootBeans, b)
	}

	// Restrict the list to descendants of the current root
	if len(rootBeans) > 0 {
		root := rootBeans[len(rootBeans)-1].ID
		filteredBeans = descendantsOf(root, filteredBeans, beansByID)
		allBeans = descendantsOf(root, allBeans, beansByID)
	}

	// Sort function for tree building
	sortFn := func(beans []*bean.Bean) {
		bean.SortByStatusPriorityAndType(beans, m.config.StatusNames(), m.config.PriorityNames(), m.config.TypeNames())
	}

	// Build tree and flatten it with collapse state
	tree := ui.BuildTree(filteredBeans, allBeans, sortFn)
	items := ui.FlattenTreeWithCollapsed(tree, m.collapsedBeans)

	// Calculate ID column width based on max ID length and tree depth
	maxIDLen := 0
	for _, b := range allBeans {
		if len(b.ID) > maxIDLen {
			maxIDLen = len(b.ID)
		}
	}
	maxDepth := ui.MaxTreeDepth(items)
	// ID column = base ID width + caret (2 chars) + tree indent (3 chars per depth level)
	idColWidth := maxIDLen + 4 // base padding + caret
	if maxDepth > 0 {
		idColWidth += maxDepth * 3 // 3 chars per depth level (├─ + space)
	}

	return beansLoadedMsg{
		items:       items,
		idColWidth:  idColWidth,
		rootBeans:   rootBeans,
		beansByID:   beansByID,
		childCounts: childCounts,
	}
}

// descendantsOf returns the beans in candidates that are (transitive) children of rootID.
func descendantsOf(rootID string, candidates []*bean.Bean, beansByID map[string]*bean.Bean) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range candidates {
		// Walk up the parent chain; seen guards against cycles
		seen := map[string]bool{b.ID: true}
		for p := b.Parent; p != "" && !seen[p]; {
			if p == rootID {
				result = append(result, b)
				break
			}
			seen[p] = true
			parent, ok := beansByID[p]
			if !ok {
				break
			}
			p = parent.Parent
		}
	}
	return result
}

// drillInto makes the given bean the new root, showing only its descendants
func (m *listModel) drillInto(id string) {
	m.rootStack = append(m.rootStack, id)
	m.cursorTarget = ""
	m.list.ResetFilter()
	m.list.Select(0)
}

// drillOut navigates one level up the hierarchy. Returns false if already at the top.
func (m *listModel) drillOut() bool {
	if len(m.rootStack) == 0 {
		return false
	}
	// Put the cursor back on the bean we came out of
	m.cursorTarget = m.rootStack[len(m.rootStack)-1]
	m.rootStack = m.rootStack[:len(m.rootStack)-1]
	m.list.ResetFilter()
	return true
}

// previewData returns the highlighted bean and its relationship context
func (m *listModel) previewData() (*bean.Bean, previewContext) {
	item, ok := m.list.SelectedItem().(beanItem)
	if !ok {
		return nil, previewContext{}
	}
	return item.bean, previewContext{
		parent:     m.beansByID[item.bean.Parent],
		childCount: m.childCounts[item.bean.ID],
	}
}

// setTagFilter sets the tag filter
func (m *listModel) setTagFilter(tag string) {
	m.tagFilter = tag
}

// clearFilter clears all active filters
func (m *listModel) clearFilter() {
	m.tagFilter = ""
}

// hasActiveFilter returns true if any filter is active
func (m *listModel) hasActiveFilter() bool {
	return m.tagFilter != ""
}

func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve space for border and footer
		m.list.SetSize(msg.Width-2, msg.Height-4)
		// Recalculate responsive columns
		m.cols = ui.CalculateResponsiveColumns(msg.Width, m.hasTags)
		m.updateDelegate()

	case beansLoadedMsg:
		items := make([]list.Item, len(msg.items))
		// Check if any beans have tags
		m.hasTags = false
		for i, flatItem := range msg.items {
			items[i] = beanItem{
				bean:        flatItem.Bean,
				cfg:         m.config,
				treePrefix:  flatItem.TreePrefix,
				matched:     flatItem.Matched,
				hasChildren: flatItem.HasChildren,
			}
			if len(flatItem.Bean.Tags) > 0 {
				m.hasTags = true
			}
		}
		m.list.SetItems(items)
		m.idColWidth = msg.idColWidth
		m.beansByID = msg.beansByID
		m.childCounts = msg.childCounts
		// Drop roots that no longer exist (e.g. deleted or archived)
		m.rootBeans = msg.rootBeans
		m.rootStack = m.rootStack[:len(msg.rootBeans)]
		if m.cursorTarget != "" {
			for i, it := range items {
				if bi, ok := it.(beanItem); ok && bi.bean.ID == m.cursorTarget {
					m.list.Select(i)
					break
				}
			}
			m.cursorTarget = ""
		}
		// Calculate responsive columns based on hasTags and width
		m.cols = ui.CalculateResponsiveColumns(m.width, m.hasTags)
		m.updateDelegate()
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "h":
				// Collapse current item if it has children
				if item, ok := m.list.SelectedItem().(beanItem); ok && item.hasChildren {
					m.collapsedBeans[item.bean.ID] = true
					return m, m.loadBeans
				}
				return m, nil
			case "l":
				// Expand current item if it has children
				if item, ok := m.list.SelectedItem().(beanItem); ok && item.hasChildren {
					delete(m.collapsedBeans, item.bean.ID)
					return m, m.loadBeans
				}
				return m, nil
			case "z":
				// Toggle collapse/expand for current item
				if item, ok := m.list.SelectedItem().(beanItem); ok && item.hasChildren {
					if m.collapsedBeans[item.bean.ID] {
						delete(m.collapsedBeans, item.bean.ID)
					} else {
						m.collapsedBeans[item.bean.ID] = true
					}
					return m, m.loadBeans
				}
				return m, nil
			case "Z":
				// Collapse all items with children
				for _, item := range m.list.Items() {
					if bi, ok := item.(beanItem); ok && bi.hasChildren {
						m.collapsedBeans[bi.bean.ID] = true
					}
				}
				return m, m.loadBeans
			case " ":
				// Toggle selection for multi-select
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					if m.selectedBeans[item.bean.ID] {
						delete(m.selectedBeans, item.bean.ID)
					} else {
						m.selectedBeans[item.bean.ID] = true
					}
				}
				return m, nil
			case "enter":
				// Drill into beans with children, open leaf beans
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					if item.hasChildren {
						m.drillInto(item.bean.ID)
						return m, m.loadBeans
					}
					return m, func() tea.Msg {
						return selectBeanMsg{bean: item.bean}
					}
				}
			case "v":
				// Open full detail view
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return selectBeanMsg{bean: item.bean}
					}
				}
			case "p":
				// Open parent picker for selected bean(s)
				if len(m.selectedBeans) > 0 {
					// Multi-select mode
					ids := make([]string, 0, len(m.selectedBeans))
					types := make([]string, 0, len(m.selectedBeans))
					for id := range m.selectedBeans {
						ids = append(ids, id)
						// Find the bean to get its type
						for _, item := range m.list.Items() {
							if bi, ok := item.(beanItem); ok && bi.bean.ID == id {
								types = append(types, bi.bean.Type)
								break
							}
						}
					}
					return m, func() tea.Msg {
						return openParentPickerMsg{
							beanIDs:   ids,
							beanTitle: fmt.Sprintf("%d selected beans", len(ids)),
							beanTypes: types,
						}
					}
				} else if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openParentPickerMsg{
							beanIDs:       []string{item.bean.ID},
							beanTitle:     item.bean.DisplayTitle(),
							beanTypes:     []string{item.bean.Type},
							currentParent: item.bean.Parent,
						}
					}
				}
			case "s":
				// Open status picker for selected bean(s)
				if len(m.selectedBeans) > 0 {
					// Multi-select mode
					ids := make([]string, 0, len(m.selectedBeans))
					for id := range m.selectedBeans {
						ids = append(ids, id)
					}
					return m, func() tea.Msg {
						return openStatusPickerMsg{
							beanIDs:   ids,
							beanTitle: fmt.Sprintf("%d selected beans", len(ids)),
						}
					}
				} else if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openStatusPickerMsg{
							beanIDs:       []string{item.bean.ID},
							beanTitle:     item.bean.DisplayTitle(),
							currentStatus: item.bean.Status,
						}
					}
				}
			case "t":
				// Open type picker for selected bean(s)
				if len(m.selectedBeans) > 0 {
					// Multi-select mode
					ids := make([]string, 0, len(m.selectedBeans))
					for id := range m.selectedBeans {
						ids = append(ids, id)
					}
					return m, func() tea.Msg {
						return openTypePickerMsg{
							beanIDs:   ids,
							beanTitle: fmt.Sprintf("%d selected beans", len(ids)),
						}
					}
				} else if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openTypePickerMsg{
							beanIDs:     []string{item.bean.ID},
							beanTitle:   item.bean.DisplayTitle(),
							currentType: item.bean.Type,
						}
					}
				}
			case "T":
				// Open tag editor for selected bean(s)
				if len(m.selectedBeans) > 0 {
					// Multi-select mode
					ids := make([]string, 0, len(m.selectedBeans))
					for id := range m.selectedBeans {
						ids = append(ids, id)
					}
					return m, func() tea.Msg {
						return openTagEditorMsg{
							beanIDs:   ids,
							beanTitle: fmt.Sprintf("%d selected beans", len(ids)),
						}
					}
				} else if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openTagEditorMsg{
							beanIDs:   []string{item.bean.ID},
							beanTitle: item.bean.DisplayTitle(),
						}
					}
				}
			case "P":
				// Open priority picker for selected bean(s)
				if len(m.selectedBeans) > 0 {
					// Multi-select mode
					ids := make([]string, 0, len(m.selectedBeans))
					for id := range m.selectedBeans {
						ids = append(ids, id)
					}
					return m, func() tea.Msg {
						return openPriorityPickerMsg{
							beanIDs:   ids,
							beanTitle: fmt.Sprintf("%d selected beans", len(ids)),
						}
					}
				} else if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openPriorityPickerMsg{
							beanIDs:         []string{item.bean.ID},
							beanTitle:       item.bean.DisplayTitle(),
							currentPriority: item.bean.Priority,
						}
					}
				}
			case ">", "<":
				// Raise/lower priority one step for selected bean(s)
				delta := 1
				if msg.String() == "<" {
					delta = -1
				}
				return m, m.shiftPriority(delta)
			case "b":
				// Open blocking picker for selected bean
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openBlockingPickerMsg{
							beanID:     item.bean.ID,
							beanTitle:  item.bean.DisplayTitle(),
							currentIDs: item.bean.Blocking,
						}
					}
				}
			case "B":
				// Open blocked-by picker for selected bean
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openBlockingPickerMsg{
							beanID:     item.bean.ID,
							beanTitle:  item.bean.DisplayTitle(),
							blockedBy:  true,
							currentIDs: item.bean.BlockedBy,
						}
					}
				}
			case "c":
				// Open create modal
				return m, func() tea.Msg {
					return openCreateModalMsg{}
				}
			case "C":
				// Open create modal with selected bean(s) as parent (create child)
				var parentID, parentTitle string
				if len(m.selectedBeans) > 0 {
					// Multi-select: use first selected bean as parent
					// (Note: a bean can only have one parent)
					for id := range m.selectedBeans {
						parentID = id
						// Find the bean to get its title
						for _, item := range m.list.Items() {
							if bi, ok := item.(beanItem); ok && bi.bean.ID == id {
								parentTitle = bi.bean.DisplayTitle()
								break
							}
						}
						break
					}
					if len(m.selectedBeans) > 1 {
						parentTitle = fmt.Sprintf("%s (+%d more)", parentTitle, len(m.selectedBeans)-1)
					}
				} else if item, ok := m.list.SelectedItem().(beanItem); ok {
					// No multi-select: use cursor position
					parentID = item.bean.ID
					parentTitle = item.bean.DisplayTitle()
				}

				if parentID != "" {
					return m, func() tea.Msg {
						return openCreateModalMsg{
							parentID:    parentID,
							parentTitle: parentTitle,
						}
					}
				}
			case "e":
				// Open the edit form for the bean under the cursor
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openEditModalMsg{bean: item.bean}
					}
				}
			case "E":
				// Open $EDITOR for the bean under the cursor
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openEditorMsg{
							beanID:   item.bean.ID,
							beanPath: item.bean.Path,
						}
					}
				}
			case "!":
				// Open launcher picker for selected bean(s)
				if len(m.selectedBeans) > 0 {
					// Multi-select: collect all IDs
					ids := make([]string, 0, len(m.selectedBeans))
					for id := range m.selectedBeans {
						ids = append(ids, id)
					}
					return m, func() tea.Msg {
						return openLauncherPickerMsg{
							beanIDs:   ids,
							beanTitle: fmt.Sprintf("%d selected beans", len(ids)),
						}
					}
				} else if item, ok := m.list.SelectedItem().(beanItem); ok {
					// Single bean
					return m, func() tea.Msg {
						return openLauncherPickerMsg{
							beanIDs:   []string{item.bean.ID},
							beanTitle: item.bean.DisplayTitle(),
						}
					}
				} else {
					// No bean selected - use first bean in list if available
					items := m.list.Items()
					if len(items) > 0 {
						if item, ok := items[0].(beanItem); ok {
							return m, func() tea.Msg {
								return openLauncherPickerMsg{
									beanIDs:   []string{item.bean.ID},
									beanTitle: item.bean.DisplayTitle(),
								}
							}
						}
					}
				}
			case "Y":
				// Archive all completed beans (asks for confirmation)
				return m, func() tea.Msg {
					return archiveCompletedMsg{}
				}
			case "esc", "backspace":
				// First clear selection if any beans are selected
				if len(m.selectedBeans) > 0 {
					clear(m.selectedBeans)
					return m, nil
				}
				// Let the list clear an applied text filter
				if m.list.FilterState() == list.FilterApplied {
					break
				}
				// Then navigate up the hierarchy
				if m.drillOut() {
					return m, m.loadBeans
				}
				// Then clear active filter if any
				if m.hasActiveFilter() {
					return m, func() tea.Msg {
						return clearFilterMsg{}
					}
				}
			}
		}
	}

	// Always forward to the list component
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// updateDelegate updates the list delegate with current responsive columns
func (m *listModel) updateDelegate() {
	delegate := itemDelegate{
		cfg:            m.config,
		hasTags:        m.hasTags,
		width:          m.width,
		cols:           m.cols,
		idColWidth:     m.idColWidth,
		selectedBeans:  &m.selectedBeans,
		collapsedBeans: &m.collapsedBeans,
	}
	m.list.SetDelegate(delegate)
}

func (m listModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}

	if m.width == 0 {
		return "Loading..."
	}

	return m.viewContent() + "\n" + m.Footer()
}

// viewContent renders the bordered list without the footer
// (m.height-2 lines tall, m.width wide).
func (m listModel) viewContent() string {
	m.list.Title = m.breadcrumbTitle(m.width - 6)

	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorMuted).
		Width(m.width - 2).
		Height(m.height - 4)

	return border.Render(m.list.View())
}

// breadcrumbTitle renders the list title, e.g. "Beans › Parent › Child [tag: x]",
// eliding outer levels to fit maxWidth.
func (m listModel) breadcrumbTitle(maxWidth int) string {
	const sep = " › "
	suffix := ""
	if m.tagFilter != "" {
		suffix = fmt.Sprintf(" [tag: %s]", m.tagFilter)
	}

	crumbs := make([]string, 0, len(m.rootBeans)+1)
	crumbs = append(crumbs, "Beans")
	for _, b := range m.rootBeans {
		crumbs = append(crumbs, b.DisplayTitle())
	}

	title := strings.Join(crumbs, sep) + suffix
	// Replace outer crumbs (after "Beans") with "…" until it fits
	for elided := 1; lipgloss.Width(title) > maxWidth && elided < len(crumbs)-1; elided++ {
		title = strings.Join(append([]string{"Beans", "…"}, crumbs[elided+1:]...), sep) + suffix
	}
	if maxWidth > 1 && lipgloss.Width(title) > maxWidth {
		title = string([]rune(title)[:maxWidth-1]) + "…"
	}
	return title
}

// Footer renders the help/status line below the list
func (m listModel) Footer() string {
	var help string

	// esc goes up a level when drilled in
	var backHelp string
	if len(m.rootStack) > 0 {
		backHelp = helpKeyStyle.Render("esc") + " " + helpStyle.Render("up") + "  "
	}

	// Show selection count if any beans are selected
	var selectionPrefix string
	if len(m.selectedBeans) > 0 {
		selectionStyle := lipgloss.NewStyle().Foreground(ui.ColorWarning).Bold(true)
		selectionPrefix = selectionStyle.Render(fmt.Sprintf("(%d selected) ", len(m.selectedBeans)))
	}

	if len(m.selectedBeans) > 0 {
		// When beans are selected, show esc to clear selection
		help = helpKeyStyle.Render("space") + " " + helpStyle.Render("toggle") + "  " +
			helpKeyStyle.Render("s") + " " + helpStyle.Render("status") + "  " +
			helpKeyStyle.Render("t") + " " + helpStyle.Render("type") + "  " +
			helpKeyStyle.Render("P") + " " + helpStyle.Render("priority") + "  " +
			helpKeyStyle.Render("!") + " " + helpStyle.Render("launch") + "  " +
			helpKeyStyle.Render("T") + " " + helpStyle.Render("tags") + "  " +
			helpKeyStyle.Render("esc") + " " + helpStyle.Render("clear selection") + "  " +
			helpKeyStyle.Render("?") + " " + helpStyle.Render("help") + "  " +
			helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")
	} else if m.hasActiveFilter() {
		help = backHelp +
			helpKeyStyle.Render("space") + " " + helpStyle.Render("select") + "  " +
			helpKeyStyle.Render("enter") + " " + helpStyle.Render("open") + "  " +
			helpKeyStyle.Render("v") + " " + helpStyle.Render("view") + "  " +
			helpKeyStyle.Render("c") + " " + helpStyle.Render("create") + "  " +
			helpKeyStyle.Render("e") + " " + helpStyle.Render("edit") + "  " +
			helpKeyStyle.Render("E") + " " + helpStyle.Render("$EDITOR") + "  " +
			helpKeyStyle.Render("s") + " " + helpStyle.Render("status") + "  " +
			helpKeyStyle.Render("t") + " " + helpStyle.Render("type") + "  " +
			helpKeyStyle.Render("P") + " " + helpStyle.Render("priority") + "  " +
			helpKeyStyle.Render("</>") + " " + helpStyle.Render("prio +/-") + "  " +
			helpKeyStyle.Render("T") + " " + helpStyle.Render("tags") + "  " +
			helpKeyStyle.Render("p") + " " + helpStyle.Render("parent") + "  " +
			helpKeyStyle.Render("b") + " " + helpStyle.Render("blocking") + "  " +
			helpKeyStyle.Render("B") + " " + helpStyle.Render("blocked by") + "  " +
			helpKeyStyle.Render("!") + " " + helpStyle.Render("launch") + "  " +
			helpKeyStyle.Render("esc") + " " + helpStyle.Render("clear filter") + "  " +
			helpKeyStyle.Render("?") + " " + helpStyle.Render("help") + "  " +
			helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")
	} else {
		help = backHelp +
			helpKeyStyle.Render("space") + " " + helpStyle.Render("select") + "  " +
			helpKeyStyle.Render("enter") + " " + helpStyle.Render("open") + "  " +
			helpKeyStyle.Render("v") + " " + helpStyle.Render("view") + "  " +
			helpKeyStyle.Render("c") + " " + helpStyle.Render("create") + "  " +
			helpKeyStyle.Render("e") + " " + helpStyle.Render("edit") + "  " +
			helpKeyStyle.Render("E") + " " + helpStyle.Render("$EDITOR") + "  " +
			helpKeyStyle.Render("s") + " " + helpStyle.Render("status") + "  " +
			helpKeyStyle.Render("t") + " " + helpStyle.Render("type") + "  " +
			helpKeyStyle.Render("P") + " " + helpStyle.Render("priority") + "  " +
			helpKeyStyle.Render("</>") + " " + helpStyle.Render("prio +/-") + "  " +
			helpKeyStyle.Render("T") + " " + helpStyle.Render("tags") + "  " +
			helpKeyStyle.Render("p") + " " + helpStyle.Render("parent") + "  " +
			helpKeyStyle.Render("b") + " " + helpStyle.Render("blocking") + "  " +
			helpKeyStyle.Render("B") + " " + helpStyle.Render("blocked by") + "  " +
			helpKeyStyle.Render("!") + " " + helpStyle.Render("launch") + "  " +
			helpKeyStyle.Render("Y") + " " + helpStyle.Render("archive") + "  " +
			helpKeyStyle.Render("h/l/z") + " " + helpStyle.Render("fold") + "  " +
			helpKeyStyle.Render("/") + " " + helpStyle.Render("filter") + "  " +
			helpKeyStyle.Render("?") + " " + helpStyle.Render("help") + "  " +
			helpKeyStyle.Render("q") + " " + helpStyle.Render("quit")
	}

	var noticePrefix string
	if m.notice != "" {
		color := ui.ColorSuccess
		if m.noticeIsErr {
			color = ui.ColorDanger
		}
		noticePrefix = lipgloss.NewStyle().Foreground(color).Bold(true).Render(m.notice) + "  "
	}

	return noticePrefix + selectionPrefix + help
}

// shiftPriority moves the priority of the marked beans (or the bean under the
// cursor) by delta steps in the priority order (negative = higher priority).
// The cursor follows the highlighted bean, since re-sorting will likely move it.
func (m *listModel) shiftPriority(delta int) tea.Cmd {
	var beans []*bean.Bean
	if len(m.selectedBeans) > 0 {
		for id := range m.selectedBeans {
			if item, ok := m.findBeanItem(id); ok {
				beans = append(beans, item.bean)
			}
		}
	} else if item, ok := m.list.SelectedItem().(beanItem); ok {
		beans = append(beans, item.bean)
	}
	if len(beans) == 0 {
		return nil
	}

	names := m.config.PriorityNames()
	for _, b := range beans {
		next := shiftedPriority(b.Priority, delta, names)
		if next == b.Priority {
			continue
		}
		if _, err := m.resolver.Mutation().UpdateBean(context.Background(), b.ID, model.UpdateBeanInput{
			Priority: &next,
		}); err != nil {
			// Continue with other beans even if one fails
			continue
		}
	}

	if item, ok := m.list.SelectedItem().(beanItem); ok {
		m.cursorTarget = item.bean.ID
	}
	return m.loadBeans
}

// shiftedPriority returns the priority delta steps away from current in names
// (ordered highest to lowest), clamped to the ends. Empty means normal.
func shiftedPriority(current string, delta int, names []string) string {
	if current == "" {
		current = "normal"
	}
	idx := -1
	for i, n := range names {
		if n == current {
			idx = i
			break
		}
	}
	if idx < 0 {
		return current
	}
	return names[max(0, min(len(names)-1, idx+delta))]
}

func (m listModel) findBeanItem(id string) (beanItem, bool) {
	for _, item := range m.list.Items() {
		if bi, ok := item.(beanItem); ok && bi.bean.ID == id {
			return bi, true
		}
	}
	return beanItem{}, false
}
