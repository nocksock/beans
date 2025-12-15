# Subprocess Management Research Documentation

**Research Period**: December 2024  
**Question**: How to implement detachable/attachable subprocess management for interactive CLI tools in Go?

## Quick Links

- **🎯 [Start Here: Overview](./01-subprocess-management-overview.md)**
- **✅ [Final Recommendations](./08-recommendations.md)**

## TL;DR

**Problem**: Need to run interactive CLI tools (like build commands) that can be:
- Started in background
- Detached from (without killing)
- Reattached to (brought to foreground)
- Run multiple simultaneously

**Finding**: No pure Go library exists for this

**Solution**: Use `abduco` (50KB C binary) bundled with application

**Timeline**: 2-4 hours to implement

---

## Document Structure

### 1. [Overview](./01-subprocess-management-overview.md)
- Research summary
- Key findings
- Quick reference table
- Navigation guide

### 2. [Bubble Tea Subprocess Handling](./02-bubbletea-subprocess-handling.md)
- Built-in `tea.ExecProcess` capabilities
- Limitations for our use case
- Integration patterns
- What we need beyond Bubble Tea

### 3. [abduco: Detailed Analysis](./03-abduco-detailed-analysis.md) ⭐
- What abduco is and how it works
- Architecture (client-server model)
- Command reference
- Go integration examples
- Bundling strategy
- Pros/cons comparison
- Platform availability

### 4. [Go PTY Libraries](./04-go-pty-libraries.md)
- `creack/pty` - the standard library
- Alternative PTY libraries
- Building session management on top
- Implementation complexity analysis
- Why building custom is hard

### 5. [Real-World Go Projects](./05-go-projects-analysis.md)
- `gob` - Process manager with Bubble Tea (most relevant!)
- Common patterns in Go projects
- Architecture comparisons
- Lessons learned
- Code examples

### 6. [Terminal Multiplexers](./06-terminal-multiplexers.md)
- Lightweight C-based options (mtm, dvtm, abduco)
- Go-based multiplexers (3mux)
- Industry standards (tmux, screen)
- Comparison matrix
- Bundling considerations

### 7. [Go Package Ecosystem Search](./07-go-packages-search.md)
- Exhaustive package search results
- Why nothing exists in pure Go
- Categories analyzed
- Alternative approaches
- Community validation

### 8. [Final Recommendations](./08-recommendations.md) ✅
- Decision framework
- Implementation plan with code
- Comparison matrix
- Risk mitigation
- Next steps

---

## Key Research Findings

### 1. No Pure Go Solution Exists ❌

After searching:
- pkg.go.dev (all packages)
- GitHub (extensive search)  
- awesome-go lists
- Real-world projects

**Result**: No Go library provides session management with attach/detach

### 2. Existing Options 🔧

| Option | Time | Code | Reliability | Maintenance |
|--------|------|------|-------------|-------------|
| **abduco (bundled)** | Hours | ~100 LOC | Proven | Minimal |
| **tmux (shell out)** | Hours | ~100 LOC | Proven | Minimal |
| **Custom (build)** | Weeks | ~1000 LOC | Unknown | Ongoing |

### 3. Recommended Approach ⭐

**Use abduco by bundling the static binary**

**Why**:
- ✅ 2-4 hour implementation
- ✅ 50KB binary size (negligible)
- ✅ 10+ years of production use
- ✅ Perfect fit for requirements
- ✅ Minimal maintenance

**Implementation**: See [recommendations document](./08-recommendations.md)

---

## Quick Reference

### For Readers Who Want...

**"Just tell me what to use"**  
→ [Final Recommendations](./08-recommendations.md)

**"How does abduco work?"**  
→ [abduco Detailed Analysis](./03-abduco-detailed-analysis.md)

**"What if I want to build my own?"**  
→ [Go PTY Libraries](./04-go-pty-libraries.md)

**"What did other projects do?"**  
→ [Real-World Go Projects](./05-go-projects-analysis.md)

**"Why doesn't a Go package exist?"**  
→ [Go Package Ecosystem Search](./07-go-packages-search.md)

**"What are all my options?"**  
→ [Terminal Multiplexers](./06-terminal-multiplexers.md)

---

## Code Examples

### Using abduco (Recommended)

```go
// Create detached session
sessionMgr.Start("build-123", "make build")

// Attach to running session (brings to foreground)
tea.ExecProcess(sessionMgr.AttachCommand("build-123"), nil)

// List sessions
sessions, _ := sessionMgr.List()

// Kill session
sessionMgr.Kill("build-123")
```

Full implementation: [Recommendations](./08-recommendations.md#phase-1-binary-bundling-30-min)

### Using tmux (Alternative)

```go
// Create session
exec.Command("tmux", "new-session", "-d", "-s", "name", "cmd").Run()

// Attach
exec.Command("tmux", "attach", "-t", "name").Run()
```

---

## Research Methodology

### Search Coverage

1. **Package Registries**
   - pkg.go.dev (all Go packages)
   - Go module index
   - awesome-go lists

2. **GitHub**
   - Keyword searches: "go session manager", "go pty attach", etc.
   - Star filters (>100 stars)
   - Recent activity filters

3. **Real Projects**
   - Bubble Tea applications
   - Process managers
   - Terminal multiplexers

4. **Web Research**
   - abduco documentation
   - tmux/screen comparison
   - PTY handling guides

### Parallel Research

Used 4 parallel research agents exploring:
1. abduco capabilities and integration
2. Go PTY libraries and session management
3. Real-world Go projects with process management
4. Terminal multiplexers and alternatives

---

## For beans Implementation

### Requirements Met ✅

- ✅ Run interactive CLI tools (PTY support)
- ✅ Detach from running processes
- ✅ Reattach to running processes
- ✅ Full terminal control when attached
- ✅ Multiple simultaneous processes
- ✅ Processes survive TUI exit
- ✅ Processes do NOT survive system reboot (as designed)
- ✅ Simple, built-in feel

### Implementation Checklist

- [ ] Compile abduco binaries (Linux, macOS)
- [ ] Embed binaries in Go app
- [ ] Write session manager wrapper
- [ ] Integrate with Bubble Tea TUI
- [ ] Add keyboard shortcuts (r=run, a=attach, k=kill)
- [ ] Test on multiple platforms
- [ ] Document for users

**Time estimate**: 2-4 hours  
**See**: [Implementation Plan](./08-recommendations.md#implementation-plan)

---

## Contributing

This research is complete and documented. Future updates might cover:
- Windows support (ConPTY)
- Alternative session managers
- Performance comparisons
- User feedback integration

---

## License

This documentation is part of the beans project.

---

**Start reading**: [Overview →](./01-subprocess-management-overview.md)  
**Skip to solution**: [Final Recommendations →](./08-recommendations.md)
