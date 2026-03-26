# ADR-005: Backend Installation Strategy

**Status**: Proposed  
**Date**: 2024-12-15  
**Deciders**: beans project team  
**Technical Story**: Whether to bundle session manager binaries or require user installation

## Context

Session managers (tmux, abduco, screen) are external programs. We must decide:

**Option A**: Bundle binaries with beans
- Embed static binaries in Go binary
- Extract to user cache on first run
- ~50KB per platform
- Requires cross-compilation setup

**Option B**: Require user installation
- Document installation instructions
- Check availability at runtime
- Clear error messages if missing
- Lighter binary, simpler build

Research findings (see `docs/research/03-abduco-detailed-analysis.md`):
- Bundling is technically feasible (~50KB overhead)
- Cross-compilation adds build complexity
- Multiple platforms to support (Linux x64, macOS x64, macOS ARM64)

## Decision

**Require user installation** (Option B).

### Rationale

1. **Build simplicity**: Standard `go build` works without extra steps
2. **Binary size**: Keep beans binary small
3. **User choice**: Users pick their preferred backend
4. **Packaging**: System package managers handle updates
5. **Maintenance**: Don't need to track upstream releases

### User Experience

When backend not available:
1. Clear error message on first session attempt
2. Installation instructions for user's platform
3. Graceful degradation (launchers still work without sessions)
4. Documentation covers installation

Example error:
```
Error: tmux not found

Session management requires a session manager backend.

To use sessions, install one of:
  • tmux: brew install tmux  (or: apt install tmux)
  • abduco: brew install abduco
  • screen: apt install screen

Or disable sessions:
  session: false  # in launcher config

Documentation: https://github.com/hmans/beans/docs/session-management.md
```

## Consequences

### Positive

- **Simple build**: No cross-compilation needed
- **Small binary**: No embedded binaries (~50KB saved per platform)
- **User control**: Users choose which backend to install
- **System updates**: Backend updates via package manager
- **Clear requirements**: Explicit about dependencies
- **Easier CI/CD**: Standard Go build in CI

### Negative

- **Installation friction**: Users must install backend separately
- **Platform variations**: Installation differs by OS
- **Version issues**: Can't guarantee specific backend version
- **Documentation burden**: Must document installation for each platform
- **Discovery**: Users might not realize they need to install anything

### Neutral

- **Error messages**: Need good UX for missing backend
- **Fallback behavior**: Must work gracefully without backend

## Alternatives Considered

### Bundle abduco only
Embed just abduco (~50KB), fall back to user's tmux/screen.

**Rejected because**:
- Still requires cross-compilation
- Many users prefer tmux (already installed)
- Maintenance burden for one backend

### Hybrid approach
Bundle for some platforms, require installation for others.

**Rejected because**:
- Inconsistent UX
- Complex build system
- Documentation complexity

### Docker-based
Provide Docker image with all backends.

**Rejected because**:
- Too heavy for CLI tool
- Requires Docker
- Wrong abstraction level

## Implementation Details

### Availability Check

```go
func (a *ScriptAdapter) Available() bool {
    _, err := exec.LookPath(a.name)
    return err == nil
}

func (f *Factory) NewManager() (SessionManager, error) {
    mgr := createAdapter(config.Backend)
    
    if !mgr.Available() {
        return nil, &BackendNotFoundError{
            Backend: config.Backend,
            Instructions: getInstallInstructions(config.Backend),
        }
    }
    
    return mgr, nil
}
```

### Installation Instructions

Maintain in code:
```go
var installInstructions = map[string]map[string]string{
    "tmux": {
        "darwin": "brew install tmux",
        "linux": "apt install tmux  (or: yum install tmux)",
    },
    "abduco": {
        "darwin": "brew install abduco",
        "linux": "apt install abduco  (or: build from source)",
    },
    "screen": {
        "darwin": "brew install screen",
        "linux": "apt install screen  (usually pre-installed)",
    },
}
```

### Documentation

Create `docs/session-management.md` with:
- Installation instructions per platform
- Recommended backends
- Configuration examples
- Troubleshooting

Include in README:
```markdown
## Session Management

beans supports detachable sessions for long-running commands.

**Installation**: Requires tmux, abduco, or screen
- macOS: `brew install tmux`
- Linux: `apt install tmux` or `yum install tmux`

See [Session Management Guide](docs/session-management.md) for details.
```

### Graceful Degradation

If backend not available:
1. First session attempt shows error with instructions
2. Launchers with `session: true` show warning
3. Launchers with `session: false` continue working
4. User can still use beans without sessions (not a hard requirement)

## Platform Support

Document support matrix:

| Backend | macOS | Linux | FreeBSD | Windows |
|---------|-------|-------|---------|---------|
| tmux    | ✅     | ✅     | ✅       | ❌       |
| abduco  | ✅     | ✅     | ✅       | ❌       |
| screen  | ✅     | ✅     | ✅       | ❌       |

Windows: Session management not supported. Launchers work but without detach/attach.

## Future Reconsideration

If many users request bundling:
1. Evaluate build complexity vs user benefit
2. Consider abduco only (smallest, simplest)
3. Could be separate build variant (`beans-full` vs `beans`)
4. Re-evaluate after user feedback

## References

- Research: `docs/research/03-abduco-detailed-analysis.md`
- Research: Bundle complexity analysis
- ADR-001: Session Adapter Pattern
- ADR-002: Shell Script Implementation
