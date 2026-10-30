# CLI-to-CLI Launch Patterns: Technical Summary

## Core Approaches

### 1. **exec.Command() with Run() [RECOMMENDED for "return to parent"]**
```go
cmd := exec.Command("child-cli")
cmd.Stdin = os.Stdin
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
err := cmd.Run() // Blocks until child exits
// Control returns here
```

**How it works:**
- Parent spawns child process
- Child gets full control of terminal I/O
- Parent blocks/waits for child
- When child exits, parent resumes

**Key requirements:**
- Must connect Stdin/Stdout/Stderr explicitly
- Parent remains in memory (uses resources)

---

### 2. **syscall.Exec() [Process replacement]**
```go
syscall.Exec("/path/to/child", []string{"child"}, os.Environ())
// Never returns - parent is REPLACED
```

**How it works:**
- Replaces current process image with child
- Parent ceases to exist
- Same PID, new program

**When to use:**
- Don't need to return to parent
- Want to save memory (no parent overhead)
- Shell-like behavior (bash uses this)

---

## Terminal Control Gotchas

### 3. **Raw/Cooked Terminal Modes**
If parent uses raw terminal mode (like TUI apps):

```go
import "golang.org/x/term"

// Before launching child
oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))

// Launch child
cmd.Run()

// After child exits - MUST RESTORE
term.Restore(int(os.Stdin.Fd()), oldState)
```

**Problem:** If you don't restore, terminal stays in raw mode (appears broken)

---

### 4. **Signal Handling**
Signals (Ctrl+C, Ctrl+Z) need proper handling:

```go
// Child automatically gets signals if properly connected
// But if parent catches signals, may need to:
signal.Ignore(syscall.SIGINT) // While child runs
// Or forward signals to child process
```

---

### 5. **Process Groups & Job Control**
For proper Ctrl+Z support:

```go
cmd := exec.Command("child")
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setpgid: true,  // New process group
}
```

---

## Common Patterns in Real Tools

### Git (launching editor)
- Uses `exec.Command()` with full terminal control
- Respects EDITOR environment variable
- Blocks until editor exits
- No special terminal restoration (assumes editor handles it)

### Shell (bash, zsh)
- Uses `fork()` + `exec()` for external commands
- Uses `syscall.Exec()` for exec built-in
- Process group management for job control

### Less/More (pagers)
- Take over terminal completely
- Save/restore terminal state
- Handle SIGWINCH (window resize)

---

## Bubbletea/TUI Specific Considerations

If parent is a Bubbletea app:

1. **Suspend the Bubbletea program**
```go
p.Send(tea.Suspend) // Suspends TUI
```

2. **Launch child CLI**
```go
cmd := exec.Command("child-cli")
cmd.Stdin = os.Stdin
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
cmd.Run()
```

3. **Resume automatically**
- Bubbletea handles terminal restoration
- TUI redraws on resume

**Alternative: tea.ExecProcess**
```go
cmd := exec.Command("child-cli")
return tea.ExecProcess(cmd, func(err error) tea.Msg {
    return editorFinishedMsg{err}
})
```
This is the CORRECT way in Bubbletea - it handles suspend/resume automatically.

---

## Summary Checklist

✅ Use `exec.Command()` + `cmd.Run()` for return-to-parent pattern
✅ Always set `cmd.Stdin/Stdout/Stderr = os.Stdin/Stdout/Stderr`
✅ In TUI apps: use `tea.ExecProcess()` or manually suspend/resume
✅ Save/restore terminal state if parent modified it
✅ Child inherits environment automatically (or customize with `cmd.Env`)
✅ No special signal handling needed (usually)

❌ Don't use `syscall.Exec()` if you need to return to parent
❌ Don't forget to restore terminal state
❌ Don't assume child will clean up parent's terminal settings

---

## Quick Reference

| Need | Solution |
|------|----------|
| Launch editor, return to parent | `exec.Command()` + `cmd.Run()` |
| Replace current process | `syscall.Exec()` |
| From Bubbletea app | `tea.ExecProcess()` |
| Save terminal state | `term.MakeRaw()` + `term.Restore()` |
| Process groups | `cmd.SysProcAttr.Setpgid = true` |

---

## Quick Decision Tree

- **TUI app (Bubbletea)?** → Use `tea.ExecProcess()`
- **Regular CLI?** → Use `exec.Command()` + `cmd.Run()` with I/O connected
- **Never returning?** → Use `syscall.Exec()`
