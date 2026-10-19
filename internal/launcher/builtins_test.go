package launcher

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltinsAreWellFormed(t *testing.T) {
	builtins := Builtins()
	if len(builtins) == 0 {
		t.Fatal("no builtins embedded")
	}
	for _, b := range builtins {
		t.Run(b.Name, func(t *testing.T) {
			if !strings.HasPrefix(b.Script, "#!/usr/bin/env bash\n") {
				t.Error("script must start with #!/usr/bin/env bash")
			}
			if b.Description == "" {
				t.Error("missing description (second line, '# ...')")
			}
			if out, err := exec.Command("bash", "-n", "-c", b.Script).CombinedOutput(); err != nil {
				t.Errorf("bash -n: %v\n%s", err, out)
			}
		})
	}
}

func TestLookupBuiltin(t *testing.T) {
	if b, ok := LookupBuiltin("tmux-window"); !ok || b.Name != "tmux-window" {
		t.Errorf("LookupBuiltin(tmux-window) = (%+v, %v)", b, ok)
	}
	if _, ok := LookupBuiltin("nope"); ok {
		t.Error("LookupBuiltin(nope) found")
	}
}

// stubTools puts scripts named after tools on PATH; each logs its arguments,
// one per line, to <dir>/<tool>.log and prints stdout.
func stubTools(t *testing.T, stdout map[string]string, tools ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, tool := range tools {
		script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + filepath.Join(dir, tool+".log") + "\"\nprintf '%s' '" + stdout[tool] + "'\n"
		if err := os.WriteFile(filepath.Join(dir, tool), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

func readLog(t *testing.T, dir, tool string) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, tool+".log"))
	if err != nil {
		t.Fatalf("%s was not called: %v", tool, err)
	}
	return strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
}

// runBuiltin runs the named builtin like `beans launch` would, for bean
// beans-abc1, and returns its stdout.
func runBuiltin(t *testing.T, name string, args ...string) string {
	t.Helper()
	b, ok := LookupBuiltin(name)
	if !ok {
		t.Fatalf("no builtin %q", name)
	}
	root := t.TempDir()
	l := Launch{Name: "builtin." + name, Script: b.Script, Args: args, Builtin: true,
		ProjectRoot: root, BeansDir: filepath.Join(root, ".beans"), BeanID: "beans-abc1", BeanTitle: "Do it"}
	cmd := CreateShellCommand(l)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("builtin %s: %v\n%s", name, err, stderr.String())
	}
	return string(out)
}

func TestTmuxWindow(t *testing.T) {
	stubs := stubTools(t, nil, "tmux")
	runBuiltin(t, "tmux-window", "--cwd", "/work", "--", "echo", "a b")

	got := strings.Join(readLog(t, stubs, "tmux"), " ")
	for _, want := range []string{"new-window -c /work -n beans-abc1", "-e BEANS_ID=beans-abc1", `echo a\ b`} {
		if !strings.Contains(got, want) {
			t.Errorf("tmux args %q missing %q", got, want)
		}
	}
}

func TestTmuxSession(t *testing.T) {
	stubs := stubTools(t, nil, "tmux")
	t.Setenv("TMUX", "")
	runBuiltin(t, "tmux-session", "--cwd", "/work")

	// The stub succeeds for has-session, so the last call attaches
	if got := strings.Join(readLog(t, stubs, "tmux"), " "); got != "attach-session -t =beans-abc1" {
		t.Errorf("last tmux call = %q", got)
	}
}

func TestKittyWindow(t *testing.T) {
	stubs := stubTools(t, nil, "kitty")
	runBuiltin(t, "kitty-window", "--cwd", "/work", "--", "echo", "hi")

	got := strings.Join(readLog(t, stubs, "kitty"), " ")
	for _, want := range []string{"@ launch --type=window --cwd=/work", "--env BEANS_ID=beans-abc1", "echo hi"} {
		if !strings.Contains(got, want) {
			t.Errorf("kitty args %q missing %q", got, want)
		}
	}
}

