# Session Management Implementation Plan

**Created**: 2024-12-15  
**Epic**: beans-ru01  
**Total Beans**: 17 implementation tasks

## Overview

This document provides a high-level overview of the session management implementation for beans. Full architectural decisions are documented in ADRs (see `docs/adr/`).

## Architecture Decision Records (ADRs)

All architectural decisions documented in Nygard format:

1. **ADR-001**: Session Manager Adapter Pattern - Core architecture decision
2. **ADR-002**: Shell Script Implementation Strategy - Implementation approach with default backend
3. **ADR-003**: Session Naming Convention - How to name sessions (bean-ID-title)
4. **ADR-004**: Launcher Integration - Integration with existing launcher system
5. **ADR-005**: Backend Installation Strategy - Require user installation vs bundling

## Implementation Phases

### Phase 1: Foundation (Critical Priority)

**Beans**: beans-cc5u, beans-8wkw, beans-ucah

1. Write ADRs (✅ COMPLETED)
2. Define SessionManager interface and types
3. Implement session name generation and sanitization

**Blocking**: All subsequent work depends on these

### Phase 2: Core Implementation (High Priority)

**Beans**: beans-w287, beans-s4vz, beans-hldx, beans-rvpc

4. Implement ScriptAdapter for shell-based backends
5. Add session manager configuration to schema
6. Create factory to instantiate session managers from config
7. Create default configs for tmux, abduco, screen

**Dependencies**: Blocked by Phase 1  
**Blocks**: All integration work

### Phase 3: Integration (Normal Priority)

**Beans**: beans-xh3x, beans-5z86, beans-qy91

8. Integrate session management with launcher system
9. Add session management to TUI (r/a/k keybindings)
10. Show session status indicators in TUI bean list

**Dependencies**: Blocked by Phase 2  
**Deliverable**: Working end-to-end session management

### Phase 4: Testing & Documentation (Normal Priority)

**Beans**: beans-9n4a, beans-p748, beans-mdto, beans-fdmx

11. Write unit tests for session management components
12. Write integration tests for session launchers
13. Write user guide for session management
14. Manual testing checklist

**Dependencies**: Blocked by Phase 3  
**Deliverable**: Tested, documented feature

### Phase 5: Polish (Low Priority)

**Beans**: beans-exug, beans-ukrs, beans-t79y

15. Implement optional SendInput and GetOutput methods
16. Improve error messages and user feedback
17. Update CHANGELOG for release

**Dependencies**: Blocked by relevant Phase 3/4 beans  
**Deliverable**: Production-ready feature

## Key Design Decisions

### Adapter Pattern
- Interface-based design supporting multiple backends
- Shell script execution via Go templates
- User-configurable via `.beans.yml`

### Session Naming
- Template: `bean-{{.ID}}-{{.Title}}`
- Example: `bean-abc123-fix-ui-rendering-bug`
- Sanitization: lowercase, alphanumeric + hyphens, 40 char limit

### Launcher Integration
- Optional `session: bool` field on launchers
- Backward compatible (defaults to false)
- Same launcher concept, two execution modes

### Backend Strategy
- Require user installation (tmux/abduco/screen)
- No binary bundling (keep build simple)
- Clear error messages with installation instructions

## Configuration Example

```yaml
# Session manager configuration
session_manager:
  backend: tmux
  name_template: "bean-{{.ID}}-{{.Title}}"

# Session manager definitions
session_managers:
  - name: tmux
    description: "Use tmux for session management"
    commands:
      start: "tmux new-session -d -s {{.Name}} {{.Command}}"
      attach: "tmux attach-session -t {{.Name}}"
      list: "tmux list-sessions"
      exists: "tmux has-session -t {{.Name}}"
      kill: "tmux kill-session -t {{.Name}}"

# Launchers can use sessions
launchers:
  - name: build
    exec: "make build"
    session: true      # Run in detached session
  
  - name: test
    exec: "go test ./..."
    session: false     # Direct execution (default)
```

## Implementation Tracking

Track progress using beans:

```bash
# List all session management beans
beans list --parent beans-ru01

# View specific phase
beans list --parent beans-ru01 --priority critical
beans list --parent beans-ru01 --priority high

# Start work on a bean
beans update beans-8wkw --status in-progress

# Mark complete
beans update beans-8wkw --status completed
```

## Estimated Timeline

- **Phase 1**: 4-6 hours
- **Phase 2**: 6-8 hours
- **Phase 3**: 8-10 hours
- **Phase 4**: 6-8 hours
- **Phase 5**: 4-6 hours

**Total**: 28-38 hours of implementation work

## Dependencies Graph

```
beans-cc5u (ADRs) ✅
    ├─→ beans-8wkw (Interface) [CRITICAL]
    │       ├─→ beans-w287 (ScriptAdapter) [HIGH]
    │       │       ├─→ beans-hldx (Factory) [HIGH]
    │       │       │       └─→ beans-xh3x (Launcher Integration) [NORMAL]
    │       │       │               └─→ beans-5z86 (TUI Integration) [NORMAL]
    │       │       │                       └─→ beans-qy91 (Status Display) [NORMAL]
    │       │       └─→ beans-exug (SendInput/GetOutput) [LOW]
    │       └─→ beans-s4vz (Config) [HIGH]
    │               ├─→ beans-hldx (Factory) [HIGH]
    │               └─→ beans-rvpc (Defaults) [HIGH]
    └─→ beans-ucah (Naming) [CRITICAL]
            └─→ beans-w287 (ScriptAdapter) [HIGH]

Testing: beans-9n4a, beans-p748 (depend on core beans)
Docs: beans-mdto (depends on TUI integration)
Manual: beans-fdmx (depends on status display)
```

## Next Steps

1. ✅ ADRs completed
2. ⏭️ Start Phase 1 critical beans (beans-8wkw, beans-ucah)
3. ⏭️ Review and discuss architectural decisions
4. ⏭️ Begin implementation following dependency order

## References

- **Research**: `docs/research/` - Comprehensive research findings
- **ADRs**: `docs/adr/` - Architectural decisions
- **Epic**: beans-ru01 - Parent bean tracking all work
- **Config Example**: `.beans.yml` - Commented examples ready for testing

## Dogfooding

The `.beans.yml` file includes commented examples of session manager configuration. Once implemented, we can use session management to develop session management! 🎯

```yaml
# Future: Run builds in sessions
launchers:
  - name: test-sessions
    exec: "go test ./internal/session/..."
    session: true
```

---

For detailed implementation guidance, see individual bean descriptions and ADRs.
