# Real-World Go Projects with Process Management

## Overview

Analysis of existing Go projects that implement or require process detachment/attachment capabilities.

## Key Finding

**Most Go projects shell out to tmux or screen** rather than implementing custom session management. No widely-used library provides this functionality.

## Notable Projects

### 1. gob - Process Manager with Bubble Tea TUI ⭐ MOST RELEVANT

**Repository**: https://github.com/juanibiapina/gob  
**Relevance**: 🟢 Extremely High - Exact same use case!

**What it is**:
- Go-based process manager with daemon architecture
- Bubble Tea TUI for interaction
- Background job management
- Real-time output streaming

**Architecture**:
```
CLI/TUI Client <-> Unix Socket <-> Daemon Process
                                      |
                                  Job Manager
                                      |
                                  SQLite State
```

**Key Features**:
- **Daemon process**: Long-running background server
- **Multiple clients**: CLI and TUI can connect simultaneously  
- **Job persistence**: SQLite-backed state survives restarts
- **Real-time sync**: Changes appear instantly across clients
- **MCP server**: AI agents can control jobs

**How They Solve Detach/Attach**:
1. Jobs run in daemon with dedicated PTYs
2. Clients connect via Unix socket
3. Output buffering allows log "replay"
4. Clean separation: daemon, jobs, UI

**Code Structure**:
```
gob/
├── cmd/
│   ├── daemon.go      # Daemon process
│   ├── client.go      # CLI client
│   └── tui.go         # Bubble Tea UI
├── internal/
│   ├── job/           # Job management
│   ├── daemon/        # Daemon server
│   ├── protocol/      # Client-server protocol
│   └── storage/       # SQLite persistence
```

**Lessons for beans**:
- ✅ Daemon architecture works well
- ✅ Unix socket IPC is simple and reliable
- ✅ SQLite provides good persistence
- ✅ Bubble Tea integrates cleanly
- ⚠️ Still significant code complexity (~3000+ LOC)

**Architecture Diagram**:
```
┌─────────────┐
│ beans TUI   │
└──────┬──────┘
       │
       │ Unix Socket
       │
┌──────▼──────┐     ┌─────────────┐
│   Daemon    │────▶│   SQLite    │
│   Server    │     │  (state DB) │
└──────┬──────┘     └─────────────┘
       │
       │ Manages
       │
   ┌───▼────┐ ┌────────┐ ┌────────┐
   │ Job 1  │ │ Job 2  │ │ Job 3  │
   │  PTY   │ │  PTY   │ │  PTY   │
   └────────┘ └────────┘ └────────┘
```

### 2. Terminal Multiplexers as Tools

Most projects use terminal multiplexers programmatically:

#### wezterm (Rust, but instructive)
- Embeds terminal emulator
- Full PTY control
- Complex implementation (~100k LOC)

#### alacritty
- Terminal emulator only
- No multiplexing
- PTY handling code is complex

### 3. Process Supervisors (No Interactive Attach)

#### ShinyTrinkets/overseer
**Repository**: https://github.com/ShinyTrinkets/overseer  
**Stars**: 124

**Features**:
- Process lifecycle management
- Real-time log streaming
- Auto-restart with backoff
- State watching

**What it doesn't do**:
- ❌ No interactive attach (no PTY)
- ❌ No terminal control
- ❌ Captures output only

**Use case**: Background services, not interactive tools

#### kokizzu/goproc  
**Repository**: https://github.com/kokizzu/goproc  
**Stars**: 26

Similar to overseer - manages processes but no interactive attach.

### 4. Container-based Approaches

Some projects use containers for isolation:

#### Docker attach
```bash
docker run -it --name mysession ubuntu bash
# Detach: Ctrl+P Ctrl+Q
docker attach mysession
```

**Pros**:
- Strong isolation
- Well-tested
- Network transparency

**Cons**:
- Heavy dependency
- Requires Docker daemon
- Overkill for simple use case

### 5. SSH-based Remote Execution

#### upterm
**Repository**: https://github.com/owenthereal/upterm  
**Stars**: 1,100+

- Secure terminal sharing
- SSH-based
- GitHub user authentication
- Not for local session management

#### tmate
- Fork of tmux
- Instant terminal sharing
- SSH-based

## Common Patterns Observed

### Pattern 1: Shell Out to tmux/screen

**Most common approach**:
```go
// Create session
exec.Command("tmux", "new-session", "-d", "-s", "name", "cmd").Run()

// Attach
exec.Command("tmux", "attach", "-t", "name").Run()

// List
exec.Command("tmux", "list-sessions").Output()
```

**Projects using this**: sesh, smug, many others

### Pattern 2: Daemon with Unix Sockets

