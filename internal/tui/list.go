package tui

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
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
	idColWidth     int                  // ID column width: the longest ID
	selectedBeans  map[string]bool      // marked beans for multi-select (shared with listModel)
	collapsedBeans map[string]bool      // collapsed beans (shared with listModel)
	titleScroll    *titleScroll         // the selected row's title scrolling (shared with listModel)
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
	selected := index == m.Index()
	cfg := d.rowConfig(item, m.Width())
	cfg.IsSelected = selected
	if selected {
		cfg.TitleShift = d.titleScroll.shift
	}
	fmt.Fprint(w, ui.RenderBeanRow(item.bean.ID, item.bean.Status, item.bean.Type, item.bean.DisplayTitle(), cfg))
}

// rowConfig is how item's row is rendered in a list of the given width.
func (d itemDelegate) rowConfig(item beanItem, width int) ui.BeanRowConfig {
	colors := d.cfg.GetBeanColors(item.bean.Status, item.bean.Type, item.bean.Priority)

	// The title takes the width the other columns leave
	idWidth := d.cols.ID
	if d.idColWidth > 0 {
		idWidth = d.idColWidth
	}
	baseWidth := d.cols.Type + d.cols.Status + ui.ColWidthSize + idWidth + 3 // cursor, space after the title, margin
	if d.cols.ShowTags {
		baseWidth += d.cols.Tags
	}

	return ui.BeanRowConfig{
		TitleMuted:    !item.bean.HasTitle(),
		StatusColor:   colors.StatusColor,
		TypeColor:     colors.TypeColor,
		PriorityColor: colors.PriorityColor,
		Priority:      item.bean.Priority,
		Size:          item.bean.Size,
		SizeColor:     ui.SizeColor(d.cfg, item.bean.Size),
		ShowSize:      true,
		IsArchive:     colors.IsArchive,
		MaxTitleWidth: max(1, width-baseWidth), // 0 would disable truncation
		ShowCursor:    true,
		IsMarked:      d.selectedBeans[item.bean.ID],
		Tags:          item.bean.Tags,
		ShowTags:      d.cols.ShowTags,
		TagsColWidth:  d.cols.Tags,
		MaxTags:       d.cols.MaxTags,
		TreePrefix:    item.treePrefix,
		Dimmed:        !item.matched,
		HasChildren:   item.hasChildren,
		IsCollapsed:   d.collapsedBeans[item.bean.ID],
		Compact:       d.cols.Compact,
	}
}

