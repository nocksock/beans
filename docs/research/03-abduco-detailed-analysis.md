# abduco: Detailed Analysis

## What is abduco?

**abduco** is a lightweight session management tool written in C that provides attach/detach functionality for terminal applications. It's conceptually similar to GNU Screen or tmux but with a much simpler, more minimal design.

**Repository**: https://github.com/martanne/abduco  
**License**: ISC (very permissive)  
**Binary Size**: ~50KB (static build)  
**Lines of Code**: ~2000 LOC

## Architecture

### Client-Server Model

```
┌─────────────┐                    ┌──────────────┐                    ┌─────────────┐
│   User's    │  Unix Socket       │    abduco    │    PTY             │   Command   │
│  Terminal   │◄──────────────────►│    Server    │◄──────────────────►│  (bash,app) │
└─────────────┘                    └──────────────┘                    └─────────────┘
      ↑                                    ↑
      │                                    │
   stdin/out                            PTY master
   (raw mode)                          (line buffered)
```

### Components

1. **Server Process**:
   - Spawns user command in a PTY
   - Daemonizes itself
   - Listens on Unix domain socket
   - Manages session lifecycle

2. **Client Process**:
   - Connects to server socket
   - Relays I/O between terminal and PTY
   - Handles user input (including detach key)

3. **Session State**:
   - Socket file in `~/.abduco/` or `/tmp/abduco/$USER/`
   - Exit status stored for terminated sessions
   - No terminal state preservation (by design)

## How Attach/Detach Works

### Creating a Session

```bash
abduco -c session-name command
```

**Process**:
1. Forks server process
2. Server creates PTY using `forkpty()`
3. Server creates Unix domain socket
4. Server spawns command in PTY
5. Client connects and relays I/O

### Detaching

**Default key**: `Ctrl+\`

**Process**:
1. Client catches detach key
2. Sends `MSG_DETACH` packet to server
3. Client closes socket and exits
4. Server keeps running with command
5. Socket remains available

### Reattaching

```bash
abduco -a session-name
```

**Process**:
1. New client connects to existing socket
2. Server sends session info
3. Client sends terminal size
4. Server adjusts PTY size via `SIGWINCH`
5. I/O relay resumes

### Multiple Clients

- Multiple clients can attach simultaneously
- All receive the same output
- Any can send input
- Read-only mode available with `-r` flag

## Command Reference

### Basic Operations

```bash
# List all sessions
abduco

# Create and attach
abduco -c session-name command

# Create without attaching
abduco -n session-name command

# Attach to existing
abduco -a session-name

# Create or attach (if exists)
abduco -A session-name command

# Read-only attach
abduco -r -a session-name

# Custom detach key (Ctrl+Q)
abduco -e ^q -a session-name

# Pass-through mode (send stdin, get output, exit)
abduco -p session-name
```

### Session Status Indicators

When listing sessions:
- `*` = Active (has connected clients)
- `+` = Terminated (command exited, status available)
- ` ` = Detached (no clients, still running)

## Go Integration

### Basic Wrapper

```go
package session

import (
    "os/exec"
    "bytes"
    "bufio"
    "strings"
)

type Session struct {
    Name       string
    PID        int
    Status     string
    HasClients bool
}

type Manager struct {
    abducoPath string
}

func NewManager(abducoPath string) *Manager {
    return &Manager{abducoPath: abducoPath}
}

// Create detached session
func (m *Manager) Create(name, command string) error {
    cmd := exec.Command(m.abducoPath, "-n", name, "sh", "-c", command)
    return cmd.Run()
}

// Attach to session (returns exec.Cmd for use with tea.ExecProcess)
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
    
    return m.parseSessions(out)
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

// Parse abduco output
func (m *Manager) parseSessions(output []byte) ([]Session, error) {
    var sessions []Session
    scanner := bufio.NewScanner(bytes.NewReader(output))
    
    // Skip header
    if !scanner.Scan() {
        return sessions, nil
    }
    
    for scanner.Scan() {
        line := scanner.Text()
        // Parse: [*+ ] date time PID name
        // Example: "*  Mon 2024-12-15 10:30:00  12345  my-session"
        
        fields := strings.Fields(line)
        if len(fields) < 5 {
            continue
        }
        
        status := fields[0]
        pid := fields[4]
        name := strings.Join(fields[5:], " ")
        
        sessions = append(sessions, Session{
            Name:       name,
            Status:     status,
            HasClients: status == "*",
        })
    }
    
    return sessions, nil
}
```

### Bubble Tea Integration

```go
import tea "github.com/charmbracelet/bubbletea"

