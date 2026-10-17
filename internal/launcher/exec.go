package launcher

import (
	"errors"
	"io"
	"os"
	"os/exec"
)

// CreateShellCommand creates a command that runs a launch's script via sh -c.
func CreateShellCommand(l Launch) *exec.Cmd {
	cmd := exec.Command("sh", "-c", l.Script)
	cmd.Env = l.Env()
	cmd.Dir = l.WorkDir()
	return cmd
}

// CreateLoggedExecCommand is like CreateShellCommand, but additionally tees the
// command's stderr into a temporary log file. Stdin and stdout are left unset so
// tea.ExecProcess attaches them to the terminal and interactive launchers keep
// working. The caller must close the returned file after the command exits.
func CreateLoggedExecCommand(l Launch) (*exec.Cmd, *os.File, error) {
	cmd := CreateShellCommand(l)

	logFile, err := os.CreateTemp("", "beans-launch-*.log")
	if err != nil {
		return nil, nil, err
	}
	cmd.Stderr = io.MultiWriter(os.Stderr, logFile)

	return cmd, logFile, nil
}

// CreateBackgroundCommand is like CreateShellCommand, but for running without
// the terminal: stdin is left unset (reads get EOF), stdout and stderr both go
// to a temporary log file, and the command is detached from the TUI's process
// group so a Ctrl-C there doesn't reach it. The caller must close the returned
// file after the command exits.
func CreateBackgroundCommand(l Launch) (*exec.Cmd, *os.File, error) {
	cmd := CreateShellCommand(l)

	logFile, err := os.CreateTemp("", "beans-launch-*.log")
	if err != nil {
		return nil, nil, err
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	detach(cmd)

	return cmd, logFile, nil
}

// ExitCode extracts the process exit code from the error returned by running a
// command. It returns 0 for a nil error and -1 if the command could not be run.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