// listModel is the model for the bean list view
type listModel struct {
	list     list.Model
	resolver *graph.Resolver
	config   *config.Config
	width    int
	height   int
	err      error

	project string // compacted project path for the title, e.g. "a/md.new"; "" shows "Beans"

	// Responsive column state
	hasTags    bool                 // whether any beans have tags
	cols       ui.ResponsiveColumns // calculated responsive columns
	idColWidth int                  // ID column width: the longest ID

	// Active filters
	query        query           // parsed filter query, see parseQuery
	queryInput   textinput.Model // filter query editor, focused while typing
	showArchived bool            // if false, beans with the archived status are hidden

	// Multi-select state
	selectedBeans map[string]bool // IDs of beans marked for multi-edit

	// Collapse/fold state
	collapsedBeans map[string]bool // IDs of beans whose children are hidden

	// Scrolling of the selected row's cut-off title (shared with the delegate)
	titleScroll *titleScroll

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

// mainListKeyMap returns the bubbles list key map for the bean list.
func mainListKeyMap() list.KeyMap {
	k := keymap.List
	return listKeyMap(k.Up, k.Down, k.PrevPage, k.NextPage, k.GoToStart, k.GoToEnd, k.Filter)
}

func newListModel(resolver *graph.Resolver, cfg *config.Config) listModel {
	selectedBeans := make(map[string]bool)
	collapsedBeans := make(map[string]bool)
	scroll := &titleScroll{}
	delegate := itemDelegate{
		cfg:            cfg,
		selectedBeans:  selectedBeans,
		collapsedBeans: collapsedBeans,
		titleScroll:    scroll,
	}

	l := list.New([]list.Item{}, delegate, 0, 0)
	// The pane frame shows the title; the title bar only shows the filter input
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.KeyMap = mainListKeyMap()
	l.Styles.Title = listTitleStyle

	qi := textinput.New()
	qi.Prompt = "/"
	qi.Placeholder = "text field:value -field:value sort:field[:asc|desc]"
	qi.PromptStyle = lipgloss.NewStyle().Foreground(ui.ColorPrimary)
	qi.PlaceholderStyle = lipgloss.NewStyle().Foreground(ui.ColorMuted)

	// In a cascading config, start with the scope as an explicit filter so it
	// can be seen and cleared like any other query.
	initialQuery := ""
	if cfg != nil {
		if scope := cfg.Scope(); scope != "" {
			initialQuery = "prefix:" + scope
		}
	}
	qi.SetValue(initialQuery)

	return listModel{
		list:           l,
		resolver:       resolver,
		config:         cfg,
		query:          parseQuery(initialQuery, cfg),
		queryInput:     qi,
		selectedBeans:  selectedBeans,
		collapsedBeans: collapsedBeans,
		titleScroll:    scroll,
	}
}

// beansLoadedMsg is sent when beans are loaded
type beansLoadedMsg struct {
	items       []ui.FlatItem         // flattened tree items
	idColWidth  int                   // ID column width: the longest ID
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
	// Build filter from the query and archived visibility
	showArchived := m.showArchived || slices.Contains(m.query.Filter.Status, config.StatusArchived)
	filter := *m.query.Filter
	if !showArchived {
		filter.ExcludeStatus = append(slices.Clip(filter.ExcludeStatus), config.StatusArchived)
	}
	// Query filtered beans
	filteredBeans, err := m.resolver.Query().Beans(context.Background(), &filter)
	if err != nil {
		return errMsg{err}
	}
	filteredBeans = slices.DeleteFunc(filteredBeans, func(b *bean.Bean) bool {
		return !matchesText(b, m.query.Text)
	})

	// Query all beans for tree context (ancestors)
	allBeans, err := m.resolver.Query().Beans(context.Background(), nil)
	if err != nil {
		return errMsg{err}
	}

	// beansByID keeps archived beans so relationships still resolve; the tree
	// and child counts only use visible beans, so archived parents stay hidden
	beansByID := make(map[string]*bean.Bean, len(allBeans))
	childCounts := make(map[string]int)
	visibleBeans := allBeans[:0:0]
	for _, b := range allBeans {
		beansByID[b.ID] = b
		if !showArchived && m.config.IsArchived(b.Status) {
			continue
		}
		visibleBeans = append(visibleBeans, b)
		if b.Parent != "" {
			childCounts[b.Parent]++
		}
	}
	allBeans = visibleBeans

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
	orders := bean.SortOrdersFrom(m.config)
	sortFn := func(beans []*bean.Bean) {
		bean.SortBy(beans, m.query.Sort.Field, m.query.Sort.Desc, orders)
	}

	// Build tree and flatten it with collapse state
	tree := ui.BuildTree(filteredBeans, allBeans, sortFn)
	items := ui.FlattenTreeWithCollapsed(tree, m.collapsedBeans)

	// The ID column fits the longest ID
	maxIDLen := 0
	for _, b := range allBeans {
		if len(b.ID) > maxIDLen {
			maxIDLen = len(b.ID)
		}
	}
	return beansLoadedMsg{
		items:       items,
		idColWidth:  maxIDLen,
		rootBeans:   rootBeans,
		beansByID:   beansByID,
		childCounts: childCounts,
	}
}

// scope returns the config's scope (see config.Config.Scope).
func (m listModel) scope() string {
	if m.config == nil {
		return ""
	}
	return m.config.Scope()
}

// prefixFilter returns the prefix: filter value, or "" if there is none.
func (m listModel) prefixFilter() string {
	if m.query.Filter == nil || m.query.Filter.IDPrefix == nil {
		return ""
	}
	return *m.query.Filter.IDPrefix
}

// setPrefixFilter replaces the query's prefix: token, or removes it if prefix
// is empty, keeping every other token.
func (m *listModel) setPrefixFilter(prefix string) {
	tokens := slices.DeleteFunc(strings.Fields(m.queryInput.Value()), func(t string) bool {
		field, _, ok := strings.Cut(strings.ToLower(t), ":")
		return ok && field == "prefix"
	})
	if prefix != "" {
		tokens = append(tokens, "prefix:"+prefix)
	}
	m.setQuery(strings.Join(tokens, " "))
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
	return true
}

// previewData returns the highlighted bean and its relationship context
func (m *listModel) previewData() (*bean.Bean, previewContext) {
	item, ok := m.list.SelectedItem().(beanItem)
	if !ok {
		return nil, previewContext{}
	}
	return item.bean, m.previewContextFor(item.bean)
}

// previewContextFor returns the relationship context of b among the loaded beans
func (m *listModel) previewContextFor(b *bean.Bean) previewContext {
	return previewContext{
		parent:     m.beansByID[b.Parent],
		childCount: m.childCounts[b.ID],
	}
}

// setQuery replaces the filter query
func (m *listModel) setQuery(input string) {
	m.queryInput.SetValue(input)
	m.query = parseQuery(input, m.config)
}

// setTagFilter makes tag the query's only included tag
func (m *listModel) setTagFilter(tag string) {
	m.setQuery(withTag(m.queryInput.Value(), tag))
}

// hasActiveFilter returns true if a filter query is applied
func (m *listModel) hasActiveFilter() bool {
	return strings.TrimSpace(m.queryInput.Value()) != ""
}

// editingQuery reports whether keys go to the filter query input
func (m *listModel) editingQuery() bool {
	return m.queryInput.Focused()
}

// updateQueryInput handles keys while the filter query is being edited.
// The list reloads on every change so results update as you type.
func (m listModel) updateQueryInput(msg tea.KeyMsg) (listModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.queryInput.Blur()
		return m, nil
	case tea.KeyEsc:
		m.queryInput.Blur()
		m.setQuery("")
		return m, m.loadBeans
	}
	before := m.queryInput.Value()
	var cmd tea.Cmd
	m.queryInput, cmd = m.queryInput.Update(msg)
	if m.queryInput.Value() == before {
		return m, cmd
	}
	m.query = parseQuery(m.queryInput.Value(), m.config)
	return m, tea.Batch(cmd, m.loadBeans)
}

