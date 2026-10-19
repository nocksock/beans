# Builtin Launches Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use mine:subagent-driven-development (recommended) or mine:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship composable builtin launch scripts inside the beans binary and make `beans launch <name> [id] [-- args...]` able to call configured launches, builtins (`builtin.X`) and the next definition up the config cascade (`parent.X`).

**Architecture:** Builtins are bash scripts embedded with `go:embed` in `internal/launcher/builtin/`. A resolver in `internal/launcher` maps a name to a script, using the merged config, the per-layer launcher definitions that `internal/config` now keeps, and the builtins. Every launch exports `BEANS_BIN`, `BEANS_LAUNCH_CONFIG`, `BEANS_LAUNCH_LAYER` and `BEANS_LAUNCH_DEPTH` so nested `beans launch` calls resolve against the same config, find `parent.` relative to their caller and stop runaway recursion.

**Tech Stack:** Go 1.25, cobra/pflag, `embed`, bash, git, tmux/kitty (stubbed in tests).

**Spec:** the plan section of bean `beans-ppxk` (`/home/nr/projects/oss/beans/.beans/beans-ppxk--builtin-launchers.md`).

**Worktree:** `/home/nr/projects/oss/beans/beans-ppxk`, branch `task/beans-ppxk`. All paths below are relative to it. Run `go test ./...` from there.

**Bean bookkeeping (every task):** after each task, tick the matching item in the bean's `### Tasks` list and append a work log line (`YYYY-MM-DD HH:MM: <message>`) to the bean file in the hub (`/home/nr/projects/oss/beans/.beans/beans-ppxk--builtin-launchers.md`). Commit bean edits in the hub (`git -C /home/nr/projects/oss/beans add .beans && git -C /home/nr/projects/oss/beans commit -m "chore: update beans-ppxk"`), code in the worktree.

**CLI detail to keep in mind:** pflag stops parsing at the first `--` and passes everything after it verbatim, including later `--`. Script flags therefore always go after `--`:

    beans launch tmux-window -- --cwd "$dir" -- "$BEANS_BIN" launch agent -- claude

---

## File Structure

| File | Responsibility |
|---|---|
| `internal/config/launchers.go` (new) | Per-layer launcher definitions, `Launcher`, `ParentLauncher`, `LoadedPath` |
| `internal/config/launchers_test.go` (new) | Tests for the above |
| `internal/config/config.go` | Store `launcherLayers` in `fromLayers`; reject reserved name prefixes in `validate` |
| `internal/launcher/launch.go` | `Launch` gains `Name`, `Args`, `Builtin`, `ConfigPath`, `Depth`; env for nested calls |
| `internal/launcher/exec.go` | Pass `$0`/`$@`, pick `bash` for builtins |
| `internal/launcher/builtins.go` (new) | Embedded builtin registry |
| `internal/launcher/builtin/*.sh` (new) | `tmux-window`, `tmux-session`, `kitty-window`, `worktree`, `prompt`, `agent` |
| `internal/launcher/builtins_test.go` (new) | Registry + per-script tests with stubbed tools |
| `internal/launcher/resolve.go` (new) | `Resolve(cfg, name, callerLayer)` |
| `internal/launcher/resolve_test.go` (new) | Resolution table |
| `cmd/launch.go` | New CLI shape, ID chain, `--` args, depth guard, `--list` with builtins |
| `cmd/root.go` | Pin config and beans dir for nested `beans launch` |
| `cmd/launch_test.go` | Updated tests |
| `internal/tui/tui.go` | `newLaunch` passes `ConfigPath` |
| `README.md` | Launch docs |

---

### Task 1: Config keeps per-layer launchers

**Files:**
- Create: `internal/config/launchers.go`, `internal/config/launchers_test.go`
- Modify: `internal/config/config.go` (`Config` struct ~line 106, `fromLayers` ~line 237, `validate` ~line 311)

- [ ] **Step 1: Write the failing tests**

`internal/config/launchers_test.go`:

```go
package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParentLauncher(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	writeTree(t, xdg, map[string]string{
		"beans/config.yml": "launchers:\n  - name: tmux\n    exec: user-tmux\n  - name: only-user\n    exec: user-only\n",
	})
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		".beans.yml":     "cascade: true\nlaunchers:\n  - name: tmux\n    exec: root-tmux\n",
		"pkg/.beans.yml": "launchers:\n  - name: tmux\n    exec: pkg-tmux\n",
	})
	pkg := filepath.Join(root, "pkg")
	userDir := filepath.Join(xdg, "beans")

	cfg, err := LoadFromDirectory(pkg)
	if err != nil {
		t.Fatalf("LoadFromDirectory() error = %v", err)
	}

	tests := []struct {
		name     string
		launcher string
		layer    string
		wantExec string
		wantDir  string
		wantOK   bool
		wantErr  string
	}{
		{name: "parent of leaf is root", launcher: "tmux", layer: pkg, wantExec: "root-tmux", wantDir: root, wantOK: true},
		{name: "parent of root is user", launcher: "tmux", layer: root, wantExec: "user-tmux", wantDir: userDir, wantOK: true},
		{name: "nothing above user", launcher: "tmux", layer: userDir},
		{name: "skips layers without it", launcher: "only-user", layer: pkg, wantExec: "user-only", wantDir: userDir, wantOK: true},
		{name: "unknown name", launcher: "nope", layer: pkg},
		{name: "unknown layer", launcher: "tmux", layer: "/elsewhere", wantErr: "not part of this config"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, ok, err := cfg.ParentLauncher(tt.launcher, tt.layer)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ParentLauncher() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParentLauncher() error = %v", err)
			}
			if ok != tt.wantOK || l.Exec != tt.wantExec || l.Dir != tt.wantDir {
				t.Errorf("ParentLauncher() = (%q in %q, %v), want (%q in %q, %v)", l.Exec, l.Dir, ok, tt.wantExec, tt.wantDir, tt.wantOK)
			}
		})
	}
}

func TestLauncher(t *testing.T) {
	cfg := Default()
	cfg.Launchers = []Launcher{{Name: "claude", Exec: "claude"}}

	if l, ok := cfg.Launcher("claude"); !ok || l.Exec != "claude" {
		t.Errorf("Launcher(claude) = (%+v, %v)", l, ok)
	}
	if _, ok := cfg.Launcher("nope"); ok {
		t.Error("Launcher(nope) found")
	}
}

func TestLoadedPath(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{".beans.yml": "beans:\n  prefix: x-\n"})

	cfg, err := LoadFromDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, ".beans.yml"); cfg.LoadedPath() != want {
		t.Errorf("LoadedPath() = %q, want %q", cfg.LoadedPath(), want)
	}

	empty, err := LoadFromDirectory(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if empty.LoadedPath() != "" {
		t.Errorf("LoadedPath() without a file = %q, want empty", empty.LoadedPath())
	}
}

func TestValidateRejectsReservedLauncherNames(t *testing.T) {
	for _, name := range []string{"builtin.tmux", "parent.tmux"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeTree(t, root, map[string]string{".beans.yml": "launchers:\n  - name: " + name + "\n    exec: x\n"})
			if _, err := LoadFromDirectory(root); err == nil || !strings.Contains(err.Error(), "reserved") {
				t.Errorf("LoadFromDirectory() error = %v, want reserved-name error", err)
			}
		})
	}
}
```

