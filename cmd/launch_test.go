package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/config"
	launcherexec "github.com/hmans/beans/internal/launcher"
)

func TestFirstLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"single id", "beans-abc1\n", "beans-abc1"},
		{"no trailing newline", "beans-abc1", "beans-abc1"},
		{"leading blank lines and spaces", "\n\n  beans-abc1  \nbeans-def2\n", "beans-abc1"},
		{"empty", "", ""},
		{"only whitespace", " \n\t\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := firstLine(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("firstLine() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("firstLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLaunchBeanID(t *testing.T) {
	pipe := func(t *testing.T, content string) *os.File {
		t.Helper()
		path := filepath.Join(t.TempDir(), "stdin")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { f.Close() })
		return f
	}

	tests := []struct {
		name          string
		args          []string
		envID         string
		stdin         string
		wantID        string
		wantFromStdin bool
		wantErr       bool
	}{
		{name: "argument wins", args: []string{"beans-abc1"}, envID: "beans-env1", stdin: "beans-def2\n", wantID: "beans-abc1"},
		{name: "BEANS_ID before stdin", envID: "beans-env1", stdin: "beans-def2\n", wantID: "beans-env1"},
		{name: "from stdin", stdin: "beans-def2\n", wantID: "beans-def2", wantFromStdin: true},
		{name: "empty stdin", stdin: "", wantFromStdin: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, fromStdin, err := launchBeanID(tt.args, tt.envID, pipe(t, tt.stdin))
			if (err != nil) != tt.wantErr {
				t.Fatalf("launchBeanID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if id != tt.wantID || fromStdin != tt.wantFromStdin {
				t.Errorf("launchBeanID() = (%q, %v), want (%q, %v)", id, fromStdin, tt.wantID, tt.wantFromStdin)
			}
		})
	}
}

func TestSplitLaunchArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		dash       int
		wantPos    []string
		wantScript []string
	}{
		{"no dash", []string{"claude", "beans-abc1"}, -1, []string{"claude", "beans-abc1"}, nil},
		{"dash", []string{"tmux-window", "--cwd", "/x", "--", "cmd"}, 1, []string{"tmux-window"}, []string{"--cwd", "/x", "--", "cmd"}},
		{"dash with id", []string{"agent", "beans-abc1", "claude"}, 2, []string{"agent", "beans-abc1"}, []string{"claude"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, script := splitLaunchArgs(tt.args, tt.dash)
			if strings.Join(pos, " ") != strings.Join(tt.wantPos, " ") || strings.Join(script, " ") != strings.Join(tt.wantScript, " ") {
				t.Errorf("splitLaunchArgs() = (%q, %q), want (%q, %q)", pos, script, tt.wantPos, tt.wantScript)
			}
		})
	}
}

func TestLaunchDepth(t *testing.T) {
	tests := []struct {
		env     string
		want    int
		wantErr string
	}{
		{"", 0, ""},
		{"3", 3, ""},
		{"16", 0, "nested more than 16"},
		{"x", 0, "invalid BEANS_LAUNCH_DEPTH"},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			got, err := launchDepth(tt.env)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("launchDepth(%q) error = %v, want containing %q", tt.env, err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("launchDepth(%q) = (%d, %v), want %d", tt.env, got, err, tt.want)
			}
		})
	}
}

func TestListLaunches(t *testing.T) {
	launchers := []config.Launcher{
		{Name: "claude", Exec: "claude", Description: "Open in Claude"},
		{Name: "tmux-worktree", Exec: "tmux new-window", Multiple: true, Background: true},
	}
	builtins := []launcherexec.Builtin{{Name: "agent", Description: "Start an agent"}}

	t.Run("table", func(t *testing.T) {
		var buf bytes.Buffer
		if err := listLaunches(&buf, launchers, builtins, false); err != nil {
			t.Fatal(err)
		}
		want := "claude         Open in Claude\n" +
			"tmux-worktree  [multiple, background]\n" +
			"\n" +
			"builtin.agent  Start an agent\n"
		if buf.String() != want {
			t.Errorf("table output =\n%q\nwant\n%q", buf.String(), want)
		}
	})

	t.Run("table without launchers", func(t *testing.T) {
		var buf bytes.Buffer
		if err := listLaunches(&buf, nil, builtins, false); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(buf.String(), "No launchers configured.\n\nbuiltin.agent") {
			t.Errorf("output = %q", buf.String())
		}
	})

	t.Run("json", func(t *testing.T) {
		var buf bytes.Buffer
		if err := listLaunches(&buf, launchers, builtins, true); err != nil {
			t.Fatal(err)
		}
		var got []launcherInfo
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
		}
		if len(got) != 3 || got[1].Name != "tmux-worktree" || !got[1].Multiple || !got[1].Background {
			t.Errorf("json = %+v", got)
		}
		if got[2].Name != "builtin.agent" || !got[2].Builtin || got[2].Exec != "" {
			t.Errorf("builtin entry = %+v", got[2])
		}
	})

	t.Run("json without launches is an empty array", func(t *testing.T) {
		var buf bytes.Buffer
		if err := listLaunches(&buf, nil, nil, true); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(buf.String()) != "[]" {
			t.Errorf("json = %q, want []", buf.String())
		}
	})
}
