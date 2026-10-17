//go:build !windows

package launcher

import (
	"os/exec"
	"syscall"
)

// detach puts cmd into its own process group, so signals sent to the TUI's
// group (e.g. Ctrl-C) don't reach it.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
