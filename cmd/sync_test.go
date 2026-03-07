package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/senna-lang/logosyncx/pkg/index"
	"github.com/senna-lang/logosyncx/pkg/plan"
)

// writeSyncPlan is a helper that writes a plan file to projectRoot/plans/
// including the Body field (unlike plan.Write which is scaffold-only).
func writeSyncPlan(t *testing.T, projectRoot string, p plan.Plan) {
	t.Helper()
	plansDir := filepath.Join(projectRoot, ".logosyncx", "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		t.Fatalf("mkdir plans: %v", err)
	}
	data, err := plan.Marshal(p)
	if err != nil {
		t.Fatalf("plan.Marshal: %v", err)
	}
	if p.Body != "" {
		data = append(data, []byte(p.Body)...)
	}
	path := filepath.Join(plansDir, plan.FileName(p))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile plan: %v", err)
	}
}

// makeSyncPlan returns a minimal plan for sync tests.
func makeSyncPlan(id, topic string, date time.Time) plan.Plan {
	return plan.Plan{
		ID:       id,
		Date:     &date,
		Topic:    topic,
		Tags:     []string{},
		Agent:    "claude-code",
		Related:  []string{},
		TasksDir: ".logosyncx/tasks/" + topic,
		Body:     "## Background\n" + topic + " plan.\n",
	}
}

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

// --- runSync: empty plans ----------------------------------------------------

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

	if !strings.Contains(out, "0 plans indexed") {
		t.Errorf("expected '0 plans indexed' in output, got: %q", out)
	}
}

// --- runSync: with plans -----------------------------------------------------

func TestSync_IndexesSessions(t *testing.T) {
	dir := setupInitedProject(t)

	date := time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)
	dateMinus1 := date.Add(-24 * time.Hour)
	writeSyncPlan(t, dir, makeSyncPlan("id1", "auth-flow", date))
	writeSyncPlan(t, dir, makeSyncPlan("id2", "db-schema", dateMinus1))

	out := captureOutput(t, func() {
		if err := runSync(); err != nil {
			t.Fatalf("runSync failed: %v", err)
		}
	})

	if !strings.Contains(out, "2 plans indexed") {
		t.Errorf("expected '2 plans indexed' in output, got: %q", out)
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

	if !strings.Contains(out, "Rebuilding plan index") {
		t.Errorf("expected 'Rebuilding plan index' in output, got: %q", out)
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

	stale := index.Entry{
		ID:        "stale-id",
		Topic:     "stale-topic",
		Tags:      []string{},
		Related:   []string{},
		DependsOn: []string{},
		Date:      time.Now(),
	}
	if err := index.Append(dir, stale); err != nil {
		t.Fatalf("Append stale entry: %v", err)
	}

	realDate := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	writeSyncPlan(t, dir, makeSyncPlan("real1", "real-topic", realDate))

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

	date := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	p := plan.Plan{
		ID:       "abc123",
		Date:     &date,
		Topic:    "field-check",
		Tags:     []string{"go", "testing"},
		Agent:    "claude-code",
		Related:  []string{"20260401-prev.md"},
		TasksDir: ".logosyncx/tasks/20260410-field-check",
		Body:     "## Background\nChecking all index fields are populated correctly.\n",
	}
	writeSyncPlan(t, dir, p)

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

	idemDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	writeSyncPlan(t, dir, makeSyncPlan("idem1", "idempotent-test", idemDate))

	for range 2 {
		if err := runSync(); err != nil {
			t.Fatalf("runSync failed: %v", err)
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
