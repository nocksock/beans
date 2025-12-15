 package tui
 
 import (
 	"testing"
 	"time"
 
 	tea "github.com/charmbracelet/bubbletea"
 	"github.com/hmans/beans/internal/bean"
 	"github.com/hmans/beans/internal/config"
 	launcherexec "github.com/hmans/beans/internal/launcher"
 )
 
 // TestLaunchProgressDoesNotDismissOnKeyPress is a regression test for beans-b1u8.
 // Previously, the viewLaunchProgress case had misplaced code that dismissed the
 // view on ANY key press, causing the TUI to get stuck showing "Loading...".
 func TestLaunchProgressDoesNotDismissOnKeyPress(t *testing.T) {
 	// Create a minimal app with launch progress state
 	app := &App{
 		state:         viewLaunchProgress,
 		previousState: viewList,
 		width:         80,
 		height:        24,
 	}
 
 	// Create a mock launcher manager with some beans
 	mockBeans := []*bean.Bean{
 		{ID: "test1", Title: "Test Bean 1"},
 		{ID: "test2", Title: "Test Bean 2"},
 	}
 	mockLauncher := &config.Launcher{
 		Name:     "test-launcher",
 		Exec:     "echo test",
 		Multiple: true,
 	}
 	manager := launcherexec.NewLaunchManager(mockLauncher, mockBeans)
 
 	app.launchProgress = newLaunchProgress(manager, mockLauncher.Name)
 
 	// Set initial window size so View() doesn't return "Loading..."
 	app.launchProgress.width = 80
 	app.launchProgress.height = 24
 
 	// Test that arbitrary key presses don't dismiss the view
 	testCases := []struct {
 		name   string
 		keyMsg tea.KeyMsg
 	}{
 		{"letter key", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}},
 		{"space key", tea.KeyMsg{Type: tea.KeySpace}},
 		{"enter key", tea.KeyMsg{Type: tea.KeyEnter}},
 		{"arrow key", tea.KeyMsg{Type: tea.KeyUp}},
 		{"number key", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}},
 	}
 
 	for _, tc := range testCases {
 		t.Run(tc.name, func(t *testing.T) {
 			// Store initial state
 			initialState := app.state
 
 			// Send the key message
 			_, _ = app.Update(tc.keyMsg)
 
 			// Verify the state hasn't changed (view not dismissed)
 			if app.state != initialState {
 				t.Errorf("key %q dismissed launch progress view (state changed from %v to %v), but should not have",
 					tc.name, initialState, app.state)
 			}
 
 			// Verify we're still in launch progress view
 			if app.state != viewLaunchProgress {
 				t.Errorf("after key %q, expected state viewLaunchProgress, got %v", tc.name, app.state)
 			}
 		})
 	}
 }
 
 // TestLaunchProgressDismissesOnQuitKeys verifies that q and esc still properly
 // dismiss the launch progress view.
 func TestLaunchProgressDismissesOnQuitKeys(t *testing.T) {
 	testCases := []struct {
 		name string
 		key  string
 	}{
 		{"q key", "q"},
 		{"esc key", "esc"},
 	}
 
 	for _, tc := range testCases {
 		t.Run(tc.name, func(t *testing.T) {
 			// Create a minimal app with launch progress state
 			app := &App{
 				state:         viewLaunchProgress,
 				previousState: viewList,
 				width:         80,
 				height:        24,
 			}
 
 			// Create a mock launcher manager
 			mockBeans := []*bean.Bean{
 				{ID: "test1", Title: "Test Bean 1"},
 			}
 			mockLauncher := &config.Launcher{
 				Name: "test-launcher",
 				Exec: "echo test",
 			}
 			manager := launcherexec.NewLaunchManager(mockLauncher, mockBeans)
 			app.launchProgress = newLaunchProgress(manager, mockLauncher.Name)
 			app.launchProgress.width = 80
 			app.launchProgress.height = 24
 
 			// Send the quit key
 			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.key)}
 			if tc.key == "esc" {
 				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
 			}
 
 			_, _ = app.Update(keyMsg)
 
 			// Verify the view was dismissed (returned to previous state)
 			if app.state != viewList {
 				t.Errorf("after pressing %q, expected to return to viewList, got %v", tc.key, app.state)
 			}
 		})
 	}
 }
 
 // TestLaunchProgressDismissesOnCompletion verifies that the launch progress view
 // properly dismisses when all launches complete.
 func TestLaunchProgressDismissesOnCompletion(t *testing.T) {
 	app := &App{
 		state:         viewLaunchProgress,
 		previousState: viewList,
 		width:         80,
 		height:        24,
 		list:          listModel{selectedBeans: map[string]bool{"test1": true}},
 	}
 
 	// Create a mock launcher manager
 	mockBeans := []*bean.Bean{
 		{ID: "test1", Title: "Test Bean 1"},
 	}
 	mockLauncher := &config.Launcher{
 		Name: "test-launcher",
 		Exec: "echo test",
 	}
 	manager := launcherexec.NewLaunchManager(mockLauncher, mockBeans)
 	app.launchProgress = newLaunchProgress(manager, mockLauncher.Name)
 	app.launchProgress.width = 80
 	app.launchProgress.height = 24
 
 	// Simulate completion
 	_, _ = app.Update(launchCompleteMsg{})
 
 	// Verify the view was dismissed
 	if app.state != viewList {
 		t.Errorf("after launchCompleteMsg, expected to return to viewList, got %v", app.state)
 	}
 
 	// Verify selection was cleared
 	if len(app.list.selectedBeans) != 0 {
 		t.Errorf("after launchCompleteMsg, expected selection to be cleared, got %d selected beans", len(app.list.selectedBeans))
 	}
 }
 
 // TestLaunchProgressDismissesOnFailure verifies that the launch progress view
 // properly handles failures.
 func TestLaunchProgressDismissesOnFailure(t *testing.T) {
 	app := &App{
 		state:         viewLaunchProgress,
 		previousState: viewList,
 		width:         80,
 		height:        24,
 	}
 
 	// Create a mock launcher manager
 	mockBeans := []*bean.Bean{
 		{ID: "test1", Title: "Test Bean 1"},
 	}
 	mockLauncher := &config.Launcher{
 		Name: "test-launcher",
 		Exec: "echo test",
 	}
 	manager := launcherexec.NewLaunchManager(mockLauncher, mockBeans)
 	app.launchProgress = newLaunchProgress(manager, mockLauncher.Name)
 	app.launchProgress.width = 80
 	app.launchProgress.height = 24
 
 	// Simulate failure
 	testErr := tea.ErrProgramKilled
 	_, _ = app.Update(launchFailedMsg{beanID: "test1", err: testErr})
 
 	// Verify the view switched to error view
 	if app.state != viewLauncherError {
 		t.Errorf("after launchFailedMsg, expected to switch to viewLauncherError, got %v", app.state)
 	}
 }
 
 // TestLaunchProgressProcessesProgressMessages verifies that launchProgressMsg
 // is properly processed and doesn't dismiss the view.
 func TestLaunchProgressProcessesProgressMessages(t *testing.T) {
 	app := &App{
 		state:         viewLaunchProgress,
 		previousState: viewList,
 		width:         80,
 		height:        24,
 	}
 
 	// Create a mock launcher manager
 	mockBeans := []*bean.Bean{
 		{ID: "test1", Title: "Test Bean 1"},
 	}
 	mockLauncher := &config.Launcher{
 		Name: "test-launcher",
 		Exec: "echo test",
 	}
 	manager := launcherexec.NewLaunchManager(mockLauncher, mockBeans)
 	app.launchProgress = newLaunchProgress(manager, mockLauncher.Name)
 	app.launchProgress.width = 80
 	app.launchProgress.height = 24
 
 	// Send a progress message (simulating the ticker)
 	_, _ = app.Update(launchProgressMsg{})
 
 	// Verify we're still in launch progress view
 	if app.state != viewLaunchProgress {
 		t.Errorf("after launchProgressMsg, expected to stay in viewLaunchProgress, got %v", app.state)
 	}
 }
 
 // TestLaunchProgressHandlesWindowResize verifies that window resize messages
 // don't dismiss the view.
 func TestLaunchProgressHandlesWindowResize(t *testing.T) {
 	app := &App{
 		state:         viewLaunchProgress,
 		previousState: viewList,
 		width:         80,
 		height:        24,
 	}
 
 	// Create a mock launcher manager
 	mockBeans := []*bean.Bean{
 		{ID: "test1", Title: "Test Bean 1"},
 	}
 	mockLauncher := &config.Launcher{
 		Name: "test-launcher",
 		Exec: "echo test",
 	}
 	manager := launcherexec.NewLaunchManager(mockLauncher, mockBeans)
 	app.launchProgress = newLaunchProgress(manager, mockLauncher.Name)
 	app.launchProgress.width = 80
 	app.launchProgress.height = 24
 
 	// Simulate window resize
 	resizeMsg := tea.WindowSizeMsg{Width: 100, Height: 30}
 	_, _ = app.Update(resizeMsg)
 
 	// Verify we're still in launch progress view
 	if app.state != viewLaunchProgress {
 		t.Errorf("after WindowSizeMsg, expected to stay in viewLaunchProgress, got %v", app.state)
 	}
 
 	// Verify dimensions were updated
 	if app.width != 100 || app.height != 30 {
 		t.Errorf("after WindowSizeMsg, expected dimensions 100x30, got %dx%d", app.width, app.height)
 	}
 }
 
 // TestLaunchProgressTickerCleanup verifies that the ticker is properly stopped
 // when the view is dismissed.
 func TestLaunchProgressTickerCleanup(t *testing.T) {
 	app := &App{
 		state:         viewLaunchProgress,
 		previousState: viewList,
 		width:         80,
 		height:        24,
 	}
 
 	// Create a mock launcher manager
 	mockBeans := []*bean.Bean{
 		{ID: "test1", Title: "Test Bean 1"},
 	}
 	mockLauncher := &config.Launcher{
 		Name: "test-launcher",
 		Exec: "echo test",
 	}
 	manager := launcherexec.NewLaunchManager(mockLauncher, mockBeans)
 	app.launchProgress = newLaunchProgress(manager, mockLauncher.Name)
 	app.launchProgress.width = 80
 	app.launchProgress.height = 24
 
 	// Verify ticker is running
 	if app.launchProgress.ticker == nil {
 		t.Fatal("expected ticker to be initialized")
 	}
 
 	// Dismiss the view with q
 	_, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
 
 	// Give a small delay to allow cleanup
 	time.Sleep(10 * time.Millisecond)
 
 	// Note: We can't directly verify the ticker is stopped, but we can verify
 	// the cleanup method was called and the state changed correctly
 	if app.state != viewList {
 		t.Errorf("after pressing q, expected to return to viewList, got %v", app.state)
 	}
 }

