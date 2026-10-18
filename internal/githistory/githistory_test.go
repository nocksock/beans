package githistory

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// isolateGit keeps the user's git config (signing, hooks, ...) out of tests.
func isolateGit(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Ada")
	t.Setenv("GIT_AUTHOR_EMAIL", "ada@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Ada")
	t.Setenv("GIT_COMMITTER_EMAIL", "ada@example.com")
}

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commitAll(t *testing.T, repo, msg string) string {
	t.Helper()
	run(t, repo, "add", "-A")
	run(t, repo, "commit", "-q", "-m", msg)
	return run(t, repo, "rev-parse", "HEAD")
}

// newRepo creates a repository with a .beans directory and returns both.
func newRepo(t *testing.T) (repo, beansDir string) {
	t.Helper()
	repo = t.TempDir()
	run(t, repo, "init", "-q", "-b", "main")
	return repo, filepath.Join(repo, ".beans")
}

const body1 = "---\ntitle: First title\nstatus: todo\n---\n\nSome body text\nthat is long enough\nfor rename detection.\n"
const body2 = "---\ntitle: First title\nstatus: in-progress\n---\n\nSome body text\nthat is long enough\nfor rename detection.\n"
const body3 = "---\ntitle: Second title\nstatus: in-progress\n---\n\nSome body text\nthat is long enough\nfor rename detection.\n"

// historyRepo commits a bean, updates it, renames its slug, and commits an
// unrelated bean whose ID shares a prefix. It returns the beans directory and
// the hashes of the three commits touching the bean, oldest first.
func historyRepo(t *testing.T) (string, []string) {
	t.Helper()
	repo, beans := newRepo(t)
	write(t, filepath.Join(beans, "abc--first-title.md"), body1)
	h1 := commitAll(t, repo, "create abc")
	write(t, filepath.Join(beans, "abc--first-title.md"), body2)
	h2 := commitAll(t, repo, "start abc")
	write(t, filepath.Join(beans, "abcd--other.md"), body1)
	commitAll(t, repo, "create abcd")
	os.Remove(filepath.Join(beans, "abc--first-title.md"))
	write(t, filepath.Join(beans, "abc--second-title.md"), body3)
	h3 := commitAll(t, repo, "rename abc")
	return beans, []string{h1, h2, h3}
}

func subjects(commits []Commit) []string {
	var s []string
	for _, c := range commits {
		s = append(s, c.Subject)
	}
	return s
}

func TestLog(t *testing.T) {
	isolateGit(t)
	ctx := context.Background()

	beans, hashes := historyRepo(t)

	_, emptyBeans := newRepo(t)
	write(t, filepath.Join(emptyBeans, "abc--x.md"), body1)

	untrackedRepo, untrackedBeans := newRepo(t)
	write(t, filepath.Join(untrackedRepo, "README"), "hi")
	commitAll(t, untrackedRepo, "init")
	write(t, filepath.Join(untrackedBeans, "abc--x.md"), body1)

	noRepo := filepath.Join(t.TempDir(), ".beans")
	write(t, filepath.Join(noRepo, "abc--x.md"), body1)

	tests := []struct {
		name  string
		file  File
		limit int
		want  []string
	}{
		{"follows slug rename, newest first", File{Dir: beans, ID: "abc"}, 0, []string{"rename abc", "start abc", "create abc"}},
		{"limit", File{Dir: beans, ID: "abc"}, 2, []string{"rename abc", "start abc"}},
		{"ID prefix of another bean", File{Dir: beans, ID: "abcd"}, 0, []string{"create abcd"}},
		{"unknown bean", File{Dir: beans, ID: "zzz"}, 0, nil},
		{"untracked bean", File{Dir: untrackedBeans, ID: "abc"}, 0, nil},
		{"repo without commits", File{Dir: emptyBeans, ID: "abc"}, 0, nil},
		{"not a git repo", File{Dir: noRepo, ID: "abc"}, 0, nil},
		{"missing directory", File{Dir: filepath.Join(t.TempDir(), "nope"), ID: "abc"}, 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.file.Log(ctx, tt.limit)
			if err != nil {
				t.Fatalf("Log() error = %v", err)
			}
			if strings.Join(subjects(got), "|") != strings.Join(tt.want, "|") {
				t.Errorf("Log() subjects = %v, want %v", subjects(got), tt.want)
			}
		})
	}

	t.Run("commit fields", func(t *testing.T) {
		got, err := File{Dir: beans, ID: "abc"}.Log(ctx, 0)
		if err != nil {
			t.Fatal(err)
		}
		c := got[len(got)-1]
		if c.Hash != hashes[0] || c.Author != "Ada" || c.Date.IsZero() {
			t.Errorf("oldest commit = %+v, want hash %s by Ada with a date", c, hashes[0])
		}
	})
}

