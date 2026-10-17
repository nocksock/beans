package config

import (
	"path/filepath"
	"testing"
)

const cascadingRoot = `cascade: true
beans:
  path: .beans
  prefix: root-
  id_length: 6
launchers:
  - name: claude
    exec: claude "$BEANS_ID"
  - name: tmux
    exec: ./scripts/tmux
`

func TestLoadCascade(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		".beans.yml": cascadingRoot,
		"pkg/.beans.yml": `beans:
  prefix: pkg-
launchers:
  - name: tmux
    exec: ./pkg-tmux
  - name: local
    exec: ./local
`,
		"own/.beans.yml": "beans:\n  path: issues\n",
	})
	pkg := filepath.Join(root, "pkg")

	cfg, err := LoadFromDirectory(pkg)
	if err != nil {
		t.Fatalf("LoadFromDirectory() error = %v", err)
	}

	t.Run("leaf identity", func(t *testing.T) {
		if cfg.ConfigDir() != pkg {
			t.Errorf("ConfigDir() = %q, want %q", cfg.ConfigDir(), pkg)
		}
		if cfg.ConfigPath() != filepath.Join(pkg, ConfigFileName) {
			t.Errorf("ConfigPath() = %q", cfg.ConfigPath())
		}
	})

	t.Run("scalars", func(t *testing.T) {
		if cfg.Beans.Prefix != "pkg-" {
			t.Errorf("Prefix = %q, want pkg-", cfg.Beans.Prefix)
		}
		if cfg.Beans.IDLength != 6 {
			t.Errorf("IDLength = %d, want 6 (inherited)", cfg.Beans.IDLength)
		}
		if cfg.Beans.DefaultStatus != "todo" {
			t.Errorf("DefaultStatus = %q, want default todo", cfg.Beans.DefaultStatus)
		}
	})

	t.Run("inherited beans path resolves against root", func(t *testing.T) {
		if got, want := cfg.ResolveBeansPath(), filepath.Join(root, ".beans"); got != want {
			t.Errorf("ResolveBeansPath() = %q, want %q", got, want)
		}
	})

	t.Run("launchers merged by name with origin dirs", func(t *testing.T) {
		want := []struct{ name, exec, dir string }{
			{"claude", `claude "$BEANS_ID"`, root},
			{"tmux", "./pkg-tmux", pkg},
			{"local", "./local", pkg},
		}
		if len(cfg.Launchers) != len(want) {
			t.Fatalf("Launchers = %+v", cfg.Launchers)
		}
		for i, w := range want {
			l := cfg.Launchers[i]
			if l.Name != w.name || l.Exec != w.exec || l.Dir != w.dir {
				t.Errorf("Launchers[%d] = %+v, want %+v", i, l, w)
			}
		}
	})

	t.Run("leaf beans path resolves against leaf", func(t *testing.T) {
		own, err := LoadFromDirectory(filepath.Join(root, "own"))
		if err != nil {
			t.Fatal(err)
		}
		if got, want := own.ResolveBeansPath(), filepath.Join(root, "own", "issues"); got != want {
			t.Errorf("ResolveBeansPath() = %q, want %q", got, want)
		}
		if own.Beans.Prefix != "root-" {
			t.Errorf("Prefix = %q, want root- (inherited)", own.Beans.Prefix)
		}
	})
}

func TestLoadCascadeDefaultPathResolvesAgainstLeaf(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		".beans.yml":     "cascade: true\nbeans:\n  prefix: root-\n",
		"pkg/.beans.yml": "",
	})

	cfg, err := Load(filepath.Join(root, "pkg", ConfigFileName))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.ResolveBeansPath(), filepath.Join(root, "pkg", DefaultBeansPath); got != want {
		t.Errorf("ResolveBeansPath() = %q, want %q", got, want)
	}
	if cfg.Beans.Prefix != "root-" {
		t.Errorf("Prefix = %q, want root-", cfg.Beans.Prefix)
	}
}

func TestLoadWithoutCascadeIgnoresParent(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		".beans.yml":     "beans:\n  prefix: root-\n  id_length: 9\n",
		"pkg/.beans.yml": "beans:\n  prefix: pkg-\n",
	})

	cfg, err := LoadFromDirectory(filepath.Join(root, "pkg"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Beans.IDLength != 4 {
		t.Errorf("IDLength = %d, want default 4", cfg.Beans.IDLength)
	}
}

func TestLoadValidatesInheritedLaunchers(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		".beans.yml":     "cascade: true\nlaunchers:\n  - name: bad\n    exec: |\n      echo a\n      echo b\n",
		"pkg/.beans.yml": "",
	})

	if _, err := LoadFromDirectory(filepath.Join(root, "pkg")); err == nil {
		t.Error("expected error for inherited multi-line launcher without shebang")
	}
}