// Update handles msg, then restarts the title scrolling if the cursor moved
// to another bean.
func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	if msg, ok := msg.(titleScrollMsg); ok {
		return m, m.stepTitleScroll(msg)
	}
	m, cmd := m.handle(msg)
	return m, tea.Batch(cmd, m.restartTitleScroll())
}

func (m listModel) handle(msg tea.Msg) (listModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
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
		if m.editingQuery() {
			return m.updateQueryInput(msg)
		}
		{
			k := keymap.List
			switch {
			case key.Matches(msg, k.Filter):
				m.queryInput.CursorEnd()
				return m, m.queryInput.Focus()
			case key.Matches(msg, k.Collapse):
				// Collapse current item if it has children
				if item, ok := m.list.SelectedItem().(beanItem); ok && item.hasChildren {
					m.collapsedBeans[item.bean.ID] = true
					return m, m.loadBeans
				}
				return m, nil
			case key.Matches(msg, k.Expand):
				// Expand current item if it has children
				if item, ok := m.list.SelectedItem().(beanItem); ok && item.hasChildren {
					delete(m.collapsedBeans, item.bean.ID)
					return m, m.loadBeans
				}
				return m, nil
			case key.Matches(msg, k.ToggleCollapse):
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
			case key.Matches(msg, k.CollapseAll):
				// Collapse all items with children
				for _, item := range m.list.Items() {
					if bi, ok := item.(beanItem); ok && bi.hasChildren {
						m.collapsedBeans[bi.bean.ID] = true
					}
				}
				return m, m.loadBeans
			case key.Matches(msg, k.ToggleSelect):
				// Toggle selection for multi-select
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					if m.selectedBeans[item.bean.ID] {
						delete(m.selectedBeans, item.bean.ID)
					} else {
						m.selectedBeans[item.bean.ID] = true
					}
				}
				return m, nil
			case key.Matches(msg, k.Open):
				// Drill into beans with children, show leaf beans in the context pane
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					if item.hasChildren {
						m.drillInto(item.bean.ID)
						return m, m.loadBeans
					}
					return m, func() tea.Msg {
						return selectBeanMsg{bean: item.bean}
					}
				}
			case key.Matches(msg, k.ChangeParent):
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
			case key.Matches(msg, k.ChangeStatus):
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
			case key.Matches(msg, k.ChangeType):
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
			case key.Matches(msg, k.EditTags):
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
			case key.Matches(msg, k.ChangePriority):
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
			case key.Matches(msg, k.ChangeSize):
				// Open size picker for selected bean(s)
				if len(m.selectedBeans) > 0 {
					// Multi-select mode
					ids := make([]string, 0, len(m.selectedBeans))
					for id := range m.selectedBeans {
						ids = append(ids, id)
					}
					return m, func() tea.Msg {
						return openSizePickerMsg{
							beanIDs:   ids,
							beanTitle: fmt.Sprintf("%d selected beans", len(ids)),
						}
					}
				} else if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openSizePickerMsg{
							beanIDs:     []string{item.bean.ID},
							beanTitle:   item.bean.DisplayTitle(),
							currentSize: item.bean.Size,
						}
					}
				}
			case key.Matches(msg, k.RaisePriority):
				// Raise priority one step for selected bean(s)
				return m, m.shiftPriority(-1)
			case key.Matches(msg, k.LowerPriority):
				// Lower priority one step for selected bean(s)
				return m, m.shiftPriority(1)
			case key.Matches(msg, k.FilterByTag):
				return m, func() tea.Msg { return openTagPickerMsg{} }
			case key.Matches(msg, k.ManageBlocking):
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
			case key.Matches(msg, k.ManageBlockedBy):
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
			case key.Matches(msg, k.Create):
				// Open create modal
				return m, func() tea.Msg {
					return openCreateModalMsg{prefill: m.query.prefill()}
				}
			case key.Matches(msg, k.CreateChild):
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
							prefill:     m.query.prefill(),
						}
					}
				}
			case key.Matches(msg, k.Edit):
				// Open the edit form for the bean under the cursor
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openEditModalMsg{bean: item.bean}
					}
				}
			case key.Matches(msg, k.OpenEditor):
				// Open $EDITOR for the bean under the cursor
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg {
						return openEditorMsg{
							beanID:   item.bean.ID,
							beanPath: item.bean.Path,
						}
					}
				}
			case key.Matches(msg, k.Launch):
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
			case key.Matches(msg, k.CopyID):
				// Copy the ID of the bean under the cursor (ignores the selection)
				if item, ok := m.list.SelectedItem().(beanItem); ok {
					return m, func() tea.Msg { return copyIDMsg{beanID: item.bean.ID} }
				}
			case key.Matches(msg, k.Archive):
				// Archive all completed beans (asks for confirmation)
				return m, func() tea.Msg {
					return archiveCompletedMsg{}
				}
			case key.Matches(msg, k.ToggleArchived):
				// Toggle showing archived beans
				m.showArchived = !m.showArchived
				return m, m.loadBeans
			case key.Matches(msg, k.ToggleScope):
				// Toggle the scope's prefix: filter token
				scope := m.scope()
				if scope == "" {
					break
				}
				if m.prefixFilter() == scope {
					m.setPrefixFilter("")
				} else {
					m.setPrefixFilter(scope)
				}
				return m, m.loadBeans
			case key.Matches(msg, k.Back):
				// First clear selection if any beans are selected
				if len(m.selectedBeans) > 0 {
					clear(m.selectedBeans)
					return m, nil
				}
				// Then clear the filter query
				if m.hasActiveFilter() {
					m.setQuery("")
					return m, m.loadBeans
				}
				// Then navigate up the hierarchy
				if m.drillOut() {
					return m, m.loadBeans
				}
				// Nothing left to back out of
				return m, tea.Quit
			}
		}
	}

	// Always forward to the list component, and to the query input while it is
	// focused so its cursor keeps blinking
	m.list, cmd = m.list.Update(msg)
	if m.editingQuery() {
		var inputCmd tea.Cmd
		m.queryInput, inputCmd = m.queryInput.Update(msg)
		cmd = tea.Batch(cmd, inputCmd)
	}
	return m, cmd
}

