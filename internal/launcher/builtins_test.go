package launcher

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
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

// stubTools puts scripts named after tools on PATH. Each call appends its
// arguments, one per line, followed by a "--end--" line, to <dir>/<tool>.log
// and then runs the tool's body from bodies (a sh snippet, may be empty).
func stubTools(t *testing.T, bodies map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for tool, body := range bodies {
		script := "#!/bin/sh\nprintf '%s\\n' \"$@\" --end-- >> \"" + filepath.Join(dir, tool+".log") + "\"\n" + body + "\n"
		if err := os.WriteFile(filepath.Join(dir, tool), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

// readCalls returns the argument lists of every call to a stubbed tool.
func readCalls(t *testing.T, dir, tool string) [][]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, tool+".log"))
	if err != nil {
		t.Fatalf("%s was not called: %v", tool, err)
	}
	var calls [][]string
	var call []string
	for _, line := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		if line == "--end--" {
			calls = append(calls, call)
			call = nil
			continue
		}
		call = append(call, line)
	}
	return calls
}

// lastCall returns the arguments of the last call to a stubbed tool.
func lastCall(t *testing.T, dir, tool string) []string {
	t.Helper()
	calls := readCalls(t, dir, tool)
	return calls[len(calls)-1]
}

func lookupBuiltin(t *testing.T, name string) Builtin {
	t.Helper()
	b, ok := LookupBuiltin(name)
	if !ok {
		t.Fatalf("no builtin %q", name)
	}
	return b
}

// runBuiltin runs the named builtin like `beans launch` would, for bean
// beans-abc1 ("Do it"), in dir (the test's cwd if empty), and returns its stdout.
func runBuiltinIn(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	b := lookupBuiltin(t, name)
	root := t.TempDir()
	l := Launch{Name: "builtin." + name, Script: b.Script, Args: args, Builtin: true,
		ProjectRoot: root, BeansDir: filepath.Join(root, ".beans"), BeanID: "beans-abc1", BeanTitle: "Do it"}
	cmd := CreateShellCommand(l)
	cmd.Dir = dir
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("builtin %s: %v\n%s", name, err, stderr.String())
	}
	return string(out)
}

func runBuiltin(t *testing.T, name string, args ...string) string {
	t.Helper()
	return runBuiltinIn(t, "", name, args...)
}

// containsSeq reports whether args contains want as a contiguous run.
func containsSeq(args, want []string) bool {
	for i := 0; i+len(want) <= len(args); i++ {
		if slices.Equal(args[i:i+len(want)], want) {
			return true
		}
	}
	return false
}

// checkForwardedEnv checks that a host builtin passed the bean's env on with
// flag (-e or --env), as single arguments, and nothing else. withChain says
// whether the launch-chain variables (BEANS_LAUNCH_*) go the same way.
func checkForwardedEnv(t *testing.T, args []string, flag string, withChain bool) {
	t.Helper()
	for _, want := range []string{"BEANS_ID=beans-abc1", "BEANS_TASK=Do it"} {
		if !containsSeq(args, []string{flag, want}) {
			t.Errorf("args %q missing %s %q", args, flag, want)
		}
	}
	if got := containsSeq(args, []string{flag, "BEANS_LAUNCH_DEPTH=1"}); got != withChain {
		t.Errorf("args %q: %s BEANS_LAUNCH_DEPTH=1 forwarded = %v, want %v", args, flag, got, withChain)
	}
	for _, a := range args {
		if strings.HasPrefix(a, "BEANS_PROMPT=") {
			t.Errorf("args %q forward unrelated BEANS_PROMPT", args)
		}
	}
}

func TestTmuxWindow(t *testing.T) {
	stubs := stubTools(t, map[string]string{"tmux": ""})
	t.Setenv("BEANS_PROMPT", "stray")
	runBuiltin(t, "tmux-window", "--cwd", "/work", "--", "echo", "a b")

	got := lastCall(t, stubs, "tmux")
	if !containsSeq(got, []string{"new-window", "-c", "/work", "-n", "beans-abc1"}) {
		t.Errorf("tmux args %q missing new-window -c /work -n beans-abc1", got)
	}
	// The command goes as separate arguments, so tmux runs it without a shell
	if !slices.Equal(got[len(got)-3:], []string{"--", "echo", "a b"}) {
		t.Errorf("tmux args %q should end with -- echo 'a b'", got)
	}
	checkForwardedEnv(t, got, "-e", true)
}

