package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExecResult holds a command and its cleanup function
type ExecResult struct {
	Cmd     *exec.Cmd
	Cleanup func()
}

// CreateExecCommand creates a command for executing an exec script.
// For multi-line scripts, it creates a temp file and returns cleanup function.
// For single-line scripts, it executes via sh -c.
func CreateExecCommand(execScript, beanID, beansRoot, beanPath string) ExecResult {
	// Check if this is multi-line (contains newline)
	if !strings.Contains(execScript, "\n") {
		// Single-line: execute via sh -c
		cmd := exec.Command("sh", "-c", execScript)
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("BEANS_ROOT=%s", beansRoot),
			fmt.Sprintf("BEANS_ID=%s", beanID),
			fmt.Sprintf("BEANS_TASK=%s", beanPath),
		)
		cmd.Dir = beansRoot
		return ExecResult{Cmd: cmd, Cleanup: func() {}}
	}

	// Multi-line: create temp file
	tmpFile, err := os.CreateTemp("", "beans-launcher-*.sh")
	if err != nil {
		// Return command that will fail with good error message
		cmd := exec.Command("sh", "-c", fmt.Sprintf("echo 'Error: failed to create temp script: %v' >&2 && exit 1", err))
		return ExecResult{Cmd: cmd, Cleanup: func() {}}
	}

	tmpPath := tmpFile.Name()

	// Write script content
	if _, err := tmpFile.WriteString(execScript); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		cmd := exec.Command("sh", "-c", fmt.Sprintf("echo 'Error: failed to write script: %v' >&2 && exit 1", err))
		return ExecResult{Cmd: cmd, Cleanup: func() {}}
	}
	tmpFile.Close()

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		cmd := exec.Command("sh", "-c", fmt.Sprintf("echo 'Error: failed to make script executable: %v' >&2 && exit 1", err))
		return ExecResult{Cmd: cmd, Cleanup: func() {}}
	}

	// Create command to execute temp file
	cmd := exec.Command(tmpPath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("BEANS_ROOT=%s", beansRoot),
		fmt.Sprintf("BEANS_ID=%s", beanID),
		fmt.Sprintf("BEANS_TASK=%s", beanPath),
	)
	cmd.Dir = beansRoot

	// Return with cleanup function
	cleanup := func() {
		os.Remove(tmpPath)
	}

	return ExecResult{Cmd: cmd, Cleanup: cleanup}
}

// extractMainCommand extracts the primary executable from a command string.
// For shell commands, this returns the first space-separated token.
// Examples:
//
//	"opencode run ..." -> "opencode"
//	"/usr/bin/tool --flag" -> "/usr/bin/tool"
//	".beans/scripts/tool.sh arg" -> ".beans/scripts/tool.sh"
func extractMainCommand(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return command
	}
	return parts[0]
}

// extractShebangInterpreter extracts the interpreter from a shebang line.
// Returns empty string if no shebang or invalid format.
func extractShebangInterpreter(script string) string {
	lines := strings.Split(script, "\n")
	if len(lines) == 0 {
		return ""
	}

	firstLine := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(firstLine, "#!") {
		return ""
	}

	// Extract interpreter path (e.g., "#!/bin/bash" -> "/bin/bash")
	// Handle "#!/usr/bin/env python3" -> "python3"
	shebang := strings.TrimPrefix(firstLine, "#!")
	shebang = strings.TrimSpace(shebang)

	// If using env, return the actual command
	if strings.HasPrefix(shebang, "/usr/bin/env ") || strings.HasPrefix(shebang, "/bin/env ") {
		parts := strings.Fields(shebang)
		if len(parts) >= 2 {
			return parts[1] // Return "python3" from "/usr/bin/env python3"
		}
	}

	// Otherwise return the full path
	parts := strings.Fields(shebang)
	if len(parts) > 0 {
		return parts[0]
	}

	return shebang
}

// IsLauncherAvailable checks if a launcher's command/interpreter is available.
// For single-line: checks if main command exists
// For multi-line: checks if shebang interpreter exists
func IsLauncherAvailable(execScript string) bool {
	// Multi-line: check shebang interpreter
	if strings.Contains(execScript, "\n") {
		interpreter := extractShebangInterpreter(execScript)
		if interpreter == "" {
			return false // No shebang = invalid
		}

		// Check if interpreter exists
		if filepath.IsAbs(interpreter) {
			_, err := os.Stat(interpreter)
			return err == nil
		}

		// Check in PATH
		_, err := exec.LookPath(interpreter)
		return err == nil
	}

	// Single-line: check main command
	mainCmd := extractMainCommand(execScript)
	if mainCmd == "" {
		return false
	}

	// If it looks like a local file path, check if it exists
	if filepath.IsAbs(mainCmd) || strings.Contains(mainCmd, string(filepath.Separator)) {
		_, err := os.Stat(mainCmd)
		return err == nil
	}

	// For command names, check if in PATH
	_, err := exec.LookPath(mainCmd)
	return err == nil
}
