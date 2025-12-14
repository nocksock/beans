package launcher

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateExecCommand_EnvironmentVariables(t *testing.T) {
	tests := []struct {
		name        string
		execScript  string
		beansDir    string
		beanID      string
		beanTitle   string
		wantEnvVars map[string]string
	}{
		{
			name:       "single-line script has all env vars",
			execScript: "echo $BEANS_ROOT $BEANS_DIR $BEANS_ID $BEANS_TASK",
			beansDir:   "/project/.beans",
			beanID:     "beans-xyz",
			beanTitle:  "Test Task",
			wantEnvVars: map[string]string{
				"BEANS_ROOT": "/project",
				"BEANS_DIR":  "/project/.beans",
				"BEANS_ID":   "beans-xyz",
				"BEANS_TASK": "Test Task",
			},
		},
		{
			name:       "multi-line script has all env vars",
			execScript: "#!/bin/bash\necho $BEANS_ROOT $BEANS_DIR $BEANS_ID $BEANS_TASK",
			beansDir:   "/home/user/project/.beans",
			beanID:     "beans-abc",
			beanTitle:  "Another Task",
			wantEnvVars: map[string]string{
				"BEANS_ROOT": "/home/user/project",
				"BEANS_DIR":  "/home/user/project/.beans",
				"BEANS_ID":   "beans-abc",
				"BEANS_TASK": "Another Task",
			},
		},
		{
			name:       "nested beansDir path",
			execScript: "env",
			beansDir:   "/home/user/workspace/.beans",
			beanID:     "beans-123",
			beanTitle:  "Task Title",
			wantEnvVars: map[string]string{
				"BEANS_ROOT": "/home/user/workspace",
				"BEANS_DIR":  "/home/user/workspace/.beans",
				"BEANS_ID":   "beans-123",
				"BEANS_TASK": "Task Title",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, result, err := CreateExecCommand(tt.execScript, tt.beansDir, tt.beanID, tt.beanTitle)
			if err != nil {
				t.Fatalf("CreateExecCommand() error = %v", err)
			}
			defer result.Cleanup()

			if cmd == nil {
				t.Fatal("CreateExecCommand() returned nil cmd")
			}

			// Check that all expected env vars are set
			envMap := envSliceToMap(cmd.Env)
			for key, wantValue := range tt.wantEnvVars {
				gotValue, exists := envMap[key]
				if !exists {
					t.Errorf("Environment variable %s not set", key)
					continue
				}
				if gotValue != wantValue {
					t.Errorf("Environment variable %s = %v, want %v", key, gotValue, wantValue)
				}
			}
		})
	}
}

func TestCreateExecCommand_SingleLine(t *testing.T) {
	cmd, result, err := CreateExecCommand("echo hello", "/project/.beans", "beans-123", "Test")
	if err != nil {
		t.Fatalf("CreateExecCommand() error = %v", err)
	}
	defer result.Cleanup()

	if cmd == nil {
		t.Fatal("CreateExecCommand() returned nil cmd")
	}

	// Check it's using sh -c
	if cmd.Path == "" {
		t.Error("Command path is empty")
	}
	if !strings.Contains(cmd.Path, "sh") {
		t.Errorf("Expected sh command, got %s", cmd.Path)
	}

	// Check args
	if len(cmd.Args) < 3 {
		t.Fatalf("Expected at least 3 args [sh, -c, script], got %v", cmd.Args)
	}
	if cmd.Args[1] != "-c" {
		t.Errorf("Expected second arg to be -c, got %s", cmd.Args[1])
	}
	if cmd.Args[2] != "echo hello" {
		t.Errorf("Expected third arg to be 'echo hello', got %s", cmd.Args[2])
	}

	// Check working directory
	if cmd.Dir != "/project" {
		t.Errorf("Working directory = %v, want /project", cmd.Dir)
	}
}

func TestCreateExecCommand_MultiLine(t *testing.T) {
	script := "#!/bin/bash\necho hello\necho world"
	cmd, result, err := CreateExecCommand(script, "/project/.beans", "beans-456", "Multi Test")
	if err != nil {
		t.Fatalf("CreateExecCommand() error = %v", err)
	}
	defer result.Cleanup()

	if cmd == nil {
		t.Fatal("CreateExecCommand() returned nil cmd")
	}

	// Check stdin is set
	if cmd.Stdin == nil {
		t.Error("Stdin should be set for multi-line script")
	}

	// Check working directory
	if cmd.Dir != "/project" {
		t.Errorf("Working directory = %v, want /project", cmd.Dir)
	}
}