func TestTmuxWindowRelativeCwd(t *testing.T) {
	stubs := stubTools(t, map[string]string{"tmux": ""})
	base := t.TempDir()
	if err := os.Mkdir(filepath.Join(base, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	runBuiltinIn(t, base, "tmux-window", "--cwd", "sub", "--", "true")

	want, _ := filepath.EvalSymlinks(filepath.Join(base, "sub"))
	got := lastCall(t, stubs, "tmux")
	if !containsSeq(got, []string{"-c", want}) {
		t.Errorf("tmux args %q missing -c %s", got, want)
	}
}

func TestTmuxSession(t *testing.T) {
	// has-session fails, so the session gets created. Every call records the
	// BEANS_* variables tmux itself runs with, which a server it starts would
	// copy into its global environment.
	tmux := `env | grep '^BEANS_' >> "$0.env"; case $1 in has-session) exit 1 ;; esac`
	chain := []string{"env", "BEANS_LAUNCH_CONFIG=", "BEANS_LAUNCH_LAYER=", "BEANS_LAUNCH_DEPTH=1"}
	tests := []struct {
		name     string
		tmuxEnv  string
		args     []string
		wantCmd  []string
		wantLast []string
	}{
		{name: "attach", args: []string{"--cwd", "/work"}, wantLast: []string{"attach-session", "-t", "=beans-abc1"}},
		{name: "switch", tmuxEnv: "/tmp/tmux-1/default,1,0", args: []string{"--cwd", "/work", "--", "echo", "a b"},
			wantCmd: append(append([]string{"--"}, chain...), "echo", "a b"), wantLast: []string{"switch-client", "-t", "=beans-abc1"}},
		// A single argument is a shell command, as tmux would run it
		{name: "shell command", args: []string{"--cwd", "/work", "--", "vim; bash"},
			wantCmd: append(append([]string{"--"}, chain...), "/bin/sh", "-c", "vim; bash"), wantLast: []string{"attach-session", "-t", "=beans-abc1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubs := stubTools(t, map[string]string{"tmux": tmux})
			t.Setenv("TMUX", tt.tmuxEnv)
			t.Setenv("SHELL", "/bin/sh")
			t.Setenv("BEANS_PROMPT", "stray")
			runBuiltin(t, "tmux-session", tt.args...)

			calls := readCalls(t, stubs, "tmux")
			if len(calls) != 3 {
				t.Fatalf("tmux calls = %q, want has-session, new-session and attach/switch", calls)
			}
			newSession := calls[1]
			if !containsSeq(newSession, []string{"new-session", "-d", "-s", "beans-abc1", "-c", "/work"}) {
				t.Errorf("new-session args = %q", newSession)
			}
			// -e sets the session's environment, which every later window
			// (and, without a command, the session's own shell) inherits
			checkForwardedEnv(t, newSession, "-e", false)
			if tt.wantCmd != nil {
				if !slices.Equal(newSession[len(newSession)-len(tt.wantCmd):], tt.wantCmd) {
					t.Errorf("new-session args %q should end with %q", newSession, tt.wantCmd)
				}
			} else {
				if slices.Contains(newSession, "--") {
					t.Errorf("new-session args %q should have no command", newSession)
				}
				for _, a := range newSession {
					if strings.Contains(a, "BEANS_LAUNCH_") {
						t.Errorf("new-session args %q pass launch-chain variable %q to the interactive shell", newSession, a)
					}
				}
			}
			if !slices.Equal(calls[2], tt.wantLast) {
				t.Errorf("last tmux call = %q, want %q", calls[2], tt.wantLast)
			}
			tmuxEnv, _ := os.ReadFile(filepath.Join(stubs, "tmux.env"))
			if strings.Contains(string(tmuxEnv), "BEANS_LAUNCH_") {
				t.Errorf("tmux ran with launch-chain variables, a new server would keep them:\n%s", tmuxEnv)
			}
		})
	}
}

// scriptLines returns a script's lines with continuations joined.
func scriptLines(script string) []string {
	lines := strings.Split(strings.ReplaceAll(script, "\\\n", " "), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return lines
}

// forwardedVars returns the variable names a host builtin forwards, from its
// `for var in ...; do` loops.
func forwardedVars(t *testing.T, script string) []string {
	t.Helper()
	var names []string
	for _, line := range scriptLines(script) {
		rest, ok := strings.CutPrefix(line, "for var in ")
		if !ok {
			continue
		}
		list, _, ok := strings.Cut(rest, ";")
		if !ok {
			t.Fatalf("unexpected loop %q", line)
		}
		names = append(names, strings.Fields(list)...)
	}
	slices.Sort(names)
	return names
}

// unsetVars returns the BEANS_* variables a builtin's `unset` lines remove,
// and whether it has any.
func unsetVars(script string) ([]string, bool) {
	var names []string
	found := false
	for _, line := range scriptLines(script) {
		rest, ok := strings.CutPrefix(line, "unset ")
		if !ok || !strings.HasPrefix(rest, "BEANS_") {
			continue
		}
		found = true
		names = append(names, strings.Fields(rest)...)
	}
	slices.Sort(names)
	return names, found
}

func TestHostBuiltinsForwardLaunchEnv(t *testing.T) {
	base := len(os.Environ())
	var want []string
	for _, kv := range (Launch{}).Env()[base:] {
		name, _, _ := strings.Cut(kv, "=")
		want = append(want, name)
	}
	slices.Sort(want)

	for _, name := range []string{"tmux-window", "tmux-session", "kitty-window"} {
		t.Run(name, func(t *testing.T) {
			script := lookupBuiltin(t, name).Script
			if got := forwardedVars(t, script); !slices.Equal(got, want) {
				t.Errorf("forwarded variables = %q, want Launch.Env's %q", got, want)
			}
			if got, ok := unsetVars(script); ok && !slices.Equal(got, want) {
				t.Errorf("unset variables = %q, want Launch.Env's %q", got, want)
			}
		})
	}
}

func TestTmuxSessionExisting(t *testing.T) {
	stubs := stubTools(t, map[string]string{"tmux": ""})
	t.Setenv("TMUX", "")
	runBuiltin(t, "tmux-session", "--cwd", "/work")

	// has-session succeeds, so the session is only attached
	calls := readCalls(t, stubs, "tmux")
	if len(calls) != 2 || !slices.Equal(calls[1], []string{"attach-session", "-t", "=beans-abc1"}) {
		t.Errorf("tmux calls = %q", calls)
	}
}

func TestKittyWindow(t *testing.T) {
	stubs := stubTools(t, map[string]string{"kitty": ""})
	t.Setenv("BEANS_PROMPT", "stray")
	runBuiltin(t, "kitty-window", "--cwd", "/work", "--", "echo", "hi")

	got := lastCall(t, stubs, "kitty")
	if !containsSeq(got, []string{"@", "launch", "--type=window", "--cwd=/work"}) {
		t.Errorf("kitty args %q missing @ launch --type=window --cwd=/work", got)
	}
	if !slices.Equal(got[len(got)-2:], []string{"echo", "hi"}) {
		t.Errorf("kitty args %q should end with echo hi", got)
	}
	checkForwardedEnv(t, got, "--env", true)
}

func TestHostBuiltinsRejectMissingFlagValue(t *testing.T) {
	stubTools(t, map[string]string{"tmux": "", "kitty": "", "git": ""})
	for _, tt := range []struct{ name, flag string }{
		{"tmux-window", "--cwd"}, {"tmux-window", "--name"}, {"tmux-session", "--cwd"},
		{"kitty-window", "--cwd"}, {"worktree", "--dir"}, {"worktree", "--base"}, {"prompt", "--worktree"},
	} {
		t.Run(tt.name+" "+tt.flag, func(t *testing.T) {
			b := lookupBuiltin(t, tt.name)
			l := Launch{Name: "builtin." + tt.name, Script: b.Script, Args: []string{tt.flag}, Builtin: true,
				ProjectRoot: t.TempDir(), BeanID: "beans-abc1"}
			out, err := CreateShellCommand(l).CombinedOutput()
			if err == nil {
				t.Fatalf("expected failure, got success:\n%s", out)
			}
			if !strings.Contains(string(out), tt.flag+" needs a value") {
				t.Errorf("output %q should say %s needs a value", out, tt.flag)
			}
		})
	}
}

// gitEnv keeps git independent of the user's global and system config.
var gitEnv = []string{"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
	"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t"}

func TestWorktree(t *testing.T) {
	repo := t.TempDir()
	git := func(t *testing.T, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
		cmd.Env = append(os.Environ(), gitEnv...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git(t, "init", "-q", "-b", "main")
	git(t, "commit", "-q", "--allow-empty", "-m", "init")

	b := lookupBuiltin(t, "worktree")
	run := func(t *testing.T, cwd string, args ...string) string {
		t.Helper()
		l := Launch{Name: "builtin.worktree", Script: b.Script, Args: args, Builtin: true, ProjectRoot: repo, BeanID: "beans-abc1"}
		cmd := CreateShellCommand(l)
		cmd.Env = append(cmd.Env, gitEnv...)
		cmd.Dir = cwd
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("worktree %v: %v\n%s", args, err, stderr.String())
		}
		return strings.TrimSpace(string(out))
	}
	branchIn := func(t *testing.T, dir string) string {
		t.Helper()
		cmd := exec.Command("git", "-C", dir, "branch", "--show-current")
		cmd.Env = append(os.Environ(), gitEnv...)
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("git branch in %s: %v", dir, err)
		}
		return strings.TrimSpace(string(out))
	}
	removeWorktree := func(t *testing.T, dir string) {
		t.Helper()
		git(t, "worktree", "remove", dir)
	}

	t.Run("default location", func(t *testing.T) {
		dir := run(t, "")
		want, _ := filepath.EvalSymlinks(filepath.Join(repo, ".worktrees", "beans-abc1"))
		if dir != want {
			t.Errorf("path = %q, want %q", dir, want)
		}
		if run(t, "") != dir {
			t.Error("second run should reuse the worktree")
		}
	})

	t.Run("branch checked out elsewhere is reused", func(t *testing.T) {
		want, _ := filepath.EvalSymlinks(filepath.Join(repo, ".worktrees", "beans-abc1"))
		other := filepath.Join(t.TempDir(), "other")
		if dir := run(t, "", "--dir", other); dir != want {
			t.Errorf("path = %q, want the existing worktree %q", dir, want)
		}
		removeWorktree(t, want)
	})

	t.Run("custom dir reuses the existing branch", func(t *testing.T) {
		custom := filepath.Join(t.TempDir(), "wt")
		dir := run(t, "", "--dir", custom, "--base", "main")
		if !strings.HasSuffix(dir, "/wt") {
			t.Errorf("path = %q, want .../wt", dir)
		}
		if got := branchIn(t, custom); got != "task/beans-abc1" {
			t.Errorf("branch in %s = %q, want task/beans-abc1", custom, got)
		}
		removeWorktree(t, custom)
	})

	t.Run("relative dir is relative to the caller's cwd", func(t *testing.T) {
		cwd := t.TempDir()
		dir := run(t, cwd, "--dir", "rel/wt")
		want, _ := filepath.EvalSymlinks(filepath.Join(cwd, "rel", "wt"))
		if dir != want {
			t.Errorf("path = %q, want %q", dir, want)
		}
		if got := branchIn(t, want); got != "task/beans-abc1" {
			t.Errorf("branch in %s = %q, want task/beans-abc1", want, got)
		}
		if run(t, cwd, "--dir", "rel/wt") != dir {
			t.Error("second run should reuse the worktree")
		}
		removeWorktree(t, want)
	})
}

// fakeBeans writes a stand-in for the beans binary that answers
// `show --prompt`, `show --json` (with jsonPadding bytes after the status) and
// `launch prompt`, logging its args.
func fakeBeans(t *testing.T, status string, jsonPadding int) (bin, logPath string) {
	t.Helper()
	dir := t.TempDir()
	logPath = filepath.Join(dir, "beans.log")
	bin = filepath.Join(dir, "beans")
	script := `#!/bin/sh
printf '%s\n' "$*" >> "` + logPath + `"
case "$*" in
  *"show --prompt"*) echo "BEAN PROMPT" ;;
  *"show --json"*)
    printf '{\n  "id": "beans-abc1",\n  "status": "` + status + `",\n  "body": "'
    head -c ` + strconv.Itoa(jsonPadding) + ` /dev/zero | tr '\0' '\n'
    echo '"}' ;;
  *"launch prompt"*) echo "FULL PROMPT $*" ;;
esac
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin, logPath
}

func beansCommand(t *testing.T, bin, name string, args ...string) *exec.Cmd {
	t.Helper()
	b := lookupBuiltin(t, name)
	l := Launch{Name: "builtin." + name, Script: b.Script, Args: args, Builtin: true,
		ProjectRoot: "/repo", BeansDir: "/repo/.beans", ConfigPath: "/repo/.beans.yml", BeanID: "beans-abc1"}
	cmd := CreateShellCommand(l)
	cmd.Env = append(cmd.Env, "BEANS_BIN="+bin)
	return cmd
}

func runWithBeans(t *testing.T, bin, name string, args ...string) string {
	t.Helper()
	cmd := beansCommand(t, bin, name, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, stderr.String())
	}
	return string(out)
}

func TestPrompt(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		padding int
		args    []string
		want    []string
		notWant []string
	}{
		{name: "plain", status: "todo", want: []string{"BEAN PROMPT"}, notWant: []string{"## tmux window", "## Worktree", "## Draft"}},
		{name: "tmux", status: "todo", args: []string{"--tmux"}, want: []string{"## tmux window", "tmux rename-window"}},
		{name: "worktree", status: "todo", args: []string{"--worktree", "/wt"}, want: []string{"## Worktree", "/wt", "task/beans-abc1"}},
		{name: "draft", status: "draft", want: []string{"## Draft: plan first", "beans update beans-abc1 --status in-progress"}},
		{name: "large draft", status: "draft", padding: 200_000, want: []string{"## Draft: plan first"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bin, logPath := fakeBeans(t, tt.status, tt.padding)
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
			for _, want := range []string{
				"--config /repo/.beans.yml --beans-path /repo/.beans show --prompt beans-abc1",
				"--config /repo/.beans.yml --beans-path /repo/.beans show --json beans-abc1",
			} {
				if !strings.Contains(string(calls), want) {
					t.Errorf("beans not pinned to the launch's config, missing %q:\n%s", want, calls)
				}
			}
		})
	}
}

func TestAgent(t *testing.T) {
	// Each agent records whether the launch-chain variables reached it
	env := `echo "${BEANS_LAUNCH_CONFIG-unset} ${BEANS_LAUNCH_LAYER-unset} ${BEANS_LAUNCH_DEPTH-unset} ${BEANS_ID-unset}" > "$0.env"`
	stubs := stubTools(t, map[string]string{"claude": env, "opencode": env, "crush": env})
	bin, _ := fakeBeans(t, "todo", 0)

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
			if got := lastCall(t, stubs, tt.agent); !slices.Equal(got, tt.want) {
				t.Errorf("%s args = %q, want %q", tt.agent, got, tt.want)
			}
			// The agent's shell is no longer part of the launch chain
			gotEnv, err := os.ReadFile(filepath.Join(stubs, tt.agent+".env"))
			if err != nil {
				t.Fatal(err)
			}
			if want := "unset unset unset beans-abc1\n"; string(gotEnv) != want {
				t.Errorf("agent env = %q, want %q", gotEnv, want)
			}
		})
	}
}

func TestAgentUnknown(t *testing.T) {
	bin, logPath := fakeBeans(t, "todo", 0)
	cmd := beansCommand(t, bin, "agent", "vim", "--tmux")
	out, err := cmd.CombinedOutput()
	if code := ExitCode(err); code != 2 {
		t.Errorf("exit code = %d, want 2\n%s", code, out)
	}
	if !strings.Contains(string(out), "unknown agent 'vim'") {
		t.Errorf("output %q should name the unknown agent", out)
	}
	if _, err := os.Stat(logPath); err == nil {
		t.Error("beans should not be called for an unknown agent")
	}
}
