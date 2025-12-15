# ADR-001: Session Manager Adapter Pattern

**Status**: Proposed  
**Date**: 2024-12-15  
**Deciders**: beans project team  
**Technical Story**: Implement detachable/attachable process management for interactive CLI tools

## Context

beans users need to run long-running interactive commands (like builds) that can be:
- Started in the background (without blocking the TUI)
- Detached from (return to TUI while process continues)
- Reattached to (bring process to foreground for interaction)
- Run multiple simultaneously

Research (see `docs/research/`) found no pure Go library provides this functionality. Available options are:
1. Shell out to existing session managers (tmux, abduco, screen)
2. Build custom solution from scratch (~1000 LOC, 1-2 weeks)
3. Use container-based approach (too heavy)

## Decision

We will implement a **session manager adapter pattern** that:

1. **Defines a common interface** for session operations (Start, Attach, List, Kill, etc.)
2. **Supports multiple backends** through adapters (tmux, abduco, screen, custom)
3. **Uses shell script execution** rather than Go libraries (none exist)
4. **Integrates with existing launcher system** via optional `session: bool` field
5. **Requires user installation** of backend (no binary bundling)

### Architecture

```
SessionManager (interface)
    ├─ ScriptAdapter (tmux)
    ├─ ScriptAdapter (abduco)
    ├─ ScriptAdapter (screen)
    └─ ScriptAdapter (custom)
```

Each adapter executes shell commands configured via `.beans.yml`.

## Consequences

### Positive

- **Flexibility**: Users can choose their preferred backend
- **Extensibility**: Easy to add new backends via configuration
- **Simplicity**: ~50-100 LOC per adapter vs ~1000 LOC custom implementation
- **Reliability**: Leverages battle-tested tools (tmux is 15+ years old)
- **Speed**: Can implement in 15-20 hours vs weeks for custom
- **Familiar**: Users already know tmux/screen/abduco

### Negative

- **External dependency**: Requires user to install session manager
- **Parsing complexity**: Need to parse backend-specific output formats
- **Limited control**: Can't customize behavior deeply
- **Unix-only**: No Windows support (would need different approach)
- **Documentation burden**: Need to document each supported backend

### Neutral

- **Configuration surface**: More config options (but opt-in)
- **Backend availability**: Some backends less common (abduco) than others (tmux)

## Alternatives Considered

### 1. Bundle abduco binary
- **Pro**: No user installation required
- **Con**: 50KB per platform, cross-compilation complexity, maintenance burden
- **Decision**: Rejected - prefer simpler approach initially

### 2. Build custom in pure Go
- **Pro**: Full control, no external dependency, Windows possible
- **Con**: 1-2 weeks work, ~1000 LOC, ongoing maintenance, unknown edge cases
- **Decision**: Rejected - not worth the time investment

### 3. Require tmux only
- **Pro**: Most widely available, simpler config
- **Con**: Less flexibility, some users prefer abduco/screen
- **Decision**: Rejected - adapter pattern provides flexibility at low cost

### 4. No session management
- **Pro**: Simplest implementation
- **Con**: Doesn't solve user need for long-running processes
- **Decision**: Rejected - feature is valuable

## Implementation Notes

- Start with tmux, abduco, screen as built-in backends
- Document each backend clearly
- Provide good error messages when backend not available
- Consider fallback to direct execution if no backend configured

## References

- Research documentation: `docs/research/`
- Related ADRs: ADR-002 (Implementation), ADR-004 (Integration)
- Similar patterns: Launcher system in beans
