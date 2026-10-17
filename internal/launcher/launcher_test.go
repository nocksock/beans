package launcher

import (
	"strings"
	"testing"
)

func TestCreateExecCommand_EnvironmentVariables(t *testing.T) {
	tests := []struct {
		name        string
		execScript  string
		projectRoot string
		configDir   string
		beansDir    string
		beanID      string
		beanTitle   string
		wantEnvVars map[string]string
	}{
		{
			name:        "single-line script has all env vars",
			execScript:  "echo $BEANS_ROOT $BEANS_DIR $BEANS_ID $BEANS_TASK",
			projectRoot: "/project",
			beansDir:    "/project/.beans",
			beanID:      "beans-xyz",
			beanTitle:   "Test Task",
			wantEnvVars: map[string]string{
				"BEANS_ROOT": "/project",
				"BEANS_DIR":  "/project/.beans",
				"BEANS_ID":   "beans-xyz",
				"BEANS_TASK": "Test Task",
			},
		},
		{
			name:        "multi-line script has all env vars",
			execScript:  "#!/bin/bash\necho $BEANS_ROOT $BEANS_DIR $BEANS_ID $BEANS_TASK",
			projectRoot: "/home/user/project",
			beansDir:    "/home/user/project/.beans",
			beanID:      "beans-abc",
			beanTitle:   "Another Task",
			wantEnvVars: map[string]string{
				"BEANS_ROOT": "/home/user/project",
				"BEANS_DIR":  "/home/user/project/.beans",
				"BEANS_ID":   "beans-abc",
				"BEANS_TASK": "Another Task",
			},
		},
		{
			name:        "inherited launcher with beans dir in parent",
			execScript:  "env",
			projectRoot: "/home/user/workspace/pkg",
			configDir:   "/home/user/workspace",
			beansDir:    "/home/user/workspace/.beans",
			beanID:      "beans-123",
			beanTitle:   "Task Title",
			wantEnvVars: map[string]string{
				"BEANS_ROOT":       "/home/user/workspace/pkg",
				"BEANS_CONFIG_DIR": "/home/user/workspace",
				"BEANS_DIR":        "/home/user/workspace/.beans",
				"BEANS_ID":         "beans-123",
				"BEANS_TASK":       "Task Title",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, result, err := CreateExecCommand(Launch{
				Script:      tt.execScript,
				ProjectRoot: tt.projectRoot,
				ConfigDir:   tt.configDir,
				BeansDir:    tt.beansDir,
				BeanID:      tt.beanID,
				BeanTitle:   tt.beanTitle,
			})
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
	cmd, result, err := CreateExecCommand(Launch{Script: "echo hello", ProjectRoot: "/project", BeansDir: "/project/.beans", BeanID: "beans-123", BeanTitle: "Test"})
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
	cmd, result, err := CreateExecCommand(Launch{Script: script, ProjectRoot: "/project", BeansDir: "/project/.beans", BeanID: "beans-456", BeanTitle: "Multi Test"})
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
	_, result, err := CreateExecCommand(Launch{Script: script, ProjectRoot: "/project", BeansDir: "/project/.beans", BeanID: "beans-789", BeanTitle: "No Shebang"})
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
	_, result, err := CreateExecCommand(Launch{Script: script, ProjectRoot: "/project", BeansDir: "/project/.beans", BeanID: "beans-999", BeanTitle: "Invalid Shebang"})
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
