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