- [ ] **Step 2: Run the tests and watch them fail**

Run: `go test ./internal/config/ -run 'TestParentLauncher|TestLauncher$|TestLoadedPath|TestValidateRejectsReserved'`
Expected: compile error, `cfg.ParentLauncher undefined` (and `Launcher`, `LoadedPath`).

- [ ] **Step 3: Implement**

`internal/config/launchers.go`:

```go
package config

import (
	"fmt"
	"slices"

	"gopkg.in/yaml.v3"
)

// launcherLayer is the launchers one config file declares, before merging.
type launcherLayer struct {
	dir       string
	launchers []Launcher
}

// decodeLauncherLayers reads the launchers each layer declares (root-first),
// each tagged with its layer's directory.
func decodeLauncherLayers(layers []layer) ([]launcherLayer, error) {
	result := make([]launcherLayer, 0, len(layers))
	for _, l := range layers {
		ll := launcherLayer{dir: l.dir}
		if items, ok := l.data["launchers"]; ok {
			raw, err := yaml.Marshal(items)
			if err != nil {
				return nil, err
			}
			if err := yaml.Unmarshal(raw, &ll.launchers); err != nil {
				return nil, fmt.Errorf("launchers in %s: %w", l.dir, err)
			}
			for i := range ll.launchers {
				ll.launchers[i].Dir = l.dir
			}
		}
		result = append(result, ll)
	}
	return result, nil
}

// Launcher returns the launcher with the given name from the merged config.
func (c *Config) Launcher(name string) (Launcher, bool) {
	i := slices.IndexFunc(c.Launchers, func(l Launcher) bool { return l.Name == name })
	if i < 0 {
		return Launcher{}, false
	}
	return c.Launchers[i], true
}

// ParentLauncher returns the launcher name as declared by the nearest config
// layer above the one in layerDir: the definition a launcher in layerDir
// overrides. ok is false when no layer above declares it.
func (c *Config) ParentLauncher(name, layerDir string) (l Launcher, ok bool, err error) {
	i := slices.IndexFunc(c.launcherLayers, func(ll launcherLayer) bool { return ll.dir == layerDir })
	if i < 0 {
		return Launcher{}, false, fmt.Errorf("config directory %s is not part of this config", layerDir)
	}
	for i--; i >= 0; i-- {
		for _, l := range c.launcherLayers[i].launchers {
			if l.Name == name {
				return l, true, nil
			}
		}
	}
	return Launcher{}, false, nil
}

// LoadedPath returns the config file this config was loaded from, or "" when
// no project config file was found.
func (c *Config) LoadedPath() string {
	return c.configPath
}
```

In `internal/config/config.go`, add to the `Config` struct after `statuses`:

```go
	// launcherLayers holds each config file's own launchers, root-first (not serialized).
	launcherLayers []launcherLayer `yaml:"-"`
```

In `fromLayers`, after `cfg.recordOrigins(layers)`:

```go
	if cfg.launcherLayers, err = decodeLauncherLayers(layers); err != nil {
		return nil, err
	}
```

In `validate`, inside the existing `for _, launcher := range c.Launchers` loop, before the shebang check:

```go
		if strings.HasPrefix(launcher.Name, "builtin.") || strings.HasPrefix(launcher.Name, "parent.") {
			return fmt.Errorf("launcher '%s': the prefixes builtin. and parent. are reserved", launcher.Name)
		}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/config/`
Expected: PASS (all, including existing cascade tests).

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(config): keep per-layer launchers for parent lookups

- Add Config.Launcher, Config.ParentLauncher and Config.LoadedPath
- Reject launcher names starting with builtin. or parent.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Launch carries name, args, builtin flag and nested-call env

**Files:**
- Modify: `internal/launcher/launch.go`, `internal/launcher/exec.go`
- Test: `internal/launcher/launch_test.go`, `internal/launcher/exec_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/launcher/launch_test.go`:

```go
func TestLaunchEnvForNestedCalls(t *testing.T) {
	tests := []struct {
		name   string
		launch Launch
		want   []string
	}{
		{
			name:   "configured launch",
			launch: Launch{ProjectRoot: "/repo/pkg", ConfigDir: "/repo", ConfigPath: "/repo/pkg/.beans.yml", Depth: 2},
			want:   []string{"BEANS_LAUNCH_CONFIG=/repo/pkg/.beans.yml", "BEANS_LAUNCH_LAYER=/repo", "BEANS_LAUNCH_DEPTH=3"},
		},
		{
			name:   "builtin has no layer",
			launch: Launch{ProjectRoot: "/repo/pkg", Builtin: true},
			want:   []string{"BEANS_LAUNCH_LAYER=", "BEANS_LAUNCH_DEPTH=1", "BEANS_CONFIG_DIR=/repo/pkg"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.launch.Env()
			for _, want := range tt.want {
				if !slices.Contains(env, want) {
					t.Errorf("Env() missing %q", want)
				}
			}
			if !slices.ContainsFunc(env, func(e string) bool { return strings.HasPrefix(e, "BEANS_BIN=/") }) {
				t.Error("Env() missing absolute BEANS_BIN")
			}
		})
	}
}

func TestBuiltinRunsInCallersDirectory(t *testing.T) {
	if got := (Launch{ProjectRoot: "/repo", ConfigDir: "/repo", Builtin: true}).WorkDir(); got != "" {
		t.Errorf("WorkDir() = %q, want empty (current directory)", got)
	}
}
```

