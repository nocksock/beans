# ADR-004: Launcher Integration

**Status**: Proposed  
**Date**: 2024-12-15  
**Deciders**: beans project team  
**Technical Story**: How session management integrates with existing launcher system

## Context

beans already has a launcher system that executes commands for beans. The launcher system:
- Executes shell scripts defined in `.beans.yml`
- Supports single or multiple bean execution
- Captures output and status
- Runs in foreground (blocking)

Session management adds the ability to:
- Run commands in background (detached)
- Attach/detach interactively
- Keep processes running after TUI exits

We need to decide how these two systems relate.

## Decision

Session management will **integrate with launchers** via an optional `session: bool` field.

### Configuration

```yaml
launchers:
  - name: build
    exec: "make build"
    session: true      # NEW: Run in detached session
    multiple: false
    
  - name: test
    exec: "go test ./..."
    session: false     # Direct execution (current behavior)
    multiple: false
```

### Behavior

When `session: true`:
1. Command runs in session manager (tmux/abduco/screen)
2. Starts detached (returns to TUI immediately)
3. User can attach later with `a` key
4. Process continues even if TUI exits
5. User can detach and reattach multiple times

When `session: false` (default):
1. Command runs directly (current behavior)
2. Blocks TUI until completion
3. Output captured and displayed
4. Process terminates with TUI

### Launcher Types

This creates two modes:

**Interactive Launchers** (`session: true`):
- Long-running builds
- Development servers
- Watch/hot-reload processes
- Commands needing user interaction

**Batch Launchers** (`session: false`):
- Quick tests
- Linters
- Formatters
- CI/CD-style commands

## Consequences

### Positive

- **Backward compatible**: Existing launchers work unchanged (default `session: false`)
- **Clear opt-in**: Users explicitly choose session management
- **Familiar pattern**: Similar to existing launcher configuration
- **Single concept**: Launcher remains the unit of execution
- **Flexible**: Some launchers use sessions, others don't
- **Consistent UX**: Same mental model for all commands

### Negative

- **Configuration complexity**: More fields to understand
- **Two execution paths**: Code must handle both modes
- **Mode switching**: Can't dynamically choose at runtime (must reconfigure)
- **Multiple field**: Need to think about `session` + `multiple` interaction

### Neutral

- **Discovery**: Users need to learn about `session: true` option

## Alternatives Considered

### Separate session commands
Create separate `session_commands:` section distinct from launchers.

**Rejected because**:
- Duplicates configuration
- Two concepts where one suffices
- Confusing which to use when

### Always use sessions
Make all launchers use sessions by default.

**Rejected because**:
- Breaking change
- Not all commands need it (quick tests)
- Loss of output capture for batch commands

### Manual session management
Don't integrate with launchers; users manually create sessions.

**Rejected because**:
- Poor UX
- Defeats purpose of launcher system
- More steps to accomplish common task

### Runtime mode selection
Let user choose per-execution whether to use session.

**Rejected because**:
- More complex UX (extra prompt)
- Less explicit (behavior varies)
- Configuration documents intent better

## Implementation Details

### Launcher Execution Path

```go
func (m *LaunchManager) startLaunch(launch *BeanLaunch) error {
    if m.launcher.Session {
        // Session path
        sessionName := generateSessionName(launch.Bean)
        return m.sessionMgr.Start(sessionName, m.launcher.Exec)
    } else {
        // Direct path (current behavior)
        cmd := exec.Command("sh", "-c", m.launcher.Exec)
        return cmd.Start()
    }
}
```

### Multiple Field Interaction

When both `session: true` and `multiple: true`:
- Creates separate session per bean
- Each session independently attachable
- Sessions named: `bean-abc123-fix-bug`, `bean-def456-add-feature`, etc.

This is powerful for:
- Running builds for multiple features in parallel
- Each session independently monitorable
- No interference between sessions

### Output Capture

**Session mode**: Output not captured in beans (lives in session)
- User attaches to see output
- Could add `GetOutput()` later for viewing in TUI

**Direct mode**: Output captured as before
- Displayed in TUI progress view
- Stored for error reporting

### Status Tracking

**Session mode**:
- Status = session status (active/detached/terminated)
- No "success/failure" until session terminates
- User can check by attaching

**Direct mode**:
- Status = command exit code
- Success/failure known immediately

## User Experience

### Typical Workflows

**Long-running build**:
```yaml
launchers:
  - name: build
    exec: "make build"
    session: true
```

1. User selects bean, presses `l` (launcher)
2. Chooses "build" launcher
3. Session starts, returns to TUI immediately
4. Status indicator shows session active
5. User presses `a` to attach and watch build
6. Presses detach key (Ctrl+B D) to return to TUI
7. Build continues in background
8. When done, status shows terminated

**Quick test**:
```yaml
launchers:
  - name: test
    exec: "go test ./..."
    session: false
```

1. User selects bean, presses `l`
2. Chooses "test" launcher
3. Progress view shows test output
4. When done, shows success/failure
5. Returns to TUI

## Migration Path

Existing configurations:
- All launchers default to `session: false`
- Behavior unchanged
- Users opt-in by adding `session: true`

## Future Enhancements

- Auto-attach option: `session: true, auto_attach: true`
- Session timeout: `session_timeout: 1h` (kill if inactive)
- Session recording: Save output for later review
- TUI output viewer: Show session output without attaching

## References

- ADR-001: Session Adapter Pattern
- ADR-002: Shell Script Implementation
- Existing: `internal/launcher/` implementation
