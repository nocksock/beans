# Go PTY Libraries

## Overview

PTY (Pseudo-Terminal) libraries provide the foundation for terminal-based process interaction. While they enable creating and managing pseudo-terminals, **none provide built-in session management** (detach/attach functionality).

## Primary Library: github.com/creack/pty

### Status & Popularity

- **Stars**: 1,900+
- **Importers**: 24,500+ repositories
- **License**: MIT
- **Maintenance**: Actively maintained (last update Oct 2024)
- **Repository**: https://github.com/creack/pty

### What It Provides

1. **PTY Creation**: Create pseudo-terminal master/slave pairs
2. **Process Spawning**: Start commands attached to a PTY
3. **Window Size Management**: Handle terminal resizing (SIGWINCH)
4. **Cross-platform**: Linux, macOS, BSD, Solaris

### What It Does NOT Provide

- ❌ Session persistence
- ❌ Detach/attach functionality
- ❌ Multiple client support
- ❌ Output buffering/replay
- ❌ State management

### Basic Usage

```go
package main

import (
    "io"
    "os"
    "os/exec"
    "github.com/creack/pty"
)

func main() {
    // Start a command with PTY
    cmd := exec.Command("bash")
    ptmx, err := pty.Start(cmd)
    if err != nil {
        panic(err)
    }
    defer ptmx.Close()
    
    // Copy I/O
    go io.Copy(ptmx, os.Stdin)
    io.Copy(os.Stdout, ptmx)
    
    cmd.Wait()
}
```

### Core API

#### pty.Open()
```go
func Open() (pty, tty *os.File, err error)
```
Creates a new PTY master/slave pair.

#### pty.Start()
```go
func Start(cmd *exec.Cmd) (*os.File, error)
```
Starts a command attached to a new PTY. Returns the PTY master file.

#### pty.Getsize() / pty.Setsize()
```go
func Getsize(t *os.File) (rows, cols uint16, err error)
func Setsize(t *os.File, ws *Winsize) error
```
Get and set terminal window size.

#### pty.InheritSize()
```go
func InheritSize(dst, src *os.File) error
```
Copy window size from one terminal to another.

### Window Resize Handling

```go
import (
    "os"
    "os/signal"
    "syscall"
    "github.com/creack/pty"
)

func handleResize(ptmx *os.File) {
    ch := make(chan os.Signal, 1)
    signal.Notify(ch, syscall.SIGWINCH)
    
    go func() {
        for range ch {
            if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
                log.Printf("resize error: %v", err)
            }
        }
    }()
    
    // Initial size
    pty.InheritSize(os.Stdin, ptmx)
}
```

## Alternative PTY Libraries

### github.com/aymanbagabas/go-pty

- **Stars**: 57
- **Importers**: 5
- **Repository**: https://github.com/aymanbagabas/go-pty
- **Status**: Active (Jan 2024)
- **Difference**: Another PTY implementation, similar API to creack/pty

### github.com/charmbracelet/x/xpty

- **Stars**: 251 (parent repo)
- **Repository**: https://github.com/charmbracelet/x/tree/main/xpty
- **Status**: Active (Charm project)
- **What it adds**: Higher-level PTY wrapper with better cross-platform support
- **Platform support**: Includes Windows ConPTY support

```go
import "github.com/charmbracelet/x/xpty"

pty, err := xpty.New()
if err != nil {
    return err
}
defer pty.Close()

cmd := exec.Command("bash")
if err := pty.Start(cmd); err != nil {
    return err
}
```

## Building Session Management with PTY

To create detach/attach functionality, you'd need to combine PTY with:

### 1. Daemon Process

```go
type Daemon struct {
    sessions map[string]*Session
    listener net.Listener
}

type Session struct {
    name    string
    pty     *os.File
    cmd     *exec.Cmd
    output  *RingBuffer
    clients map[string]net.Conn
}

func (d *Daemon) Start() error {
    // Listen on Unix socket
    listener, err := net.Listen("unix", socketPath)
    if err != nil {
        return err
    }
    d.listener = listener
    
    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }
        go d.handleClient(conn)
    }
}
```

### 2. Session Storage

```go
func (d *Daemon) CreateSession(name, command string) error {
    // Create PTY
    cmd := exec.Command("sh", "-c", command)
    ptmx, err := pty.Start(cmd)
    if err != nil {
        return err
    }
    
    // Create session
    session := &Session{
        name:    name,
        pty:     ptmx,
        cmd:     cmd,
        output:  NewRingBuffer(1024 * 1024), // 1MB buffer
        clients: make(map[string]net.Conn),
    }
    
    d.sessions[name] = session
    
    // Start output capture
    go d.captureOutput(session)
    
    return nil
}
```

### 3. Output Buffering

```go
type RingBuffer struct {
    buf  []byte
    size int
    pos  int
    mu   sync.Mutex
}

func (rb *RingBuffer) Write(p []byte) (n int, err error) {
    rb.mu.Lock()
    defer rb.mu.Unlock()
    
    for _, b := range p {
        rb.buf[rb.pos] = b
        rb.pos = (rb.pos + 1) % rb.size
    }
    return len(p), nil
}

func (rb *RingBuffer) Read() []byte {
    rb.mu.Lock()
    defer rb.mu.Unlock()
    
    // Return buffered content
    return rb.buf
}
```