// setSize sizes the list to fill a pane of width x height (inside its frame).
func (m *listModel) setSize(width, height int) {
	if width == m.width && height == m.height {
		return
	}
	m.width, m.height = width, height
	m.list.SetSize(width, height)
	m.cols = ui.CalculateResponsiveColumns(width, m.hasTags)
	m.updateDelegate()
}

// updateDelegate updates the list delegate with current responsive columns
func (m *listModel) updateDelegate() {
	m.list.SetDelegate(m.delegate())
}

// delegate renders the rows with the current columns.
func (m listModel) delegate() itemDelegate {
	return itemDelegate{
		cfg:            m.config,
		hasTags:        m.hasTags,
		width:          m.width,
		cols:           m.cols,
		idColWidth:     m.idColWidth,
		selectedBeans:  m.selectedBeans,
		collapsedBeans: m.collapsedBeans,
		titleScroll:    m.titleScroll,
	}
}

// View renders the list content at the size set by setSize.
func (m listModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err)
	}
	return m.list.View()
}

// title is the pane title, e.g. "Beans › Parent › Child [tag: x]", with the
// project path in place of "Beans" in a nested project ("a/md.new › Parent").
func (m listModel) title() string {
	return m.breadcrumbTitle(m.width - 4)
}

// breadcrumbTitle renders the list title, e.g. "Beans › Parent › Child [tag: x]",
// eliding outer levels to fit maxWidth.
func (m listModel) breadcrumbTitle(maxWidth int) string {
	const sep = " › "
	suffix := ""
	if q := strings.Join(strings.Fields(m.queryInput.Value()), " "); q != "" {
		suffix = fmt.Sprintf(" [%s]", q)
	}
	if m.showArchived {
		suffix += " [+archived]"
	}

	root := m.project
	if root == "" {
		root = "Beans"
	}
	crumbs := make([]string, 0, len(m.rootBeans)+1)
	crumbs = append(crumbs, root)
	for _, b := range m.rootBeans {
		crumbs = append(crumbs, b.DisplayTitle())
	}

	title := strings.Join(crumbs, sep) + suffix
	// Replace outer crumbs (after the root) with "…" until it fits
	for elided := 1; lipgloss.Width(title) > maxWidth && elided < len(crumbs)-1; elided++ {
		title = strings.Join(append([]string{root, "…"}, crumbs[elided+1:]...), sep) + suffix
	}
	if maxWidth > 1 && lipgloss.Width(title) > maxWidth {
		title = string([]rune(title)[:maxWidth-1]) + "…"
	}
	return title
}

