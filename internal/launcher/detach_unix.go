//go:build !windows

package launcher

import (
	"os/exec"
	"syscall"
)

// detach starts cmd in its own session without a controlling terminal, so
// signals sent to the TUI (e.g. Ctrl-C) don't reach it, and programs that try
// to prompt on /dev/tty fail right away instead of being stopped forever.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
