# ADR-003: Session Naming Convention

**Status**: Proposed  
**Date**: 2024-12-15  
**Deciders**: beans project team  
**Technical Story**: How to generate unique, readable session names for beans

## Context

When beans creates sessions for commands, we need to name them. The name appears in:
- Session manager listings (`tmux ls`, `abduco`, etc.)
- User's terminal when manually attaching
- beans TUI when showing active sessions

Requirements:
1. **Unique**: Different beans must have different session names
2. **Readable**: Users should recognize which bean a session belongs to
3. **Safe**: Work across different session managers (alphanumeric safe)
4. **Reasonable length**: Not too long for terminal display

Options considered:
- **Option A**: Simple prefix with ID: `bean-abc123`
- **Option B**: Include sanitized title: `bean-abc123-fix-ui-bug`
- **Option C**: Configurable template: User decides format

## Decision

Use **Option B** with a configurable template (defaulting to Option B format).

### Default Template

```
bean-{{.ID}}-{{.Title}}
```

Example outputs:
- `bean-abc123-fix-ui-rendering-bug`
- `bean-def456-add-dark-mode-feature`
- `bean-ghi789-refactor-authentication`

### Template Variables

Available in session name template:
- `{{.ID}}`: Bean unique ID (always present)
- `{{.Title}}`: Bean title (sanitized)
- `{{.Type}}`: Bean type (feature, bug, task, etc.)
- `{{.Status}}`: Bean status (todo, in-progress, etc.)

### Sanitization Rules

Title sanitization:
1. Convert to lowercase
2. Replace spaces with hyphens
3. Remove non-alphanumeric characters (except hyphens)
4. Collapse multiple hyphens to single hyphen
5. Trim hyphens from start/end
6. Limit to 40 characters

Examples:
```
"Fix UI Bug" → "fix-ui-bug"
"Add Feature: Dark Mode!" → "add-feature-dark-mode"
"Refactor (Part 1)" → "refactor-part-1"
"Very Long Title That Goes On And On And On" → "very-long-title-that-goes-on-and-on-an"
```

### Configuration

Users can override in `.beans.yml`:

```yaml
session_manager:
  name_template: "bean-{{.ID}}-{{.Title}}"  # Default
  # or: "{{.Type}}-{{.ID}}"
  # or: "beans-{{.Status}}-{{.ID}}"
```

## Consequences

### Positive

- **Human-readable**: Easy to identify beans from session list
- **Unique**: Bean ID ensures uniqueness
- **Flexible**: Users can customize if needed
- **Terminal-friendly**: Sanitization ensures compatibility
- **Debuggable**: Clear what each session is for

### Negative

- **Length**: Longer names take more terminal space
- **Complexity**: Sanitization logic needed
- **Collisions possible**: If title truncated and IDs similar (very rare)
- **Update lag**: If bean title changes, session name doesn't update

### Neutral

- **Case sensitivity**: Different backends handle differently (use lowercase to be safe)

## Alternatives Considered

### Option A: Simple ID only (`bean-abc123`)
- **Pro**: Short, guaranteed unique, simple
- **Con**: Not human-readable, hard to identify which bean
- **Rejected**: Readability matters for user experience

### UUID-based naming
- **Pro**: Guaranteed globally unique
- **Con**: Completely unreadable (`bean-a1b2c3d4-e5f6-...`)
- **Rejected**: Defeats purpose of readable names

### Include timestamp
- **Example**: `bean-abc123-fix-bug-20241215-143022`
- **Pro**: Allows multiple sessions per bean
- **Con**: Very long, timestamp not useful for short-lived sessions
- **Rejected**: Length outweighs benefit

## Implementation Notes

### Collision Handling

If a session already exists with the same name:
1. Check if it's for the same bean (could be old session)
2. Prompt user: "Session already exists. Kill old session? (y/n)"
3. Or: Append timestamp only when collision detected

### Name Length Limits

Different session managers have different limits:
- tmux: ~255 characters (practical limit)
- screen: ~100 characters
- abduco: ~100 characters

Our default (ID + 40-char title) = ~50 characters total, well under all limits.

### Bean Title Changes

If user renames a bean:
- Existing sessions keep old name (don't auto-rename)
- New sessions use new name
- User can manually kill old session if desired

This is simpler than trying to track and rename sessions.

## Edge Cases

1. **Empty title**: Fall back to `bean-{{.ID}}`
2. **Special characters only**: Sanitize to empty → use ID only
3. **Duplicate sanitized titles**: ID makes them unique
4. **Very short titles**: Keep them short (no minimum length)

## Examples

```go
// Bean: ID=abc123, Title="Fix UI Rendering Bug"
name := "bean-abc123-fix-ui-rendering-bug"

// Bean: ID=def456, Title="Add Dark Mode 🌙"
name := "bean-def456-add-dark-mode"

// Bean: ID=ghi789, Title="Feature: User Authentication & Authorization"
name := "bean-ghi789-feature-user-authentication-"  // truncated
```

## References

- ADR-001: Session Adapter Pattern
- ADR-002: Shell Script Implementation
- Research: Session managers handle names differently
