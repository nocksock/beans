package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	launcherexec "github.com/hmans/beans/internal/launcher"
)

// viewState represents which view is currently active
type viewState int

const (
	viewList viewState = iota
	viewDetail
	viewTagPicker
	viewParentPicker
	viewStatusPicker
	viewTypePicker
	viewBlockingPicker
	viewPriorityPicker
	viewSizePicker
	viewCreateModal
	viewHelpOverlay
	viewLauncherPicker
	viewLauncherError
	viewNoLaunchers
	viewNoLaunchersMulti
	viewConfigureLaunchers
	viewConfigWriteError
	viewTagEditor
	viewArchiveConfirm
	viewDepNav
)

// Two-column layout constants
const (
	twoColumnMinWidth = 120 // minimum terminal width for the list + preview layout
	previewMaxWidth   = 80  // preview pane width cap (bean bodies are written for ~80 columns)
	listMinWidth      = 40  // list pane never gets narrower than this
)

// paneWidths returns the (list, preview) widths for the two-column layout.
func paneWidths(totalWidth int) (int, int) {
	preview := min(previewMaxWidth, totalWidth-listMinWidth)
	return totalWidth - preview, preview
}

// beansChangedMsg is sent when beans change on disk (via file watcher)
type beansChangedMsg struct{}

// openTagPickerMsg requests opening the tag picker
type openTagPickerMsg struct{}

// tagSelectedMsg is sent when a tag is selected from the picker
type tagSelectedMsg struct {
	tag string
}

// clearFilterMsg is sent to clear any active filter
type clearFilterMsg struct{}

// openEditorMsg requests opening the editor for a bean
type openEditorMsg struct {
	beanID   string
	beanPath string
}

// editorFinishedMsg is sent when the editor closes
type editorFinishedMsg struct {
	err error
}

// openParentPickerMsg requests opening the parent picker for bean(s)
type openParentPickerMsg struct {
	beanIDs       []string // IDs of beans to update
	beanTitle     string   // Display title (single title or "N selected beans")
	beanTypes     []string // Types of the beans (to filter eligible parents)
	currentParent string   // Only meaningful for single bean
}

// App is the main TUI application model
type App struct {
	state              viewState
	list               listModel
	detail             detailModel
	depNav             depNavModel
	preview            previewModel
	tagPicker          tagPickerModel
	parentPicker       parentPickerModel
	statusPicker       statusPickerModel
	typePicker         typePickerModel
	blockingPicker     blockingPickerModel
	tagEditor          tagEditorModel
	archiveConfirm     archiveConfirmModel
	priorityPicker     priorityPickerModel
	sizePicker         sizePickerModel
	createModal        createModalModel
	helpOverlay        helpOverlayModel
	launcherPicker     launcherPickerModel
	launcherError      launcherErrorModel
	noLaunchers        noLaunchersModel
	noLaunchersMulti   noLaunchersMultiModel
	configureLaunchers configureLaunchersModel
	configWriteError   configWriteErrorModel
	log                logPanelModel
	history            []detailModel // stack of previous detail views for back navigation
	core               *beancore.Core
	resolver           *graph.Resolver
	config             *config.Config
	width              int
	height             int
	program            *tea.Program // reference to program for sending messages from watcher

	// Key chord state - the keys typed so far of a chord like "g t"
	pendingKeys string

	// Modal state - tracks view behind modal pickers
	previousState viewState

	// Dependency navigator state - the view it was opened from, and whether
	// the detail view was opened from it (so going back returns to it)
	depNavOrigin     viewState
	detailFromDepNav bool
	depNavHistory    int         // detail history depth when the navigator opened a bean
	depNavDetail     detailModel // detail view the navigator was opened from

	// Editor state - tracks bean being edited to update updated_at on save
	editingBeanID      string
	editingBeanModTime time.Time

	// Quitting flag - set before running external commands to prevent View output
	quitting bool

	// noticeSeq identifies the latest footer notice, so stale clears are ignored
	noticeSeq int

	// keyWarnings are the key binding conflicts found at startup
	keyWarnings []string
}

// New creates a new TUI application. It uses the key bindings from cfg, or
// the defaults if they are invalid (Run reports that error).
func New(core *beancore.Core, cfg *config.Config) *App {
	resolver := &graph.Resolver{Core: core}
	warnings, err := applyKeymap(cfg)
	if err != nil {
		keymap = DefaultKeyMap()
	}
	return &App{
		state:       viewList,
		core:        core,
		resolver:    resolver,
		config:      cfg,
		list:        newListModel(resolver, cfg),
		preview:     newPreviewModel(cfg),
		keyWarnings: warnings,
	}
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	if len(a.keyWarnings) > 0 {
		return tea.Batch(a.list.Init(), a.showNotice(keyWarningNotice(a.keyWarnings), true))
	}
	return a.list.Init()
}