### 4. Client Protocol

```go
type Message struct {
    Type    string
    Session string
    Data    []byte
}

const (
    MsgCreate = "create"
    MsgAttach = "attach"
    MsgDetach = "detach"
    MsgData   = "data"
    MsgResize = "resize"
)

func (d *Daemon) handleClient(conn net.Conn) {
    decoder := json.NewDecoder(conn)
    encoder := json.NewEncoder(conn)
    
    for {
        var msg Message
        if err := decoder.Decode(&msg); err != nil {
            return
        }
        
        switch msg.Type {
        case MsgCreate:
            d.CreateSession(msg.Session, string(msg.Data))
        case MsgAttach:
            d.AttachClient(msg.Session, conn)
        case MsgData:
            d.SendToSession(msg.Session, msg.Data)
        }
    }
}
```

## Implementation Complexity

### Minimal Viable Session Manager

**Components needed**:
1. Daemon process (~200 LOC)
2. Unix socket server (~100 LOC)
3. PTY management per session (~150 LOC)
4. Output buffering (ring buffer) (~100 LOC)
5. Client protocol (~150 LOC)
6. Signal handling (~50 LOC)
7. Error recovery (~50 LOC)
8. Client library (~200 LOC)

**Total**: ~1000 lines of well-tested code

**Time estimate**: 1-2 weeks for robust implementation

### Edge Cases to Handle

1. **Session cleanup**: What happens when daemon crashes?
2. **Socket cleanup**: Handling stale socket files
3. **PTY size sync**: Multiple clients with different terminal sizes
4. **Output race conditions**: Multiple clients reading simultaneously
5. **Graceful shutdown**: Properly closing PTYs and processes
6. **Memory management**: Ring buffer overflow handling
7. **Client disconnection**: Detecting and cleaning up dead clients

## Comparison: Custom vs abduco

| Aspect | Custom (creack/pty) | abduco |
|--------|---------------------|--------|
| Code to write | ~1000 LOC | ~50 LOC wrapper |
| Time to implement | 1-2 weeks | 2 hours |
| Testing needed | Extensive | Minimal |
| Edge cases | You discover them | Already solved |
| Maintenance | Ongoing | Minimal |
| Platform support | Linux, macOS, BSD | Linux, macOS, BSD |
| Binary size impact | 0 (pure Go) | +50KB |
| Flexibility | Complete | Limited |
| Reliability | Unknown | Proven (10+ years) |

## When to Build Custom

Build a custom session manager if:

1. ✅ You need features abduco doesn't provide
2. ✅ You want Windows support (via ConPTY)
3. ✅ You need deep integration with your app
4. ✅ You want to avoid any external dependencies
5. ✅ You have time for proper implementation and testing
6. ✅ You enjoy building infrastructure

## Recommended Learning Path

If building custom:

1. **Start simple**: Get basic PTY creation working
2. **Add I/O**: Bidirectional copy between PTY and terminal
3. **Handle resize**: Implement SIGWINCH handling
4. **Build daemon**: Unix socket server
5. **Add protocol**: Client-server message format
6. **Implement sessions**: Create/attach/detach logic
7. **Buffer output**: Ring buffer for replay
8. **Test thoroughly**: All edge cases

## Code Example: Minimal PTY Manager

```go
package session

import (
    "io"
    "os/exec"
    "github.com/creack/pty"
)

type Manager struct {
    sessions map[string]*Session
}

type Session struct {
    name string
    pty  *os.File
    cmd  *exec.Cmd
}

func (m *Manager) Create(name, command string) error {
    cmd := exec.Command("sh", "-c", command)
    ptmx, err := pty.Start(cmd)
    if err != nil {
        return err
    }
    
    m.sessions[name] = &Session{
        name: name,
        pty:  ptmx,
        cmd:  cmd,
    }
    
    return nil
}

func (m *Manager) Attach(name string) error {
    session, exists := m.sessions[name]
    if !exists {
        return fmt.Errorf("session not found")
    }
    
    // This is where it gets complex:
    // - Need to handle terminal raw mode
    // - Need to capture Ctrl+\ for detach
    // - Need to handle resize
    // - Need to multiplex I/O if multiple clients
    
    go io.Copy(session.pty, os.Stdin)
    io.Copy(os.Stdout, session.pty)
    
    return nil
}
```

**Note**: The above is a simplified example. A production implementation requires significantly more code for proper detach handling, cleanup, etc.

## Conclusion

**creack/pty is excellent for PTY operations**, but building session management on top requires substantial additional work. For the beans use case, **abduco provides the complete solution** with proven reliability.

---

**Previous**: [← abduco Analysis](./03-abduco-detailed-analysis.md)  
**Next**: [Go Projects Analysis →](./05-go-projects-analysis.md)
