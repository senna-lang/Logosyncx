// Package gitutil provides helpers for automating git operations via go-git.
// It covers git add (staging new/modified files) and git rm (staging deletions).
package gitutil

import (
	"fmt"
	"path/filepath"

	gogit "github.com/go-git/go-git/v5"
)

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
