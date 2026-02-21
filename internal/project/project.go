// Package project provides utilities for locating the Logosyncx project root.
package project

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrNotInitialized is returned when no .logosyncx/ directory can be found
// by walking up the directory tree from the current working directory.
var ErrNotInitialized = errors.New("not a logosyncx project (run `logos init` first)")

// FindRoot walks up the directory tree from the current working directory
// until it finds a directory containing .logosyncx/, then returns that
// directory as the project root. Returns ErrNotInitialized if not found.
func FindRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return findRootFrom(cwd)
}

// FindRootFrom is like FindRoot but starts from the given directory.
// Exported for use in tests.
func FindRootFrom(dir string) (string, error) {
	return findRootFrom(dir)
}

func findRootFrom(dir string) (string, error) {
	current := filepath.Clean(dir)
	for {
		candidate := filepath.Join(current, ".logosyncx")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached the filesystem root without finding .logosyncx/.
			return "", ErrNotInitialized
		}
		current = parent
	}
}
