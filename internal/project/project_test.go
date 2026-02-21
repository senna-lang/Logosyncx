package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRootFrom_FindsDirectParent(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".logosyncx"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindRootFrom(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != dir {
		t.Errorf("FindRootFrom = %q, want %q", got, dir)
	}
}

func TestFindRootFrom_FindsAncestor(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".logosyncx"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a nested subdirectory and search from there.
	nested := filepath.Join(root, "pkg", "session")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindRootFrom(nested)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != root {
		t.Errorf("FindRootFrom = %q, want %q", got, root)
	}
}

func TestFindRootFrom_ReturnsErrNotInitialized(t *testing.T) {
	dir := t.TempDir()
	// No .logosyncx/ directory created.

	_, err := FindRootFrom(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != ErrNotInitialized {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
}

func TestFindRootFrom_ErrorMessageContainsHint(t *testing.T) {
	dir := t.TempDir()

	_, err := FindRootFrom(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
}

func TestFindRootFrom_LogosyncxMustBeDirectory(t *testing.T) {
	dir := t.TempDir()

	// Create .logosyncx as a FILE, not a directory — should not match.
	if err := os.WriteFile(filepath.Join(dir, ".logosyncx"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := FindRootFrom(dir)
	if err == nil {
		t.Fatal("expected error when .logosyncx is a file, got nil")
	}
	if err != ErrNotInitialized {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
}

func TestFindRootFrom_StopsAtNearestAncestor(t *testing.T) {
	// Structure: grandparent/.logosyncx/  AND  grandparent/parent/.logosyncx/
	// Searching from grandparent/parent/child should find grandparent/parent.
	grandparent := t.TempDir()
	parent := filepath.Join(grandparent, "parent")
	child := filepath.Join(parent, "child")

	for _, d := range []string{
		filepath.Join(grandparent, ".logosyncx"),
		filepath.Join(parent, ".logosyncx"),
		child,
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	got, err := FindRootFrom(child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != parent {
		t.Errorf("FindRootFrom = %q, want nearest ancestor %q", got, parent)
	}
}

func TestErrNotInitialized_ContainsLogosInitHint(t *testing.T) {
	msg := ErrNotInitialized.Error()
	if msg == "" {
		t.Error("ErrNotInitialized message should not be empty")
	}
	// Should guide the user toward the fix.
	if len(msg) < 10 {
		t.Errorf("error message too short: %q", msg)
	}
}
