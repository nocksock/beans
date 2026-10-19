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