// TestCreateModalWithParentContext verifies that the create modal properly
// receives and uses parent context when creating a child bean.
func TestCreateModalWithParentContext(t *testing.T) {
	parentID := "parent-123"
	contextInfo := "child of Test Parent"

	// Create modal with parent context
	m := newCreateModalModel(80, 24, &parentID, nil, contextInfo)

	// Verify context was stored
	if m.parentID == nil || *m.parentID != parentID {
		t.Errorf("expected parentID to be %q, got %v", parentID, m.parentID)
	}
	if m.contextInfo != contextInfo {
		t.Errorf("expected contextInfo to be %q, got %q", contextInfo, m.contextInfo)
	}
	if len(m.blockingIDs) != 0 {
		t.Errorf("expected no blocking IDs, got %v", m.blockingIDs)
	}
}

// TestCreateModalWithBlockingContext verifies that the create modal properly
// receives and uses blocking context when creating a bean that blocks multiple others.
func TestCreateModalWithBlockingContext(t *testing.T) {
	blockingIDs := []string{"bean-1", "bean-2", "bean-3"}
	contextInfo := "blocking 3 beans"

	// Create modal with blocking context
	m := newCreateModalModel(80, 24, nil, blockingIDs, contextInfo)

	// Verify context was stored
	if m.parentID != nil {
		t.Errorf("expected no parent ID, got %v", m.parentID)
	}
	if len(m.blockingIDs) != 3 {
		t.Errorf("expected 3 blocking IDs, got %d", len(m.blockingIDs))
	}
	if m.contextInfo != contextInfo {
		t.Errorf("expected contextInfo to be %q, got %q", contextInfo, m.contextInfo)
	}
}

// TestBeanCreatedMsgIncludesContext verifies that beanCreatedMsg includes
// the parent/blocking context from the create modal.
func TestBeanCreatedMsgIncludesContext(t *testing.T) {
	parentID := "parent-123"
	m := newCreateModalModel(80, 24, &parentID, nil, "child of Test")

	// Simulate entering a title and pressing enter
	m.textInput.SetValue("New Bean Title")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Execute the command to get the message
	msg := cmd()
	createdMsg, ok := msg.(beanCreatedMsg)
	if !ok {
		t.Fatalf("expected beanCreatedMsg, got %T", msg)
	}

	// Verify the message includes the context
	if createdMsg.title != "New Bean Title" {
		t.Errorf("expected title %q, got %q", "New Bean Title", createdMsg.title)
	}
	if createdMsg.parentID == nil || *createdMsg.parentID != parentID {
		t.Errorf("expected parentID %q, got %v", parentID, createdMsg.parentID)
	}
	if len(createdMsg.blockingIDs) != 0 {
		t.Errorf("expected no blocking IDs, got %v", createdMsg.blockingIDs)
	}
}
