package sessionpoller

import (
	"os"
	"strings"
	"testing"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   int
	}{
		{name: "success", script: "true", want: 0},
		{name: "failure", script: "exit 3", want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := CreateExecCommand(tt.script, t.TempDir()+"/.beans", "beans-x", "X")
			if err != nil {
				t.Fatal(err)
			}
			if got := ExitCode(cmd.Run()); got != tt.want {
				t.Errorf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}

	t.Run("command not startable", func(t *testing.T) {
		cmd, _ := CreateExecCommand("true", "/nonexistent-dir/.beans", "beans-x", "X")
		if got := ExitCode(cmd.Run()); got != -1 {
			t.Errorf("ExitCode() = %d, want -1", got)
		}
	})
}

func TestCreateLoggedExecCommand(t *testing.T) {
	cmd, logFile, err := CreateLoggedExecCommand(`echo out; echo "err $BEANS_ID" >&2; exit 2`, t.TempDir()+"/.beans", "beans-x", "X")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(logFile.Name())

	// Stdin/stdout must stay unset so tea.ExecProcess attaches the terminal
	if cmd.Stdin != nil || cmd.Stdout != nil {
		t.Errorf("stdin/stdout should be nil, got %v / %v", cmd.Stdin, cmd.Stdout)
	}

	if got := ExitCode(cmd.Run()); got != 2 {
		t.Errorf("exit code = %d, want 2", got)
	}
	logFile.Close()

	data, err := os.ReadFile(logFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(data)); got != "err beans-x" {
		t.Errorf("log = %q, want only stderr %q", got, "err beans-x")
	}
}