// footer renders the list's hints for the footer, or the filter query input
// while it is being edited
func (m listModel) footer() string {
	if m.editingQuery() {
		return m.queryFooter()
	}

	var help string

	k := keymap.List

	// esc goes up a level when drilled in
	var backHelp string
	if len(m.rootStack) > 0 {
		backHelp = hint(k.Back, "up")
	}

	// Show selection count if any beans are selected
	var selectionPrefix string
	if len(m.selectedBeans) > 0 {
		selectionStyle := lipgloss.NewStyle().Foreground(ui.ColorWarning).Bold(true)
		selectionPrefix = selectionStyle.Render(fmt.Sprintf("(%d selected) ", len(m.selectedBeans)))
	}

	archivedHelp := "show archived"
	if m.showArchived {
		archivedHelp = "hide archived"
	}

	// The scope toggle only matters in a scoped config
	var scopeHelp string
	if scope := m.scope(); scope != "" {
		if m.prefixFilter() == scope {
			scopeHelp = hint(k.ToggleScope, "all scopes")
		} else {
			scopeHelp = hint(k.ToggleScope, scope+"* only")
		}
	}

	if len(m.selectedBeans) > 0 {
		// When beans are selected, show esc to clear selection
		help = hints(
			hint(k.ToggleSelect, "toggle"),
			hint(k.ChangeStatus, ""),
			hint(k.ChangeType, ""),
			hint(k.ChangePriority, ""),
			hint(k.ChangeSize, ""),
			hint(k.Launch, ""),
			hint(k.EditTags, ""),
			hint(k.Back, "clear selection"),
		)
	} else if m.hasActiveFilter() {
		help = hints(
			backHelp,
			hint(k.ToggleSelect, ""),
			hint(k.Open, "open"),
			hint(k.Create, ""),
			hint(k.Edit, ""),
			hint(k.OpenEditor, ""),
			hint(k.ChangeStatus, ""),
			hint(k.ChangeType, ""),
			hint(k.ChangePriority, ""),
			hint(k.ChangeSize, ""),
			groupHint("prio +/-", k.RaisePriority, k.LowerPriority),
			hint(k.EditTags, ""),
			hint(k.ChangeParent, ""),
			hint(k.ManageBlocking, ""),
			hint(k.ManageBlockedBy, ""),
			hint(k.Launch, ""),
			hint(k.Filter, "edit filter"),
			hint(k.Back, "clear filter"),
		)
	} else {
		help = hints(
			backHelp,
			hint(k.ToggleSelect, ""),
			hint(k.Open, "open"),
			hint(k.Create, ""),
			hint(k.Edit, ""),
			hint(k.OpenEditor, ""),
			hint(k.ChangeStatus, ""),
			hint(k.ChangeType, ""),
			hint(k.ChangePriority, ""),
			hint(k.ChangeSize, ""),
			groupHint("prio +/-", k.RaisePriority, k.LowerPriority),
			hint(k.EditTags, ""),
			hint(k.ChangeParent, ""),
			hint(k.ManageBlocking, ""),
			hint(k.ManageBlockedBy, ""),
			hint(k.Launch, ""),
			hint(k.CopyID, ""),
			hint(k.Archive, ""),
			hint(k.ToggleArchived, archivedHelp),
			scopeHelp,
			groupHint("fold", k.Collapse, k.Expand, k.ToggleCollapse),
			hint(k.Filter, "filter"),
		)
	}

	return selectionPrefix + help
}

// noticeView renders the transient footer notice, if any.
func (m listModel) noticeView() string {
	if m.notice == "" {
		return ""
	}
	color := ui.ColorSuccess
	if m.noticeIsErr {
		color = ui.ColorDanger
	}
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(m.notice) + "  "
}

// queryFooter renders the filter query input with any warnings about ignored tokens
func (m listModel) queryFooter() string {
	line := m.queryInput.View()
	if len(m.query.Warnings) > 0 {
		warn := lipgloss.NewStyle().Foreground(ui.ColorDanger)
		line += "  " + warn.Render(strings.Join(m.query.Warnings, ", "))
	}
	keyHint := func(k, desc string) string { return helpKeyStyle.Render(k) + " " + helpStyle.Render(desc) + "  " }
	return line + "  " + hints(keyHint("enter", "apply"), keyHint("esc", "clear"))
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
