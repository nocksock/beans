package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph"
	"github.com/hmans/beans/internal/graph/model"
	launcherexec "github.com/hmans/beans/internal/launcher"
	"github.com/hmans/beans/internal/tui/layout"
)

// viewState represents which view is currently active
type viewState int

const (
	viewList viewState = iota // the panes, with no modal open
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
)

// beansChangedMsg is sent when beans change on disk (via file watcher)
type beansChangedMsg struct{}

// openTagPickerMsg requests opening the tag picker
type openTagPickerMsg struct{}

// tagSelectedMsg is sent when a tag is selected from the picker
type tagSelectedMsg struct {
	tag string
}

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
	ws                 workspace // pane layout and focus
	panes              map[layout.ID]pane
	list               listModel
	ctx                contextModel
	logDetail          logDetailModel
	ctxPane            contextPane // shows ctx or logDetail
	deps               depNavModel
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
	keybinds           keybindsModel
	launcherPicker     launcherPickerModel
	launcherError      launcherErrorModel
	noLaunchers        noLaunchersModel
	noLaunchersMulti   noLaunchersMultiModel
	configureLaunchers configureLaunchersModel
	configWriteError   configWriteErrorModel
	log                logPanelModel
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

	// depsShown is whether the dependency graph was visible after the last
	// update, to root it at the context's bean when it appears
	depsShown bool

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
	a := &App{
		state:       viewList,
		ws:          newWorkspace(),
		core:        core,
		resolver:    resolver,
		config:      cfg,
		list:        newListModel(resolver, cfg),
		ctx:         newContextModel(cfg),
		deps:        newDepNavModel(resolver, cfg),
		keyWarnings: warnings,
	}
	a.ctxPane = contextPane{bean: &a.ctx, run: &a.logDetail}
	a.panes = a.newPanes()
	return a
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	if len(a.keyWarnings) > 0 {
		return tea.Batch(a.list.Init(), a.showNotice(keyWarningNotice(a.keyWarnings), true))
	}
	return a.list.Init()
}

// Update handles messages, then fits the panes to the resulting layout
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := a.update(msg)
	a.syncPanes()
	return a, cmd
}

