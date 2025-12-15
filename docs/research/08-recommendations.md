# Final Recommendations

## Executive Summary

After comprehensive research, we have **three viable options** for implementing detachable/attachable subprocess management in beans:

1. **abduco (bundled)** - Recommended ⭐
2. **tmux (shell out)** - Alternative
3. **Custom (build)** - Advanced

## The Winning Option: abduco (Bundled)

### Why abduco?

✅ **Perfect fit for requirements**:
- Interactive CLI tools ← Has full PTY support
- Detach/attach ← Core functionality
- Multiple sessions ← Built-in
- Simple ← Minimal, focused
- Built-in feel ← Can bundle 50KB binary

✅ **Advantages over alternatives**:
- Lighter than tmux (~50KB vs ~1MB)
- Permissive license (ISC)
- Battle-tested (10+ years)
- Easy to integrate (~50 LOC wrapper)

✅ **No pure Go alternative exists**:
- Extensive ecosystem search found nothing
- Building custom = 1-2 weeks + maintenance

### Implementation Complexity

**Time to implement**: 2-4 hours

**Code volume**: ~100 lines

**Components**:
1. Binary extraction (~20 LOC)
2. Session manager wrapper (~50 LOC)
3. Bubble Tea integration (~30 LOC)

### Implementation Plan

#### Phase 1: Binary Bundling (30 min)

```go
// internal/session/abduco.go
package session

import (
    _ "embed"
    "os"
    "path/filepath"
    "runtime"
)

//go:embed bin/abduco-linux-amd64
var abducoLinux []byte

//go:embed bin/abduco-darwin-amd64
var abducoDarwin []byte

//go:embed bin/abduco-darwin-arm64
var abducoDarwinARM []byte

func extractAbduco() (string, error) {
    cacheDir, _ := os.UserCacheDir()
    binDir := filepath.Join(cacheDir, "beans", "bin")
    abducoPath := filepath.Join(binDir, "abduco")
    
    // Check if exists
    if _, err := os.Stat(abducoPath); err == nil {
        return abducoPath, nil
    }
    
    // Create directory
    os.MkdirAll(binDir, 0755)
    
    // Select binary
    var data []byte
    switch runtime.GOOS {
    case "linux":
        data = abducoLinux
    case "darwin":
        if runtime.GOARCH == "arm64" {
            data = abducoDarwinARM
        } else {
            data = abducoDarwin
        }
    default:
        return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
    }
    
    // Write binary
    if err := os.WriteFile(abducoPath, data, 0755); err != nil {
        return "", err
    }
    
    return abducoPath, nil
}
```

#### Phase 2: Session Manager (1 hour)

```go
// internal/session/manager.go
package session

import (
    "bufio"
    "bytes"
    "fmt"
    "os/exec"
    "strings"
    tea "github.com/charmbracelet/bubbletea"
)

type Manager struct {
    abducoPath string
}

type Session struct {
    Name       string
    Status     string  // "*" active, "+" terminated, " " detached
    HasClients bool
}

func NewManager() (*Manager, error) {
    path, err := extractAbduco()
    if err != nil {
        return nil, err
    }
    return &Manager{abducoPath: path}, nil
}

// Create detached session
func (m *Manager) Start(name, command string) error {
    cmd := exec.Command(m.abducoPath, "-n", name, "sh", "-c", command)
    return cmd.Run()
}

// Get attach command (for use with tea.ExecProcess)
func (m *Manager) AttachCommand(name string) *exec.Cmd {
    return exec.Command(m.abducoPath, "-a", name)
}

// List all sessions
func (m *Manager) List() ([]Session, error) {
    cmd := exec.Command(m.abducoPath)
    out, err := cmd.Output()
    if err != nil {
        return nil, err
    }
    return parseSessions(out), nil
}

// Check if session exists
func (m *Manager) Exists(name string) (bool, error) {
    sessions, err := m.List()
    if err != nil {
        return false, err
    }
    for _, s := range sessions {
        if s.Name == name {
            return true, nil
        }
    }
    return false, nil
}

// Kill session
func (m *Manager) Kill(name string) error {
    // Send exit command via pass-through mode
    cmd := exec.Command(m.abducoPath, "-p", name)
    cmd.Stdin = strings.NewReader("exit\n")
    return cmd.Run()
}

func parseSessions(output []byte) []Session {
    var sessions []Session
    scanner := bufio.NewScanner(bytes.NewReader(output))
    
    // Skip header
    if !scanner.Scan() {
        return sessions
    }
    
    for scanner.Scan() {
        line := scanner.Text()
        if len(line) == 0 {
            continue
        }
        
        status := string(line[0])
        fields := strings.Fields(line)
        if len(fields) < 5 {
            continue
        }
        
        name := strings.Join(fields[5:], " ")
        
        sessions = append(sessions, Session{
            Name:       name,
            Status:     status,
            HasClients: status == "*",
        })
    }
    
    return sessions
}
```

