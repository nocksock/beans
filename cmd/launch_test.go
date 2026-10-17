package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/config"
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
		stdin         string
		wantID        string
		wantFromStdin bool
		wantErr       bool
	}{
		{name: "argument wins over stdin", args: []string{"beans-abc1"}, stdin: "beans-def2\n", wantID: "beans-abc1"},
		{name: "from stdin", stdin: "beans-def2\n", wantID: "beans-def2", wantFromStdin: true},
		{name: "empty stdin", stdin: "", wantFromStdin: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, fromStdin, err := launchBeanID(tt.args, pipe(t, tt.stdin))
			if (err != nil) != tt.wantErr {
				t.Fatalf("launchBeanID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if id != tt.wantID || fromStdin != tt.wantFromStdin {
				t.Errorf("launchBeanID() = (%q, %v), want (%q, %v)", id, fromStdin, tt.wantID, tt.wantFromStdin)
			}
		})
	}
}

func TestFindLauncher(t *testing.T) {
	launchers := []config.Launcher{{Name: "claude", Exec: "claude"}, {Name: "tmux", Exec: "tmux"}}

	tests := []struct {
		name      string
		launchers []config.Launcher
		key       string
		wantErr   string
	}{
		{name: "found", launchers: launchers, key: "tmux"},
		{name: "unknown lists available", launchers: launchers, key: "nope", wantErr: "available: claude, tmux"},
		{name: "none configured", key: "claude", wantErr: "no launchers are configured"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := findLauncher(tt.launchers, tt.key)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("findLauncher() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("findLauncher() error = %v", err)
			}
			if l.Name != tt.key {
				t.Errorf("findLauncher() = %q, want %q", l.Name, tt.key)
			}
		})
	}
}

func TestListLaunchers(t *testing.T) {
	launchers := []config.Launcher{
		{Name: "claude", Exec: "claude", Description: "Open in Claude"},
		{Name: "tmux-worktree", Exec: "tmux new-window", Multiple: true, Background: true},
	}

	t.Run("table", func(t *testing.T) {
		var buf bytes.Buffer
		if err := listLaunchers(&buf, launchers, false); err != nil {
			t.Fatal(err)
		}
		want := "claude         Open in Claude\n" +
			"tmux-worktree  [multiple, background]\n"
		if buf.String() != want {
			t.Errorf("table output =\n%q\nwant\n%q", buf.String(), want)
		}
	})

	t.Run("table without launchers", func(t *testing.T) {
		var buf bytes.Buffer
		if err := listLaunchers(&buf, nil, false); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(buf.String(), "No launchers configured") {
			t.Errorf("output = %q", buf.String())
		}
	})

	t.Run("json", func(t *testing.T) {
		var buf bytes.Buffer
		if err := listLaunchers(&buf, launchers, true); err != nil {
			t.Fatal(err)
		}
		var got []launcherInfo
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
		}
		if len(got) != 2 || got[1].Name != "tmux-worktree" || !got[1].Multiple || !got[1].Background {
			t.Errorf("json = %+v", got)
		}
	})

	t.Run("json without launchers is an empty array", func(t *testing.T) {
		var buf bytes.Buffer
		if err := listLaunchers(&buf, nil, true); err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(buf.String()) != "[]" {
			t.Errorf("json = %q, want []", buf.String())
		}
	})
}
