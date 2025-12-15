# ADR-002: Shell Script Implementation Strategy

**Status**: Proposed  
**Date**: 2024-12-15  
**Deciders**: beans project team  
**Technical Story**: Implementation approach for session manager adapters

## Context

Having decided on an adapter pattern (ADR-001), we need to choose how to implement the adapters. Three options emerged:

**Option A**: Shell script templates with Go templating (`{{.Name}}`, `{{.Command}}`)
- Configuration-driven
- User can customize commands
- Requires parsing shell output
- Simple Go wrapper code (~50-100 LOC per adapter)

**Option B**: Pure Go implementations
- Type-safe
- More control over behavior
- Requires understanding each backend's protocol
- More code (~200-300 LOC per adapter)

**Option C**: Hybrid approach
- Built-in Go implementations for common backends
- Shell script fallback for custom backends
- Most complex, but most flexible

## Decision

We will use **Option A: Shell script templates** with Go's `text/template` package.

### Configuration Format

Session managers defined in `.beans.yml`:

```yaml
session_managers:
  - name: tmux
    commands:
      start: "tmux new-session -d -s {{.Name}} {{.Command}}"
      attach: "tmux attach-session -t {{.Name}}"
      list: "tmux list-sessions"
      exists: "tmux has-session -t {{.Name}}"
      kill: "tmux kill-session -t {{.Name}}"
```

### Template Variables

Available template variables:
- `{{.Name}}`: Session name (generated from bean ID and title)
- `{{.Command}}`: Command to execute
- `{{.Input}}`: Input to send (for SendInput operation)

### Default Backend

The system will ship with default configurations for:
- `tmux` (most widely available)
- `abduco` (lightweight alternative)
- `screen` (classic option)

**Default backend**: tmux (configured in default config)

Users can override or add custom backends.

## Consequences

### Positive

- **Low buy-in cost**: Simple to understand and implement
- **User customizable**: Power users can tweak commands
- **Extensible**: Easy to add new backends without code changes
- **Familiar pattern**: Similar to how shells/scripts already work
- **Quick iteration**: Can test/debug by running commands manually
- **Documentation friendly**: Shell commands are self-documenting

### Negative

- **Output parsing**: Need to parse each backend's output format
- **Error handling**: Shell errors less structured than Go errors
- **Testing complexity**: Harder to unit test shell execution
- **Security**: Need to be careful with template injection
- **Platform specific**: Shell commands may differ across OS

### Neutral

- **Performance**: Shell exec overhead negligible for interactive use
- **Type safety**: Lose some compile-time safety (trade-off for flexibility)

## Alternatives Considered

### Pure Go implementations (Option B)
- **Rejected because**: 
  - Much more code to write and maintain
  - Need to understand internals of each backend
  - Less flexible for users
  - Longer implementation time
  
### Hybrid approach (Option C)
- **Deferred**: Consider for future if shell scripts prove limiting
- Could add as enhancement without breaking existing configs

## Implementation Details

### ScriptAdapter Structure

```go
type ScriptAdapter struct {
    name     string
    commands ScriptCommands
    templates map[string]*template.Template
}

func (a *ScriptAdapter) Start(name, command string) error {
    cmd := a.renderTemplate("start", map[string]string{
        "Name": name,
        "Command": command,
    })
    return exec.Command("sh", "-c", cmd).Run()
}
```

### Output Parsing

Each backend's `list` output will need a parser. Start with simple approaches:
- Line-based parsing
- Field splitting
- Regex for complex formats

Can be enhanced later without breaking API.

### Error Messages

Commands that fail should provide:
1. Original shell command attempted
2. Exit code
3. stderr output
4. Suggestion to check backend installation

## Security Considerations

- Template injection: Use Go's template escaping
- Command injection: Use `exec.Command("sh", "-c", cmd)` not string concatenation
- User input: Sanitize session names (see ADR-003)

## Migration Path

Future: Could add pure Go implementations for specific backends while maintaining shell script interface. This would be transparent to users.

## References

- ADR-001: Session Adapter Pattern
- ADR-003: Session Naming Convention
- Similar pattern: Launcher exec scripts in beans
