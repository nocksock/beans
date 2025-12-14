package launcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateExecCommand_SingleLine(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatal(err)
	}

	beanID := "abc123"
	beanPath := filepath.Join(beansDir, "beans-"+beanID+".md")

	cmd, result, err := CreateExecCommand("echo hello", beansDir, beanID, beanPath)
	if err != nil {
		t.Fatalf("CreateExecCommand() error = %v, want nil", err)
	}
	defer result.Cleanup()

	// Verify command structure - check that path ends with "sh"
	if !strings.HasSuffix(cmd.Path, "/sh") {
		t.Errorf("CreateExecCommand() cmd.Path = %v, want path ending with /sh", cmd.Path)
	}

	if len(cmd.Args) < 3 || cmd.Args[1] != "-c" || cmd.Args[2] != "echo hello" {
		t.Errorf("CreateExecCommand() cmd.Args = %v, want [sh -c echo hello]", cmd.Args)
	}

	// Verify working directory is project root (parent of .beans)
	expectedDir := tmpDir
	if cmd.Dir != expectedDir {
		t.Errorf("CreateExecCommand() cmd.Dir = %v, want %v", cmd.Dir, expectedDir)
	}

	// Verify environment variables
	envMap := make(map[string]string)
	for _, e := range cmd.Env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if envMap["BEANS_ROOT"] != tmpDir {
		t.Errorf("BEANS_ROOT = %v, want %v", envMap["BEANS_ROOT"], tmpDir)
	}
	if envMap["BEANS_DIR"] != beansDir {
		t.Errorf("BEANS_DIR = %v, want %v", envMap["BEANS_DIR"], beansDir)
	}
	if envMap["BEANS_ID"] != beanID {
		t.Errorf("BEANS_ID = %v, want %v", envMap["BEANS_ID"], beanID)
	}
	if envMap["BEANS_TASK"] != beanPath {
		t.Errorf("BEANS_TASK = %v, want %v", envMap["BEANS_TASK"], beanPath)
	}
}

func TestCreateExecCommand_MultiLineWithShebang(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatal(err)
	}

	beanID := "def456"
	beanPath := filepath.Join(beansDir, "beans-"+beanID+".md")

	script := "#!/bin/bash\necho hello\necho world"
	cmd, result, err := CreateExecCommand(script, beansDir, beanID, beanPath)
	if err != nil {
		t.Fatalf("CreateExecCommand() error = %v, want nil", err)
	}
	defer result.Cleanup()

	// Verify command uses bash
	if !strings.Contains(cmd.Path, "bash") {
		t.Errorf("CreateExecCommand() cmd.Path = %v, want bash", cmd.Path)
	}

	// Verify stdin contains the script
	if cmd.Stdin == nil {
		t.Error("CreateExecCommand() cmd.Stdin = nil, want script reader")
	}

	// Verify working directory
	expectedDir := tmpDir
	if cmd.Dir != expectedDir {
		t.Errorf("CreateExecCommand() cmd.Dir = %v, want %v", cmd.Dir, expectedDir)
	}

	// Verify environment variables
	envMap := make(map[string]string)
	for _, e := range cmd.Env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	if envMap["BEANS_TASK"] != beanPath {
		t.Errorf("BEANS_TASK = %v, want %v", envMap["BEANS_TASK"], beanPath)
	}
}

func TestCreateExecCommand_MultiLineWithEnvShebang(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatal(err)
	}

	beanID := "ghi789"
	beanPath := filepath.Join(beansDir, "beans-"+beanID+".md")

	script := "#!/usr/bin/env python3\nprint('hello')"
	cmd, result, err := CreateExecCommand(script, beansDir, beanID, beanPath)
	if err != nil {
		t.Fatalf("CreateExecCommand() error = %v, want nil", err)
	}
	defer result.Cleanup()

	// Verify command uses env
	if !strings.Contains(cmd.Path, "env") {
		t.Errorf("CreateExecCommand() cmd.Path = %v, want env", cmd.Path)
	}

	// Verify args include python3
	found := false
	for _, arg := range cmd.Args {
		if arg == "python3" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("CreateExecCommand() cmd.Args = %v, want to contain python3", cmd.Args)
	}
}

func TestCreateExecCommand_MultiLineNoShebang(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatal(err)
	}

	beanID := "jkl012"
	beanPath := filepath.Join(beansDir, "beans-"+beanID+".md")

	script := "echo hello\necho world"
	_, _, err := CreateExecCommand(script, beansDir, beanID, beanPath)
	if err == nil {
		t.Error("CreateExecCommand() error = nil, want error for multi-line script without shebang")
	}

	if !strings.Contains(err.Error(), "shebang") {
		t.Errorf("CreateExecCommand() error = %v, want error mentioning shebang", err)
	}
}

func TestCreateExecCommand_InvalidShebang(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatal(err)
	}

	beanID := "mno345"
	beanPath := filepath.Join(beansDir, "beans-"+beanID+".md")

	script := "#!\necho hello"
	_, _, err := CreateExecCommand(script, beansDir, beanID, beanPath)
	if err == nil {
		t.Error("CreateExecCommand() error = nil, want error for invalid shebang")
	}

	if !strings.Contains(err.Error(), "invalid shebang") {
		t.Errorf("CreateExecCommand() error = %v, want error mentioning invalid shebang", err)
	}
}

func TestCreateExecCommand_CleanupIsNoOp(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatal(err)
	}

	beanID := "pqr678"
	beanPath := filepath.Join(beansDir, "beans-"+beanID+".md")

	_, result, err := CreateExecCommand("echo test", beansDir, beanID, beanPath)
	if err != nil {
		t.Fatalf("CreateExecCommand() error = %v, want nil", err)
	}

	// Cleanup should not panic
	result.Cleanup()
	result.Cleanup() // Should be safe to call multiple times
}
