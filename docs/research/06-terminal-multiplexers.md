# Lightweight Terminal Multiplexers

## Overview

Research on minimal terminal multiplexers that could be embedded or called programmatically from Go.

## C-based Solutions

### 1. mtm (Micro Terminal Multiplexer) - Smallest Option

**Repository**: https://github.com/deadpixi/mtm  
**License**: GPL v3  
**Size**: ~1000 lines of C code  
**Status**: "Finished" - stable, minimal updates

**Key Features**:
- Extreme minimalism (smallest viable multiplexer)
- Simple keybindings (Ctrl-g prefix)
- Split panes (horizontal/vertical)
- Mouse support (resize, select, scrollwheel)
- Search functionality
- No session management (by design)

**Terminal Type**: Advertises as `screen-bce`

**Dependencies**: Only ncursesw

**Installation**:
```bash
# Homebrew (macOS)
brew install mtm

# From source
git clone https://github.com/deadpixi/mtm
cd mtm && make && sudo make install
```

**Pros**:
- ✅ Smallest codebase (easy to understand)
- ✅ Simple to build and embed
- ✅ Portable (just needs ncurses)
- ✅ Stable (considered "finished")

**Cons**:
- ❌ GPL license (restrictive)
- ❌ No session management
- ❌ Would need separate solution for detach/attach

**Use Case**: If you only need split panes, not sessions

### 2. dvtm (Dynamic Virtual Terminal Manager)

**Repository**: https://github.com/martanne/dvtm  
**License**: MIT/ISC (permissive)  
**Size**: ~4000 lines of C  
**Author**: Same as abduco (designed to work together)

**Key Features**:
- Tiling window manager for terminals
- dwm-inspired (dynamic window management)
- Multiple layouts (vertical, horizontal, grid, fullscreen)
- Tagging system
- Status bar via named pipes
- Copy mode (pipes to external editor)

**Typical Usage**: `abduco -c session dvtm`

**Installation**:
```bash
# Most package managers
sudo apt install dvtm      # Debian/Ubuntu
sudo pacman -S dvtm        # Arch
brew install dvtm          # macOS
```

**Pros**:
- ✅ Permissive license (MIT/ISC)
- ✅ Well-established project
- ✅ Pairs perfectly with abduco
- ✅ More features than mtm

**Cons**:
- ⚠️ Larger than mtm
- ⚠️ More complex keybindings
- ⚠️ Requires abduco for sessions

**Use Case**: Full terminal multiplexing + session management (with abduco)

### 3. abduco (Session Manager)

**See**: [Detailed Analysis](./03-abduco-detailed-analysis.md)

**Purpose**: Session management only (pairs with dvtm)

**The Unix Philosophy Combo**:
```bash
# Session management + terminal multiplexing
abduco -c work dvtm

# Detach: Ctrl+\
# Reattach: abduco -a work
```

## Go-based Solutions

### 1. 3mux - Most Popular Go Multiplexer

**Repository**: https://github.com/aaronjanse/3mux  
**Stars**: 1,838  
**License**: MIT  
**Status**: Active

**Key Features**:
- i3-inspired keybindings
- Built-in session management
- Search, mouse support, scrollback
- Some tmux/screen binding support
- Pure Go implementation

**Size**: Medium-large Go project

**Architecture**:
```go
// Main components
type Universe struct {
    sessions []*Session
}

type Session struct {
    root *Node
}

type Node struct {
    // Terminal or split container
}
```

**Pros**:
- ✅ Pure Go (no external dependencies)
- ✅ MIT license (could fork/embed)
- ✅ Modern, user-friendly
- ✅ Active development

**Cons**:
- ⚠️ Large codebase to understand
- ⚠️ Not designed as library
- ⚠️ Would need significant refactoring to embed

**Integration Potential**: Medium - would require extracting core logic

### 2. Other Go Multiplexers

#### tuios
- **Stars**: 2,114
- **Purpose**: Terminal UI OS
- **Use Case**: Too opinionated/full-featured for embedding

#### sunder  
- **Stars**: 32
- **Purpose**: Minimalist tmux alternative
- **Use Case**: Small but not actively maintained

## tmux and screen (Industry Standards)

### tmux

**Repository**: https://github.com/tmux/tmux  
**License**: ISC  
**Size**: ~100k LOC (C)  
**Availability**: Extremely high

**Key Features**:
- Full-featured terminal multiplexer
- Robust session management
- Scripting support
- Control mode for automation

**For Programmatic Use**:
```bash
# Create session
tmux new-session -d -s name "command"

# Attach
tmux attach-session -t name

# Control mode (machine-readable)
tmux -CC attach-session -t name
```

**Go Wrappers**:
- **wricardo/gomux** (36 stars) - Create sessions/windows/panes
- Most are thin CLI wrappers

**Pros**:
- ✅ Extremely robust
- ✅ Widely installed
- ✅ Well-documented
- ✅ Control mode for automation

**Cons**:
- ⚠️ Heavy (~1MB binary)
- ⚠️ Complex for simple needs
- ⚠️ Users must install

### screen

**License**: GPL  
**Size**: ~800KB binary  
**Availability**: Very high (older systems)

**Similar to tmux** but older, less featured

## Comparison Matrix

