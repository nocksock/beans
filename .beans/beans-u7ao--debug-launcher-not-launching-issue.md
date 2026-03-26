---
# beans-u7ao
title: Debug launcher not launching issue
status: in-progress
type: bug
priority: normal
created_at: 2026-03-26T08:51:48Z
updated_at: 2026-03-26T08:52:23Z
---

## Context

When pressing `!` to launch a bean, nothing happens. No picker opens, no error shown, no launch occurs.

## Debug Plan

### Hypothesis

The issue is likely in one of these areas:
1. **Message routing** - `launcherSelectedMsg` not reaching the handler
2. **Session manager initialization** - abduco path not found
3. **State transitions** - view not switching properly

### Proposed Tests

```go
// Test: Verify openLauncherPickerMsg transitions state
func TestLauncherPickerOpens(t *testing.T) {
    app := &App{
        state:      viewList,
        sessionMgr: session.NewSessionManager(),
        config: &config.Config{
            Launchers: []config.Launcher{
                {Name: "test", Exec: "echo test"},
            },
        },
        // ... other fields
    }
    
    // Send openLauncherPickerMsg
    app.Update(openLauncherPickerMsg{beanIDs: []string{"test"}, beanTitle: "Test"})
    
    // Should be in launcher picker state
    if app.state != viewLauncherPicker {
        t.Errorf("expected viewLauncherPicker, got %v", app.state)
    }
}

// Test: Verify launcherSelectedMsg for non-session launcher
func TestLauncherSelectionLaunches(t *testing.T) {
    app := &App{
        state:      viewLauncherPicker,
        previousState: viewList,
        sessionMgr: session.NewSessionManager(),
        config: &config.Config{/* ... */},
    }
    
    // Send launcherSelectedMsg
    app.Update(launcherSelectedMsg{
        launcher: launcher{name: "test", exec: "echo hello"},
        beanIDs: []string{"test1"},
    })
    
    // Should start launch or show progress
}
```

### Implementation Steps

1. **Run existing tests** - `go test ./internal/tui/...` to see if any fail
2. **Add integration test** for launcher flow in tui_test.go
3. **Add nil checks** where needed in message handlers
4. **Verify session manager initialization** works correctly
5. **Build and test manually** with `go run . --beans-path /tmp/test-beans`

## Related

Parent: beans-e3dd (Refactor session management to use abduco)

## Checklist

- [ ] Run existing TUI tests
- [ ] Add unit test for openLauncherPickerMsg handler
- [ ] Add unit test for launcherSelectedMsg handler
- [ ] Fix any failing tests
- [ ] Manual test with session launcher
- [ ] Manual test with non-session launcher
