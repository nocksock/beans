package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain isolates tests from the developer's own user config.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "beans-config-test")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_CONFIG_HOME", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// setUserConfig points the user config at a temp dir and writes content to it
// (no file if content is empty). Returns the user config path.
func setUserConfig(t *testing.T, content string) string {
	t.Helper()
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	path := filepath.Join(xdg, "beans", UserConfigFileName)
	if content != "" {
		writeTree(t, xdg, map[string]string{"beans/" + UserConfigFileName: content})
	}
	return path
}

func TestUserConfigPath(t *testing.T) {
	t.Run("XDG_CONFIG_HOME", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/xdg")
		if got, want := UserConfigPath(), filepath.Join("/xdg", "beans", "config.yml"); got != want {
			t.Errorf("UserConfigPath() = %q, want %q", got, want)
		}
	})
	t.Run("falls back to ~/.config", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "/home/someone")
		if got, want := UserConfigPath(), filepath.Join("/home/someone", ".config", "beans", "config.yml"); got != want {
			t.Errorf("UserConfigPath() = %q, want %q", got, want)
		}
	})
}

func TestLoadUserConfig(t *testing.T) {
	tests := []struct {
		name       string
		user       string
		project    string // .beans.yml content; empty means no project config
		wantErr    bool
		wantPrefix string
		wantIDLen  int
		wantWarn   bool
	}{
		{
			name:       "no user config",
			project:    "beans:\n  prefix: proj-\n",
			wantPrefix: "proj-",
			wantIDLen:  4,
		},
		{
			name:       "user config is the base layer",
			user:       "beans:\n  prefix: user-\n  id_length: 7\ntui:\n  disable_launcher_warning: true\n",
			project:    "beans:\n  prefix: proj-\n",
			wantPrefix: "proj-",
			wantIDLen:  7,
			wantWarn:   true,
		},
		{
			name:       "user config applies without project config",
			user:       "beans:\n  prefix: user-\n",
			wantPrefix: "user-",
			wantIDLen:  4,
		},
		{
			name:    "beans.path is rejected",
			user:    "beans:\n  path: /somewhere\n",
			project: "beans:\n  prefix: proj-\n",
			wantErr: true,
		},
		{
			name:       "cascade is ignored",
			user:       "cascade: false\nbeans:\n  id_length: 9\n",
			project:    "beans:\n  prefix: proj-\n",
			wantPrefix: "proj-",
			wantIDLen:  9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setUserConfig(t, tt.user)
			dir := t.TempDir()
			if tt.project != "" {
				writeTree(t, dir, map[string]string{ConfigFileName: tt.project})
			}

			cfg, err := LoadFromDirectory(dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("LoadFromDirectory() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadFromDirectory() error = %v", err)
			}
			if cfg.Beans.Prefix != tt.wantPrefix {
				t.Errorf("Prefix = %q, want %q", cfg.Beans.Prefix, tt.wantPrefix)
			}
			if cfg.Beans.IDLength != tt.wantIDLen {
				t.Errorf("IDLength = %d, want %d", cfg.Beans.IDLength, tt.wantIDLen)
			}
			if cfg.TUI.DisableLauncherWarning != tt.wantWarn {
				t.Errorf("DisableLauncherWarning = %v, want %v", cfg.TUI.DisableLauncherWarning, tt.wantWarn)
			}
			if cfg.ConfigDir() != dir {
				t.Errorf("ConfigDir() = %q, want %q", cfg.ConfigDir(), dir)
			}
			if got := cfg.ResolveBeansPath(); got != filepath.Join(dir, DefaultBeansPath) {
				t.Errorf("ResolveBeansPath() = %q, want it inside the project", got)
			}
		})
	}
}

func TestLoadKeybinding(t *testing.T) {
	setUserConfig(t, `keybinding:
  global:
    quit: [q, ctrl+q]
    help: "?"
  list:
    changeStatus: S
    archive: <disabled>
`)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{ConfigFileName: `keybinding:
  list:
    changeStatus: [x, "g s"]
    create: []
`})

	cfg, err := LoadFromDirectory(dir)
	if err != nil {
		t.Fatalf("LoadFromDirectory() error = %v", err)
	}

	tests := []struct {
		ctx, action string
		want        KeyList
	}{
		{"global", "quit", KeyList{"q", "ctrl+q"}},
		{"global", "help", KeyList{"?"}},
		{"list", "changeStatus", KeyList{"x", "g s"}}, // project overrides user
		{"list", "archive", KeyList{}},                // <disabled>
		{"list", "create", KeyList{}},                 // empty list
	}
	for _, tt := range tests {
		got, ok := cfg.Keybinding[tt.ctx][tt.action]
		if !ok {
			t.Errorf("%s.%s missing", tt.ctx, tt.action)
			continue
		}
		if got == nil || len(got) != len(tt.want) {
			t.Errorf("%s.%s = %#v, want %#v", tt.ctx, tt.action, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("%s.%s = %#v, want %#v", tt.ctx, tt.action, got, tt.want)
			}
		}
	}
}

func TestLoadKeybindingInvalid(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"map value", "keybinding:\n  list:\n    create: {a: b}\n"},
		{"empty key", "keybinding:\n  list:\n    create: \"\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeTree(t, dir, map[string]string{ConfigFileName: tt.content})
			if _, err := LoadFromDirectory(dir); err == nil {
				t.Error("LoadFromDirectory() error = nil, want error")
			}
		})
	}
}

func TestReloadPicksUpUserConfig(t *testing.T) {
	userPath := setUserConfig(t, "")
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{ConfigFileName: "beans:\n  prefix: proj-\n"})

	cfg, err := LoadFromDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(userPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userPath, []byte("beans:\n  id_length: 8\n"), 0644); err != nil {
		t.Fatal(err)
	}

	reloaded, err := cfg.Reload()
	if err != nil {
		t.Fatalf("Reload() error = %v", err)
	}
	if reloaded.Beans.IDLength != 8 {
		t.Errorf("IDLength = %d, want 8 from the user config", reloaded.Beans.IDLength)
	}
}