| Solution | Size | License | Session Mgmt | Go Native | Embeddable | Availability |
|----------|------|---------|--------------|-----------|------------|--------------|
| **mtm** | 1K LOC | GPL v3 | ❌ | ❌ | Medium | Low |
| **dvtm** | 4K LOC | MIT/ISC | ❌ (needs abduco) | ❌ | Medium | Medium |
| **abduco** | 2K LOC | ISC | ✅ | ❌ | Easy | Medium |
| **dvtm+abduco** | 6K LOC | MIT/ISC | ✅ | ❌ | Medium | Medium |
| **3mux** | Large | MIT | ✅ | ✅ | Hard | Low |
| **tmux** | 100K LOC | ISC | ✅ | ❌ | No | Very High |
| **screen** | Large | GPL | ✅ | ❌ | No | High |

## Recommendations by Use Case

### 1. Session Management Only (Our Need)

**Best choice**: **abduco**
- Minimal (~50KB)
- Permissive license
- Does exactly what we need
- Easy to bundle

**Why not the others**:
- mtm: No session management
- dvtm: Need abduco anyway
- tmux: Overkill
- 3mux: Complex to embed

### 2. Split Panes + Sessions

**Best choice**: **dvtm + abduco**
- Unix philosophy (composable)
- Both permissive licenses
- ~100KB total
- Works together perfectly

**Alternative**: **tmux** (if users have it)

### 3. Pure Go Solution

**Best choice**: Extract from **3mux** or build custom
- Requires significant refactoring
- ~1000+ LOC even for minimal version
- Not worth it for simple needs

### 4. Most Available

**Best choice**: **tmux**
- Widely installed already
- No bundling needed
- Full-featured
- Programmatic control mode

## Building Custom with Go

If you want pure Go multiplexing:

### Minimal Components Needed

```go
// 1. PTY Management (creack/pty)
type Pane struct {
    PTY *os.File
    Cmd *exec.Cmd
}

// 2. Layout Management
type Layout interface {
    Split(direction Direction)
    Focus(pane int)
    Resize(pane int, size Size)
}

// 3. Terminal Rendering
type Renderer interface {
    Draw(panes []*Pane)
    HandleInput(key Key)
}

// 4. Session Storage
type SessionStore interface {
    Save(session *Session) error
    Load(id string) (*Session, error)
}
```

**Estimated effort**: 2-3 weeks for basic functionality

## Integration with Go Applications

### Pattern 1: Shell Out (Simplest)

```go
// Start dvtm in abduco session
cmd := exec.Command("abduco", "-c", "session", "dvtm")
cmd.Stdin = os.Stdin
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
cmd.Run()
```

### Pattern 2: Embed Binary

```go
//go:embed bin/abduco
var abducoBinary []byte

//go:embed bin/dvtm  
var dvtmBinary []byte

func extractBinaries() error {
    // Extract both to ~/.cache/myapp/
}
```

### Pattern 3: Library Integration (Complex)

```go
// Would require significant refactoring of 3mux or similar
import "github.com/aaronjanse/3mux/lib"

mux := lib.NewMultiplexer()
mux.CreateSession("work")
mux.Split(lib.Horizontal)
```

*Note: 3mux doesn't currently expose a library API*

## Bundling Considerations

### Binary Sizes

```
abduco:         ~50KB  (static)
dvtm:           ~60KB  (static)
mtm:            ~40KB  (static)
tmux:           ~1MB   (dynamic, with libs)
3mux:           ~5MB   (Go binary)
```

### Compilation

#### abduco + dvtm (static)

```bash
# abduco
git clone https://github.com/martanne/abduco
cd abduco
make LDFLAGS=-static
strip abduco  # ~50KB

# dvtm
git clone https://github.com/martanne/dvtm
cd dvtm
make LDFLAGS=-static
strip dvtm  # ~60KB
```

#### Cross-compilation

```bash
# For Linux
CC=x86_64-linux-musl-gcc make LDFLAGS=-static

# For macOS
# Must compile on macOS (no easy cross-compile)
```

## Final Recommendation

**For beans specifically**:

### Option 1: abduco Only (Recommended)
- **Why**: We only need session management, not split panes
- **Size**: 50KB
- **Complexity**: Minimal
- **Code**: ~50 LOC wrapper

### Option 2: abduco + dvtm
- **Why**: If we want split panes in future
- **Size**: 110KB total
- **Complexity**: Low-medium
- **Code**: ~100 LOC wrapper

### Option 3: tmux
- **Why**: Users likely have it
- **Size**: 0 (external)
- **Complexity**: Minimal
- **Code**: ~50 LOC wrapper
- **Downside**: Must be installed

### Not Recommended: 3mux or Custom
- **Why**: Too much complexity for our needs
- **Effort**: Weeks vs hours
- **Maintenance**: Ongoing vs minimal

## Code Example: Using dvtm + abduco

```go
package session

import (
    "os/exec"
    "path/filepath"
)

type Manager struct {
    abducoPath string
    dvtmPath   string
}

func NewManager() (*Manager, error) {
    abduco, err := extractAbduco()
    if err != nil {
        return nil, err
    }
    
    dvtm, err := extractDvtm()
    if err != nil {
        return nil, err
    }
    
    return &Manager{
        abducoPath: abduco,
        dvtmPath:   dvtm,
    }, nil
}

// Start session with multiplexer
func (m *Manager) CreateWithMultiplexer(name string) *exec.Cmd {
    return exec.Command(
        m.abducoPath,
        "-c", name,
        m.dvtmPath,
    )
}

// Start session with single command
func (m *Manager) Create(name, command string) error {
    cmd := exec.Command(
        m.abducoPath,
        "-n", name,
        "sh", "-c", command,
    )
    return cmd.Run()
}
```

---

**Previous**: [← Go Projects Analysis](./05-go-projects-analysis.md)  
**Next**: [Go Packages Search →](./07-go-packages-search.md)