**Used by**: gob, spm

```go
// Daemon listens
listener, _ := net.Listen("unix", socketPath)

// Client connects
conn, _ := net.Dial("unix", socketPath)

// Protocol: JSON messages
json.NewEncoder(conn).Encode(message)
```

### Pattern 3: Web-based Terminal

**Used by**: gotty, ttyd

```go
// WebSocket + xterm.js
upgrader.Upgrade(w, r, nil)
ws.WriteMessage(websocket.TextMessage, output)
```

## Architecture Comparison

### gob Architecture (Daemon + TUI)

**Complexity**: Medium-High (~3000 LOC)

**Components**:
1. Daemon server (Unix socket)
2. Job manager (PTY per job)
3. SQLite storage
4. Protocol layer
5. CLI client
6. TUI client

**Pros**:
- ✅ True multi-client support
- ✅ Persistent state
- ✅ Clean separation
- ✅ Extensible

**Cons**:
- ⚠️ Complex setup
- ⚠️ More code to maintain
- ⚠️ Daemon lifecycle management

### abduco Architecture (Simple Shell-out)

**Complexity**: Low (~50 LOC)

**Components**:
1. Wrapper functions
2. Output parsing

**Pros**:
- ✅ Minimal code
- ✅ Battle-tested
- ✅ Simple deployment

**Cons**:
- ⚠️ External binary
- ⚠️ Less flexibility

## Bubble Tea Integration Patterns

### Pattern 1: TUI as Client (gob style)

```go
type App struct {
    daemonConn net.Conn
    jobs       []Job
}

func (a *App) Init() tea.Cmd {
    return tea.Batch(
        a.connectDaemon(),
        a.pollJobs(),
    )
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case jobsMsg:
        a.jobs = msg.jobs
    case startJobMsg:
        a.sendCommand(StartJob{ID: msg.id})
    }
}
```

### Pattern 2: TUI Shells Out (abduco style)

```go
type App struct {
    sessionMgr *SessionManager
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "a" { // attach
            return a, tea.ExecProcess(
                a.sessionMgr.AttachCmd(sessionID),
                nil,
            )
        }
    }
}
```

## Lessons Learned

### 1. Session Management is Complex

Real-world projects either:
- Use existing tools (tmux/screen)
- Implement full daemon architecture
- Or don't support true detach/attach

### 2. Daemon Architecture is Standard

Projects needing multi-client support use:
- Long-running daemon process
- Unix socket for IPC
- State persistence (SQLite/files)

### 3. PTY Management is Tricky

Projects that handle PTYs directly:
- Have extensive edge case handling
- Deal with signal management
- Handle terminal state carefully

### 4. Simple Solutions Win

Most projects prefer:
- Shelling out to proven tools
- Over custom implementations
- Trade flexibility for reliability

## Recommendations Based on Real Projects

### For beans Specifically

**Option 1: Follow gob's Architecture** (if you want full control)
- Implement daemon with Unix socket
- Use SQLite for persistence
- Full integration with Bubble Tea
- **Time**: 1-2 weeks
- **Code**: ~3000 LOC

**Option 2: Shell Out to abduco** (simplest)
- Wrap abduco commands
- Minimal code
- **Time**: 1 day
- **Code**: ~50-100 LOC

**Option 3: Shell Out to tmux** (most available)
- Use tmux programmatically
- Widely installed
- **Time**: 1 day  
- **Code**: ~50-100 LOC

## Code Example: gob-style Architecture

```go
// Daemon
type Daemon struct {
    jobs     map[string]*Job
    listener net.Listener
    db       *sql.DB
}

func (d *Daemon) Start() error {
    l, _ := net.Listen("unix", "/tmp/beans.sock")
    for {
        conn, _ := l.Accept()
        go d.handleClient(conn)
    }
}

// Client (in TUI)
type Client struct {
    conn net.Conn
}

func (c *Client) StartJob(id, cmd string) error {
    msg := Message{Type: "start", Job: id, Cmd: cmd}
    return json.NewEncoder(c.conn).Encode(msg)
}

func (c *Client) AttachJob(id string) error {
    // This is where it gets complex...
    // Need to handle PTY I/O over socket
}
```

## Conclusion

**Real-world projects validate our findings**:
1. No standard Go library for session management
2. Most projects shell out to tmux/screen
3. Custom implementations require significant effort
4. Daemon architecture is proven for multi-client scenarios

**For beans**: Given the research, **abduco or tmux shell-out is the pragmatic choice**, unless you want to invest in a full daemon architecture like gob.

---

**Previous**: [← Go PTY Libraries](./04-go-pty-libraries.md)  
**Next**: [Terminal Multiplexers →](./06-terminal-multiplexers.md)
