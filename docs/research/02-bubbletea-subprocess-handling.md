# Bubble Tea Subprocess Handling

## Overview

Bubble Tea has built-in support for spawning interactive subprocesses, but **does not provide session persistence** (detach/attach functionality).

## Built-in Capabilities

### 1. ExecProcess - Interactive Subprocess Execution

**Purpose**: Launch interactive programs (like `$EDITOR`) that need full terminal control.

**How it works**:
```go
func openEditor() tea.Cmd {
    editor := os.Getenv("EDITOR")
    if editor == "" {
        editor = "vim"
    }
    c := exec.Command(editor)
    return tea.ExecProcess(c, func(err error) tea.Msg {
        return editorFinishedMsg{err}
    })
}
```

**Process flow**:
1. `ReleaseTerminal()` - Releases stdin/stdout control to subprocess
2. Subprocess runs with full terminal access
3. `RestoreTerminal()` - Recaptures terminal control when subprocess exits
4. Callback message sent to Update() function

### 2. Example from Official Repository

```go
// From: github.com/charmbracelet/bubbletea/examples/exec/main.go

type editorFinishedMsg struct{ err error }

func openEditor() tea.Cmd {
    editor := os.Getenv("EDITOR")
    if editor == "" {
        editor = "vim"
    }
    c := exec.Command(editor)
    return tea.ExecProcess(c, func(err error) tea.Msg {
        return editorFinishedMsg{err}
    })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "e":
            return m, openEditor()
        }
    case editorFinishedMsg:
        if msg.err != nil {
            m.err = msg.err
            return m, tea.Quit
        }
    }
    return m, nil
}
```

## Core API Reference

### tea.ExecProcess

```go
func ExecProcess(c *exec.Cmd, fn ExecCallback) Cmd
```

**Parameters**:
- `c *exec.Cmd` - The command to execute
- `fn ExecCallback` - Callback function called when process exits

**Returns**: `tea.Cmd` that can be returned from Update()

### tea.ExecCallback

```go
type ExecCallback func(error) Msg
```

Called when the subprocess exits, with any error that occurred.

## Limitations for Our Use Case

### ❌ No Session Persistence
- Process exits when Bubble Tea app exits
- Cannot detach and leave process running
- Cannot reattach to a running process

### ❌ No Background Execution
- Process blocks the TUI while running
- Cannot start multiple processes simultaneously
- Cannot switch between processes

### ✅ Good for One-shot Commands
- Perfect for launching `$EDITOR`
- Good for interactive commands that complete and return
- Handles terminal state correctly

## What We Need Beyond Bubble Tea

For our use case (detachable/attachable build processes), we need:

1. **Session Management** - Process survives TUI detachment
2. **Multiple Sessions** - Run multiple builds simultaneously
3. **Reattachment** - Reconnect to running process
4. **Output Buffering** - See historical output when reattaching

**Conclusion**: Bubble Tea provides the foundation for interactive subprocess execution, but we need additional session management on top (abduco, tmux, or custom implementation).

## Integration Pattern with Session Manager

```go
// Start process in detached session
func (m model) startBuild() tea.Cmd {
    return func() tea.Msg {
        // Use session manager (abduco/tmux/custom)
        sessionMgr.Start("bean-123", "make build")
        return buildStartedMsg{id: "bean-123"}
    }
}

// Attach to running session
func (m model) attachToBuild(id string) tea.Cmd {
    // Use tea.ExecProcess with session manager's attach command
    return tea.ExecProcess(
        sessionMgr.AttachCommand(id),
        func(err error) tea.Msg {
            return buildDetachedMsg{id: id, err: err}
        },
    )
}
```

---

**Previous**: [← Overview](./01-subprocess-management-overview.md)  
**Next**: [abduco Analysis →](./03-abduco-detailed-analysis.md)
