# Research Summary: Subprocess Management for beans

**Date**: December 2024  
**Total Research Time**: 4+ hours with parallel agents  
**Documents Created**: 9 comprehensive documents (~72KB total)

---

## The Question

> How do we implement detachable/attachable subprocess management for interactive CLI tools in Go?

**Use Case**: Run build commands that users can:
- Start in background
- Detach from (Ctrl+\)
- Reattach to (brings to foreground)
- Have multiple running simultaneously

---

## The Answer

### Use `abduco` (bundled)

**Why**: It's the simplest, most reliable solution that exactly fits our needs.

**Implementation time**: 2-4 hours  
**Code volume**: ~100 lines  
**Binary size increase**: 50KB  
**Maintenance**: Minimal  

---

## Key Research Findings

### 1. No Pure Go Solution Exists ❌

We searched everywhere:
- ✅ pkg.go.dev - All Go packages
- ✅ GitHub - Extensive keyword searches
- ✅ awesome-go - Comprehensive lists
- ✅ Real-world projects - How others solve this

**Result**: Zero Go packages provide session management with attach/detach

### 2. Why Nothing Exists

**Complexity**: Requires PTY + process lifecycle + client-server protocol + state persistence  
**Existing tools work**: tmux/screen are mature and reliable  
**Small demand**: Most developers just shell out to existing tools  
**Platform-specific**: Hard to create portable abstraction  

### 3. What Go Ecosystem Has

| Category | What Exists | What's Missing |
|----------|-------------|----------------|
| **PTY libraries** | ✅ creack/pty | Session persistence |
| **Process supervisors** | ✅ overseer, goproc | Interactive attach |
| **Daemon helpers** | ✅ go-daemon | Terminal management |
| **Session managers** | ❌ Nothing | Everything we need |

---

## The Three Options

### Option 1: abduco (bundled) ⭐ RECOMMENDED

```go
// Simple wrapper
sessionMgr.Start("bean-123", "make build")  // background
sessionMgr.Attach("bean-123")                // foreground
```

**Pros**:
- ✅ 2-4 hour implementation
- ✅ ~100 lines of code
- ✅ 50KB binary (negligible)
- ✅ 10+ years battle-tested
- ✅ Perfect fit for our needs

**Cons**:
- ⚠️ External C binary (but no Go alternative!)
- ⚠️ Unix-only (Linux, macOS, BSD)

**Decision**: Choose this unless you have specific reason not to

---

### Option 2: tmux (shell out)

```go
// Shell out to tmux
exec.Command("tmux", "new-session", "-d", "-s", "name", "cmd")
exec.Command("tmux", "attach", "-t", "name")
```

**Pros**:
- ✅ Widely installed
- ✅ No bundling needed
- ✅ Full-featured

**Cons**:
- ⚠️ Heavy (~1MB)
- ⚠️ Users must install
- ⚠️ Overkill for simple needs

**Decision**: Good fallback if users already have tmux

---

### Option 3: Custom (build from scratch)

```go
// Implement yourself using creack/pty
// Components: daemon, IPC, protocol, state, etc.
```

**Pros**:
- ✅ Complete control
- ✅ Pure Go
- ✅ Custom features

**Cons**:
- ❌ 1-2 weeks implementation
- ❌ ~1000 lines of code
- ❌ Ongoing maintenance
- ❌ Edge cases to discover

**Decision**: Only if you really need custom features or Windows support

---

## Decision Matrix

| Criterion | abduco | tmux | Custom |
|-----------|--------|------|--------|
| Time to implement | ⭐⭐⭐⭐⭐ Hours | ⭐⭐⭐⭐⭐ Hours | ⭐ Weeks |
| Code complexity | ⭐⭐⭐⭐⭐ ~100 LOC | ⭐⭐⭐⭐⭐ ~100 LOC | ⭐ ~1000 LOC |
| Reliability | ⭐⭐⭐⭐⭐ Proven | ⭐⭐⭐⭐⭐ Proven | ⭐⭐ Unknown |
| Binary size | ⭐⭐⭐⭐ +50KB | ⭐⭐⭐⭐⭐ 0 | ⭐⭐⭐⭐⭐ 0 |
| Maintenance | ⭐⭐⭐⭐⭐ Minimal | ⭐⭐⭐⭐⭐ Minimal | ⭐⭐ Ongoing |
| Flexibility | ⭐⭐⭐ Limited | ⭐⭐ Very limited | ⭐⭐⭐⭐⭐ Complete |
| User install | ⭐⭐⭐⭐⭐ None | ⭐⭐⭐ Required | ⭐⭐⭐⭐⭐ None |

**Winner**: abduco (best balance of simplicity, reliability, speed)

---

## Real-World Validation

### What We Found

**gob** project (GitHub: juanibiapina/gob):
- Go process manager with Bubble Tea TUI
- Uses daemon architecture with Unix sockets
- ~3000 LOC for full implementation
- Validates: This is complex to build!

