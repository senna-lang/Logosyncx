package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/senna-lang/logosyncx/pkg/index"
	"github.com/senna-lang/logosyncx/pkg/session"
)

// --- runSync: not initialized ------------------------------------------------

func TestSync_NotInitialized_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	if err := runSync(); err == nil {
		t.Fatal("expected error when project not initialized, got nil")
	}
}

// --- runSync: empty sessions -------------------------------------------------

func TestSync_EmptySessions_CreatesEmptyIndex(t *testing.T) {
	dir := setupInitedProject(t)

	out := captureOutput(t, func() {
		if err := runSync(); err != nil {
			t.Fatalf("runSync failed: %v", err)
		}
	})

	// Index file should now exist.
	indexPath := filepath.Join(dir, ".logosyncx", "index.jsonl")
	if _, err := os.Stat(indexPath); err != nil {
		t.Errorf("expected index.jsonl to exist after sync, got: %v", err)
	}

	// Output should mention 0 sessions.
	if !strings.Contains(out, "0 sessions indexed") {
		t.Errorf("expected '0 sessions indexed' in output, got: %q", out)
	}
}

// --- runSync: with sessions --------------------------------------------------

func TestSync_IndexesSessions(t *testing.T) {
	dir := setupInitedProject(t)

	date := time.Date(2025, 2, 20, 10, 0, 0, 0, time.UTC)
	sessions := []session.Session{
		{
			ID:      "id1",
			Date:    date,
			Topic:   "auth-flow",
			Tags:    []string{"auth", "jwt"},
			Agent:   "claude-code",
			Related: []string{},
			Body:    "## Summary\nJWT authentication decisions.\n",
		},
		{
			ID:      "id2",
			Date:    date.Add(-24 * time.Hour),
			Topic:   "db-schema",
			Tags:    []string{"postgres"},
			Agent:   "claude-code",
			Related: []string{},
			Body:    "## Summary\nPostgreSQL schema design.\n",
		},
	}
	for _, s := range sessions {
		if _, err := session.Write(dir, s); err != nil {
			t.Fatalf("session.Write: %v", err)
		}
	}

	out := captureOutput(t, func() {
		if err := runSync(); err != nil {
			t.Fatalf("runSync failed: %v", err)
		}
	})

	if !strings.Contains(out, "2 sessions indexed") {
		t.Errorf("expected '2 sessions indexed' in output, got: %q", out)
	}

	entries, err := index.ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries in index, got %d", len(entries))
	}
}

// --- runSync: output messages ------------------------------------------------

func TestSync_PrintsRebuildingMessage(t *testing.T) {
	setupInitedProject(t)

	out := captureOutput(t, func() {
		if err := runSync(); err != nil {
			t.Fatalf("runSync failed: %v", err)
		}
	})

	if !strings.Contains(out, "Rebuilding session index") {
		t.Errorf("expected 'Rebuilding session index' in output, got: %q", out)
	}
}

func TestSync_PrintsDoneMessage(t *testing.T) {
	setupInitedProject(t)

	out := captureOutput(t, func() {
		if err := runSync(); err != nil {
			t.Fatalf("runSync failed: %v", err)
		}
	})

	if !strings.Contains(out, "Done.") {
		t.Errorf("expected 'Done.' in output, got: %q", out)
	}
}

// --- runSync: overwrites existing stale index --------------------------------

func TestSync_OverwritesStaleIndex(t *testing.T) {
	dir := setupInitedProject(t)

	// Write a stale entry directly into the index.
	stale := index.Entry{
		ID:      "stale-id",
		Topic:   "stale-topic",
		Tags:    []string{},
		Related: []string{},
		Date:    time.Now(),
	}
	if err := index.Append(dir, stale); err != nil {
		t.Fatalf("Append stale entry: %v", err)
	}

	// Write one real session.
	s := session.Session{
		ID:      "real1",
		Date:    time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		Topic:   "real-topic",
		Tags:    []string{},
		Agent:   "claude-code",
		Related: []string{},
		Body:    "## Summary\nReal session.\n",
	}
	if _, err := session.Write(dir, s); err != nil {
		t.Fatalf("session.Write: %v", err)
	}

	if err := runSync(); err != nil {
		t.Fatalf("runSync failed: %v", err)
	}

	entries, err := index.ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after sync (stale should be gone), got %d", len(entries))
	}
	if entries[0].Topic == "stale-topic" {
		t.Error("stale entry should have been removed by sync")
	}
	if entries[0].Topic != "real-topic" {
		t.Errorf("expected 'real-topic', got %q", entries[0].Topic)
	}
}

// --- runSync: index entry fields ---------------------------------------------

func TestSync_IndexEntry_HasCorrectFields(t *testing.T) {
	dir := setupInitedProject(t)

	date := time.Date(2025, 4, 10, 9, 0, 0, 0, time.UTC)
	s := session.Session{
		ID:      "abc123",
		Date:    date,
		Topic:   "field-check",
		Tags:    []string{"go", "testing"},
		Agent:   "claude-code",
		Related: []string{"2025-04-01_prev.md"},
		Body:    "## Summary\nChecking all index fields are populated correctly.\n",
	}
	if _, err := session.Write(dir, s); err != nil {
		t.Fatalf("session.Write: %v", err)
	}

	if err := runSync(); err != nil {
		t.Fatalf("runSync failed: %v", err)
	}

	entries, err := index.ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.ID != "abc123" {
		t.Errorf("ID = %q, want 'abc123'", e.ID)
	}
	if e.Topic != "field-check" {
		t.Errorf("Topic = %q, want 'field-check'", e.Topic)
	}
	if len(e.Tags) != 2 || e.Tags[0] != "go" || e.Tags[1] != "testing" {
		t.Errorf("Tags = %v, want [go testing]", e.Tags)
	}
	if e.Agent != "claude-code" {
		t.Errorf("Agent = %q, want 'claude-code'", e.Agent)
	}
	if e.Filename == "" {
		t.Error("Filename should not be empty")
	}
	if e.Excerpt == "" {
		t.Error("Excerpt should not be empty")
	}
	if !e.Date.Equal(date) {
		t.Errorf("Date = %v, want %v", e.Date, date)
	}
}

// --- runSync: idempotent -----------------------------------------------------

func TestSync_Idempotent(t *testing.T) {
	dir := setupInitedProject(t)

	s := session.Session{
		ID:      "idem1",
		Date:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Topic:   "idempotent-test",
		Tags:    []string{},
		Agent:   "claude-code",
		Related: []string{},
		Body:    "## Summary\nIdempotent test.\n",
	}
	if _, err := session.Write(dir, s); err != nil {
		t.Fatalf("session.Write: %v", err)
	}

	// Run sync twice.
	for i := 0; i < 2; i++ {
		if err := runSync(); err != nil {
			t.Fatalf("runSync (run %d) failed: %v", i+1, err)
		}
	}

	entries, err := index.ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry after two syncs (not duplicated), got %d", len(entries))
	}
}
