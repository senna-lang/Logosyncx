// Package gitutil provides helpers for automating git operations via go-git
// and os/exec.  It covers git add (staging), git rm (staging deletions),
// git commit, git push, and git status queries.
package gitutil

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v5"
)

// StatusCode is a single-character git status indicator (e.g. 'M', 'A', '?', ' ').
type StatusCode byte

const (
	StatusUnmodified StatusCode = ' '
	StatusUntracked  StatusCode = '?'
	StatusAdded      StatusCode = 'A'
	StatusModified   StatusCode = 'M'
	StatusDeleted    StatusCode = 'D'
	StatusRenamed    StatusCode = 'R'
)

// FileStatus holds the staging-area and worktree status of a single file,
// as reported by git status --porcelain.
type FileStatus struct {
	Path     string     // path relative to the repository root
	Staging  StatusCode // index (staging area) status
	Worktree StatusCode // working tree status
}

// StatusUnderDir returns the git status of every file whose path starts with
// prefix (relative to projectRoot, e.g. ".logosyncx/").
// It uses the system git binary so that sparse-checkout, worktree, and other
// local git configuration are honoured automatically.
func StatusUnderDir(projectRoot, prefix string) ([]FileStatus, error) {
	cmd := exec.Command("git", "status", "--porcelain", "--", prefix)
	cmd.Dir = projectRoot
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git status: %w\n%s", err, errOut.String())
	}

	var entries []FileStatus
	for _, line := range strings.Split(out.String(), "\n") {
		// Each porcelain line is at least "XY PATH" (4 chars minimum).
		if len(line) < 4 {
			continue
		}
		x := StatusCode(line[0])
		y := StatusCode(line[1])
		// line[2] is always a space in porcelain v1 format.
		path := strings.TrimSpace(line[3:])
		if path == "" {
			continue
		}
		entries = append(entries, FileStatus{
			Path:     path,
			Staging:  x,
			Worktree: y,
		})
	}
	return entries, nil
}

// Add stages the file at filePath in the git repository that contains
// projectRoot. filePath must be an absolute path; it is converted to a
// path relative to the repository worktree root before staging.
//
// If the file is not inside a git repository, or go-git cannot open it,
// the error is returned but logos save still succeeds — git add is
// best-effort and the user can stage the file manually.
func Add(projectRoot, filePath string) error {
	repo, err := gogit.PlainOpenWithOptions(projectRoot, &gogit.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return fmt.Errorf("open git repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	// Convert the absolute file path to a path relative to the worktree root.
	repoRoot := worktree.Filesystem.Root()
	rel, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		return fmt.Errorf("compute relative path: %w", err)
	}

	if _, err := worktree.Add(rel); err != nil {
		return fmt.Errorf("git add %s: %w", rel, err)
	}

	return nil
}

// Commit creates a git commit with the given message in the repository that
// contains projectRoot.  The commit is performed via the system git binary so
// that the user's configured author identity and credential helpers are
// honoured transparently.
//
// An error is returned when git is not available, the working directory is not
// a repository, or the commit itself fails (e.g. nothing staged).
func Commit(projectRoot, message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = projectRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit: %w\n%s", err, out.String())
	}
	return nil
}

// Push pushes the current branch to its upstream remote using the system git
// binary so that SSH keys, credential helpers, and proxy settings configured
// by the user are all respected.
//
// An error is returned when git is not available, the working directory is not
// a repository, or the push fails (e.g. no upstream configured, auth error).
func Push(projectRoot string) error {
	cmd := exec.Command("git", "push")
	cmd.Dir = projectRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push: %w\n%s", err, out.String())
	}
	return nil
}

// Remove stages the deletion of the file at filePath in the git repository
// that contains projectRoot.  filePath must be an absolute path.
//
// Like Add, this is best-effort: the caller should treat a non-nil error as a
// warning and still consider the underlying file operation successful.
func Remove(projectRoot, filePath string) error {
	repo, err := gogit.PlainOpenWithOptions(projectRoot, &gogit.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return fmt.Errorf("open git repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	repoRoot := worktree.Filesystem.Root()
	rel, err := filepath.Rel(repoRoot, filePath)
	if err != nil {
		return fmt.Errorf("compute relative path: %w", err)
	}

	if _, err := worktree.Remove(rel); err != nil {
		return fmt.Errorf("git rm %s: %w", rel, err)
	}

	return nil
}