func TestCreateExecCommand_MultiLineNoShebang(t *testing.T) {
	script := "echo hello\necho world"
	_, result, err := CreateExecCommand(script, "/project/.beans", "beans-789", "No Shebang")
	if result != nil {
		defer result.Cleanup()
	}

	if err == nil {
		t.Error("Expected error for multi-line script without shebang")
	}
	if err != nil && !strings.Contains(err.Error(), "shebang") {
		t.Errorf("Expected shebang error, got: %v", err)
	}
}

func TestCreateExecCommand_InvalidShebang(t *testing.T) {
	script := "#!\necho hello"
	_, result, err := CreateExecCommand(script, "/project/.beans", "beans-999", "Invalid Shebang")
	if result != nil {
		defer result.Cleanup()
	}

	if err == nil {
		t.Error("Expected error for invalid shebang")
	}
	if err != nil && !strings.Contains(err.Error(), "shebang") {
		t.Errorf("Expected shebang error, got: %v", err)
	}
}

// envSliceToMap converts os.Environ() style slice to map
func envSliceToMap(envSlice []string) map[string]string {
	result := make(map[string]string)
	for _, entry := range envSlice {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

// mockCmd wraps exec.Cmd for testing without executing
func mockCmd(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

// SECURITY TESTS

// TestShellInjectionViaBeanTitleSemicolon tests that bean titles with semicolons
// cannot be used for shell injection via environment variables
func TestShellInjectionViaBeanTitleSemicolon(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create temp beansDir: %v", err)
	}

	// Malicious bean title attempting shell injection
	maliciousBeanTitle := "normal task; touch /tmp/pwned-semicolon"
	beanID := "test-bean-1"

	// Simple script that echoes the BEANS_TASK variable
	// If shell injection works, the touch command would execute
	execScript := "echo $BEANS_TASK"

	cmd, result, err := CreateExecCommand(execScript, beansDir, beanID, maliciousBeanTitle)
	if err != nil {
		t.Fatalf("CreateExecCommand failed: %v", err)
	}
	defer result.Cleanup()

	// Capture output
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	// Run the command
	if err := cmd.Run(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	// The output should contain the literal string with semicolon, not execute injection
	output := strings.TrimSpace(stdout.String())
	if output != maliciousBeanTitle {
		t.Errorf("unexpected output: got %q, want %q", output, maliciousBeanTitle)
	}

	// Verify the injection didn't work - the file should not exist
	if _, err := os.Stat("/tmp/pwned-semicolon"); err == nil {
		t.Error("shell injection succeeded: /tmp/pwned-semicolon file was created")
		os.Remove("/tmp/pwned-semicolon") // cleanup
	}
}

// TestShellInjectionViaCommandSubstitution tests that bean titles with $()
// command substitution cannot execute arbitrary commands
func TestShellInjectionViaCommandSubstitution(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create temp beansDir: %v", err)
	}

	maliciousBeanTitle := "task $(touch /tmp/pwned-cmdsub)"
	beanID := "test-bean-2"
	execScript := "echo $BEANS_TASK"

	cmd, result, err := CreateExecCommand(execScript, beansDir, beanID, maliciousBeanTitle)
	if err != nil {
		t.Fatalf("CreateExecCommand failed: %v", err)
	}
	defer result.Cleanup()

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output != maliciousBeanTitle {
		t.Errorf("unexpected output: got %q, want %q", output, maliciousBeanTitle)
	}

	// Verify injection didn't work
	if _, err := os.Stat("/tmp/pwned-cmdsub"); err == nil {
		t.Error("shell injection via $() succeeded: /tmp/pwned-cmdsub file was created")
		os.Remove("/tmp/pwned-cmdsub")
	}
}

// TestShellInjectionViaBackticks tests that bean titles with backticks
// cannot execute arbitrary commands
func TestShellInjectionViaBackticks(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create temp beansDir: %v", err)
	}

	maliciousBeanTitle := "task `touch /tmp/pwned-backtick`"
	beanID := "test-bean-3"
	execScript := "echo $BEANS_TASK"

	cmd, result, err := CreateExecCommand(execScript, beansDir, beanID, maliciousBeanTitle)
	if err != nil {
		t.Fatalf("CreateExecCommand failed: %v", err)
	}
	defer result.Cleanup()

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output != maliciousBeanTitle {
		t.Errorf("unexpected output: got %q, want %q", output, maliciousBeanTitle)
	}

	// Verify injection didn't work
	if _, err := os.Stat("/tmp/pwned-backtick"); err == nil {
		t.Error("shell injection via backticks succeeded: /tmp/pwned-backtick file was created")
		os.Remove("/tmp/pwned-backtick")
	}
}

// TestPathTraversalInBeansDir tests that path traversal attempts in beans directory
// don't lead to command execution outside the intended scope
func TestPathTraversalInBeansDir(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create temp beansDir: %v", err)
	}

	// Attempt path traversal in beansDir parameter
	maliciousBeansDir := filepath.Join(tmpDir, ".beans", "..", "..", "etc")
	beanID := "test-bean-4"
	beanTitle := "normal task"
	execScript := "pwd"

	cmd, result, err := CreateExecCommand(execScript, maliciousBeansDir, beanID, beanTitle)
	if err != nil {
		t.Fatalf("CreateExecCommand failed: %v", err)
	}
	defer result.Cleanup()

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	// Verify working directory is NOT /etc
	output := strings.TrimSpace(stdout.String())
	if strings.Contains(output, "/etc") && !strings.Contains(output, tmpDir) {
		t.Errorf("path traversal succeeded: command executed in %q", output)
	}
}

