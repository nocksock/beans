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

			cfg, err := initConfig("proj", initCmd.Flags())
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
