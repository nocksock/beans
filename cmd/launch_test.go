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

func TestLaunchPin(t *testing.T) {
	env := func(m map[string]string) func(string) string {
		return func(k string) string { return m[k] }
	}
	launchEnv := map[string]string{
		"BEANS_LAUNCH_DEPTH":  "1",
		"BEANS_LAUNCH_CONFIG": "/p/.beans.yml",
		"BEANS_DIR":           "/p/.beans",
		"BEANS_ROOT":          "/p",
	}
	noConfigEnv := map[string]string{
		"BEANS_LAUNCH_DEPTH":  "1",
		"BEANS_LAUNCH_CONFIG": "",
		"BEANS_DIR":           "/p/.beans",
		"BEANS_ROOT":          "/p",
	}
	tests := []struct {
		name       string
		configFlag string
		beansFlag  string
		env        map[string]string
		want       launchPaths
	}{
		{name: "top level uses flags", configFlag: "/q/.beans.yml", env: map[string]string{"BEANS_DIR": "/p/.beans"},
			want: launchPaths{configPath: "/q/.beans.yml"}},
		{name: "top level without flags", env: map[string]string{"BEANS_LAUNCH_CONFIG": "/p/.beans.yml"}, want: launchPaths{}},
		{name: "nested is pinned", env: launchEnv,
			want: launchPaths{configPath: "/p/.beans.yml", beansPath: "/p/.beans", pinned: true}},
		{name: "nested without config file searches from BEANS_ROOT", env: noConfigEnv,
			want: launchPaths{beansPath: "/p/.beans", searchDir: "/p", pinned: true}},
		{name: "--config unpins", configFlag: "/q/.beans.yml", env: launchEnv,
			want: launchPaths{configPath: "/q/.beans.yml"}},
		{name: "--beans-path unpins", beansFlag: "/q/.beans", env: launchEnv,
			want: launchPaths{beansPath: "/q/.beans"}},
		{name: "both flags", configFlag: "/q/.beans.yml", beansFlag: "/q/.beans", env: launchEnv,
			want: launchPaths{configPath: "/q/.beans.yml", beansPath: "/q/.beans"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := launchPin(tt.configFlag, tt.beansFlag, env(tt.env))
			if got != tt.want {
				t.Errorf("launchPin() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCheckLaunchFlags(t *testing.T) {
	tests := []struct {
		name    string
		list    bool
		asJSON  bool
		nargs   int
		wantErr string
	}{
		{name: "list", list: true},
		{name: "list json", list: true, asJSON: true},
		{name: "launch", nargs: 2},
		{name: "list with args", list: true, nargs: 1, wantErr: "--list takes no arguments"},
		{name: "json without list", asJSON: true, nargs: 1, wantErr: "--json only works with --list"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkLaunchFlags(tt.list, tt.asJSON, tt.nargs)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("checkLaunchFlags() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("checkLaunchFlags() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestPrepareLaunch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	pkg := filepath.Join(root, "pkg")
	for path, content := range map[string]string{
		filepath.Join(root, ".beans.yml"): "cascade: true\nlaunchers:\n  - name: prompt\n    exec: root-prompt\n",
		filepath.Join(pkg, ".beans.yml"):  "launchers:\n  - name: prompt\n    exec: pkg-prompt\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := config.LoadFromDirectory(pkg)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		args       []string
		dash       int
		pinned     bool
		env        map[string]string
		wantName   string
		wantScript string // prefix
		wantID     string
		wantDepth  int
		wantArgs   []string
		wantErr    string
		wantUsage  bool
	}{
		{name: "configured name", args: []string{"prompt", "beans-a"}, dash: -1, wantName: "prompt", wantScript: "pkg-prompt", wantID: "beans-a"},
		{name: "builtin fallback with script args", args: []string{"agent", "beans-a", "claude", "-x"}, dash: 2,
			wantName: "builtin.agent", wantScript: "#!/usr/bin/env bash", wantID: "beans-a", wantArgs: []string{"claude", "-x"}},
		{name: "parent. with pinned layer", args: []string{"parent.prompt", "beans-a"}, dash: -1, pinned: true,
			env:      map[string]string{"BEANS_LAUNCH_LAYER": root, "BEANS_LAUNCH_DEPTH": "1"},
			wantName: "builtin.prompt", wantScript: "#!/usr/bin/env bash", wantID: "beans-a", wantDepth: 1},
		{name: "parent. not pinned ignores BEANS_LAUNCH_LAYER", args: []string{"parent.prompt", "beans-a"}, dash: -1,
			env:      map[string]string{"BEANS_LAUNCH_LAYER": "/other/project", "BEANS_LAUNCH_DEPTH": "1"},
			wantName: "prompt", wantScript: "root-prompt", wantID: "beans-a", wantDepth: 1},
		{name: "ID from env", args: []string{"prompt"}, dash: -1, env: map[string]string{"BEANS_ID": "beans-env"},
			wantName: "prompt", wantScript: "pkg-prompt", wantID: "beans-env"},
		{name: "depth error", args: []string{"prompt", "beans-a"}, dash: -1, pinned: true,
			env: map[string]string{"BEANS_LAUNCH_DEPTH": "16"}, wantErr: "nested more than 16"},
		{name: "unknown launch", args: []string{"nope", "beans-a"}, dash: -1, wantErr: `unknown launch "nope"`},
		{name: "no launch", args: nil, dash: -1, wantErr: "no launch given", wantUsage: true},
		{name: "too many args", args: []string{"prompt", "beans-a", "extra"}, dash: -1, wantErr: "too many arguments", wantUsage: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdin, err := os.CreateTemp(t.TempDir(), "stdin")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { stdin.Close() })
			getenv := func(k string) string { return tt.env[k] }

			plan, err := prepareLaunch(cfg, tt.args, tt.dash, tt.pinned, getenv, stdin)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("prepareLaunch() error = %v, want containing %q", err, tt.wantErr)
				}
				if isLaunchUsageError(err) != tt.wantUsage {
					t.Errorf("isLaunchUsageError() = %v, want %v", isLaunchUsageError(err), tt.wantUsage)
				}
				return
			}
			if err != nil {
				t.Fatalf("prepareLaunch() error = %v", err)
			}
			if plan.resolved.Name != tt.wantName || !strings.HasPrefix(plan.resolved.Script, tt.wantScript) {
				t.Errorf("resolved = {%q %q}, want {%q %q…}", plan.resolved.Name, plan.resolved.Script, tt.wantName, tt.wantScript)
			}
			if plan.id != tt.wantID || plan.fromStdin || plan.depth != tt.wantDepth {
				t.Errorf("plan = {id %q stdin %v depth %d}, want {id %q stdin false depth %d}", plan.id, plan.fromStdin, plan.depth, tt.wantID, tt.wantDepth)
			}
			if strings.Join(plan.scriptArgs, " ") != strings.Join(tt.wantArgs, " ") {
				t.Errorf("scriptArgs = %q, want %q", plan.scriptArgs, tt.wantArgs)
			}
		})
	}
}
