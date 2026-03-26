# Go Package Ecosystem Search Results

## Summary

Exhaustive search of pkg.go.dev, GitHub, and awesome-go for packages providing session management with attach/detach capabilities.

## Key Finding

**No Go package provides abduco/dtach-style session management.**

The ecosystem has:
- ✅ PTY libraries (create pseudo-terminals)
- ✅ Process supervisors (manage background processes)
- ✅ Daemon helpers (daemonize processes)
- ❌ **Session managers with attach/detach** ← THIS DOESN'T EXIST

## Categories Searched

### 1. PTY Libraries

#### github.com/creack/pty ⭐ PRIMARY

- **Importers**: 24,500+
- **Stars**: 1,900+
- **Maintenance**: Active
- **What it does**: Create and manage pseudo-terminals
- **What it doesn't do**: Session persistence, attach/detach

```go
import "github.com/creack/pty"

cmd := exec.Command("bash")
ptmx, err := pty.Start(cmd)
// ptmx is *os.File representing PTY master
```

#### github.com/aymanbagabas/go-pty

- **Importers**: 5
- **Similar to creack/pty**

#### github.com/charmbracelet/x/xpty

- **Part of Charm ecosystem**
- **Cross-platform PTY (including Windows ConPTY)**
- **Still no session management**

### 2. Process Supervisors

#### github.com/ShinyTrinkets/overseer

- **Stars**: 124
- **Purpose**: Process lifecycle management
- **Features**:
  - Start/stop processes
  - Real-time log streaming
  - Auto-restart with backoff
  - State watching

**What it doesn't have**:
- ❌ Interactive attach (no PTY)
- ❌ User input to process
- ❌ Terminal control

**Use case**: Background services, not interactive tools

```go
import "github.com/ShinyTrinkets/overseer"

cmd := overseer.NewCmd("bash")
cmd.Start()
// Can watch logs, but can't interact
```

#### github.com/kokizzu/goproc

- **Stars**: 26
- **Similar limitations to overseer**
- **Good for**: Daemon management
- **Not good for**: Interactive sessions

#### github.com/rainycape/governator

- **Stars**: 44
- **Purpose**: Production process supervision
- **Similar limitations**: No interactive attach

### 3. Daemon Helpers

#### github.com/sevlyar/go-daemon

- **Importers**: 607
- **Purpose**: Daemonize Go processes

```go
import "github.com/sevlyar/go-daemon"

ctx := &daemon.Context{
    PidFileName: "sample.pid",
    LogFileName: "sample.log",
}
d, err := ctx.Reborn()
```

**What it does**: Make your process a daemon  
**What it doesn't do**: Session management

#### github.com/takama/daemon

- **Importers**: 191
- **Purpose**: System service management
- **Targets**: systemd, launchd, etc.

### 4. Session Management (HTTP Only)

These packages provide **web session management**, not terminal sessions:

- github.com/alexedwards/scs - HTTP sessions
- github.com/icza/session - Web sessions
- github.com/gorilla/sessions - Cookie sessions
- github.com/Kwynto/gosession - HTTP sessions

**Not relevant for our use case**

### 5. Task/Job Queues

#### github.com/hibiken/asynq

- **Stars**: 9,000+
- **Purpose**: Distributed task queue
- **Based on**: Redis

**Not relevant**: For async jobs, not interactive processes

#### github.com/RichardKnop/machinery

- **Purpose**: Asynchronous task queue
- **Not relevant**: Different use case

### 6. IPC Libraries

#### github.com/godbus/dbus

- **Purpose**: D-Bus bindings
- **Use case**: System service communication
- **Not relevant for our needs**

#### NATS

- **Purpose**: Message bus
- **Use case**: Distributed systems
- **Not relevant**: Too heavy, wrong abstraction

### 7. CLI Frameworks

#### github.com/spf13/cobra

- **Purpose**: CLI application framework
- **No session management**

#### github.com/urfave/cli

- **Purpose**: CLI helpers
- **No session management**

## Tools Found (Not Libraries)

### sesh (tmux session manager)

- **Stars**: 1,500+
- **Type**: CLI tool
- **Cannot import**: Not a library

