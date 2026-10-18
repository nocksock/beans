// Package githistory reads the git history of a bean's file by shelling out
// to git.
//
// A bean is matched by its ID rather than by its current filename, so the
// history survives renames of the slug part of the filename (which happen
// whenever the title changes). Missing git, a beans directory outside any
// repository and untracked beans all yield an empty history instead of an
// error.
package githistory

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Commit is a commit that touched a bean's file.
type Commit struct {
	Hash    string
	Author  string
	Date    time.Time
	Subject string
}

// ErrInvalidHash is returned for commit references that aren't a hex hash.
var ErrInvalidHash = errors.New("invalid commit hash")

// File is a bean's file in the repository that contains the beans directory.
type File struct {
	// Dir is the beans directory. Git runs inside it, so it resolves the
	// repository (or worktree) holding the beans, which may differ from the
	// one holding the code.
	Dir string
	// ID is the bean's ID.
	ID string
}

var hashPattern = regexp.MustCompile(`^[0-9a-fA-F]{4,64}$`)

// Field and record separators for the log format.
const (
	fieldSep  = "\x1f"
	recordSep = "\x1e"
)

// Log returns the commits that touched the bean's file, newest first.
// limit <= 0 returns all commits.
func (f File) Log(ctx context.Context, limit int) ([]Commit, error) {
	args := []string{"log", "--no-color", "--format=%H" + fieldSep + "%an" + fieldSep + "%aI" + fieldSep + "%s" + recordSep}
	if limit > 0 {
		args = append(args, fmt.Sprintf("--max-count=%d", limit))
	}
	out, ok, err := f.git(ctx, append(args, f.pathspecs()...)...)
	if !ok || err != nil {
		return nil, err
	}
	return parseLog(out)
}

// Diff returns the diff of the bean's file in the given commit, or "" if the
// commit didn't touch it. Renames show up as such.
func (f File) Diff(ctx context.Context, hash string) (string, error) {
	if !hashPattern.MatchString(hash) {
		return "", fmt.Errorf("%w: %q", ErrInvalidHash, hash)
	}
	args := append([]string{"show", "--no-color", "--format=", "--find-renames", hash}, f.pathspecs()...)
	out, _, err := f.git(ctx, args...)
	return out, err
}

// pathspecs match every filename the bean can have had, in any subdirectory.
func (f File) pathspecs() []string {
	return []string{
		"--",
		":(glob)**/" + f.ID + ".md",
		":(glob)**/" + f.ID + "--*.md",
		":(glob)**/" + f.ID + ".*.md",
	}
}

// git runs a git command in the beans directory. ok is false, without an
// error, when git isn't installed or the directory isn't in a repository
// (or the repository has no commits yet).
func (f File) git(ctx context.Context, args ...string) (out string, ok bool, err error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", f.Dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) || isNoHistory(stderr.String()) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), true, nil
}

// isNoHistory reports whether git's stderr says there's no history to read.
func isNoHistory(stderr string) bool {
	return strings.Contains(stderr, "not a git repository") ||
		strings.Contains(stderr, "does not have any commits yet") ||
		strings.Contains(stderr, "No such file or directory")
}

func parseLog(out string) ([]Commit, error) {
	var commits []Commit
	for _, record := range strings.Split(out, recordSep) {
		record = strings.TrimSpace(record)
		if record == "" {
			continue
		}
		fields := strings.SplitN(record, fieldSep, 4)
		if len(fields) != 4 {
			return nil, fmt.Errorf("unexpected git log record: %q", record)
		}
		date, err := time.Parse(time.RFC3339, fields[2])
		if err != nil {
			return nil, fmt.Errorf("parsing commit date: %w", err)
		}
		commits = append(commits, Commit{Hash: fields[0], Author: fields[1], Date: date, Subject: fields[3]})
	}
	return commits, nil
}