Append to `internal/launcher/exec_test.go`:

```go
func TestCreateShellCommandPassesArgs(t *testing.T) {
	tests := []struct {
		name    string
		builtin bool
		script  string
	}{
		{"sh", false, `printf '%s|' "$0" "$@"`},
		{"bash builtin", true, `[[ -n $BASH_VERSION ]] && printf '%s|' "$0" "$@"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Launch{Name: "demo", Script: tt.script, Args: []string{"a b", "--", "c"}, ProjectRoot: t.TempDir(), Builtin: tt.builtin}
			out, err := CreateShellCommand(l).Output()
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if got, want := string(out), "demo|a b|--|c|"; got != want {
				t.Errorf("output = %q, want %q", got, want)
			}
		})
	}
}
```

(Ensure `strings` is imported in `launch_test.go`; it already is.)

- [ ] **Step 2: Run the tests and watch them fail**

Run: `go test ./internal/launcher/ -run 'TestLaunchEnvForNestedCalls|TestBuiltinRunsInCallersDirectory|TestCreateShellCommandPassesArgs'`
Expected: compile error, unknown fields `ConfigPath`, `Depth`, `Builtin`, `Name`, `Args`.

- [ ] **Step 3: Implement**

Replace `internal/launcher/launch.go` with:

```go
package launcher

import (
	"fmt"
	"os"
)

// Launch describes running a script for a bean.
type Launch struct {
	Name        string   // name the launch was invoked as; the script's $0
	Script      string
	Args        []string // the script's positional parameters
	Builtin     bool     // an embedded builtin: runs with bash in the caller's directory
	ProjectRoot string   // directory of the project's config file (BEANS_ROOT)
	ConfigDir   string   // directory of the config file that declared Script; empty means ProjectRoot
	ConfigPath  string   // config file nested `beans launch` calls load; empty when there is none
	BeansDir    string   // beans directory (BEANS_DIR)
	BeanID      string
	BeanTitle   string
	Depth       int // how many launches this one is nested in
}

// WorkDir is where the script runs: the directory of the config file that
// declared it, so relative paths in inherited configs keep working. Builtins
// run where they were called from, which "" means for exec.Cmd.
func (l Launch) WorkDir() string {
	if l.Builtin {
		return ""
	}
	return l.configDir()
}

func (l Launch) configDir() string {
	if l.ConfigDir != "" {
		return l.ConfigDir
	}
	return l.ProjectRoot
}

// layer is the config directory `parent.` lookups of nested calls start
// from; builtins belong to no config layer.
func (l Launch) layer() string {
	if l.Builtin {
		return ""
	}
	return l.configDir()
}