#### Phase 3: Bubble Tea Integration (1 hour)

```go
// internal/tui/tui.go
type model struct {
    sessionMgr *session.Manager
    beans      []bean.Bean
    selected   int
}

func (m model) Init() tea.Cmd {
    // Initialize session manager
    mgr, err := session.NewManager()
    if err != nil {
        return func() tea.Msg { return errMsg{err} }
    }
    m.sessionMgr = mgr
    
    return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "r": // Run build
            bean := m.beans[m.selected]
            sessionName := "bean-" + bean.ID
            
            // Start in background
            if err := m.sessionMgr.Start(sessionName, "make build"); err != nil {
                return m, func() tea.Msg { return errMsg{err} }
            }
            
            return m, nil
            
        case "a": // Attach to build
            bean := m.beans[m.selected]
            sessionName := "bean-" + bean.ID
            
            // Check if session exists
            exists, _ := m.sessionMgr.Exists(sessionName)
            if !exists {
                return m, func() tea.Msg { 
                    return errMsg{fmt.Errorf("no running build")} 
                }
            }
            
            // Attach using tea.ExecProcess
            return m, tea.ExecProcess(
                m.sessionMgr.AttachCommand(sessionName),
                func(err error) tea.Msg {
                    return buildDetachedMsg{sessionName, err}
                },
            )
            
        case "k": // Kill build
            bean := m.beans[m.selected]
            sessionName := "bean-" + bean.ID
            
            if err := m.sessionMgr.Kill(sessionName); err != nil {
                return m, func() tea.Msg { return errMsg{err} }
            }
            
            return m, nil
        }
    
    case buildDetachedMsg:
        // User detached from build (pressed Ctrl+\)
        return m, nil
    }
    
    return m, nil
}

type buildDetachedMsg struct {
    session string
    err     error
}
```

#### Phase 4: Build Process (30 min)

```bash
# scripts/build-abduco.sh

#!/bin/bash
set -e

# Clone abduco
git clone https://github.com/martanne/abduco /tmp/abduco
cd /tmp/abduco

# Build for each platform
echo "Building abduco for Linux amd64..."
make clean
make LDFLAGS=-static
strip abduco
cp abduco $REPO/internal/session/bin/abduco-linux-amd64

echo "Building abduco for macOS amd64..."
# Must run on macOS or use cross-compiler
# On macOS:
make clean
make
strip abduco
cp abduco $REPO/internal/session/bin/abduco-darwin-amd64

echo "Building abduco for macOS arm64..."
# On Apple Silicon Mac:
make clean
make
strip abduco
cp abduco $REPO/internal/session/bin/abduco-darwin-arm64

echo "Done!"
```

### Testing Plan

```go
// internal/session/manager_test.go

func TestSessionManager(t *testing.T) {
    mgr, err := NewManager()
    require.NoError(t, err)
    
    // Test start
    err = mgr.Start("test-session", "sleep 10")
    require.NoError(t, err)
    
    // Test exists
    exists, err := mgr.Exists("test-session")
    require.NoError(t, err)
    assert.True(t, exists)
    
    // Test list
    sessions, err := mgr.List()
    require.NoError(t, err)
    assert.Len(t, sessions, 1)
    assert.Equal(t, "test-session", sessions[0].Name)
    
    // Test kill
    err = mgr.Kill("test-session")
    require.NoError(t, err)
    
    // Cleanup
    time.Sleep(100 * time.Millisecond)
}
```

## Alternative: tmux (Shell Out)

### When to Choose

- Users likely have tmux installed
- Don't want to bundle binaries
- Okay with heavier dependency

### Implementation

```go
type TmuxManager struct{}

func (m *TmuxManager) Start(name, command string) error {
    cmd := exec.Command("tmux", "new-session", "-d", "-s", name, command)
    return cmd.Run()
}

func (m *TmuxManager) AttachCommand(name string) *exec.Cmd {
    return exec.Command("tmux", "attach-session", "-t", name)
}

func (m *TmuxManager) List() ([]Session, error) {
    out, err := exec.Command("tmux", "list-sessions").Output()
    // Parse tmux format
}
```

**Pros**: No bundling, full-featured  
**Cons**: Must be installed, ~1MB binary

## Advanced: Custom Implementation

### When to Choose

