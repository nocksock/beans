package graph

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestBeanHistory(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Ada")
	t.Setenv("GIT_AUTHOR_EMAIL", "ada@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Ada")
	t.Setenv("GIT_COMMITTER_EMAIL", "ada@example.com")
	ctx := context.Background()

	resolver, core := setupTestResolver(t)
	b := createTestBean(t, core, "hist-1", "History Bean", "todo")
	br := resolver.Bean()

	t.Run("not a git repo", func(t *testing.T) {
		got, err := br.History(ctx, b, nil)
		if err != nil || len(got) != 0 {
			t.Errorf("History() = %v, %v; want empty, nil", got, err)
		}
	})

	repo := filepath.Dir(core.Root())
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "create hist-1")
	b.Status = "in-progress"
	if err := core.Update(b); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "commit", "-q", "-am", "start hist-1")

	limit := 1
	tests := []struct {
		name  string
		limit *int
		want  []string
	}{
		{"all commits", nil, []string{"start hist-1", "create hist-1"}},
		{"limit", &limit, []string{"start hist-1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := br.History(ctx, b, tt.limit)
			if err != nil {
				t.Fatalf("History() error = %v", err)
			}
			var subjects []string
			for _, c := range got {
				subjects = append(subjects, c.Subject)
			}
			if strings.Join(subjects, "|") != strings.Join(tt.want, "|") {
				t.Errorf("History() subjects = %v, want %v", subjects, tt.want)
			}
		})
	}

	t.Run("diff", func(t *testing.T) {
		history, err := br.History(ctx, b, &limit)
		if err != nil {
			t.Fatal(err)
		}
		fromCommit, err := resolver.BeanCommit().Diff(ctx, history[0])
		if err != nil {
			t.Fatal(err)
		}
		fromBean, err := br.Diff(ctx, b, history[0].Hash)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(fromCommit, "+status: in-progress") {
			t.Errorf("BeanCommit.diff missing status change:\n%s", fromCommit)
		}
		if fromBean != fromCommit {
			t.Errorf("Bean.diff = %q, want %q", fromBean, fromCommit)
		}
	})
}