// Update handles messages
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		var pending bool
		if msg, pending = a.resolveChord(keyMsg); pending {
			return a, nil
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// The list is sized for its pane; other views get the full size
		cmd = a.resizeList()
		if a.state == viewList {
			return a, cmd
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keymap.Global.ForceQuit):
			return a, tea.Quit
		case key.Matches(msg, keymap.Global.Help):
			// Open help overlay if not already showing it (and not in a picker/modal)
			if a.state == viewList || a.state == viewDetail {
				a.previousState = a.state
				a.helpOverlay = newHelpOverlayModel(a.width, a.height)
				a.state = viewHelpOverlay
				return a, a.helpOverlay.Init()
			}
		case key.Matches(msg, keymap.Global.Quit):
			if a.canQuit() {
				return a, tea.Quit
			}
		case key.Matches(msg, keymap.Global.ReloadConfig):
			if (a.state == viewList || a.state == viewDetail) && a.canQuit() {
				return a, func() tea.Msg { return reloadConfigMsg{} }
			}
		case key.Matches(msg, keymap.List.ToggleLog):
			if a.state == viewList && a.canQuit() {
				a.log.visible = !a.log.visible
				return a, a.resizeList()
			}
		case key.Matches(msg, keymap.List.OpenLog):
			if a.state == viewList && a.canQuit() {
				return a, a.openLatestLog()
			}
		}

	case beansChangedMsg:
		// Beans changed on disk - refresh
		if a.state == viewDepNav && !a.depNav.reload() {
			a.state = a.depNavOrigin
		}
		if a.state == viewDetail {
			// Try to reload the current bean via GraphQL
			updatedBean, err := a.resolver.Query().Bean(context.Background(), a.detail.bean.ID)
			if err != nil || updatedBean == nil {
				// Bean was deleted - return to list
				a.state = viewList
				a.history = nil
			} else {
				// Recreate detail view with fresh bean data
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		// Trigger list refresh
		return a, a.list.loadBeans

	case configChangedMsg, reloadConfigMsg:
		return a, a.reloadConfig()

	case clearNoticeMsg:
		if msg.seq == a.noticeSeq {
			a.list.notice = ""
		}
		return a, nil

	case openTagPickerMsg:
		// Collect all tags with their counts
		tags := a.collectTagsWithCounts()
		if len(tags) == 0 {
			// No tags in system, don't open picker
			return a, nil
		}
		a.tagPicker = newTagPickerModel(tags, a.width, a.height)
		a.state = viewTagPicker
		return a, a.tagPicker.Init()

	case tagSelectedMsg:
		a.state = viewList
		a.list.setTagFilter(msg.tag)
		return a, a.list.loadBeans

	case openParentPickerMsg:
		// Check if all bean types can have parents
		for _, beanType := range msg.beanTypes {
			if beancore.ValidParentTypes(beanType) == nil {
				// At least one bean type (e.g., milestone) cannot have parents - don't open the picker
				return a, nil
			}
		}
		a.previousState = a.state // Remember where we came from for the modal background
		a.parentPicker = newParentPickerModel(msg.beanIDs, msg.beanTitle, msg.beanTypes, msg.currentParent, a.resolver, a.config, a.width, a.height)
		a.state = viewParentPicker
		return a, a.parentPicker.Init()

	case closeParentPickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a, a.list.loadBeans

	case openStatusPickerMsg:
		a.previousState = a.state
		a.statusPicker = newStatusPickerModel(msg.beanIDs, msg.beanTitle, msg.currentStatus, a.config, a.width, a.height)
		a.state = viewStatusPicker
		return a, a.statusPicker.Init()

	case closeStatusPickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a, a.list.loadBeans

	case statusSelectedMsg:
		// Update all beans' status via GraphQL mutations
		for _, beanID := range msg.beanIDs {
			_, err := a.resolver.Mutation().UpdateBean(context.Background(), beanID, model.UpdateBeanInput{
				Status: &msg.status,
			})
			if err != nil {
				// Continue with other beans even if one fails
				continue
			}
		}
		// Return to the previous view and refresh
		a.state = a.previousState
		// Clear selection after batch edit
		clear(a.list.selectedBeans)
		if a.state == viewDetail && len(msg.beanIDs) == 1 {
			updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanIDs[0])
			if updatedBean != nil {
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		return a, a.list.loadBeans

	case openTypePickerMsg:
		a.previousState = a.state
		a.typePicker = newTypePickerModel(msg.beanIDs, msg.beanTitle, msg.currentType, a.config, a.width, a.height)
		a.state = viewTypePicker
		return a, a.typePicker.Init()

	case closeTypePickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a, a.list.loadBeans

	case typeSelectedMsg:
		// Update all beans' type via GraphQL mutations
		for _, beanID := range msg.beanIDs {
			_, err := a.resolver.Mutation().UpdateBean(context.Background(), beanID, model.UpdateBeanInput{
				Type: &msg.beanType,
			})
			if err != nil {
				// Continue with other beans even if one fails
				continue
			}
		}
		// Return to the previous view and refresh
		a.state = a.previousState
		// Clear selection after batch edit
		clear(a.list.selectedBeans)
		if a.state == viewDetail && len(msg.beanIDs) == 1 {
			updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanIDs[0])
			if updatedBean != nil {
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		return a, a.list.loadBeans

	case openSizePickerMsg:
		a.previousState = a.state
		a.sizePicker = newSizePickerModel(msg.beanIDs, msg.beanTitle, msg.currentSize, a.width, a.height)
		a.state = viewSizePicker
		return a, a.sizePicker.Init()

	case closeSizePickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a, a.list.loadBeans

	case sizeSelectedMsg:
		// Update all beans' size via GraphQL mutations
		for _, beanID := range msg.beanIDs {
			_, err := a.resolver.Mutation().UpdateBean(context.Background(), beanID, model.UpdateBeanInput{
				Size: &msg.size,
			})
			if err != nil {
				// Continue with other beans even if one fails
				continue
			}
		}
		// Return to the previous view and refresh
		a.state = a.previousState
		// Clear selection after batch edit
		clear(a.list.selectedBeans)
		if a.state == viewDetail && len(msg.beanIDs) == 1 {
			updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanIDs[0])
			if updatedBean != nil {
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		return a, a.list.loadBeans

	case openPriorityPickerMsg:
		a.previousState = a.state
		a.priorityPicker = newPriorityPickerModel(msg.beanIDs, msg.beanTitle, msg.currentPriority, a.config, a.width, a.height)
		a.state = viewPriorityPicker
		return a, a.priorityPicker.Init()

	case closePriorityPickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a, a.list.loadBeans

	case prioritySelectedMsg:
		// Update all beans' priority via GraphQL mutations
		for _, beanID := range msg.beanIDs {
			_, err := a.resolver.Mutation().UpdateBean(context.Background(), beanID, model.UpdateBeanInput{
				Priority: &msg.priority,
			})
			if err != nil {
				// Continue with other beans even if one fails
				continue
			}
		}
		// Return to the previous view and refresh
		a.state = a.previousState
		// Clear selection after batch edit
		clear(a.list.selectedBeans)
		if a.state == viewDetail && len(msg.beanIDs) == 1 {
			updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanIDs[0])
			if updatedBean != nil {
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		return a, a.list.loadBeans

	case openHelpMsg:
		a.previousState = a.state
		a.helpOverlay = newHelpOverlayModel(a.width, a.height)
		a.state = viewHelpOverlay
		return a, a.helpOverlay.Init()

	case closeHelpMsg:
		a.state = a.previousState
		return a, nil

	case openTagEditorMsg:
		var beans []*bean.Bean
		for _, id := range msg.beanIDs {
			if b, err := a.resolver.Query().Bean(context.Background(), id); err == nil && b != nil {
				beans = append(beans, b)
			}
		}
		if len(beans) == 0 {
			return a, nil
		}
		a.previousState = a.state
		a.tagEditor = newTagEditorModel(beans, msg.beanTitle, a.collectTagsWithCounts(), a.width, a.height)
		a.state = viewTagEditor
		return a, a.tagEditor.Init()

	case closeTagEditorMsg:
		a.state = a.previousState
		return a, a.list.loadBeans

	case tagsConfirmedMsg:
		// Apply tag changes to each bean via GraphQL mutations
		if len(msg.toAdd) > 0 || len(msg.toRemove) > 0 {
			for _, beanID := range msg.beanIDs {
				b, err := a.resolver.Query().Bean(context.Background(), beanID)
				if err != nil || b == nil {
					continue
				}
				tags := applyTagChanges(b.Tags, msg.toAdd, msg.toRemove)
				_, err = a.resolver.Mutation().UpdateBean(context.Background(), beanID, model.UpdateBeanInput{
					Tags: tags,
				})
				if err != nil {
					// Continue with other beans even if one fails
					continue
				}
			}
		}
		// Return to the previous view and refresh
		a.state = a.previousState
		// Clear selection after batch edit
		clear(a.list.selectedBeans)
		if a.state == viewDetail && len(msg.beanIDs) == 1 {
			updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanIDs[0])
			if updatedBean != nil {
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		return a, a.list.loadBeans

	case openBlockingPickerMsg:
		a.previousState = a.state
		a.blockingPicker = newBlockingPickerModel(msg.beanID, msg.beanTitle, msg.blockedBy, msg.currentIDs, a.resolver, a.config, a.width, a.height)
		a.state = viewBlockingPicker
		return a, a.blockingPicker.Init()

	case closeBlockingPickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a, a.list.loadBeans

	case blockingConfirmedMsg:
		// Apply all blocking changes via GraphQL mutations
		addLink, removeLink := a.resolver.Mutation().AddBlocking, a.resolver.Mutation().RemoveBlocking
		if msg.blockedBy {
			addLink, removeLink = a.resolver.Mutation().AddBlockedBy, a.resolver.Mutation().RemoveBlockedBy
		}
		for _, targetID := range msg.toAdd {
			_, err := addLink(context.Background(), msg.beanID, targetID)
			if err != nil {
				// Continue with other changes even if one fails
				continue
			}
		}
		for _, targetID := range msg.toRemove {
			_, err := removeLink(context.Background(), msg.beanID, targetID)
			if err != nil {
				// Continue with other changes even if one fails
				continue
			}
		}
		// Return to previous view and refresh
		a.state = a.previousState
		if a.state == viewDetail {
			updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanID)
			if updatedBean != nil {
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		return a, a.list.loadBeans

	case archiveCompletedMsg:
		var ids []string
		beans, _ := a.resolver.Query().Beans(context.Background(), nil)
		for _, b := range beans {
			if a.config.IsArchivable(b.Status) {
				ids = append(ids, b.ID)
			}
		}
		if len(ids) == 0 {
			return a, nil
		}
		a.previousState = a.state
		a.archiveConfirm = newArchiveConfirmModel(ids,
			fmt.Sprintf("Archive %d bean(s)?", len(ids)),
			"All completed and scrapped beans will be moved to archived.",
			a.width, a.height)
		a.state = viewArchiveConfirm
		return a, nil

	case archiveBeanMsg:
		b, err := a.resolver.Query().Bean(context.Background(), msg.beanID)
		if err != nil || b == nil {
			return a, nil
		}
		if a.config.IsArchived(b.Status) {
			return a, nil
		}
		if a.config.IsArchiveStatus(b.Status) {
			return a, func() tea.Msg { return archiveConfirmedMsg{beanIDs: []string{b.ID}} }
		}
		a.previousState = a.state
		a.archiveConfirm = newArchiveConfirmModel([]string{b.ID},
			"Archive unfinished bean?",
			fmt.Sprintf("%s is %q, not completed. It will be archived anyway.", b.ID, b.Status),
			a.width, a.height)
		a.state = viewArchiveConfirm
		return a, nil

	case closeArchiveConfirmMsg:
		a.state = a.previousState
		return a, nil

	case archiveConfirmedMsg:
		status := config.StatusArchived
		for _, id := range msg.beanIDs {
			if _, err := a.resolver.Mutation().UpdateBean(context.Background(), id, model.UpdateBeanInput{
				Status: &status,
			}); err != nil {
				a.previousState = viewList
				a.launcherError = newLauncherErrorModel("Archive", err, a.width, a.height)
				a.state = viewLauncherError
				return a, a.list.loadBeans
			}
		}
		// Archiving from the detail view stays there, showing the new status
		fromDetail := a.state == viewDetail || (a.state == viewArchiveConfirm && a.previousState == viewDetail)
		if fromDetail && len(msg.beanIDs) == 1 {
			if updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanIDs[0]); updatedBean != nil {
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
				a.state = viewDetail
				return a, a.list.loadBeans
			}
		}
		a.state = viewList
		return a, a.list.loadBeans

	case openCreateModalMsg:
		a.previousState = a.state
		a.createModal = newCreateModalModel(a.config, a.width, a.height, msg.parentID, msg.parentTitle)
		a.state = viewCreateModal
		return a, a.createModal.Init()

	case closeCreateModalMsg:
		a.state = a.previousState
		return a, nil

	case openEditModalMsg:
		a.previousState = a.state
		a.createModal = newEditModalModel(a.config, a.width, a.height, msg.bean)
		a.state = viewCreateModal
		return a, a.createModal.Init()

	case beanEditedMsg:
		// Save only the changed fields; an unchanged form doesn't touch the bean
		a.state = a.previousState
		input := msg.input
		if input.Title != nil || input.Body != nil || input.Type != nil || input.Status != nil || input.Priority != nil {
			if _, err := a.resolver.Mutation().UpdateBean(context.Background(), msg.beanID, input); err != nil {
				a.launcherError = newLauncherErrorModel("Edit Bean", err, a.width, a.height)
				a.state = viewLauncherError
				return a, nil
			}
		}
		updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanID)
		if updatedBean == nil {
			return a, a.list.loadBeans
		}
		if a.state == viewDetail {
			a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
		}
		a.list.cursorTarget = updatedBean.ID
		if !msg.openEditor {
			return a, a.list.loadBeans
		}
		return a, tea.Batch(
			a.list.loadBeans,
			func() tea.Msg {
				return openEditorMsg{beanID: updatedBean.ID, beanPath: updatedBean.Path}
			},
		)

	case beanCreatedMsg:
		// Create the bean via GraphQL mutation (draft status unless chosen otherwise)
		input := model.CreateBeanInput{}
		if msg.title != "" {
			input.Title = &msg.title
		}
		status := createModalDefaultStatus
		if msg.status != "" {
			status = msg.status
		}
		input.Status = &status
		if msg.beanType != "" {
			input.Type = &msg.beanType
		}
		if msg.priority != "" {
			input.Priority = &msg.priority
		}
		if msg.size != "" {
			input.Size = &msg.size
		}
		if msg.body != "" {
			input.Body = &msg.body
		}
		if msg.parentID != "" {
			input.Parent = &msg.parentID
		}
		createdBean, err := a.resolver.Mutation().CreateBean(context.Background(), input)
		if err != nil {
			// Show error to user
			a.launcherError = newLauncherErrorModel("Create Bean", err, a.width, a.height)
			a.state = viewLauncherError
			return a, nil
		}
		// Return to list, move the cursor to the new bean, and optionally open it in the editor
		a.state = viewList
		a.list.cursorTarget = createdBean.ID
		if !msg.openEditor {
			return a, a.list.loadBeans
		}
		return a, tea.Batch(
			a.list.loadBeans,
			func() tea.Msg {
				return openEditorMsg{beanID: createdBean.ID, beanPath: createdBean.Path}
			},
		)

	case openEditorMsg:
		// Launch editor for the bean file
		editor := getEditor()
		fullPath := filepath.Join(a.core.Root(), msg.beanPath)

		// Record the bean ID and file mod time before editing
		a.editingBeanID = msg.beanID
		if info, err := os.Stat(fullPath); err == nil {
			a.editingBeanModTime = info.ModTime()
		}

		c := exec.Command(editor, fullPath)
		a.quitting = true
		return a, tea.ExecProcess(c, func(err error) tea.Msg {
			a.quitting = false
			return editorFinishedMsg{err: err}
		})

	case editorFinishedMsg:
		// Editor closed - check if file was modified and update updated_at if so
		if a.editingBeanID != "" {
			if b, err := a.core.Get(a.editingBeanID); err == nil {
				fullPath := filepath.Join(a.core.Root(), b.Path)
				if info, err := os.Stat(fullPath); err == nil {
					if info.ModTime().After(a.editingBeanModTime) {
						// File was modified - reload from disk first to get user's changes,
						// then call Update to set updated_at
						_ = a.core.Load()
						if b, err = a.core.Get(a.editingBeanID); err == nil {
							_ = a.core.Update(b)
						}
					}
				}
			}
			// Keep the cursor on the edited bean, even if edits changed its sort position
			a.list.cursorTarget = a.editingBeanID
			// Clear editing state
			a.editingBeanID = ""
			a.editingBeanModTime = time.Time{}
			return a, a.list.loadBeans
		}
		return a, nil

	case parentSelectedMsg:
		// Set the new parent via GraphQL mutation for all beans
		var parentID *string
		if msg.parentID != "" {
			parentID = &msg.parentID
		}
		for _, beanID := range msg.beanIDs {
			_, err := a.resolver.Mutation().SetParent(context.Background(), beanID, parentID)
			if err != nil {
				// Continue with other beans even if one fails
				continue
			}
		}
		// Return to the previous view and refresh
		a.state = a.previousState
		// Clear selection after batch edit
		clear(a.list.selectedBeans)
		if a.state == viewDetail && len(msg.beanIDs) == 1 {
			// Refresh the bean to show updated parent
			updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanIDs[0])
			if updatedBean != nil {
				a.detail = newDetailModel(updatedBean, a.resolver, a.config, a.width, a.height)
			}
		}
		return a, a.list.loadBeans

	case openLauncherPickerMsg:
		// Discover available launchers based on bean count
		var launchers []launcher

		if len(msg.beanIDs) > 1 {
			// Multiple beans - only show launchers with multiple: true
			launchers = discoverLaunchersForMultiple(a.config, a.core.Root())
		} else {
			// Single bean - show all launchers
			launchers = discoverLaunchers(a.config, a.core.Root())
		}

		if len(launchers) == 0 {
			// Check if this is multi-bean with no multiple-capable launchers
			if len(msg.beanIDs) > 1 {
				// Show specific error for multi-bean
				a.previousState = a.state
				a.noLaunchersMulti = newNoLaunchersMultiModel(a.width, a.height)
				a.state = viewNoLaunchersMulti
				return a, nil
			}

			// Check if any launchers are configured at all
			if !hasLaunchersConfigured(a.config) {
				// First time - offer to configure defaults
				// For configure dialog, just show the first bean ID
				firstBeanID := ""
				if len(msg.beanIDs) > 0 {
					firstBeanID = msg.beanIDs[0]
				}
				a.previousState = a.state
				a.configureLaunchers = newConfigureLaunchersModel(firstBeanID, msg.beanTitle, a.width, a.height)
				a.state = viewConfigureLaunchers
				return a, a.configureLaunchers.Init()
			}

			// Launchers configured but none available
			a.previousState = a.state
			a.noLaunchers = newNoLaunchersModel(a.width, a.height)
			a.state = viewNoLaunchers
			return a, nil
		}

		// Open launcher picker
		a.previousState = a.state
		a.launcherPicker = newLauncherPickerModel(launchers, msg.beanIDs, msg.beanTitle, a.width, a.height)
		a.state = viewLauncherPicker
		return a, a.launcherPicker.Init()

	case launcherSelectedMsg:
		// Close the launcher picker first
		a.state = a.previousState

		if len(msg.beanIDs) == 0 {
			return a, nil
		}

		if !msg.launcher.background {
			// Foreground launchers get the terminal, so only one bean can run
			return a, a.runLauncher(msg.launcher, msg.beanIDs[0], msg.beanTitle)
		}

		// Background launchers run for every bean in parallel
		cmds := make([]tea.Cmd, 0, len(msg.beanIDs)+1)
		for _, id := range msg.beanIDs {
			cmds = append(cmds, a.runLauncherInBackground(msg.launcher, id, msg.beanTitle))
		}
		cmds = append(cmds, a.resizeList())
		return a, tea.Batch(cmds...)

	case closeLauncherPickerMsg:
		a.state = a.previousState
		return a, nil

	case launchFinishedMsg:
		a.log.finish(msg.id, msg.exitCode, msg.err, time.Now())
		// Failures show the panel, so the list may need to shrink
		return a, a.resizeList()

	case launchersConfiguredMsg:
		// Append selected launchers to .beans.yml
		projectRoot := a.projectRoot()

		if err := appendLaunchersToConfig(projectRoot, msg.launchers); err != nil {
			// Show error modal
			a.previousState = viewDetail
			a.configWriteError = newConfigWriteErrorModel(err, a.width, a.height)
			a.state = viewConfigWriteError
			return a, nil
		}

		// Return to previous state (detail view), then reload config to pick
		// up the new launchers
		a.state = a.previousState
		return a, a.reloadConfig()

	case closeConfigureLaunchersMsg:
		// User cancelled - return to previous state
		a.state = a.previousState
		return a, nil

	case clearFilterMsg:
		a.list.clearFilter()
		return a, a.list.loadBeans

	case openDepNavMsg:
		a.depNavOrigin = a.state
		a.depNavDetail = a.detail
		a.detailFromDepNav = false
		a.depNav = newDepNavModel(msg.bean, a.resolver, a.config, a.width, a.height)
		a.state = viewDepNav
		return a, a.depNav.Init()

	case closeDepNavMsg:
		a.state = a.depNavOrigin
		if a.state == viewDetail {
			// The navigator may have opened other beans; restore the original one
			a.detail = a.depNavDetail
			if b, err := a.resolver.Query().Bean(context.Background(), a.detail.bean.ID); err == nil && b != nil {
				a.detail = newDetailModel(b, a.resolver, a.config, a.width, a.height)
			}
		}
		return a, a.list.loadBeans

	case selectBeanMsg:
		// Push current detail view to history if we're already viewing a bean
		if a.state == viewDetail {
			a.history = append(a.history, a.detail)
		}
		if a.state == viewDepNav {
			// Going back from this detail view returns to the navigator
			a.depNavHistory = len(a.history)
			a.detailFromDepNav = true
		}
		a.state = viewDetail
		a.detail = newDetailModel(msg.bean, a.resolver, a.config, a.width, a.height)
		return a, a.detail.Init()

	case browseBeanMsg:
		if a.state == viewDetail {
			a.browseBean(msg.delta)
		}
		return a, nil

	case backToListMsg:
		// Return to the dependency navigator if the detail view came from it
		if a.state == viewDetail && a.detailFromDepNav && len(a.history) == a.depNavHistory {
			a.detailFromDepNav = false
			if !a.depNav.reload() {
				a.state = a.depNavOrigin
				return a, nil
			}
			a.state = viewDepNav
			return a, nil
		}
		// Pop from history if available, otherwise go to list
		if len(a.history) > 0 {
			a.detail = a.history[len(a.history)-1]
			a.history = a.history[:len(a.history)-1]
			// Stay in viewDetail state
		} else {
			a.state = viewList
			return a, nil
		}
		return a, nil
	}

	// Forward all messages to the current view
	switch a.state {
	case viewList:
		a.list, cmd = a.list.Update(msg)
	case viewDetail:
		a.detail, cmd = a.detail.Update(msg)
	case viewDepNav:
		a.depNav, cmd = a.depNav.Update(msg)
	case viewTagPicker:
		a.tagPicker, cmd = a.tagPicker.Update(msg)
	case viewParentPicker:
		a.parentPicker, cmd = a.parentPicker.Update(msg)
	case viewStatusPicker:
		a.statusPicker, cmd = a.statusPicker.Update(msg)
	case viewTypePicker:
		a.typePicker, cmd = a.typePicker.Update(msg)
	case viewPriorityPicker:
		a.priorityPicker, cmd = a.priorityPicker.Update(msg)
	case viewSizePicker:
		a.sizePicker, cmd = a.sizePicker.Update(msg)
	case viewBlockingPicker:
		a.blockingPicker, cmd = a.blockingPicker.Update(msg)
	case viewTagEditor:
		a.tagEditor, cmd = a.tagEditor.Update(msg)
	case viewArchiveConfirm:
		a.archiveConfirm, cmd = a.archiveConfirm.Update(msg)
	case viewCreateModal:
		a.createModal, cmd = a.createModal.Update(msg)
	case viewHelpOverlay:
		a.helpOverlay, cmd = a.helpOverlay.Update(msg)
	case viewLauncherPicker:
		a.launcherPicker, cmd = a.launcherPicker.Update(msg)
	case viewLauncherError:
		// Any key dismisses error modal
		if _, ok := msg.(tea.KeyMsg); ok {
			a.state = a.previousState
			return a, nil
		}
	case viewNoLaunchers:
		// Any key dismisses no-launchers modal
		if _, ok := msg.(tea.KeyMsg); ok {
			a.state = a.previousState
			return a, nil
		}
	case viewNoLaunchersMulti:
		// Any key dismisses no multi-bean launchers modal
		if _, ok := msg.(tea.KeyMsg); ok {
			a.state = a.previousState
			return a, nil
		}
	case viewConfigureLaunchers:
		a.configureLaunchers, cmd = a.configureLaunchers.Update(msg)
	case viewConfigWriteError:
		// Any key dismisses error modal
		if _, ok := msg.(tea.KeyMsg); ok {
			a.state = a.previousState
			return a, nil
		}
	}

	return a, cmd
}

// browseBean replaces the detail view with the bean delta positions away from
// the current one in the list's visible order, moving the list cursor along.
// It does nothing at the ends of the list or when the bean isn't listed.
func (a *App) browseBean(delta int) {
	items := a.list.list.VisibleItems()
	for i, it := range items {
		if bi, ok := it.(beanItem); ok && bi.bean.ID == a.detail.bean.ID {
			j := i + delta
			if j < 0 || j >= len(items) {
				return
			}
			next, ok := items[j].(beanItem)
			if !ok {
				return
			}
			a.list.list.Select(j)
			a.detail = newDetailModel(next.bean, a.resolver, a.config, a.width, a.height)
			return
		}
	}
}

// isTwoColumnMode returns true if the terminal is wide enough for list + preview
func (a *App) isTwoColumnMode() bool {
	return a.width >= twoColumnMinWidth
}

// resizeList sizes the list for the full width, or the left pane in two-column
// mode, leaving room for the log panel when it's shown
func (a *App) resizeList() tea.Cmd {
	width := a.width
	if a.isTwoColumnMode() {
		width, _ = paneWidths(a.width)
	}
	height := a.height
	if a.logPanelShown() {
		height -= logPanelHeight
	}
	var cmd tea.Cmd
	a.list, cmd = a.list.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return cmd
}

// logPanelShown returns true if the log panel is toggled on and the terminal
// is tall enough to fit it below the list
func (a *App) logPanelShown() bool {
	return a.log.visible && a.height >= 3*logPanelHeight
}

// listView renders the list, with the preview pane beside it when there's room
// and the log panel below it when shown
func (a *App) listView() string {
	if a.list.err != nil || a.list.width == 0 {
		return a.list.View()
	}

	body := a.list.viewContent()
	if a.isTwoColumnMode() {
		_, previewWidth := paneWidths(a.width)
		a.preview.setBean(a.list.previewData())
		right := a.preview.View(previewWidth, lipgloss.Height(body))
		body = lipgloss.JoinHorizontal(lipgloss.Top, body, right)
	}
	if a.logPanelShown() {
		body += "\n" + a.log.View(a.width)
	}

	return body + "\n" + a.list.Footer()
}

// resolveChord tracks chord keys (see KeyMap). It returns pending=true while a
// chord is incomplete; otherwise the message to handle: the key itself, or a
// chordMsg once a chord is complete. An unknown chord is handled as a chordMsg
// too, which matches no binding.
func (a *App) resolveChord(msg tea.KeyMsg) (_ tea.Msg, pending bool) {
	bindings := a.chordBindings()
	seq := msg.String()
	if a.pendingKeys != "" {
		seq = a.pendingKeys + chordSep + seq
	}
	if isChordPrefix(seq, bindings) {
		a.pendingKeys = seq
		return nil, true
	}
	wasPending := a.pendingKeys != ""
	a.pendingKeys = ""
	if wasPending {
		return chordMsg(seq), false
	}
	return msg, false
}

// chordBindings returns the bindings whose chords can be typed in the current
// state. Chords are off while typing text (filters, text inputs).
func (a *App) chordBindings() []key.Binding {
	if a.typingText() {
		return nil
	}

	switch a.state {
	case viewList:
		return append(allBindings(keymap.Global), allBindings(keymap.List)...)
	case viewDetail:
		return append(allBindings(keymap.Global), allBindings(keymap.Detail)...)
	case viewDepNav:
		return append(allBindings(keymap.Global), allBindings(keymap.DepNav)...)
	case viewHelpOverlay:
		return allBindings(keymap.Help)
	case viewTagPicker, viewParentPicker, viewStatusPicker, viewTypePicker, viewBlockingPicker, viewPriorityPicker, viewSizePicker, viewLauncherPicker:
		return allBindings(keymap.Picker)
	case viewArchiveConfirm:
		return allBindings(keymap.ArchiveConfirm)
	case viewConfigureLaunchers:
		return allBindings(keymap.ConfigureLaunchers)
	}
	return nil
}

// typingText reports whether keys currently go to a text input.
func (a *App) typingText() bool {
	switch a.state {
	case viewCreateModal, viewTagEditor:
		return true
	case viewLauncherPicker:
		return a.launcherPicker.list.FilterState() == list.Filtering
	case viewArchiveConfirm, viewConfigureLaunchers:
		return false
	}
	return !a.canQuit()
}

// canQuit returns true if the quit key should quit the app in the current state.
// It must not quit while the user is typing text (filters, text inputs).
func (a *App) canQuit() bool {
	switch a.state {
	case viewList:
		return a.list.list.FilterState() != list.Filtering
	case viewDetail:
		return a.detail.linkList.FilterState() != list.Filtering
	case viewTagPicker:
		return a.tagPicker.list.FilterState() != list.Filtering
	case viewParentPicker:
		return a.parentPicker.list.FilterState() != list.Filtering
	case viewStatusPicker:
		return a.statusPicker.list.FilterState() != list.Filtering
	case viewTypePicker:
		return a.typePicker.list.FilterState() != list.Filtering
	case viewBlockingPicker:
		return a.blockingPicker.list.FilterState() != list.Filtering
	case viewPriorityPicker:
		return a.priorityPicker.list.FilterState() != list.Filtering
	case viewSizePicker:
		return a.sizePicker.list.FilterState() != list.Filtering
	case viewHelpOverlay, viewDepNav:
		return true
	default:
		// Text input modals (create, tag editor) and launcher dialogs
		return false
	}
}

// collectTagsWithCounts returns all tags with their usage counts
func (a *App) collectTagsWithCounts() []tagWithCount {
	beans, _ := a.resolver.Query().Beans(context.Background(), nil)
	tagCounts := make(map[string]int)
	for _, b := range beans {
		for _, tag := range b.Tags {
			tagCounts[tag]++
		}
	}

	tags := make([]tagWithCount, 0, len(tagCounts))
	for tag, count := range tagCounts {
		tags = append(tags, tagWithCount{tag: tag, count: count})
	}

	return tags
}

// View renders the current view
func (a *App) View() string {
	// Return empty when quitting to prevent View output from being printed
	// to the terminal after tea.ExecProcess runs (see bubbletea issue #431)
	if a.quitting {
		return ""
	}

	switch a.state {
	case viewList:
		return a.listView()
	case viewDetail:
		return a.detail.View()
	case viewDepNav:
		return a.depNav.View()
	case viewTagPicker:
		return a.tagPicker.View()
	case viewParentPicker:
		return a.parentPicker.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewStatusPicker:
		return a.statusPicker.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewTypePicker:
		return a.typePicker.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewPriorityPicker:
		return a.priorityPicker.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewSizePicker:
		return a.sizePicker.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewBlockingPicker:
		return a.blockingPicker.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewTagEditor:
		return a.tagEditor.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewArchiveConfirm:
		return a.archiveConfirm.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewCreateModal:
		return a.createModal.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewHelpOverlay:
		return a.helpOverlay.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewLauncherPicker:
		return a.launcherPicker.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewLauncherError:
		return a.launcherError.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewNoLaunchers:
		return a.noLaunchers.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewNoLaunchersMulti:
		return a.noLaunchersMulti.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewConfigureLaunchers:
		return a.configureLaunchers.ModalView(a.getBackgroundView(), a.width, a.height)
	case viewConfigWriteError:
		return a.configWriteError.ModalView(a.getBackgroundView(), a.width, a.height)
	}
	return ""
}

// getBackgroundView returns the view to show behind modal pickers
func (a *App) getBackgroundView() string {
	switch a.previousState {
	case viewDetail:
		return a.detail.View()
	default:
		return a.listView()
	}
}

// getEditor returns the user's preferred editor using the fallback chain:
// $VISUAL -> $EDITOR -> vi -> nano
// projectRoot returns the directory of the project's config file, falling back
// to the parent of the beans directory when no config file was loaded.
func (a *App) projectRoot() string {
	if dir := a.config.ConfigDir(); dir != "" {
		return dir
	}
	return filepath.Dir(a.core.Root())
}

// newLaunch describes running script (declared by the config file in
// configDir) for a bean of this project.
func (a *App) newLaunch(script, configDir, beanID, beanTitle string) launcherexec.Launch {
	return launcherexec.Launch{
		Script:      script,
		ProjectRoot: a.projectRoot(),
		ConfigDir:   configDir,
		BeansDir:    a.core.Root(),
		BeanID:      beanID,
		BeanTitle:   beanTitle,
	}
}

// launchFor describes running l for the bean with the given ID. fallbackTitle
// is used when the bean can't be loaded.
func (a *App) launchFor(l launcher, beanID, fallbackTitle string) launcherexec.Launch {
	title := fallbackTitle
	if b, _ := a.core.Get(beanID); b != nil {
		title = b.DisplayTitle()
	}
	return a.newLaunch(l.exec, l.dir, beanID, title)
}

// runLauncher runs l in the foreground (with the terminal handed over to it)
// and logs it in the log panel. Stderr is additionally captured to a log file
// so failures can be inspected afterwards.
func (a *App) runLauncher(l launcher, beanID, fallbackTitle string) tea.Cmd {
	cmd, logFile, err := launcherexec.CreateLoggedExecCommand(a.launchFor(l, beanID, fallbackTitle))
	if err != nil {
		return a.logStartFailure(l, beanID, err)
	}

	id := a.log.start(l.name, beanID, logFile.Name(), time.Now())
	a.quitting = true
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		logFile.Close()
		a.quitting = false
		return launchFinishedMsg{id: id, exitCode: launcherexec.ExitCode(err), err: err}
	})
}

// runLauncherInBackground runs l without the terminal and logs it in the log
// panel, with stdout and stderr captured for it.
func (a *App) runLauncherInBackground(l launcher, beanID, fallbackTitle string) tea.Cmd {
	cmd, logFile, err := launcherexec.CreateBackgroundCommand(a.launchFor(l, beanID, fallbackTitle))
	if err != nil {
		return a.logStartFailure(l, beanID, err)
	}

	id := a.log.start(l.name, beanID, logFile.Name(), time.Now())
	return func() tea.Msg {
		err := cmd.Run()
		logFile.Close()
		return launchFinishedMsg{id: id, exitCode: launcherexec.ExitCode(err), err: err}
	}
}

// logStartFailure logs a run whose command couldn't even be created.
func (a *App) logStartFailure(l launcher, beanID string, err error) tea.Cmd {
	id := a.log.start(l.name, beanID, "", time.Now())
	return func() tea.Msg {
		return launchFinishedMsg{id: id, exitCode: -1, err: err}
	}
}

// openLatestLog opens the latest run's full output in the user's editor.
func (a *App) openLatestLog() tea.Cmd {
	e := a.log.latest()
	if e == nil || e.logPath == "" {
		return nil
	}
	c := exec.Command(getEditor(), e.logPath)
	a.quitting = true
	return tea.ExecProcess(c, func(error) tea.Msg {
		a.quitting = false
		return nil
	})
}

func getEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	// Fallback chain: vi is more universal, nano as last resort
	if _, err := exec.LookPath("vi"); err == nil {
		return "vi"
	}
	return "nano"
}

// Run starts the TUI application with file watching
func Run(core *beancore.Core, cfg *config.Config) error {
	if _, _, err := NewKeyMap(cfg.Keybinding); err != nil {
		return fmt.Errorf("keybinding: %w", err)
	}
	// Must happen before the program takes over the terminal
	detectMarkdownStyle()

	app := New(core, cfg)
	p := tea.NewProgram(app, tea.WithAltScreen())

	// Store reference to program for sending messages from watcher
	app.program = p

	// Start file watching
	if err := core.Watch(func() {
		// Send message to TUI when beans change
		if app.program != nil {
			app.program.Send(beansChangedMsg{})
		}
	}); err != nil {
		return err
	}
	defer core.Unwatch()

	// Reload the config whenever a config file changes
	stopConfigWatch, err := config.Watch(app.projectRoot(), func() {
		app.program.Send(configChangedMsg{})
	})
	if err != nil {
		return err
	}
	defer stopConfigWatch()

	// Finished runs' output is only kept for this session
	defer app.log.cleanup()

	_, err = p.Run()
	return err
}