type model struct {
    sessionMgr *Manager
    sessions   []Session
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "r": // run build
            beanID := m.getCurrentBeanID()
            sessionName := "bean-" + beanID
            
            // Start in background
            if err := m.sessionMgr.Create(sessionName, "make build"); err != nil {
                // handle error
            }
            return m, m.refreshSessions()
            
        case "a": // attach to build
            beanID := m.getCurrentBeanID()
            sessionName := "bean-" + beanID
            
            // Attach using tea.ExecProcess
            return m, tea.ExecProcess(
                m.sessionMgr.AttachCommand(sessionName),
                func(err error) tea.Msg {
                    return buildDetachedMsg{err: err}
                },
            )
        }
    }
    return m, nil
}
```

## Bundling Strategy

### Static Compilation

```bash
# Clone abduco
git clone https://github.com/martanne/abduco
cd abduco

# Compile statically
make LDFLAGS=-static

# Result: ~50KB static binary
ls -lh abduco
```

### Embedding in Go

```go
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
    // Get cache directory
    cacheDir, err := os.UserCacheDir()
    if err != nil {
        cacheDir = os.TempDir()
    }
    
    abducoDir := filepath.Join(cacheDir, "beans", "bin")
    abducoPath := filepath.Join(abducoDir, "abduco")
    
    // Check if already extracted
    if _, err := os.Stat(abducoPath); err == nil {
        return abducoPath, nil
    }
    
    // Create directory
    if err := os.MkdirAll(abducoDir, 0755); err != nil {
        return "", err
    }
    
    // Select binary for platform
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
        return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
    }
    
    // Write binary
    if err := os.WriteFile(abducoPath, data, 0755); err != nil {
        return "", err
    }
    
    return abducoPath, nil
}
```

## Pros and Cons

### Advantages ✅

- **Minimal**: Only ~2000 LOC, easy to understand
- **Permissive license**: ISC (like BSD)
- **Small binary**: ~50KB static
- **Battle-tested**: 10+ years of production use
- **Simple integration**: Just shell out with exec.Command()
- **Perfect fit**: Does exactly what we need, nothing more
- **Multiple clients**: Can attach from multiple terminals
- **Read-only mode**: Safe observation of running sessions

### Disadvantages ⚠️

- **External dependency**: Not pure Go (but no Go alternative exists)
- **Unix-only**: No Windows support (relies on Unix sockets and PTY)
- **Limited distribution**: Not as widely available as tmux
- **No multiplexing**: One command per session (but we don't need this)
- **No scrollback**: Doesn't preserve terminal history (by design)
- **Socket-based**: Requires filesystem access

## Comparison with Alternatives

| Feature | abduco | tmux | screen | Custom Go |
|---------|--------|------|--------|-----------|
| Binary size | 50KB | ~1MB | ~800KB | 0 (embedded) |
| Session mgmt | ✅ | ✅ | ✅ | DIY |
| Multiple panes | ❌ | ✅ | ✅ | DIY |
| Scrollback | ❌ | ✅ | ✅ | DIY |
| Availability | Limited | High | High | N/A |
| Simplicity | High | Low | Medium | N/A |
| License | ISC | ISC | GPL | N/A |

## Platform Availability

### Package Managers

```bash
# Arch Linux
pacman -S abduco

# Alpine Linux
apk add abduco

# Fedora/RHEL
dnf install abduco

# FreeBSD
pkg install abduco

# macOS (Homebrew)
brew install abduco

# From source (any Linux)
git clone https://github.com/martanne/abduco
cd abduco && make && sudo make install
```

### Distribution Status

- ✅ Available: Arch, Alpine, Fedora, FreeBSD, macOS (Homebrew)
- ⚠️ Old version: Debian, Ubuntu (0.1 vs 0.6 current)
- ❌ Not default: Most distributions require manual install

## Recommendation

**Use abduco by bundling the static binary.** It's the simplest, most reliable solution that exactly fits our use case.

---

**Previous**: [← Bubble Tea Subprocess Handling](./02-bubbletea-subprocess-handling.md)  
**Next**: [Go PTY Libraries →](./04-go-pty-libraries.md)