// Env returns the process environment plus the BEANS_* variables, including
// the ones nested `beans launch` calls use to find the same config.
func (l Launch) Env() []string {
	bin, _ := os.Executable()
	return append(os.Environ(),
		fmt.Sprintf("BEANS_ROOT=%s", l.ProjectRoot),
		fmt.Sprintf("BEANS_CONFIG_DIR=%s", l.configDir()),
		fmt.Sprintf("BEANS_DIR=%s", l.BeansDir),
		fmt.Sprintf("BEANS_ID=%s", l.BeanID),
		fmt.Sprintf("BEANS_TASK=%s", l.BeanTitle),
		fmt.Sprintf("BEANS_BIN=%s", bin),
		fmt.Sprintf("BEANS_LAUNCH_CONFIG=%s", l.ConfigPath),
		fmt.Sprintf("BEANS_LAUNCH_LAYER=%s", l.layer()),
		fmt.Sprintf("BEANS_LAUNCH_DEPTH=%d", l.Depth+1),
	)
}
```

In `internal/launcher/exec.go`, replace `CreateShellCommand` (add `"cmp"` to the imports):

```go
// CreateShellCommand creates a command that runs a launch's script via sh -c
// (bash -c for builtins), with the launch's name as $0 and its args as $@.
func CreateShellCommand(l Launch) *exec.Cmd {
	shell := "sh"
	if l.Builtin {
		shell = "bash"
	}
	args := append([]string{"-c", l.Script, cmp.Or(l.Name, "beans-launch")}, l.Args...)
	cmd := exec.Command(shell, args...)
	cmd.Env = l.Env()
	cmd.Dir = l.WorkDir()
	return cmd
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/launcher/`
Expected: PASS. If `TestLaunchEnvConfigDirDefaultsToProjectRoot` or `TestLaunchWorkDir` fail, they test the unchanged non-builtin path and must still pass as is; fix the code, not the test.

- [ ] **Step 5: Commit**

```bash
git add internal/launcher/
git commit -m "feat(launcher): pass args to launch scripts and export nested-call env

- Launch gains Name, Args, Builtin, ConfigPath and Depth
- Scripts get \$0/\$@; builtins run with bash in the caller's directory
- Export BEANS_BIN, BEANS_LAUNCH_CONFIG, BEANS_LAUNCH_LAYER, BEANS_LAUNCH_DEPTH

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Builtin registry and host builtins (tmux-window, tmux-session, kitty-window)

**Files:**
- Create: `internal/launcher/builtins.go`, `internal/launcher/builtins_test.go`, `internal/launcher/builtin/tmux-window.sh`, `internal/launcher/builtin/tmux-session.sh`, `internal/launcher/builtin/kitty-window.sh`

- [ ] **Step 1: Write the failing tests**

`internal/launcher/builtins_test.go`:

```go
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
```

- [ ] **Step 2: Run the tests and watch them fail**

Run: `go test ./internal/launcher/ -run 'TestBuiltins|TestLookupBuiltin|TestTmux|TestKitty'`
Expected: compile error, `undefined: Builtins`.

- [ ] **Step 3: Implement the registry**

`internal/launcher/builtins.go`:

```go
package launcher

import (
	"embed"
	"path"
	"strings"
)

//go:embed builtin/*.sh
var builtinFS embed.FS

// Builtin is a launch script shipped inside the beans binary.
type Builtin struct {
	Name        string
	Description string // the script's second line, without "# "
	Script      string
}

// Builtins returns all builtin launches, sorted by name.
func Builtins() []Builtin {
	entries, err := builtinFS.ReadDir("builtin")
	if err != nil {
		panic(err) // embedded at build time
	}
	builtins := make([]Builtin, 0, len(entries))
	for _, e := range entries {
		raw, err := builtinFS.ReadFile(path.Join("builtin", e.Name()))
		if err != nil {
			panic(err)
		}
		builtins = append(builtins, newBuiltin(strings.TrimSuffix(e.Name(), ".sh"), string(raw)))
	}
	return builtins
}

// LookupBuiltin returns the builtin with the given name.
func LookupBuiltin(name string) (Builtin, bool) {
	raw, err := builtinFS.ReadFile(path.Join("builtin", name+".sh"))
	if err != nil {
		return Builtin{}, false
	}
	return newBuiltin(name, string(raw)), true
}

func newBuiltin(name, script string) Builtin {
	var description string
	if lines := strings.SplitN(script, "\n", 3); len(lines) > 1 {
		description = strings.TrimSpace(strings.TrimPrefix(lines[1], "#"))
	}
	return Builtin{Name: name, Description: description, Script: script}
}
```

(`ReadDir` returns entries sorted by name.)

- [ ] **Step 4: Write the host scripts**

`internal/launcher/builtin/tmux-window.sh`:

```bash
#!/usr/bin/env bash
# Run a command in a new tmux window: tmux-window [--cwd DIR] [--name NAME] -- CMD...
set -euo pipefail

cwd=$PWD
name=${BEANS_ID:-beans}
while [[ $# -gt 0 ]]; do
    case $1 in
        --cwd) cwd=$2; shift 2 ;;
        --name) name=$2; shift 2 ;;
        --) shift; break ;;
        *) break ;;
    esac
done
[[ $# -gt 0 ]] || { echo "usage: tmux-window [--cwd DIR] [--name NAME] -- CMD..." >&2; exit 2; }

# The window's shell comes from the tmux server, so the bean's env is passed on
env_args=()
for var in $(compgen -e | grep '^BEANS_'); do
    env_args+=(-e "$var=${!var}")
done

tmux new-window -c "$cwd" -n "$name" "${env_args[@]}" "$(printf '%q ' "$@")"
```

`internal/launcher/builtin/tmux-session.sh`:

```bash
#!/usr/bin/env bash
# Open or switch to a tmux session for the bean: tmux-session [--cwd DIR] [-- CMD...]
set -euo pipefail

: "${BEANS_ID:?BEANS_ID not set}"
cwd=$PWD
while [[ $# -gt 0 ]]; do
    case $1 in
        --cwd) cwd=$2; shift 2 ;;
        --) shift; break ;;
        *) break ;;
    esac
done

session=${BEANS_ID//[^a-zA-Z0-9_-]/}
if ! tmux has-session -t "=$session" 2>/dev/null; then
    # The session's shell comes from the tmux server, so the bean's env is passed on
    env_args=()
    for var in $(compgen -e | grep '^BEANS_'); do
        env_args+=(-e "$var=${!var}")
    done
    cmd=()
    [[ $# -gt 0 ]] && cmd=("$(printf '%q ' "$@")")
    tmux new-session -d -s "$session" -c "$cwd" "${env_args[@]}" "${cmd[@]}"
fi

if [[ -n ${TMUX:-} ]]; then
    tmux switch-client -t "=$session"
else
    tmux attach-session -t "=$session"
fi
```

`internal/launcher/builtin/kitty-window.sh`:

```bash
#!/usr/bin/env bash
# Run a command in a new kitty window: kitty-window [--cwd DIR] -- CMD...
set -euo pipefail

cwd=$PWD
while [[ $# -gt 0 ]]; do
    case $1 in
        --cwd) cwd=$2; shift 2 ;;
        --) shift; break ;;
        *) break ;;
    esac
done
[[ $# -gt 0 ]] || { echo "usage: kitty-window [--cwd DIR] -- CMD..." >&2; exit 2; }

# The window's shell comes from kitty, so the bean's env is passed on
env_args=()
for var in $(compgen -e | grep '^BEANS_'); do
    env_args+=(--env "$var=${!var}")
done

kitty @ launch --type=window --cwd="$cwd" "${env_args[@]}" "$@"
```

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/launcher/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/launcher/
git commit -m "feat(launcher): embed builtin launches, add tmux and kitty hosts

- Registry of go:embed'ed bash scripts with descriptions
- tmux-window, tmux-session and kitty-window builtins

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: worktree builtin

**Files:**
- Create: `internal/launcher/builtin/worktree.sh`
- Test: `internal/launcher/builtins_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/launcher/builtins_test.go`:

```go
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
```

- [ ] **Step 2: Run the test and watch it fail**

Run: `go test ./internal/launcher/ -run TestWorktree`
Expected: FAIL, `no builtin` / worktree exits non-zero (script doesn't exist; `LookupBuiltin` returns an empty script).

- [ ] **Step 3: Write the script**

`internal/launcher/builtin/worktree.sh`:

```bash
#!/usr/bin/env bash
# Ensure a git worktree on branch task/<bean-id> and print its path: worktree [--base BRANCH] [--dir DIR]
set -euo pipefail

: "${BEANS_ID:?BEANS_ID not set}"
repo=${BEANS_ROOT:-$PWD}
base=""
dir=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --base) base=$2; shift 2 ;;
        --dir) dir=$2; shift 2 ;;
        *) echo "usage: worktree [--base BRANCH] [--dir DIR]" >&2; exit 2 ;;
    esac
done
dir=${dir:-$repo/.worktrees/$BEANS_ID}
branch=task/$BEANS_ID

if [[ ! -d $dir ]]; then
    git -C "$repo" worktree prune
    if git -C "$repo" show-ref --verify --quiet "refs/heads/$branch"; then
        git -C "$repo" worktree add --quiet "$dir" "$branch" >&2
    else
        # Without --base, the branch starts at the repo's current HEAD
        git -C "$repo" worktree add --quiet -b "$branch" "$dir" ${base:+"$base"} >&2
    fi
fi

cd "$dir" && pwd -P
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/launcher/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/launcher/
git commit -m "feat(launcher): add worktree builtin

- Creates or reuses a worktree on task/<bean-id>, prints its path
- --base and --dir override the start point and location

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: prompt and agent builtins

**Files:**
- Create: `internal/launcher/builtin/prompt.sh`, `internal/launcher/builtin/agent.sh`
- Test: `internal/launcher/builtins_test.go`

Both call beans through `$BEANS_BIN`. `Env()` sets `BEANS_BIN` from `os.Executable()`; the tests append `BEANS_BIN=<stub>` to `cmd.Env` after it, and the later entry wins.

- [ ] **Step 1: Write the failing tests**

Append to `internal/launcher/builtins_test.go`:

```go
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
```

- [ ] **Step 2: Run the tests and watch them fail**

Run: `go test ./internal/launcher/ -run 'TestPrompt|TestAgent'`
Expected: FAIL (scripts missing).

- [ ] **Step 3: Write the scripts**

`internal/launcher/builtin/prompt.sh`:

```bash
#!/usr/bin/env bash
# Print the agent prompt for the bean: prompt [--tmux] [--worktree DIR]
set -euo pipefail

: "${BEANS_ID:?BEANS_ID not set}"
tmux_window=false
worktree=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --tmux) tmux_window=true; shift ;;
        --worktree) worktree=$2; shift 2 ;;
        *) echo "usage: prompt [--tmux] [--worktree DIR]" >&2; exit 2 ;;
    esac
