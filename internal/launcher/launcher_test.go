package launcher

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractMainCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
	}{
		{
			name:    "simple command",
			command: "opencode",
			want:    "opencode",
		},
		{
			name:    "command with args",
			command: "opencode run \"Work on task $BEANS_ID\"",
			want:    "opencode",
		},
		{
			name:    "absolute path",
			command: "/usr/bin/tool --flag",
			want:    "/usr/bin/tool",
		},
		{
			name:    "relative path with args",
			command: ".beans/scripts/tool.sh arg1 arg2",
			want:    ".beans/scripts/tool.sh",
		},
		{
			name:    "command with quotes",
			command: "claude \"Work on task $BEANS_ID\"",
			want:    "claude",
		},
		{
			name:    "empty command",
			command: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMainCommand(tt.command)
			if got != tt.want {
				t.Errorf("extractMainCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractShebangInterpreter(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   string
	}{
		{
			name:   "bash shebang",
			script: "#!/bin/bash\necho hello",
			want:   "/bin/bash",
		},
		{
			name:   "env python",
			script: "#!/usr/bin/env python3\nprint('hi')",
			want:   "python3",
		},
		{
			name:   "env ruby",
			script: "#!/usr/bin/env ruby\nputs 'hi'",
			want:   "ruby",
		},
		{
			name:   "bin env variant",
			script: "#!/bin/env python3\nprint('hi')",
			want:   "python3",
		},
		{
			name:   "sh shebang",
			script: "#!/bin/sh\necho test",
			want:   "/bin/sh",
		},
		{
			name:   "no shebang",
			script: "echo hello\necho world",
			want:   "",
		},
		{
			name:   "empty script",
			script: "",
			want:   "",
		},
		{
			name:   "shebang with extra spaces",
			script: "#!  /bin/bash  \necho test",
			want:   "/bin/bash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractShebangInterpreter(tt.script)
			if got != tt.want {
				t.Errorf("extractShebangInterpreter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsLauncherAvailable(t *testing.T) {
	// Create temp file for path testing
	tmpDir := t.TempDir()
	testScript := filepath.Join(tmpDir, "test.sh")
	if err := os.WriteFile(testScript, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		execScript string
		want       bool
	}{
		{
			name:       "single-line with PATH command (ls)",
			execScript: "ls -la",
			want:       true,
		},
		{
			name:       "single-line with non-existent command",
			execScript: "nonexistent-command-xyz-123",
			want:       false,
		},
		{
			name:       "single-line with existing file path",
			execScript: testScript,
			want:       true,
		},
		{
			name:       "single-line with non-existent file path",
			execScript: filepath.Join(tmpDir, "nonexistent.sh"),
			want:       false,
		},
		{
			name:       "multi-line with env bash",
			execScript: "#!/usr/bin/env bash\necho hello",
			want:       commandExists("bash"), // bash may or may not be in PATH
		},
		{
			name:       "multi-line with valid sh shebang",
			execScript: "#!/bin/sh\necho hello",
			want:       true,
		},
		{
			name:       "multi-line with env python (if installed)",
			execScript: "#!/usr/bin/env python3\nprint('hi')",
			want:       commandExists("python3"), // May or may not be installed
		},
		{
			name:       "multi-line without shebang",
			execScript: "echo hello\necho world",
			want:       false,
		},
		{
			name:       "multi-line with invalid interpreter",
			execScript: "#!/usr/bin/nonexistent-interpreter\necho test",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLauncherAvailable(tt.execScript)
			if got != tt.want {
				t.Errorf("IsLauncherAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateExecCommand(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		execScript string
		beanID     string
		beansRoot  string
		beanPath   string
		wantEnv    map[string]string
		wantDir    string
	}{
		{
			name:       "single-line command",
			execScript: "echo hello",
			beanID:     "abc123",
			beansRoot:  "/project",
			beanPath:   "/project/.beans/beans-abc123.md",
			wantEnv: map[string]string{
				"BEANS_ID":   "abc123",
				"BEANS_ROOT": "/project",
				"BEANS_TASK": "/project/.beans/beans-abc123.md",
			},
			wantDir: "/project",
		},
		{
			name:       "multi-line script",
			execScript: "#!/bin/bash\necho hello\necho world",
			beanID:     "def456",
			beansRoot:  tmpDir,
			beanPath:   filepath.Join(tmpDir, ".beans", "beans-def456.md"),
			wantEnv: map[string]string{
				"BEANS_ID":   "def456",
				"BEANS_ROOT": tmpDir,
				"BEANS_TASK": filepath.Join(tmpDir, ".beans", "beans-def456.md"),
			},
			wantDir: tmpDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreateExecCommand(tt.execScript, tt.beanID, tt.beansRoot, tt.beanPath)
			defer result.Cleanup()

			// Verify command is not nil
			if result.Cmd == nil {
				t.Fatal("CreateExecCommand returned nil Cmd")
			}

			// Verify working directory
			if result.Cmd.Dir != tt.wantDir {
				t.Errorf("Cmd.Dir = %v, want %v", result.Cmd.Dir, tt.wantDir)
			}

			// Verify environment variables are set
			env := make(map[string]string)
			for _, e := range result.Cmd.Env {
				parts := strings.SplitN(e, "=", 2)
				if len(parts) == 2 {
					env[parts[0]] = parts[1]
				}
			}

			for key, wantVal := range tt.wantEnv {
				if gotVal, ok := env[key]; !ok {
					t.Errorf("Environment variable %s not set", key)
				} else if gotVal != wantVal {
					t.Errorf("Environment variable %s = %v, want %v", key, gotVal, wantVal)
				}
			}

			// For multi-line, verify temp file was created
			if strings.Contains(tt.execScript, "\n") {
				// Verify the command is executing a temp file
				if len(result.Cmd.Args) == 0 {
					t.Error("Multi-line script should execute temp file")
				} else {
					tmpPath := result.Cmd.Args[0]
					// Verify temp file exists
					if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
						t.Errorf("Temp file does not exist: %s", tmpPath)
					}
					// Cleanup should remove it
					result.Cleanup()
					if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
						t.Errorf("Cleanup did not remove temp file: %s", tmpPath)
					}
				}
			}
		})
	}
}

// Helper function to check if a command exists
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
