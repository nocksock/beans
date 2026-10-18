package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/config"
	"github.com/spf13/pflag"
)

// parseInitFlags resets the init flags to their defaults and parses args.
func parseInitFlags(t *testing.T, args ...string) {
	t.Helper()
	flags := initCmd.Flags()
	reset := func() {
		flags.VisitAll(func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
	}
	reset()
	t.Cleanup(reset)
	if err := flags.Parse(args); err != nil {
		t.Fatalf("parse flags %v: %v", args, err)
	}
}

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    config.BeansConfig
		check   func(t *testing.T, cfg *config.Config)
		wantErr string
	}{
		{
			name: "defaults",
			want: config.BeansConfig{Path: ".beans", Prefix: "proj-", IDLength: 4, DefaultStatus: "todo", DefaultType: "task"},
		},
		{
			name: "all beans settings",
			args: []string{"--path", "~/tasks", "--prefix", "x-", "--id-length", "6", "--default-status", "draft", "--default-type", "bug"},
			want: config.BeansConfig{Path: "~/tasks", Prefix: "x-", IDLength: 6, DefaultStatus: "draft", DefaultType: "bug"},
		},
		{
			name: "empty prefix",
			args: []string{"--prefix", ""},
			want: config.BeansConfig{Path: ".beans", Prefix: "", IDLength: 4, DefaultStatus: "todo", DefaultType: "task"},
		},
		{
			name: "prompt settings",
			args: []string{"--prompt-extra", "be nice"},
			want: config.BeansConfig{Path: ".beans", Prefix: "proj-", IDLength: 4, DefaultStatus: "todo", DefaultType: "task"},
			check: func(t *testing.T, cfg *config.Config) {
				if got := cfg.PromptExtra(); got != "be nice" {
					t.Errorf("prompt extra = %q, want %q", got, "be nice")
				}
			},
		},
		{name: "empty path", args: []string{"--path", ""}, wantErr: "--path"},
		{name: "zero id length", args: []string{"--id-length", "0"}, wantErr: "--id-length"},
		{name: "unknown status", args: []string{"--default-status", "nope"}, wantErr: "--default-status"},
		{name: "unknown type", args: []string{"--default-type", "nope"}, wantErr: "--default-type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseInitFlags(t, tt.args...)

			cfg, err := initConfig(config.DefaultWithPrefix("proj-"), initCmd.Flags())
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want one mentioning %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Beans != tt.want {
				t.Errorf("beans = %+v, want %+v", cfg.Beans, tt.want)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestInitCmdWithPath(t *testing.T) {
	project := t.TempDir()
	tasks := filepath.Join(t.TempDir(), "tasks")
	t.Chdir(project)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	oldBeansPath := beansPath
	beansPath = ""
	t.Cleanup(func() { beansPath = oldBeansPath })

	parseInitFlags(t, "--path", tasks, "--prefix", "t-")
	if err := initCmd.RunE(initCmd, nil); err != nil {
		t.Fatalf("init error = %v", err)
	}

	if info, err := os.Stat(tasks); err != nil || !info.IsDir() {
		t.Fatalf("beans directory not created at %s: %v", tasks, err)
	}
	if _, err := os.Stat(filepath.Join(project, ".beans")); !os.IsNotExist(err) {
		t.Errorf("default .beans directory should not be created, stat err = %v", err)
	}

	cfg, err := config.Load(filepath.Join(project, config.ConfigFileName))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := cfg.ResolveBeansPath(); got != tasks {
		t.Errorf("beans path = %q, want %q", got, tasks)
	}
	if cfg.Beans.Prefix != "t-" {
		t.Errorf("prefix = %q, want %q", cfg.Beans.Prefix, "t-")
	}
}

func TestInitCmdRejectsPathWithBeansPath(t *testing.T) {
	oldBeansPath := beansPath
	beansPath = filepath.Join(t.TempDir(), ".beans")
	t.Cleanup(func() { beansPath = oldBeansPath })

	parseInitFlags(t, "--path", "tasks")
	err := initCmd.RunE(initCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--beans-path") {
		t.Fatalf("error = %v, want one about combining --path and --beans-path", err)
	}
}

// cascadingParent writes a cascading root config and returns the root and
// an empty child directory below it.
func cascadingParent(t *testing.T) (root, child string) {
	t.Helper()
	root = t.TempDir()
	rootConfig := "cascade: true\nbeans:\n  path: .beans\n  prefix: root-\n  id_length: 6\nstatuses:\n  todo:\n    - name: ready\n      label: Ready\n"
	if err := os.WriteFile(filepath.Join(root, config.ConfigFileName), []byte(rootConfig), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".beans"), 0755); err != nil {
		t.Fatal(err)
	}
	child = filepath.Join(root, "api")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}
	return root, child
}

// runInit runs the init command in dir with the given flags.
func runInit(t *testing.T, dir string, args ...string) error {
	t.Helper()
	t.Chdir(dir)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	oldBeansPath := beansPath
	beansPath = ""
	t.Cleanup(func() { beansPath = oldBeansPath })
	parseInitFlags(t, args...)
	return initCmd.RunE(initCmd, nil)
}

func TestInitCmdRefusesUnderCascadingParent(t *testing.T) {
	root, child := cascadingParent(t)

	err := runInit(t, child, "--prefix", "api-")
	if err == nil || !strings.Contains(err.Error(), "--inherit") {
		t.Fatalf("error = %v, want one mentioning --inherit", err)
	}
	if !strings.Contains(err.Error(), filepath.Join(root, config.ConfigFileName)) {
		t.Errorf("error = %v, want it to name the parent config", err)
	}
	if _, err := os.Stat(filepath.Join(child, config.ConfigFileName)); !os.IsNotExist(err) {
		t.Errorf("config should not be written, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(child, ".beans")); !os.IsNotExist(err) {
		t.Errorf("beans directory should not be created, stat err = %v", err)
	}
}

func TestInitCmdInheritWritesOnlyGivenSettings(t *testing.T) {
	root, child := cascadingParent(t)

	if err := runInit(t, child, "--inherit", "--prefix", "api-", "--default-status", "ready"); err != nil {
		t.Fatalf("init error = %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(child, config.ConfigFileName))
	if err != nil {
		t.Fatal(err)
	}
	want := "beans:\n  default_status: ready\n  prefix: api-\n"
	if string(raw) != want {
		t.Errorf("config = %q, want %q", raw, want)
	}
	if _, err := os.Stat(filepath.Join(child, ".beans")); !os.IsNotExist(err) {
		t.Errorf("child should use the parent's beans directory, stat err = %v", err)
	}

	cfg, err := config.Load(filepath.Join(child, config.ConfigFileName))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Beans.Prefix != "api-" || cfg.Beans.IDLength != 6 || cfg.Beans.DefaultStatus != "ready" {
		t.Errorf("beans = %+v, want prefix api-, id_length 6 from the parent, default_status ready", cfg.Beans)
	}
	if got, want := cfg.ResolveBeansPath(), filepath.Join(root, ".beans"); got != want {
		t.Errorf("beans path = %q, want parent's %q", got, want)
	}
}

func TestInitCmdInheritWithPathCreatesOwnBeansDir(t *testing.T) {
	_, child := cascadingParent(t)

	if err := runInit(t, child, "--inherit", "--path", "tasks"); err != nil {
		t.Fatalf("init error = %v", err)
	}
	if info, err := os.Stat(filepath.Join(child, "tasks")); err != nil || !info.IsDir() {
		t.Fatalf("beans directory not created: %v", err)
	}
	cfg, err := config.Load(filepath.Join(child, config.ConfigFileName))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got, want := cfg.ResolveBeansPath(), filepath.Join(child, "tasks"); got != want {
		t.Errorf("beans path = %q, want %q", got, want)
	}
}

func TestInitCmdInheritValidatesAgainstParent(t *testing.T) {
	_, child := cascadingParent(t)

	err := runInit(t, child, "--inherit", "--default-status", "nope")
	if err == nil || !strings.Contains(err.Error(), "ready") {
		t.Fatalf("error = %v, want one listing the parent's statuses", err)
	}
}

func TestInitCmdInheritRequiresCascadingParent(t *testing.T) {
	err := runInit(t, t.TempDir(), "--inherit", "--prefix", "x-")
	if err == nil || !strings.Contains(err.Error(), "cascade") {
		t.Fatalf("error = %v, want one about needing a cascading parent", err)
	}
}

func TestInitCmdCascade(t *testing.T) {
	project := t.TempDir()
	if err := runInit(t, project, "--cascade", "--prefix", "repo-"); err != nil {
		t.Fatalf("init error = %v", err)
	}

	child := filepath.Join(project, "api")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}
	got, err := config.CascadingParent(child)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(project, config.ConfigFileName); got != want {
		t.Errorf("cascading parent of child = %q, want %q", got, want)
	}

	cfg, err := config.Load(filepath.Join(project, config.ConfigFileName))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Beans.Prefix != "repo-" || cfg.Beans.IDLength != 4 {
		t.Errorf("beans = %+v, want prefix repo- and default id_length 4", cfg.Beans)
	}
}

func TestInitCmdInheritCascade(t *testing.T) {
	_, child := cascadingParent(t)
	if err := runInit(t, child, "--inherit", "--cascade", "--prefix", "api-"); err != nil {
		t.Fatalf("init error = %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(child, config.ConfigFileName))
	if err != nil {
		t.Fatal(err)
	}
	want := "beans:\n  prefix: api-\ncascade: true\n"
	if string(raw) != want {
		t.Errorf("config = %q, want %q", raw, want)
	}
}

func TestInitCmdRefusesToOverwrite(t *testing.T) {
	project := t.TempDir()
	if err := runInit(t, project, "--prefix", "one-"); err != nil {
		t.Fatalf("first init error = %v", err)
	}

	err := runInit(t, project, "--prefix", "two-")
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("second init error = %v, want one mentioning --force", err)
	}
	cfg, err := config.Load(filepath.Join(project, config.ConfigFileName))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Beans.Prefix != "one-" {
		t.Errorf("prefix = %q, want the original one- to be kept", cfg.Beans.Prefix)
	}

	if err := runInit(t, project, "--force", "--prefix", "two-"); err != nil {
		t.Fatalf("forced init error = %v", err)
	}
	cfg, err = config.Load(filepath.Join(project, config.ConfigFileName))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Beans.Prefix != "two-" {
		t.Errorf("prefix = %q, want two- after --force", cfg.Beans.Prefix)
	}
}