done

# Query the same beans the launch was started for, wherever this runs
beans=("${BEANS_BIN:-beans}")
[[ -n ${BEANS_LAUNCH_CONFIG:-} ]] && beans+=(--config "$BEANS_LAUNCH_CONFIG")
[[ -n ${BEANS_DIR:-} ]] && beans+=(--beans-path "$BEANS_DIR")

"${beans[@]}" show --prompt "$BEANS_ID"

if $tmux_window; then
    cat <<'EOF'

## tmux window

You are running in a tmux window. Before starting, rename it to `<bean-id>: <label>`, where <label> is a short summary of this bean of at most 20 characters:
`tmux rename-window -t "$TMUX_PANE" '<bean-id>: <label>'`
EOF
fi

if [[ -n $worktree ]]; then
    cat <<EOF

## Worktree

You are working in the git worktree $worktree on branch task/$BEANS_ID. Commit your work on that branch. Leave merging the branch and removing the worktree to the user unless they ask you to do it.
EOF
fi

if "${beans[@]}" show --json "$BEANS_ID" | grep -Eq '"status": *"draft"'; then
    cat <<EOF

## Draft: plan first

This bean is still a draft. Do not implement anything yet. Plan the feature first: work out the requirements and open questions with the user, then write the agreed plan into the bean body. Once the plan is done, set the status to in-progress with \`beans update $BEANS_ID --status in-progress\`.
EOF
fi
```

`internal/launcher/builtin/agent.sh`:

```bash
#!/usr/bin/env bash
# Start a coding agent on the bean in the current directory: agent <claude|opencode|crush> [PROMPT-ARGS...]
set -euo pipefail

agent=${1:?usage: agent <claude|opencode|crush> [PROMPT-ARGS...]}
shift

# Unqualified, so a project can override the prompt launch
prompt=$("${BEANS_BIN:-beans}" launch prompt -- "$@")

case $agent in
    claude) exec claude "$prompt" ;;
    opencode) exec opencode --prompt "$prompt" ;;
    # Interactive crush takes no prompt, so this runs it once, non-interactively
    crush) exec crush run "$prompt" ;;
    *) echo "agent: unknown agent '$agent', use claude, opencode or crush" >&2; exit 2 ;;
esac
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/launcher/`
Expected: PASS. `TestAgent` expects the fake's output without a trailing newline because `$(...)` strips it.

- [ ] **Step 5: Commit**

```bash
git add internal/launcher/
git commit -m "feat(launcher): add prompt and agent builtins

- prompt: bean prompt plus tmux, worktree and draft notes
- agent: starts claude, opencode or crush with the prompt launch

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Name resolution

**Files:**
- Create: `internal/launcher/resolve.go`, `internal/launcher/resolve_test.go`

- [ ] **Step 1: Write the failing test**

`internal/launcher/resolve_test.go`:

```go
package launcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/config"
)

