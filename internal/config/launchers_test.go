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