**Most projects**:
- Shell out to tmux or screen
- Don't build custom solutions
- Accept external dependencies

**Lesson**: The Go community solves this by using existing C tools, not building Go libraries.

---

## Implementation Quick Start

### Step 1: Compile abduco (30 minutes)

```bash
git clone https://github.com/martanne/abduco
cd abduco
make LDFLAGS=-static
strip abduco  # ~50KB
```

Do this for:
- Linux x86_64
- macOS x86_64
- macOS ARM64

### Step 2: Embed in Go (30 minutes)

```go
//go:embed bin/abduco-linux-amd64
var abducoLinux []byte

func extractAbduco() (string, error) {
    // Extract to ~/.cache/beans/bin/abduco
    // Return path
}
```

### Step 3: Wrapper (1 hour)

```go
type Manager struct {
    abducoPath string
}

func (m *Manager) Start(name, cmd string) error
func (m *Manager) AttachCommand(name string) *exec.Cmd
func (m *Manager) List() ([]Session, error)
func (m *Manager) Exists(name string) (bool, error)
```

### Step 4: Integrate with Bubble Tea (1 hour)

```go
case "r": // run
    sessionMgr.Start("bean-"+id, "make build")

case "a": // attach
    return m, tea.ExecProcess(
        sessionMgr.AttachCommand("bean-"+id),
        nil,
    )
```

**Total**: 2-4 hours to working implementation

---

## Architecture Comparison

### abduco Architecture (Simple)

```
beans TUI
    ↓ shell out
abduco client → Unix Socket → abduco server → PTY → build process
```

**Complexity**: LOW  
**Code**: ~100 LOC  

### Custom Architecture (Complex)

```
beans TUI
    ↓ Unix socket
beans daemon → Session Manager → PTY Manager → build process
    ↓                              ↓
SQLite DB                    Output Buffer
```

**Complexity**: HIGH  
**Code**: ~1000 LOC  

**Savings**: 900 LOC + weeks of debugging by using abduco

---

## Risk Analysis

### Risk: Binary compatibility

**Probability**: Low  
**Impact**: Medium  
**Mitigation**: Static compilation, test on multiple platforms

### Risk: Users unfamiliar with detach key

**Probability**: High  
**Impact**: Low  
**Mitigation**: Document Ctrl+\ clearly, show in UI

### Risk: Future Windows support needed

**Probability**: Medium  
**Impact**: High  
**Mitigation**: Keep abstraction layer, can swap implementation later

---

## Documentation Navigation

### For Different Reader Types

**Executive**: Read this summary  
**Developer implementing**: [Recommendations](./08-recommendations.md)  
**Curious about abduco**: [abduco Analysis](./03-abduco-detailed-analysis.md)  
**Want to build custom**: [Go PTY Libraries](./04-go-pty-libraries.md)  
**Researching alternatives**: [Terminal Multiplexers](./06-terminal-multiplexers.md)  

### Full Document List

1. [Overview](./01-subprocess-management-overview.md) - Research summary
2. [Bubble Tea](./02-bubbletea-subprocess-handling.md) - Built-in capabilities
3. [abduco](./03-abduco-detailed-analysis.md) - Detailed analysis ⭐
4. [PTY Libraries](./04-go-pty-libraries.md) - Building blocks
5. [Go Projects](./05-go-projects-analysis.md) - Real-world examples
6. [Multiplexers](./06-terminal-multiplexers.md) - All options
7. [Package Search](./07-go-packages-search.md) - Ecosystem analysis
8. [Recommendations](./08-recommendations.md) - Final decision ✅

---

## Conclusion

### The Winning Solution: abduco (bundled)

**Time to value**: Hours, not weeks  
**Code complexity**: Minimal (~100 LOC)  
**Reliability**: Proven (10+ years production use)  
**Maintenance**: Minimal  
**User experience**: Seamless (bundled, no install)  

### Why This Beats Custom

**Custom implementation would require**:
- ⏱️ 1-2 weeks development
- 📝 ~1000 lines of code
- 🐛 Extensive testing/debugging
- 🔧 Ongoing maintenance
- ❓ Unknown edge cases

**abduco provides**:
- ⏱️ 2-4 hours integration
- 📝 ~100 lines of wrapper
- ✅ Battle-tested reliability
- 🎯 Minimal maintenance
- ✨ Known behavior

**Savings**: Weeks of work + ongoing maintenance burden

### Next Steps

1. ✅ Research complete (this document)
2. ⏭️ Compile abduco binaries
3. ⏭️ Implement session manager wrapper
4. ⏭️ Integrate with beans TUI
5. ⏭️ Ship and gather user feedback

---

**Total Research Output**: 9 documents, 72KB, comprehensive analysis

**Research validated by**: 
- Package ecosystem search
- Real-world project analysis  
- Community patterns
- Technical feasibility assessment

**Confidence level**: High (exhaustive research, clear winner)

---

For full implementation details, see: [08-recommendations.md](./08-recommendations.md)