func TestResolve(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	pkg := filepath.Join(root, "pkg")
	for path, content := range map[string]string{
		filepath.Join(root, ".beans.yml"): "cascade: true\nlaunchers:\n  - name: prompt\n    exec: root-prompt\n",
		filepath.Join(pkg, ".beans.yml"):  "launchers:\n  - name: prompt\n    exec: pkg-prompt\n  - name: tmux-window\n    exec: my-window\n",
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
		name        string
		launch      string
		caller      string
		wantName    string
		wantScript  string // prefix
		wantDir     string
		wantBuiltin bool
		wantErr     string
	}{
		{name: "configured", launch: "prompt", wantName: "prompt", wantScript: "pkg-prompt", wantDir: pkg},
		{name: "config shadows builtin", launch: "tmux-window", wantName: "tmux-window", wantScript: "my-window", wantDir: pkg},
		{name: "falls back to builtin", launch: "agent", wantName: "builtin.agent", wantScript: "#!/usr/bin/env bash", wantBuiltin: true},
		{name: "builtin. skips config", launch: "builtin.tmux-window", wantName: "builtin.tmux-window", wantScript: "#!/usr/bin/env bash", wantBuiltin: true},
		{name: "parent. from leaf", launch: "parent.prompt", caller: pkg, wantName: "prompt", wantScript: "root-prompt", wantDir: root},
		{name: "parent. without caller starts at leaf", launch: "parent.prompt", wantName: "prompt", wantScript: "root-prompt", wantDir: root},
		{name: "parent. above root falls back to builtin", launch: "parent.prompt", caller: root, wantName: "builtin.prompt", wantScript: "#!/usr/bin/env bash", wantBuiltin: true},
		{name: "unknown", launch: "nope", wantErr: `unknown launch "nope"`},
		{name: "unknown builtin", launch: "builtin.nope", wantErr: `unknown builtin "nope"`},
		{name: "parent. of unknown", launch: "parent.nope", caller: pkg, wantErr: `unknown builtin "nope"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := Resolve(cfg, tt.launch, tt.caller)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Resolve() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if r.Name != tt.wantName || !strings.HasPrefix(r.Script, tt.wantScript) || r.Dir != tt.wantDir || r.Builtin != tt.wantBuiltin {
				t.Errorf("Resolve() = {%q %q… %q %v}, want {%q %q… %q %v}",
					r.Name, firstLineOf(r.Script), r.Dir, r.Builtin, tt.wantName, tt.wantScript, tt.wantDir, tt.wantBuiltin)
			}
		})
	}
}

func firstLineOf(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}
```

- [ ] **Step 2: Run the test and watch it fail**

Run: `go test ./internal/launcher/ -run TestResolve`
Expected: compile error, `undefined: Resolve`. If it fails with an import cycle, `internal/config` imports `internal/launcher`. It doesn't today (checked); keep it that way.

- [ ] **Step 3: Implement**

`internal/launcher/resolve.go`:

```go
package launcher

import (
	"fmt"
	"strings"

	"github.com/hmans/beans/internal/config"
)

const (
	builtinPrefix = "builtin."
	parentPrefix  = "parent."
)

// Resolved is a launch script found by name.
type Resolved struct {
	Name    string // configured name, or "builtin.<name>" for builtins
	Script  string
	Dir     string // directory of the config file that declared it; empty for builtins
	Builtin bool
}

// Resolve finds the script for a launch name:
//   - "builtin.X" is always the embedded builtin X
//   - "parent.X" is X as declared by the nearest config layer above
//     callerLayer (the leaf config when empty), else builtin X
//   - "X" is the configured launcher X, else builtin X
func Resolve(cfg *config.Config, name, callerLayer string) (Resolved, error) {
	if base, ok := strings.CutPrefix(name, builtinPrefix); ok {
		return resolveBuiltin(base)
	}
	if base, ok := strings.CutPrefix(name, parentPrefix); ok {
		if callerLayer == "" {
			callerLayer = cfg.ConfigDir()
		}
		l, found, err := cfg.ParentLauncher(base, callerLayer)
		if err != nil {
			return Resolved{}, err
		}
		if found {
			return fromConfig(l), nil
		}
		return resolveBuiltin(base)
	}
	if l, ok := cfg.Launcher(name); ok {
		return fromConfig(l), nil
	}
	if b, ok := LookupBuiltin(name); ok {
		return fromBuiltin(b), nil
	}
	return Resolved{}, fmt.Errorf("unknown launch %q, see beans launch --list", name)
}

func resolveBuiltin(name string) (Resolved, error) {
	b, ok := LookupBuiltin(name)
	if !ok {
		return Resolved{}, fmt.Errorf("unknown builtin %q, see beans launch --list", name)
	}
	return fromBuiltin(b), nil
}

func fromConfig(l config.Launcher) Resolved {
	return Resolved{Name: l.Name, Script: l.Exec, Dir: l.Dir}
}

func fromBuiltin(b Builtin) Resolved {
	return Resolved{Name: builtinPrefix + b.Name, Script: b.Script, Builtin: true}
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/launcher/ ./internal/config/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/launcher/
git commit -m "feat(launcher): resolve launch names to config, builtin and parent scripts

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: `beans launch <name> [id] [-- args...]`

**Files:**
- Modify: `cmd/launch.go`, `cmd/root.go`
- Test: `cmd/launch_test.go`

- [ ] **Step 1: Update the tests**

In `cmd/launch_test.go`:

1. Replace `TestLaunchBeanID` with:

```go
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
```

2. Delete `TestFindLauncher` (replaced by `launcher.TestResolve`).

3. Add:

```go
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
```

4. Replace `TestListLaunchers` with `TestListLaunches`. It passes builtins explicitly:

```go
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
}
```

Add `launcherexec "github.com/hmans/beans/internal/launcher"` to the test imports.

- [ ] **Step 2: Run the tests and watch them fail**

Run: `go test ./cmd/ -run 'TestLaunchBeanID|TestSplitLaunchArgs|TestLaunchDepth|TestListLaunches'`
Expected: compile errors (`splitLaunchArgs`, `launchDepth`, `listLaunches` undefined, `launchBeanID` arity).

- [ ] **Step 3: Implement `cmd/launch.go`**

Replace the var block, `launchCmd`, `launchBeanID`, `findLauncher`, `newCLILaunch`, `launcherInfo` and `listLaunchers`. Keep `firstLine` and `runLaunch` as they are, except for the error wording in `runLaunch` (`"running launch: %w"`, `"launch exited with code %d"`). Add `"strconv"` to the imports and drop `config` if it becomes unused (it stays used by `listLaunches`).

```go
var (
	launchList bool
	launchJSON bool
)

// maxLaunchDepth caps nested `beans launch` calls, e.g. a `prompt` override
// that calls `prompt` instead of `parent.prompt`.
const maxLaunchDepth = 16

var launchCmd = &cobra.Command{
	Use:   "launch <name> [id] [-- args...]",
	Short: "Run a launch for a bean",
	Long: `Runs a launch for a bean, attached to the terminal, and exits with its exit
code. A launch is a launcher configured in .beans.yml or a builtin:

  beans launch claude beans-abc1      configured launcher "claude"
  beans launch agent -- claude        builtin "agent" (when not configured)
  beans launch builtin.tmux-window    always the builtin
  beans launch parent.prompt          "prompt" from the config above the caller's

Arguments after -- are passed to the launch script. The bean ID (or a unique
prefix of it) is taken from the argument, else from $BEANS_ID (set inside
launches), else from the first line of stdin:

  echo beans-abc1 | beans launch claude

Use --list to show the configured launchers and the builtins.`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if launchList {
			return listLaunches(cmd.OutOrStdout(), cfg.Launchers, launcherexec.Builtins(), launchJSON)
		}

		positional, scriptArgs := splitLaunchArgs(args, cmd.ArgsLenAtDash())
		if len(positional) == 0 {
			return fmt.Errorf("no launch given, see beans launch --list")
		}
		if len(positional) > 2 {
			return fmt.Errorf("too many arguments, pass arguments for the launch after --")
		}

		depth, err := launchDepth(os.Getenv("BEANS_LAUNCH_DEPTH"))
		if err != nil {
			return err
		}
		resolved, err := launcherexec.Resolve(cfg, positional[0], os.Getenv("BEANS_LAUNCH_LAYER"))
		if err != nil {
			return err
		}

		id, fromStdin, err := launchBeanID(positional[1:], os.Getenv("BEANS_ID"), os.Stdin)
		if err != nil {
			return err
		}

		resolver := &graph.Resolver{Core: core}
		b, err := resolver.Query().Bean(context.Background(), id)
		if err != nil {
			return handleBeanQueryError(err, false, id)
		}
		if b == nil {
			return fmt.Errorf("bean not found: %s", id)
		}

		// From here on, failures are the launch's, not a usage problem.
		cmd.SilenceUsage = true
		launch := newCLILaunch(resolved, b)
		launch.Args = scriptArgs
		launch.Depth = depth
		return runLaunch(launch, fromStdin)
	},
}

// splitLaunchArgs splits args at the -- cobra saw (dash is
// cmd.ArgsLenAtDash(), -1 without one) into beans' own positional args and
// the launch script's args.
func splitLaunchArgs(args []string, dash int) (positional, scriptArgs []string) {
	if dash < 0 {
		return args, nil
	}
	return args[:dash], args[dash:]
}

// launchDepth parses BEANS_LAUNCH_DEPTH, the number of launches this call is
// nested in, and refuses to go deeper than maxLaunchDepth.
func launchDepth(env string) (int, error) {
	if env == "" {
		return 0, nil
	}
	depth, err := strconv.Atoi(env)
	if err != nil {
		return 0, fmt.Errorf("invalid BEANS_LAUNCH_DEPTH %q", env)
	}
	if depth >= maxLaunchDepth {
		return 0, fmt.Errorf("launches nested more than %d deep, does a launch call itself? (use parent.<name> to call the definition it overrides)", maxLaunchDepth)
	}
	return depth, nil
}

// launchBeanID returns the bean ID from args, else envID ($BEANS_ID), else the
// first non-empty line of stdin. fromStdin reports whether stdin was consumed.
func launchBeanID(args []string, envID string, stdin *os.File) (id string, fromStdin bool, err error) {
	if len(args) > 0 {
		return args[0], false, nil
	}
	if envID != "" {
		return envID, false, nil
	}
	if stat, err := stdin.Stat(); err == nil && stat.Mode()&os.ModeCharDevice != 0 {
		return "", false, fmt.Errorf("no bean ID given, pass it as argument, via $BEANS_ID or via stdin")
	}
	id, err = firstLine(stdin)
	if err != nil {
		return "", true, err
	}
	if id == "" {
		return "", true, fmt.Errorf("no bean ID given, stdin was empty")
	}
	return id, true, nil
}

// newCLILaunch describes running r for b in the current project.
func newCLILaunch(r launcherexec.Resolved, b *bean.Bean) launcherexec.Launch {
	projectRoot := cfg.ConfigDir()
	if projectRoot == "" {
		projectRoot = filepath.Dir(core.Root())
	}
	return launcherexec.Launch{
		Name:        r.Name,
		Script:      r.Script,
		Builtin:     r.Builtin,
		ProjectRoot: projectRoot,
		ConfigDir:   r.Dir,
		ConfigPath:  cfg.LoadedPath(),
		BeansDir:    core.Root(),
		BeanID:      b.ID,
		BeanTitle:   b.DisplayTitle(),
	}
}

// launcherInfo is a launch as listed by --list --json.
type launcherInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Exec        string `json:"exec,omitempty"`
	Multiple    bool   `json:"multiple"`
	Background  bool   `json:"background"`
	Builtin     bool   `json:"builtin,omitempty"`
}