- Need Windows support
- Want complete control
- Have 1-2 weeks available
- Enjoy building infrastructure

### What You'd Build

1. **Daemon Process** (~300 LOC)
   - Unix socket server
   - Session management
   - PTY lifecycle

2. **Client Library** (~200 LOC)
   - Protocol implementation
   - Connection handling

3. **PTY Management** (~200 LOC)
   - Using creack/pty
   - Resize handling
   - I/O multiplexing

4. **State Persistence** (~150 LOC)
   - SQLite storage
   - Session recovery

5. **Protocol** (~150 LOC)
   - Message format
   - Command handling

**Total**: ~1000 LOC + extensive testing

**Time**: 1-2 weeks + ongoing maintenance

## Comparison Matrix

| Criterion | abduco | tmux | Custom |
|-----------|--------|------|--------|
| **Implementation time** | 2-4 hours | 2-4 hours | 1-2 weeks |
| **Code volume** | ~100 LOC | ~100 LOC | ~1000 LOC |
| **Binary size** | +50KB | 0 (external) | 0 |
| **External dependency** | Bundled | User installs | None |
| **Flexibility** | Medium | Low | Complete |
| **Maintenance** | Minimal | Minimal | Ongoing |
| **Reliability** | Proven | Proven | Unknown |
| **Platform support** | Unix | Unix | Unix+Windows* |
| **Feature richness** | Minimal | Full | Custom |

*Windows support requires additional work with ConPTY

## Decision Framework

### Choose abduco if:
- ✅ Timeline is short (want to ship soon)
- ✅ 50KB binary size is acceptable
- ✅ Targeting Unix systems only
- ✅ Want proven, reliable solution
- ✅ Prefer minimal maintenance

### Choose tmux if:
- ✅ Users have tmux installed
- ✅ Don't want to bundle binaries
- ✅ Want full multiplexing features
- ✅ Need complex session management

### Choose custom if:
- ✅ Need Windows support
- ✅ Want complete control
- ✅ Have time (1-2 weeks)
- ✅ Enjoy infrastructure work
- ✅ Need unique features

## Recommended Path for beans

### Phase 1: Start with abduco (Now)

**Timeline**: 1 day  
**Deliverable**: Working detach/attach

**Why**:
- Fastest path to value
- Proven solution
- Easy to implement
- Low maintenance

### Phase 2: Gather Feedback (Weeks 1-4)

**Questions to answer**:
- Do users actually use detach/attach?
- Are there feature gaps?
- Any platform issues?
- Performance concerns?

### Phase 3: Iterate or Replace (Month 2+)

**If abduco works well**: Keep it, invest elsewhere  
**If limitations found**: Consider custom implementation  
**If Windows needed**: Build custom with ConPTY support

## Implementation Checklist

### Week 1: Core Implementation

- [ ] Compile abduco for Linux (amd64)
- [ ] Compile abduco for macOS (amd64, arm64)
- [ ] Create session package structure
- [ ] Implement binary extraction
- [ ] Implement Manager wrapper
- [ ] Write unit tests
- [ ] Integrate with Bubble Tea TUI

### Week 2: Polish & Documentation

- [ ] Add error handling
- [ ] Improve status indicators
- [ ] Document usage
- [ ] Add examples
- [ ] Test on different platforms
- [ ] Update README

### Week 3: Ship & Iterate

- [ ] Release to users
- [ ] Gather feedback
- [ ] Fix bugs
- [ ] Consider enhancements

## Risk Mitigation

### Risk: Binary compatibility issues

**Mitigation**:
- Static compilation of abduco
- Test on multiple platforms
- Fallback to user-installed abduco

### Risk: Users find limitations

**Mitigation**:
- Document detach key (Ctrl+\)
- Provide clear error messages
- Consider tmux fallback option

### Risk: Platform support gaps

**Mitigation**:
- Check OS in code
- Provide helpful error messages
- Document supported platforms

## Conclusion

**Recommended approach**: Start with abduco (bundled)

**Rationale**:
1. ✅ Fastest path (hours, not weeks)
2. ✅ Proven reliability (10+ years)
3. ✅ Perfect fit for requirements
4. ✅ Minimal maintenance burden
5. ✅ Can iterate based on feedback

**Next steps**:
1. Compile abduco binaries
2. Implement session manager wrapper
3. Integrate with beans TUI
4. Ship and gather feedback

**Success criteria**:
- Users can start builds in background
- Users can attach to running builds
- Users can detach without killing builds
- Multiple builds can run simultaneously

---

**Previous**: [← Go Packages Search](./07-go-packages-search.md)  
**Index**: [↑ Overview](./01-subprocess-management-overview.md)