func (a *App) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		var pending bool
		if msg, pending = a.resolveChord(keyMsg); pending {
			return nil
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Panes are sized by syncPanes; modals get the full size
		if a.state == viewList {
			return nil
		}

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keymap.Global.ForceQuit):
			return tea.Quit
		case key.Matches(msg, keymap.Global.Help):
			// Open the keybinds dialog from a pane (not in a picker/modal)
			if a.state == viewList && !a.typingText() {
				a.previousState = a.state
				a.keybinds = newKeybindsModel(a.keybindSections(), a.width, a.height)
				a.state = viewHelpOverlay
				return a.keybinds.Init()
			}
		case key.Matches(msg, keymap.Global.Quit):
			if a.canQuit() {
				return tea.Quit
			}
		case key.Matches(msg, keymap.Global.ReloadConfig):
			if a.state == viewList && a.canQuit() {
				return func() tea.Msg { return reloadConfigMsg{} }
			}
		case key.Matches(msg, keymap.Global.OpenLog):
			if a.state == viewList && a.canQuit() {
				if e := a.log.latest(); e != nil {
					return a.openLog(e.logPath)
				}
			}
		}
		if a.state == viewList && !a.typingText() && a.handleLayoutKey(msg) {
			return nil
		}

	case beansLoadedMsg:
		// Always for the list, even with a modal open
		a.list, cmd = a.list.Update(msg)
		a.refreshPanes()
		return cmd

	case beansChangedMsg:
		// Beans changed on disk - refresh
		// Trigger list refresh
		return a.list.loadBeans

	case configChangedMsg, reloadConfigMsg:
		return a.reloadConfig()

	case clearNoticeMsg:
		if msg.seq == a.noticeSeq {
			a.list.notice = ""
		}
		return nil

	case openTagPickerMsg:
		// Collect all tags with their counts
		tags := a.collectTagsWithCounts()
		if len(tags) == 0 {
			// No tags in system, don't open picker
			return nil
		}
		a.tagPicker = newTagPickerModel(tags, a.width, a.height)
		a.state = viewTagPicker
		return a.tagPicker.Init()

	case tagSelectedMsg:
		a.state = viewList
		a.list.setTagFilter(msg.tag)
		return a.list.loadBeans

	case openParentPickerMsg:
		// Check if all bean types can have parents
		for _, beanType := range msg.beanTypes {
			if beancore.ValidParentTypes(beanType) == nil {
				// At least one bean type (e.g., milestone) cannot have parents - don't open the picker
				return nil
			}
		}
		a.previousState = a.state // Remember where we came from for the modal background
		a.parentPicker = newParentPickerModel(msg.beanIDs, msg.beanTitle, msg.beanTypes, msg.currentParent, a.resolver, a.config, a.width, a.height)
		a.state = viewParentPicker
		return a.parentPicker.Init()

	case closeParentPickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a.list.loadBeans

	case openStatusPickerMsg:
		a.previousState = a.state
		a.statusPicker = newStatusPickerModel(msg.beanIDs, msg.beanTitle, msg.currentStatus, a.config, a.width, a.height)
		a.state = viewStatusPicker
		return a.statusPicker.Init()

	case closeStatusPickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a.list.loadBeans

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
		return a.list.loadBeans

	case openTypePickerMsg:
		a.previousState = a.state
		a.typePicker = newTypePickerModel(msg.beanIDs, msg.beanTitle, msg.currentType, a.config, a.width, a.height)
		a.state = viewTypePicker
		return a.typePicker.Init()

	case closeTypePickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a.list.loadBeans

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
		return a.list.loadBeans

	case openSizePickerMsg:
		a.previousState = a.state
		a.sizePicker = newSizePickerModel(msg.beanIDs, msg.beanTitle, msg.currentSize, a.width, a.height)
		a.state = viewSizePicker
		return a.sizePicker.Init()

	case closeSizePickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a.list.loadBeans

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
		return a.list.loadBeans

	case openPriorityPickerMsg:
		a.previousState = a.state
		a.priorityPicker = newPriorityPickerModel(msg.beanIDs, msg.beanTitle, msg.currentPriority, a.config, a.width, a.height)
		a.state = viewPriorityPicker
		return a.priorityPicker.Init()

	case closePriorityPickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a.list.loadBeans

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
		return a.list.loadBeans

	case closeHelpMsg:
		a.state = a.previousState
		return nil

	case runKeybindMsg:
		// Replay the chosen key in the pane the dialog was opened from. A
		// chord replays whole, like a completed chord.
		a.state = a.previousState
		return a.update(chordMsg(msg.key))

	case openTagEditorMsg:
		var beans []*bean.Bean
		for _, id := range msg.beanIDs {
			if b, err := a.resolver.Query().Bean(context.Background(), id); err == nil && b != nil {
				beans = append(beans, b)
			}
		}
		if len(beans) == 0 {
			return nil
		}
		a.previousState = a.state
		a.tagEditor = newTagEditorModel(beans, msg.beanTitle, a.collectTagsWithCounts(), a.width, a.height)
		a.state = viewTagEditor
		return a.tagEditor.Init()

	case closeTagEditorMsg:
		a.state = a.previousState
		return a.list.loadBeans

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
		return a.list.loadBeans

	case openBlockingPickerMsg:
		a.previousState = a.state
		a.blockingPicker = newBlockingPickerModel(msg.beanID, msg.beanTitle, msg.blockedBy, msg.currentIDs, a.resolver, a.config, a.width, a.height)
		a.state = viewBlockingPicker
		return a.blockingPicker.Init()

	case closeBlockingPickerMsg:
		// Return to previous view and refresh in case beans changed while picker was open
		a.state = a.previousState
		return a.list.loadBeans

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
		return a.list.loadBeans

	case archiveCompletedMsg:
		var ids []string
		beans, _ := a.resolver.Query().Beans(context.Background(), nil)
		for _, b := range beans {
			if a.config.IsArchivable(b.Status) {
				ids = append(ids, b.ID)
			}
		}
		if len(ids) == 0 {
			return nil
		}
		a.previousState = a.state
		a.archiveConfirm = newArchiveConfirmModel(ids,
			fmt.Sprintf("Archive %d bean(s)?", len(ids)),
			"All completed and scrapped beans will be moved to archived.",
			a.width, a.height)
		a.state = viewArchiveConfirm
		return nil

	case archiveBeanMsg:
		b, err := a.resolver.Query().Bean(context.Background(), msg.beanID)
		if err != nil || b == nil {
			return nil
		}
		if a.config.IsArchived(b.Status) {
			return nil
		}
		if a.config.IsArchiveStatus(b.Status) {
			return func() tea.Msg { return archiveConfirmedMsg{beanIDs: []string{b.ID}} }
		}
		a.previousState = a.state
		a.archiveConfirm = newArchiveConfirmModel([]string{b.ID},
			"Archive unfinished bean?",
			fmt.Sprintf("%s is %q, not completed. It will be archived anyway.", b.ID, b.Status),
			a.width, a.height)
		a.state = viewArchiveConfirm
		return nil

	case closeArchiveConfirmMsg:
		a.state = a.previousState
		return nil

	case archiveConfirmedMsg:
		status := config.StatusArchived
		for _, id := range msg.beanIDs {
			if _, err := a.resolver.Mutation().UpdateBean(context.Background(), id, model.UpdateBeanInput{
				Status: &status,
			}); err != nil {
				a.previousState = viewList
				a.launcherError = newLauncherErrorModel("Archive", err, a.width, a.height)
				a.state = viewLauncherError
				return a.list.loadBeans
			}
		}
		a.state = viewList
		return a.list.loadBeans

	case openCreateModalMsg:
		a.previousState = a.state
		a.createModal = newCreateModalModel(a.config, a.width, a.height, msg.parentID, msg.parentTitle).withPrefill(msg.prefill)
		a.createModal.prefixes = a.config.AvailablePrefixes()
		a.state = viewCreateModal
		return a.createModal.Init()

	case closeCreateModalMsg:
		a.state = a.previousState
		return nil

	case openEditModalMsg:
		a.previousState = a.state
		a.createModal = newEditModalModel(a.config, a.width, a.height, msg.bean)
		a.state = viewCreateModal
		return a.createModal.Init()

	case beanEditedMsg:
		// Save only the changed fields; an unchanged form doesn't touch the bean
		a.state = a.previousState
		input := msg.input
		if input.Title != nil || input.Body != nil || input.Type != nil || input.Status != nil || input.Priority != nil {
			if _, err := a.resolver.Mutation().UpdateBean(context.Background(), msg.beanID, input); err != nil {
				a.launcherError = newLauncherErrorModel("Edit Bean", err, a.width, a.height)
				a.state = viewLauncherError
				return nil
			}
		}
		updatedBean, _ := a.resolver.Query().Bean(context.Background(), msg.beanID)
		if updatedBean == nil {
			return a.list.loadBeans
		}
		a.list.cursorTarget = updatedBean.ID
		if !msg.openEditor {
			return a.list.loadBeans
		}
		return tea.Batch(
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
		input.Tags = msg.tags
		if msg.parentID != "" {
			input.Parent = &msg.parentID
		}
		input.Prefix = &msg.prefix
		createdBean, err := a.resolver.Mutation().CreateBean(context.Background(), input)
		if err != nil {
			// Show error to user
			a.launcherError = newLauncherErrorModel("Create Bean", err, a.width, a.height)
			a.state = viewLauncherError
			return nil
		}
		// Return to list, move the cursor to the new bean, and optionally open it in the editor
		a.state = viewList
		a.list.cursorTarget = createdBean.ID
		if !msg.openEditor {
			return a.list.loadBeans
		}
		return tea.Batch(
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
		return tea.ExecProcess(c, func(err error) tea.Msg {
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
			return a.list.loadBeans
		}
		return nil

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
		return a.list.loadBeans

	case openLauncherPickerMsg:
		// Discover available launchers based on bean count
		var launchers []launcher

		if len(msg.beanIDs) > 1 {
			// Multiple beans - only show launchers with multiple: true
			launchers = discoverLaunchersForMultiple(a.config)
		} else {
			// Single bean - show all launchers
			launchers = discoverLaunchers(a.config)
		}

		if len(launchers) == 0 {
			// Check if this is multi-bean with no multiple-capable launchers
			if len(msg.beanIDs) > 1 {
				// Show specific error for multi-bean
				a.previousState = a.state
				a.noLaunchersMulti = newNoLaunchersMultiModel(a.width, a.height)
				a.state = viewNoLaunchersMulti
				return nil
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
				return a.configureLaunchers.Init()
			}

			// Launchers configured but none available
			a.previousState = a.state
			a.noLaunchers = newNoLaunchersModel(a.width, a.height)
			a.state = viewNoLaunchers
			return nil
		}

		// Open launcher picker
		a.previousState = a.state
		a.launcherPicker = newLauncherPickerModel(launchers, msg.beanIDs, msg.beanTitle, a.width, a.height)
		a.state = viewLauncherPicker
		return a.launcherPicker.Init()

	case launcherSelectedMsg:
		// Close the launcher picker first
		a.state = a.previousState
		// Clear selection so the next launch targets the cursor again
		clear(a.list.selectedBeans)

		if len(msg.beanIDs) == 0 {
			return nil
		}

		if !msg.launcher.background {
			// Foreground launchers get the terminal, so only one bean can run
			return a.runLauncher(msg.launcher, msg.beanIDs[0], msg.beanTitle)
		}

		// Background launchers run for every bean in parallel
		cmds := make([]tea.Cmd, 0, len(msg.beanIDs)+1)
		for _, id := range msg.beanIDs {
			cmds = append(cmds, a.runLauncherInBackground(msg.launcher, id, msg.beanTitle))
		}
		return tea.Batch(cmds...)

	case closeLauncherPickerMsg:
		a.state = a.previousState
		return nil

	case launchFinishedMsg:
		a.log.finish(msg.id, msg.exitCode, msg.err, time.Now())
		return nil

	case launchersConfiguredMsg:
		// Append selected launchers to .beans.yml
		projectRoot := a.projectRoot()

		if err := appendLaunchersToConfig(projectRoot, msg.launchers); err != nil {
			// Show error modal
			a.previousState = viewList
			a.configWriteError = newConfigWriteErrorModel(err, a.width, a.height)
			a.state = viewConfigWriteError
			return nil
		}

		// Return to the previous state, then reload config to pick
		// up the new launchers
		a.state = a.previousState
		return a.reloadConfig()

	case closeConfigureLaunchersMsg:
		// User cancelled - return to previous state
		a.state = a.previousState
		return nil

	case closeTagPickerMsg:
		a.state = viewList
		return nil

	case selectBeanMsg:
		// Show the bean in the context pane and move there
		a.showInContext(msg.bean)
		a.ws.focusPane(paneContext)
		return nil

	case focusBackMsg:
		a.ws.back()
		return nil

	case titleScrollMsg:
		// Even behind a modal, so the scroll doesn't stall
		a.list, cmd = a.list.Update(msg)
		return cmd

	case focusContextMsg:
		a.ws.focusPane(paneContext)
		return nil

	case openLogMsg:
		return a.openLog(msg.path)
	}

	// Forward all messages to the current view
	switch a.state {
	case viewList:
		cmd = a.updatePanes(msg)
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
		a.keybinds, cmd = a.keybinds.Update(msg)
	case viewLauncherPicker:
		a.launcherPicker, cmd = a.launcherPicker.Update(msg)
	case viewLauncherError:
		// Any key dismisses error modal
		if _, ok := msg.(tea.KeyMsg); ok {
			a.state = a.previousState
			return nil
		}
	case viewNoLaunchers:
		// Any key dismisses no-launchers modal
		if _, ok := msg.(tea.KeyMsg); ok {
			a.state = a.previousState
			return nil
		}
	case viewNoLaunchersMulti:
		// Any key dismisses no multi-bean launchers modal
		if _, ok := msg.(tea.KeyMsg); ok {
			a.state = a.previousState
			return nil
		}
	case viewConfigureLaunchers:
		a.configureLaunchers, cmd = a.configureLaunchers.Update(msg)
	case viewConfigWriteError:
		// Any key dismisses error modal
		if _, ok := msg.(tea.KeyMsg); ok {
			a.state = a.previousState
			return nil
		}
	}

	return cmd
}

// handleLayoutKey handles the keys arranging the panes. It returns false for
// other keys.
func (a *App) handleLayoutKey(msg tea.KeyMsg) bool {
	k := keymap.Layout
	switch {
	case key.Matches(msg, k.NextPreset):
		a.ws.cyclePreset(1)
	case key.Matches(msg, k.PrevPreset):
		a.ws.cyclePreset(-1)
	case key.Matches(msg, k.FocusNext):
		a.ws.cycleFocus(1)
	case key.Matches(msg, k.FocusPrev):
		a.ws.cycleFocus(-1)
	case key.Matches(msg, k.ToggleContext):
		a.ws.toggle(paneContext)
	case key.Matches(msg, k.ToggleDepGraph):
		a.ws.toggle(paneDepGraph)
	case key.Matches(msg, k.ToggleLog):
		a.ws.toggle(paneLog)
	default:
		for id, b := range paneKeys() {
			if key.Matches(msg, b) {
				a.ws.focusPane(id)
				return true
			}
		}
		return false
	}
	return true
}

// updatePanes sends keys to the focused pane. Other messages (e.g. the list's
// filter results) go to the list, the only pane that has async work.
func (a *App) updatePanes(msg tea.Msg) tea.Cmd {
	if _, ok := msg.(tea.KeyMsg); ok {
		return a.focusedPane().update(msg)
	}
	return a.panes[paneList].update(msg)
}

func (a *App) focusedPane() pane {
	return a.panes[a.ws.focus]
}

// syncPanes brings the panes in line with the layout after an update: it
// shows the log when a run asks for it, sizes the panes to their areas, roots
// a newly shown dependency graph at the context's bean, and points the
// context at the focused list or dependency graph's bean, or the log's run.
func (a *App) syncPanes() {
	if a.log.takeReveal() {
		a.ws.show(paneLog)
	}

	// Size first, so the panes below scroll for their actual height
	for _, r := range a.ws.arrange(a.width, a.panesHeight()).Rects() {
		a.panes[r.ID].setSize(max(r.W-2, 0), max(r.H-2, 0)) // inside the frame
	}

	depsShown := a.ws.isVisible(paneDepGraph)
	if depsShown && !a.depsShown && a.ctx.bean != nil {
		a.deps.focus(a.ctx.bean)
	}
	a.depsShown = depsShown

	switch source := a.focusedPane().(type) {
	case selectionSource:
		a.showInContext(source.selection())
	case runSource:
		a.ctxPane.showRun = true
		a.logDetail.setEntry(source.selectedRun())
	default:
		if a.ctxPane.showRun && a.logDetail.entry != nil {
			// Keep up with the shown run's output
			a.logDetail.setEntry(a.log.entry(a.logDetail.entry.id))
		}
	}
}

// refreshPanes re-reads the beans shown in the context and dependency graph
// after the beans were reloaded.
func (a *App) refreshPanes() {
	if b := a.ctx.bean; b != nil {
		a.setContextBean(a.list.beansByID[b.ID])
	}
	if a.deps.root != nil && !a.deps.reload() {
		// The root is gone; start over from the context's bean
		a.deps.root = nil
		if a.ctx.bean != nil {
			a.deps.focus(a.ctx.bean)
		}
	}
}

// showInContext shows b (or nothing, if nil) in the context pane.
func (a *App) showInContext(b *bean.Bean) {
	a.ctxPane.showRun = false
	a.setContextBean(b)
}

// setContextBean sets the bean of the context pane, whether it shows it or a run.
func (a *App) setContextBean(b *bean.Bean) {
	if b == nil {
		a.ctx.setBean(nil, previewContext{})
		return
	}
	a.ctx.setBean(b, a.list.previewContextFor(b))
}

// panesHeight is the height of the panes, above the footer.
func (a *App) panesHeight() int {
	return max(a.height-1, 0)
}

// mainView renders the panes and the footer.
func (a *App) mainView() string {
	if a.list.err != nil {
		return a.list.View()
	}
	if a.width == 0 {
		return "Loading..."
	}
	panes := a.ws.arrange(a.width, a.panesHeight()).Render(a.drawPane)
	return panes + "\n" + a.footer()
}

// drawPane renders a pane in its frame, highlighted when focused.
func (a *App) drawPane(r layout.Rect) string {
	p := a.panes[r.ID]
	border := paneBorderStyle
	if r.ID == a.ws.focus {
		border = focusedPaneBorderStyle
	}
	title := p.title()
	if k := primaryKey(paneKeys()[r.ID]); k != "" {
		title = "[" + k + "] " + title
	}
	return layout.Frame(title, p.View(), r.W, r.H, border)
}

// keybindSections are the sections of the keybinds dialog: the global and
// layout bindings, then the focused pane's.
func (a *App) keybindSections() []keybindSection {
	p := a.focusedPane()
	return []keybindSection{
		{"Global", allBindings(keymap.Global)},
		{"Layout", allBindings(keymap.Layout)},
		{p.name(), p.bindings()},
	}
}

// footer renders the notice, the layout hints and the focused pane's hints,
// or just the pane's text input while typing, cut to the screen width.
func (a *App) footer() string {
	if a.focusedPane().typing() {
		// The pane's text input (e.g. the list's filter query) takes the line
		return ansi.Truncate(a.focusedPane().footer(), a.width, "…")
	}
	g, k := keymap.Global, keymap.Layout
	line := a.list.noticeView() + hints(
		groupHint("layout", k.NextPreset, k.PrevPreset),
		hint(k.FocusNext, "pane"),
		hint(k.ToggleContext, "details"),
		hint(k.ToggleDepGraph, "dependencies"),
		hint(k.ToggleLog, "log"),
		hint(g.Help, ""),
		hint(g.Quit, ""),
		helpStyle.Render("│")+"  "+a.focusedPane().footer(),
	)
	return ansi.Truncate(line, a.width, "…")
}

// resolveChord tracks chord keys (see KeyMap). It returns pending=true while a
// chord is incomplete, or when it completes as an unknown chord, which is
// dropped; otherwise the message to handle: the key itself, or a chordMsg
// once a chord is complete.
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
		chord := chordMsg(seq)
		return chord, !slices.ContainsFunc(bindings, func(b key.Binding) bool { return key.Matches(chord, b) })
	}
	return msg, false
}

// chordBindings returns the bindings whose chords can be typed in the current
// state. Chords are off while typing text (filters, text inputs), except in
// the bean form for chords that start with a non-printable key (ctrl+g t).
func (a *App) chordBindings() []key.Binding {
	if a.state == viewCreateModal {
		return unprintableChords(allBindings(keymap.CreateModal))
	}
	if a.typingText() {
		return nil
	}

	switch a.state {
	case viewList:
		shared := append(allBindings(keymap.Global), allBindings(keymap.Layout)...)
		return append(shared, a.focusedPane().bindings()...)
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
	case viewHelpOverlay:
		return a.keybinds.typing()
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
		return !a.focusedPane().typing()
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
	case viewHelpOverlay:
		return !a.keybinds.typing()
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
		return a.mainView()
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
		return a.keybinds.ModalView(a.getBackgroundView(), a.width, a.height)
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
	return a.mainView()
}

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

// openLog opens a run's output in the user's editor.
func (a *App) openLog(path string) tea.Cmd {
	if path == "" {
		return nil
	}
	c := exec.Command(getEditor(), path)
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