### smug (tmux session manager)

- **Stars**: 788
- **Type**: CLI tool
- **Cannot import**: Not a library

### pmgo (PM2 for Go)

- **Stars**: ~200
- **Type**: CLI tool
- **Not a library**

## Why Nothing Exists

After extensive research, here's why no Go package provides this:

### 1. Complexity

Session management requires:
- PTY handling
- Process lifecycle
- Client-server protocol
- State persistence
- Signal handling
- Error recovery

**Too complex for a small library**, too specific for a large framework.

### 2. Existing Solutions Work

- tmux/screen are mature, reliable
- Most developers just shell out
- No strong demand for Go alternative

### 3. Platform Specificity

- PTY behavior differs across Unix variants
- Windows requires different approach (ConPTY)
- Hard to create portable abstraction

### 4. Maintenance Burden

- Terminal handling has many edge cases
- Continuous maintenance required
- Small user base doesn't justify effort

## Alternative Approaches Discovered

### 1. Container-based

```go
// Use Docker for isolation
exec.Command("docker", "run", "-it", "--name", "session", "image", "cmd")
exec.Command("docker", "attach", "session")
```

**Pros**: Strong isolation, well-tested  
**Cons**: Heavy, requires Docker

### 2. SSH-based

```go
// Use SSH multiplexing
exec.Command("ssh", "-M", "-S", socketPath, "localhost", "cmd")
exec.Command("ssh", "-S", socketPath, "localhost")
```

**Pros**: Secure, network-transparent  
**Cons**: Complex setup, requires SSH

### 3. tmux Control Mode

```go
// Use tmux programmatically
exec.Command("tmux", "-CC", "new-session", "-s", "name")
```

**Pros**: Machine-readable output  
**Cons**: Still requires tmux

## Attempted Combinations

We tried combining libraries:

### creack/pty + overseer?

- **Problem**: overseer doesn't expose PTY
- **Would need**: Fork and modify overseer

### creack/pty + go-daemon?

- **Problem**: Still need to build entire protocol
- **Result**: 1000+ LOC of custom code

### creack/pty + gorilla/websocket?

- **Works for**: Web-based terminals (like gotty)
- **Doesn't help with**: Local attach/detach

## Packages That Came Close

### 1. github.com/gravitational/teleport

- **Purpose**: SSH server with session recording
- **Has**: Session management
- **Problem**: Entire infrastructure, not a library
- **Size**: Massive project

### 2. github.com/elisescu/tty-share

- **Purpose**: Share terminal over network
- **Has**: Some session logic
- **Problem**: Network-focused, not for local use
- **Not extractable**: Tightly coupled

## Conclusion

After searching:
- ✅ pkg.go.dev (all packages)
- ✅ GitHub (extensive search)
- ✅ awesome-go (comprehensive list)
- ✅ Go module index
- ✅ Real projects (see [Go Projects Analysis](./05-go-projects-analysis.md))

**Result**: No Go package does what we need.

## Implications for beans

This means our options are:

1. **Shell out to existing tools** (abduco, tmux, screen)
   - Quickest path
   - Proven solutions
   - External dependency

2. **Build custom solution**
   - 1-2 weeks effort
   - ~1000 LOC
   - Ongoing maintenance
   - Using creack/pty as foundation

3. **Don't support detach/attach**
   - Simplest
   - Less useful for long-running builds

## Recommended Decision Path

Given that **no package exists**:

### If Timeline is Short (< 1 week)
→ **Use abduco** (bundle it)

### If You Want Pure Go
→ **Build custom** (1-2 weeks, significant effort)

### If Users Have tmux
→ **Use tmux** (no bundling needed)

## Community Validation

We found:
- **0 packages** on pkg.go.dev with "session attach"
- **0 packages** on awesome-go for this use case  
- **0 importable libraries** on GitHub
- **Many CLI tools** that solve this (all shell out to tmux/screen/abduco)

This confirms: The Go community solves this by **shelling out to C tools**, not by building Go libraries.

---

**Previous**: [← Terminal Multiplexers](./06-terminal-multiplexers.md)  
**Next**: [Final Recommendations →](./08-recommendations.md)