// listLaunches writes the configured launchers and the builtins to w, as a
// table or as JSON.
func listLaunches(w io.Writer, launchers []config.Launcher, builtins []launcherexec.Builtin, asJSON bool) error {
	if asJSON {
		infos := make([]launcherInfo, 0, len(launchers)+len(builtins))
		for _, l := range launchers {
			infos = append(infos, launcherInfo{Name: l.Name, Description: l.Description, Exec: l.Exec, Multiple: l.Multiple, Background: l.Background})
		}
		for _, b := range builtins {
			infos = append(infos, launcherInfo{Name: "builtin." + b.Name, Description: b.Description, Builtin: true})
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(infos)
	}

	if len(launchers) == 0 {
		fmt.Fprintln(w, "No launchers configured.")
	}
	rows := make([][]string, 0, len(launchers))
	for _, l := range launchers {
		var flags []string
		if l.Multiple {
			flags = append(flags, "multiple")
		}
		if l.Background {
			flags = append(flags, "background")
		}
		details := []string{l.Description}
		if len(flags) > 0 {
			details = append(details, "["+strings.Join(flags, ", ")+"]")
		}
		rows = append(rows, []string{l.Name, strings.TrimSpace(strings.Join(details, "  "))})
	}
	writeRows(w, rows)

	if len(builtins) > 0 {
		fmt.Fprintln(w)
		rows = rows[:0]
		for _, b := range builtins {
			rows = append(rows, []string{"builtin." + b.Name, b.Description})
		}
		writeRows(w, rows)
	}
	return nil
}

// writeRows writes name/details rows with the names padded to one width.
func writeRows(w io.Writer, rows [][]string) {
	width := 0
	for _, r := range rows {
		width = max(width, len(r[0]))
	}
	for _, r := range rows {
		fmt.Fprintln(w, strings.TrimRight(fmt.Sprintf("%-*s  %s", width, r[0], r[1]), " "))
	}
}

func init() {
	launchCmd.Flags().BoolVar(&launchList, "list", false, "List configured launchers and builtins")
	launchCmd.Flags().BoolVar(&launchJSON, "json", false, "Output the --list as JSON")
	rootCmd.AddCommand(launchCmd)
}
```

- [ ] **Step 4: Pin nested calls to the launch's config in `cmd/root.go`**

In `PersistentPreRunE`, right after the `~` expansion lines, add:

```go
		// Nested `beans launch` calls use the config and beans of the launch
		// that started them, wherever the script cd'ed to
		if cmd == launchCmd && os.Getenv("BEANS_LAUNCH_DEPTH") != "" {
			if configPath == "" {
				configPath = os.Getenv("BEANS_LAUNCH_CONFIG")
			}
			if beansPath == "" {
				beansPath = os.Getenv("BEANS_DIR")
			}
		}
```

- [ ] **Step 5: Run the tests**

Run: `go test ./cmd/ ./internal/...`
Expected: PASS. `go vet ./...` must be clean.

- [ ] **Step 6: Commit**

```bash
git add cmd/
git commit -m "feat(launch)!: beans launch <name> [id] [-- args...] with builtins

- Launch name is positional; --launcher is removed
- Bean ID from argument, then \$BEANS_ID, then stdin
- Arguments after -- reach the script as \$@
- Resolve configured, builtin. and parent. launches; guard nesting depth
- Nested calls reuse the launch's config and beans directory
- --list shows builtins

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 8: TUI passes the config path

**Files:**
- Modify: `internal/tui/tui.go:1217-1227` (`newLaunch`)

- [ ] **Step 1: Implement**

In `newLaunch`, add `ConfigPath: a.config.LoadedPath(),` to the `launcherexec.Launch` literal, after `ConfigDir`.

- [ ] **Step 2: Run the tests**

Run: `go test ./internal/tui/`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/tui.go
git commit -m "feat(tui): let launches started from the TUI call nested launches

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 9: Documentation

**Files:**
- Modify: `README.md` (section `## Launchers`, ~line 270)

- [ ] **Step 1: Rewrite the CLI part of the Launchers section**

Read the whole `## Launchers` section first and replace every `--launcher` usage. Add these subsections after the existing config example:

````markdown
### Running launches from the CLI

```bash
beans launch claude beans-abc1          # configured launcher
echo beans-abc1 | beans launch claude   # bean ID from stdin
beans launch --list                     # configured launchers and builtins
```

Arguments after `--` are passed to the launch script as `"$@"`. Inside a launch, `$BEANS_ID` is set, so nested calls need no bean ID.

### Builtins

beans ships launches you can use directly or compose in your own:

| Builtin | Usage |
|---|---|
| `tmux-window` | `-- [--cwd DIR] [--name NAME] -- CMD...` run a command in a new tmux window |
| `tmux-session` | `-- [--cwd DIR] [-- CMD...]` open or switch to a tmux session for the bean |
| `kitty-window` | `-- [--cwd DIR] -- CMD...` run a command in a new kitty window |
| `worktree` | `-- [--base BRANCH] [--dir DIR]` ensure a git worktree on `task/<bean-id>` (default `.worktrees/<bean-id>`) and print its path |
| `prompt` | `-- [--tmux] [--worktree DIR]` print the agent prompt for the bean |
| `agent` | `-- <claude\|opencode\|crush> [PROMPT-ARGS...]` start a coding agent with the `prompt` launch |

Names resolve like this:

- `X`: the launcher `X` from `.beans.yml`, else the builtin `X`.
- `builtin.X`: always the builtin.
- `parent.X`: `X` as defined by the next config file up the cascade (see Cascading Configuration), else the builtin. Use it to wrap the definition you override.

Builtins call each other by plain name, so overriding one piece changes every launch that uses it. For example, an agent in a new tmux window, in its own worktree, with an extra note in the prompt:

```yaml
launchers:
  - name: worktree-tmux+claude
    multiple: true
    background: true
    exec: |
      #!/bin/bash
      set -euo pipefail
      dir=$("$BEANS_BIN" launch worktree)
      "$BEANS_BIN" launch tmux-window -- --cwd "$dir" -- \
        "$BEANS_BIN" launch agent -- claude --tmux --worktree "$dir"
  - name: prompt
    exec: |
      #!/bin/bash
      "$BEANS_BIN" launch parent.prompt -- "$@"
      echo "Run the tests before you commit."
```

Launches get these extra variables for nested calls: `BEANS_BIN` (the running beans binary), `BEANS_LAUNCH_CONFIG`, `BEANS_LAUNCH_LAYER` and `BEANS_LAUNCH_DEPTH`. Nesting deeper than 16 launches fails, which catches a launch that calls itself instead of `parent.<name>`.
````

Also add `BEANS_BIN` to any env-var table in that section.

- [ ] **Step 2: Verify the example config loads**

Write the example into a temporary project and check it parses and lists:

```bash
tmp=$(mktemp -d) && cd "$tmp" && git init -q && go run /home/nr/projects/oss/beans/beans-ppxk init >/dev/null
# paste the launchers: block above into .beans.yml, then:
go run /home/nr/projects/oss/beans/beans-ppxk launch --list
```

Expected: both launchers, then the `builtin.*` section.

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: document builtin launches and the new beans launch syntax

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 10: End-to-end check

No new code. Verify the pieces work together with the real binary (CLAUDE.md: use a separate test project for anything that writes).

- [ ] **Step 1: Set up a test project**

```bash
bin=$(mktemp -d)/beans && (cd /home/nr/projects/oss/beans/beans-ppxk && go build -o "$bin" .)
tmp=$(mktemp -d) && cd "$tmp" && git init -q -b main && git commit -q --allow-empty -m init
"$bin" init >/dev/null
id=$("$bin" create "e2e" -t task --json | jq -r .bean.id)
```

(`beans create --json` prints `{"success": true, "bean": {"id": ...}}`.)

- [ ] **Step 2: Builtins and args**

```bash
"$bin" launch prompt "$id" -- --tmux | head -20      # bean prompt + tmux note
"$bin" launch worktree "$id"                         # prints $tmp/.worktrees/$id
cd "$tmp/.worktrees/$id" && BEANS_LAUNCH_DEPTH=1 BEANS_LAUNCH_CONFIG="$tmp/.beans.yml" BEANS_DIR="$tmp/.beans" BEANS_ID="$id" \
  "$bin" launch prompt | head -3                     # nested call from another cwd finds the bean
```

- [ ] **Step 3: parent. and the recursion guard**

Add to `$tmp/.beans.yml`:

```yaml
launchers:
  - name: prompt
    exec: '"$BEANS_BIN" launch parent.prompt -- "$@"; echo EXTRA'
  - name: loop
    exec: '"$BEANS_BIN" launch loop'
```

```bash
cd "$tmp"
"$bin" launch prompt "$id" | tail -1     # EXTRA
"$bin" launch loop "$id"; echo "exit $?" # "nested more than 16", non-zero exit
```

- [ ] **Step 4: tmux (only if running inside tmux)**

```bash
"$bin" launch tmux-window "$id" -- --cwd "$tmp" -- sh -c 'echo $BEANS_ID; sleep 5'
```

Expected: a new window named after the bean prints the bean ID.

- [ ] **Step 5: Full test suite**

Run: `go test ./... && go vet ./...`
Expected: all PASS.

- [ ] **Step 6: Record results**

Append the outcome to the bean's work log; tick the last task items. Commit the bean in the hub.