// TestMaliciousShebang tests that malicious shebang lines are handled safely
func TestMaliciousShebang(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create temp beansDir: %v", err)
	}

	// Attempt to inject commands via shebang
	maliciousScript := "#!/bin/sh; touch /tmp/pwned-shebang\necho hello"
	beanID := "test-bean-5"
	beanTitle := "normal task"

	cmd, result, err := CreateExecCommand(maliciousScript, beansDir, beanID, beanTitle)
	if err != nil {
		t.Fatalf("CreateExecCommand failed: %v", err)
	}
	defer result.Cleanup()

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	// Verify injection didn't work
	if _, err := os.Stat("/tmp/pwned-shebang"); err == nil {
		t.Error("shell injection via shebang succeeded: /tmp/pwned-shebang file was created")
		os.Remove("/tmp/pwned-shebang")
	}
}

// TestEnvironmentVariableInjection tests that environment variables are set correctly
// and don't allow injection
func TestEnvironmentVariableInjection(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create temp beansDir: %v", err)
	}

	beanID := "test-bean-6"
	beanTitle := "task with $PATH injection"
	execScript := "echo BEANS_TASK=$BEANS_TASK"

	cmd, result, err := CreateExecCommand(execScript, beansDir, beanID, beanTitle)
	if err != nil {
		t.Fatalf("CreateExecCommand failed: %v", err)
	}
	defer result.Cleanup()

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	expected := "BEANS_TASK=" + beanTitle
	if output != expected {
		t.Errorf("environment variable not set correctly: got %q, want %q", output, expected)
	}
}

// TestSpecialCharactersInBeanTitle tests that Unicode and emoji in bean titles are handled safely
func TestSpecialCharactersInBeanTitle(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create temp beansDir: %v", err)
	}

	beanTitle := "Fix bug 🐛 with Unicode: ñ, 中文, 🚀"
	beanID := "test-bean-7"
	execScript := "echo $BEANS_TASK"

	cmd, result, err := CreateExecCommand(execScript, beansDir, beanID, beanTitle)
	if err != nil {
		t.Fatalf("CreateExecCommand failed: %v", err)
	}
	defer result.Cleanup()

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output != beanTitle {
		t.Errorf("special characters not preserved: got %q, want %q", output, beanTitle)
	}
}

// TestScriptWithNullBytes tests that scripts with null bytes are handled safely
func TestScriptWithNullBytes(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create temp beansDir: %v", err)
	}

	// Script with null byte
	execScript := "echo hello\x00world"
	beanID := "test-bean-8"
	beanTitle := "normal task"

	cmd, result, err := CreateExecCommand(execScript, beansDir, beanID, beanTitle)
	if err != nil {
		// It's acceptable to reject scripts with null bytes
		return
	}
	defer result.Cleanup()

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	// Run command - it should not cause issues
	_ = cmd.Run() // Don't fail on error, just ensure no panic/crash
}

// TestShebangWithMultipleArguments tests that shebangs with multiple arguments work correctly
func TestShebangWithMultipleArguments(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create temp beansDir: %v", err)
	}

	script := "#!/usr/bin/env bash -e\necho test"
	beanID := "test-bean-9"
	beanTitle := "normal task"

	cmd, result, err := CreateExecCommand(script, beansDir, beanID, beanTitle)
	if err != nil {
		t.Fatalf("CreateExecCommand failed: %v", err)
	}
	defer result.Cleanup()

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("command execution failed: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output != "test" {
		t.Errorf("shebang with multiple args didn't work: got %q, want %q", output, "test")
	}
}

// TestSingleLineCommandWithTrailingNewline tests that commands are properly identified as single-line
// even with trailing newline
func TestSingleLineCommandWithTrailingNewline(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create temp beansDir: %v", err)
	}

	// This should be treated as multi-line due to the newline
	execScript := "echo test\n"
	beanID := "test-bean-10"
	beanTitle := "normal task"

	_, _, err := CreateExecCommand(execScript, beansDir, beanID, beanTitle)
	// Should error because it's multi-line without shebang
	if err == nil {
		t.Error("expected error for multi-line script (trailing newline) without shebang")
	}
}
