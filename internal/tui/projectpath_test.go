package tui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompactPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"md.new", "md.new"},
		{"apps/md.new", "a/md.new"},
		{"packages/keymap", "p/keymap"},
		{"packages/ui/button", "p/u/button"},
		{".config/apps/web", ".c/a/web"},
		{"ümlaut/x", "ü/x"},
		{"./x", "./x"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := compactPath(tt.path); got != tt.want {
				t.Errorf("compactPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestProjectLabel(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A worktree marks its root with a .git file
	worktree := filepath.Join(repo, "worktrees", "feature")
	if err := os.MkdirAll(worktree, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()

	tests := []struct {
		name string
		dir  string
		want string
	}{
		{"repo root", repo, ""},
		{"nested project", filepath.Join(repo, "apps", "md.new"), "a/md.new"},
		{"inside worktree", filepath.Join(worktree, "packages", "keymap"), "p/keymap"},
		{"worktree root", worktree, ""},
		{"no repository", outside, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := projectLabel(tt.dir); got != tt.want {
				t.Errorf("projectLabel(%q) = %q, want %q", tt.dir, got, tt.want)
			}
		})
	}
}