// TestLogHubLayout covers beans living in a different repository (or
// worktree) than the code: git must resolve the repo from the beans dir.
func TestLogHubLayout(t *testing.T) {
	isolateGit(t)
	ctx := context.Background()

	hub, beans := newRepo(t)
	write(t, filepath.Join(beans, "abc--x.md"), body1)
	commitAll(t, hub, "create abc")

	code := t.TempDir()
	run(t, code, "init", "-q", "-b", "main")
	write(t, filepath.Join(code, "main.go"), "package main\n")
	commitAll(t, code, "code")

	wt := filepath.Join(t.TempDir(), "wt")
	run(t, hub, "worktree", "add", "-q", "-b", "feature", wt)
	write(t, filepath.Join(wt, ".beans", "abc--x.md"), body2)
	commitAll(t, wt, "start abc in worktree")

	t.Chdir(code)

	tests := []struct {
		name string
		dir  string
		want []string
	}{
		{"hub repo from another cwd", beans, []string{"create abc"}},
		{"worktree of hub", filepath.Join(wt, ".beans"), []string{"start abc in worktree", "create abc"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := File{Dir: tt.dir, ID: "abc"}.Log(ctx, 0)
			if err != nil {
				t.Fatalf("Log() error = %v", err)
			}
			if strings.Join(subjects(got), "|") != strings.Join(tt.want, "|") {
				t.Errorf("Log() subjects = %v, want %v", subjects(got), tt.want)
			}
		})
	}
}

func TestDiff(t *testing.T) {
	isolateGit(t)
	ctx := context.Background()

	beans, hashes := historyRepo(t)
	unrelated := run(t, filepath.Dir(beans), "rev-parse", "HEAD~1") // "create abcd"
	f := File{Dir: beans, ID: "abc"}

	tests := []struct {
		name     string
		hash     string
		contains []string
		empty    bool
		err      error
	}{
		{"creation", hashes[0], []string{"new file mode", "+title: First title"}, false, nil},
		{"update", hashes[1], []string{"-status: todo", "+status: in-progress"}, false, nil},
		{"short hash", hashes[1][:8], []string{"+status: in-progress"}, false, nil},
		{"slug rename", hashes[2], []string{"rename from .beans/abc--first-title.md", "rename to .beans/abc--second-title.md", "+title: Second title"}, false, nil},
		{"commit not touching the bean", unrelated, nil, true, nil},
		{"option injection", "--output=/tmp/x", nil, true, ErrInvalidHash},
		{"not a hash", "HEAD", nil, true, ErrInvalidHash},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := f.Diff(ctx, tt.hash)
			if !errors.Is(err, tt.err) {
				t.Fatalf("Diff() error = %v, want %v", err, tt.err)
			}
			if tt.empty && got != "" {
				t.Errorf("Diff() = %q, want empty", got)
			}
			for _, s := range tt.contains {
				if !strings.Contains(got, s) {
					t.Errorf("Diff() missing %q in:\n%s", s, got)
				}
			}
		})
	}

	t.Run("not a git repo", func(t *testing.T) {
		got, err := File{Dir: t.TempDir(), ID: "abc"}.Diff(ctx, hashes[0])
		if err != nil || got != "" {
			t.Errorf("Diff() = %q, %v; want empty, nil", got, err)
		}
	})
}
