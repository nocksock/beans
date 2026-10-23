package tui

import (
	"os"
	"path/filepath"
	"strings"
)

// projectLabel names the project at dir (the directory of its config file) by
// its path within the enclosing git repository, compacted (see compactPath).
// It is empty when dir is the repository root or not inside a repository.
func projectLabel(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	root := gitRoot(abs)
	if root == "" {
		return ""
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == "." {
		return ""
	}
	return compactPath(filepath.ToSlash(rel))
}

// gitRoot returns the nearest directory at or above dir that contains a .git
// entry (a directory, or a file in worktrees), or "" if there is none.
func gitRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// compactPath shortens every segment of a slash-separated path but the last
// to its first character, keeping a leading dot: "apps/md.new" becomes
// "a/md.new", ".config/apps/web" becomes ".c/a/web".
func compactPath(path string) string {
	segments := strings.Split(path, "/")
	for i, s := range segments[:len(segments)-1] {
		segments[i] = compactSegment(s)
	}
	return strings.Join(segments, "/")
}

// compactSegment returns the first character of s, or the first two when s
// starts with a dot.
func compactSegment(s string) string {
	runes := []rune(s)
	n := 1
	if len(runes) > 1 && runes[0] == '.' {
		n = 2
	}
	return string(runes[:min(n, len(runes))])
}