func TestWorktree(t *testing.T) {
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q", "-b", "main")
	git("commit", "-q", "--allow-empty", "-m", "init")

	b, _ := LookupBuiltin("worktree")
	run := func(args ...string) string {
		t.Helper()
		l := Launch{Name: "builtin.worktree", Script: b.Script, Args: args, Builtin: true, ProjectRoot: repo, BeanID: "beans-abc1"}
		out, err := CreateShellCommand(l).Output()
		if err != nil {
			t.Fatalf("worktree %v: %v", args, err)
		}
		return strings.TrimSpace(string(out))
	}

	t.Run("default location", func(t *testing.T) {
		dir := run()
		want, _ := filepath.EvalSymlinks(filepath.Join(repo, ".worktrees", "beans-abc1"))
		if dir != want {
			t.Errorf("path = %q, want %q", dir, want)
		}
		if run() != dir {
			t.Error("second run should reuse the worktree")
		}
	})

	t.Run("custom dir reuses the existing branch", func(t *testing.T) {
		git("worktree", "remove", filepath.Join(repo, ".worktrees", "beans-abc1"))
		custom := filepath.Join(t.TempDir(), "wt")
		if dir := run("--dir", custom, "--base", "main"); !strings.HasSuffix(dir, "/wt") {
			t.Errorf("path = %q, want .../wt", dir)
		}
	})
}

// fakeBeans writes a stand-in for the beans binary that answers
// `show --prompt`, `show --json` and `launch prompt`, logging its args.
func fakeBeans(t *testing.T, status string) (bin, logPath string) {
	t.Helper()
	dir := t.TempDir()
	logPath = filepath.Join(dir, "beans.log")
	bin = filepath.Join(dir, "beans")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "` + logPath + `"
case "$*" in
  *"show --prompt"*) echo "BEAN PROMPT" ;;
  *"show --json"*) echo '{"id": "beans-abc1", "status": "` + status + `"}' ;;
  *"launch prompt"*) echo "FULL PROMPT $*" ;;
esac
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, logPath
}

func runWithBeans(t *testing.T, bin, name string, args ...string) string {
	t.Helper()
	b, _ := LookupBuiltin(name)
	l := Launch{Name: "builtin." + name, Script: b.Script, Args: args, Builtin: true,
		ProjectRoot: "/repo", BeansDir: "/repo/.beans", ConfigPath: "/repo/.beans.yml", BeanID: "beans-abc1"}
	cmd := CreateShellCommand(l)
	cmd.Env = append(cmd.Env, "BEANS_BIN="+bin)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %v: %v", name, args, err)
	}
	return string(out)
}

func TestPrompt(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		args    []string
		want    []string
		notWant []string
	}{
		{name: "plain", status: "todo", want: []string{"BEAN PROMPT"}, notWant: []string{"## tmux window", "## Worktree", "## Draft"}},
		{name: "tmux", status: "todo", args: []string{"--tmux"}, want: []string{"## tmux window", "tmux rename-window"}},
		{name: "worktree", status: "todo", args: []string{"--worktree", "/wt"}, want: []string{"## Worktree", "/wt", "task/beans-abc1"}},
		{name: "draft", status: "draft", want: []string{"## Draft: plan first", "beans update beans-abc1 --status in-progress"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin, logPath := fakeBeans(t, tt.status)
			out := runWithBeans(t, bin, "prompt", tt.args...)
			for _, w := range tt.want {
				if !strings.Contains(out, w) {
					t.Errorf("prompt missing %q:\n%s", w, out)
				}
			}
			for _, w := range tt.notWant {
				if strings.Contains(out, w) {
					t.Errorf("prompt contains %q:\n%s", w, out)
				}
			}
			calls, _ := os.ReadFile(logPath)
			if !strings.Contains(string(calls), "--config /repo/.beans.yml --beans-path /repo/.beans show --prompt beans-abc1") {
				t.Errorf("beans not pinned to the launch's config:\n%s", calls)
			}
		})
	}
}

func TestAgent(t *testing.T) {
	stubs := stubTools(t, nil, "claude", "opencode", "crush")
	bin, _ := fakeBeans(t, "todo")

	tests := []struct {
		agent string
		want  []string
	}{
		{"claude", []string{"FULL PROMPT launch prompt -- --tmux"}},
		{"opencode", []string{"--prompt", "FULL PROMPT launch prompt -- --tmux"}},
		{"crush", []string{"run", "FULL PROMPT launch prompt -- --tmux"}},
	}
	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			runWithBeans(t, bin, "agent", tt.agent, "--tmux")
			if got := readLog(t, stubs, tt.agent); strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Errorf("%s args = %q, want %q", tt.agent, got, tt.want)
			}
		})
	}
}
