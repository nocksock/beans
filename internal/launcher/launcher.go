package launcher

import (
	"fmt"
	"os/exec"
	"strings"
)

// ExecutionResult holds a command and its cleanup function
type ExecutionResult struct {
	Cmd     *exec.Cmd
	cleanup func()
}

// Cleanup removes temporary files created for the execution
func (r *ExecutionResult) Cleanup() {
	if r.cleanup != nil {
		r.cleanup()
	}
}

// CreateExecCommand creates a command for running a launch.
// For multi-line scripts with shebang, passes script via stdin to the interpreter.
// For single-line scripts, it executes via sh -c.
func CreateExecCommand(l Launch) (*exec.Cmd, *ExecutionResult, error) {
	execScript := l.Script
	env := l.Env()
	workDir := l.WorkDir()

	// Check if this is multi-line (contains newline)
	if !strings.Contains(execScript, "\n") {
		// Single-line: execute via sh -c
		cmd := exec.Command("sh", "-c", execScript)
		cmd.Env = env
		cmd.Dir = workDir

		result := &ExecutionResult{
			Cmd:     cmd,
			cleanup: func() {},
		}

		return cmd, result, nil
	}

	// Multi-line: extract shebang and pass script via stdin
	lines := strings.SplitN(execScript, "\n", 2)
	if !strings.HasPrefix(lines[0], "#!") {
		return nil, nil, fmt.Errorf("multi-line script must start with shebang")
	}

	// Extract interpreter from shebang
	shebang := strings.TrimPrefix(lines[0], "#!")
	shebang = strings.TrimSpace(shebang)

	// Parse shebang into command and args
	// e.g., "/bin/bash" or "/usr/bin/env bash" or "/usr/bin/env python3"
	parts := strings.Fields(shebang)
	if len(parts) == 0 {
		return nil, nil, fmt.Errorf("invalid shebang: %s", lines[0])
	}

	// Build command - pass script content via stdin
	var cmd *exec.Cmd
	if len(parts) == 1 {
		cmd = exec.Command(parts[0])
	} else {
		cmd = exec.Command(parts[0], parts[1:]...)
	}

	cmd.Env = env
	cmd.Dir = workDir

	// Pass the script (including shebang line) via stdin
	cmd.Stdin = strings.NewReader(execScript)

	// Note: Stdout/Stderr will be set by caller (either os.Stdout or captured buffer)

	result := &ExecutionResult{
		Cmd:     cmd,
		cleanup: func() {}, // No cleanup needed
	}

	return cmd, result, nil
}
