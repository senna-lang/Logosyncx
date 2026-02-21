// Package task provides tests for the JSONL task index.
package task

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/senna-lang/logosyncx/pkg/config"
)

// --- helpers -----------------------------------------------------------------

// setupTaskIndex creates a temp directory with a .logosyncx/tasks/ structure
// and returns the project root and a Store. It does NOT create task-index.jsonl.
func setupTaskIndex(t *testing.T) (string, *Store) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".logosyncx", "tasks"), 0o755); err != nil {
		t.Fatalf("mkdir tasks: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".logosyncx", "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	cfg := config.Default("test-project")
	return dir, NewStore(dir, &cfg)
}

// makeTaskEntry returns a minimal TaskJSON for testing.
func makeTaskEntry(id, title string, status Status, date time.Time) TaskJSON {
	return TaskJSON{
		ID:       id,
		Filename: date.Format("2006-01-02") + "_" + slugify(title) + ".md",
		Date:     date,
		Title:    title,
		Status:   status,
		Priority: PriorityMedium,
		Tags:     []string{},
	}
}

// writeTaskToStore writes a task file via the store and returns the TaskJSON
// that was appended to the index.
func writeTaskToStore(t *testing.T, s *Store, title, statusStr string, date time.Time) TaskJSON {
	t.Helper()
	tk := &Task{
		Title:    title,
		Status:   Status(statusStr),
		Priority: PriorityMedium,
		Date:     date,
		Tags:     []string{},
		Body:     "## What\nTest task: " + title + ".\n",
	}
	_, err := s.Save(tk, tk.Body)
	if err != nil {
		t.Fatalf("Save %q: %v", title, err)
	}
	return tk.ToJSON()
}

// --- TaskIndexFilePath -------------------------------------------------------

func TestTaskIndexFilePath_ReturnsExpectedPath(t *testing.T) {
	got := TaskIndexFilePath("/project")
	want := "/project/.logosyncx/task-index.jsonl"
	if got != want {
		t.Errorf("TaskIndexFilePath = %q, want %q", got, want)
	}
}

// --- ReadAllTaskIndex --------------------------------------------------------

func TestReadAllTaskIndex_FileNotExist_ReturnsErrNotExist(t *testing.T) {
	dir, _ := setupTaskIndex(t)
	_, err := ReadAllTaskIndex(dir)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}

func TestReadAllTaskIndex_EmptyFile_ReturnsNoEntries(t *testing.T) {
	dir, _ := setupTaskIndex(t)
	if err := os.WriteFile(TaskIndexFilePath(dir), []byte{}, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestReadAllTaskIndex_OneEntry(t *testing.T) {
	dir, _ := setupTaskIndex(t)
	date := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	e := makeTaskEntry("t-abc123", "refactor auth", StatusOpen, date)

	if err := AppendTaskIndex(dir, e); err != nil {
		t.Fatalf("AppendTaskIndex: %v", err)
	}

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.ID != "t-abc123" {
		t.Errorf("ID = %q, want 't-abc123'", got.ID)
	}
	if got.Title != "refactor auth" {
		t.Errorf("Title = %q, want 'refactor auth'", got.Title)
	}
	if got.Status != StatusOpen {
		t.Errorf("Status = %q, want %q", got.Status, StatusOpen)
	}
	if !got.Date.Equal(date) {
		t.Errorf("Date = %v, want %v", got.Date, date)
	}
}

func TestReadAllTaskIndex_MultipleEntries(t *testing.T) {
	dir, _ := setupTaskIndex(t)
	date := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	for i, title := range []string{"task-a", "task-b", "task-c"} {
		ids := []string{"t-001", "t-002", "t-003"}
		e := makeTaskEntry(ids[i], title, StatusOpen, date.Add(time.Duration(i)*24*time.Hour))
		if err := AppendTaskIndex(dir, e); err != nil {
			t.Fatalf("AppendTaskIndex %s: %v", title, err)
		}
	}

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestReadAllTaskIndex_SkipsBlankLines(t *testing.T) {
	dir, _ := setupTaskIndex(t)
	date := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	e := makeTaskEntry("t-x1", "some task", StatusOpen, date)
	if err := AppendTaskIndex(dir, e); err != nil {
		t.Fatalf("AppendTaskIndex: %v", err)
	}

	// Append blank lines manually.
	f, err := os.OpenFile(TaskIndexFilePath(dir), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	_, _ = f.WriteString("\n\n")
	f.Close()

	entries, readErr := ReadAllTaskIndex(dir)
	if readErr != nil {
		t.Fatalf("ReadAllTaskIndex: %v", readErr)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry (blank lines skipped), got %d", len(entries))
	}
}

func TestReadAllTaskIndex_MalformedLine_ReturnsError(t *testing.T) {
	dir, _ := setupTaskIndex(t)
	if err := os.WriteFile(TaskIndexFilePath(dir), []byte("not valid json\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := ReadAllTaskIndex(dir)
	if err == nil {
		t.Error("expected error for malformed JSON line, got nil")
	}
}

// --- AppendTaskIndex ---------------------------------------------------------

func TestAppendTaskIndex_CreatesFileIfNotExists(t *testing.T) {
	dir, _ := setupTaskIndex(t)
	date := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	e := makeTaskEntry("t-new1", "new task", StatusOpen, date)

	if err := AppendTaskIndex(dir, e); err != nil {
		t.Fatalf("AppendTaskIndex: %v", err)
	}
	if _, err := os.Stat(TaskIndexFilePath(dir)); err != nil {
		t.Errorf("expected task-index.jsonl to exist, got: %v", err)
	}
}

func TestAppendTaskIndex_MultipleCallsAccumulate(t *testing.T) {
	dir, _ := setupTaskIndex(t)
	date := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	for i, title := range []string{"alpha", "beta", "gamma"} {
		ids := []string{"t-a1", "t-b2", "t-g3"}
		e := makeTaskEntry(ids[i], title, StatusOpen, date.Add(time.Duration(i)*time.Hour))
		if err := AppendTaskIndex(dir, e); err != nil {
			t.Fatalf("AppendTaskIndex %q: %v", title, err)
		}
	}

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries after 3 appends, got %d", len(entries))
	}
}

func TestAppendTaskIndex_PreservesExistingEntries(t *testing.T) {
	dir, _ := setupTaskIndex(t)
	date := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	e1 := makeTaskEntry("t-first", "first task", StatusOpen, date)
	e2 := makeTaskEntry("t-second", "second task", StatusInProgress, date.Add(time.Hour))

	if err := AppendTaskIndex(dir, e1); err != nil {
		t.Fatalf("first AppendTaskIndex: %v", err)
	}
	if err := AppendTaskIndex(dir, e2); err != nil {
		t.Fatalf("second AppendTaskIndex: %v", err)
	}

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if entries[0].ID != "t-first" {
		t.Errorf("first entry ID = %q, want 't-first'", entries[0].ID)
	}
	if entries[1].ID != "t-second" {
		t.Errorf("second entry ID = %q, want 't-second'", entries[1].ID)
	}
}

func TestAppendTaskIndex_PreservesAllFields(t *testing.T) {
	dir, _ := setupTaskIndex(t)
	date := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)
	e := TaskJSON{
		ID:       "t-full",
		Filename: "2025-03-01_full-task.md",
		Date:     date,
		Title:    "Full Task",
		Status:   StatusInProgress,
		Priority: PriorityHigh,
		Session:  "2025-02-28_auth.md",
		Tags:     []string{"refactor", "backend"},
		Assignee: "alice",
		Excerpt:  "Refactor the auth module.",
	}
	if err := AppendTaskIndex(dir, e); err != nil {
		t.Fatalf("AppendTaskIndex: %v", err)
	}

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got := entries[0]
	if got.Priority != PriorityHigh {
		t.Errorf("Priority = %q, want %q", got.Priority, PriorityHigh)
	}
	if got.Session != "2025-02-28_auth.md" {
		t.Errorf("Session = %q, want '2025-02-28_auth.md'", got.Session)
	}
	if got.Assignee != "alice" {
		t.Errorf("Assignee = %q, want 'alice'", got.Assignee)
	}
	if got.Excerpt != "Refactor the auth module." {
		t.Errorf("Excerpt = %q, want 'Refactor the auth module.'", got.Excerpt)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "refactor" {
		t.Errorf("Tags = %v, want [refactor backend]", got.Tags)
	}
}

// --- RebuildTaskIndex --------------------------------------------------------

func TestRebuildTaskIndex_EmptyTasks_CreatesEmptyIndex(t *testing.T) {
	dir, store := setupTaskIndex(t)
	n, err := store.RebuildTaskIndex()
	if err != nil {
		t.Fatalf("RebuildTaskIndex: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 tasks indexed, got %d", n)
	}
	// Index file should exist even when empty.
	if _, statErr := os.Stat(TaskIndexFilePath(dir)); statErr != nil {
		t.Errorf("task-index.jsonl should exist after RebuildTaskIndex, got: %v", statErr)
	}
}

func TestRebuildTaskIndex_IndexesAllTasks(t *testing.T) {
	dir, store := setupTaskIndex(t)
	date := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)

	writeTaskToStore(t, store, "task-alpha", "open", date)
	writeTaskToStore(t, store, "task-beta", "in_progress", date.Add(time.Hour))

	// Truncate index to force a full rebuild.
	if err := os.WriteFile(TaskIndexFilePath(dir), []byte{}, 0o644); err != nil {
		t.Fatalf("truncate index: %v", err)
	}

	n, err := store.RebuildTaskIndex()
	if err != nil {
		t.Fatalf("RebuildTaskIndex: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 tasks indexed, got %d", n)
	}

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries in index, got %d", len(entries))
	}
}

func TestRebuildTaskIndex_OverwritesExistingIndex(t *testing.T) {
	dir, store := setupTaskIndex(t)
	date := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)

	// Write a stale entry directly.
	stale := makeTaskEntry("t-stale", "old task", StatusDone, date)
	if err := AppendTaskIndex(dir, stale); err != nil {
		t.Fatalf("AppendTaskIndex stale: %v", err)
	}

	// Write one real task and rebuild.
	writeTaskToStore(t, store, "new task", "open", date.Add(time.Hour))

	// Manually rebuild (Save already calls it, but force it again cleanly).
	// First remove index to simulate a fresh rebuild scenario.
	if err := os.Remove(TaskIndexFilePath(dir)); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove index: %v", err)
	}

	n, err := store.RebuildTaskIndex()
	if err != nil {
		t.Fatalf("RebuildTaskIndex: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 task indexed after rebuild, got %d", n)
	}

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (stale overwritten), got %d", len(entries))
	}
	if entries[0].Title == "old task" {
		t.Error("stale entry should have been removed by RebuildTaskIndex")
	}
	if entries[0].Title != "new task" {
		t.Errorf("expected 'new task', got %q", entries[0].Title)
	}
}

func TestRebuildTaskIndex_PopulatesExcerpt(t *testing.T) {
	dir, store := setupTaskIndex(t)
	date := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)

	tk := &Task{
		Title:    "excerpt task",
		Status:   StatusOpen,
		Priority: PriorityMedium,
		Date:     date,
		Tags:     []string{},
		Body:     "## What\nThis excerpt should appear in the task index.\n",
	}
	if _, err := store.Save(tk, tk.Body); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Truncate to force rebuild.
	if err := os.WriteFile(TaskIndexFilePath(dir), []byte{}, 0o644); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	if _, err := store.RebuildTaskIndex(); err != nil {
		t.Fatalf("RebuildTaskIndex: %v", err)
	}

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Excerpt == "" {
		t.Error("expected non-empty excerpt after RebuildTaskIndex")
	}
}

func TestRebuildTaskIndex_NoTasksDir_ReturnsZero(t *testing.T) {
	// Project root has .logosyncx/ but no tasks/ subdir.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".logosyncx"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfg := config.Default("test-project")
	store := NewStore(dir, &cfg)

	n, err := store.RebuildTaskIndex()
	if err != nil {
		t.Fatalf("RebuildTaskIndex with no tasks dir: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 tasks, got %d", n)
	}
}

// --- Save maintains index ----------------------------------------------------

func TestSave_AppendsToTaskIndex(t *testing.T) {
	dir, store := setupTaskIndex(t)
	date := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)

	writeTaskToStore(t, store, "index task", "open", date)

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after Save, got %d", len(entries))
	}
	if entries[0].Title != "index task" {
		t.Errorf("Title = %q, want 'index task'", entries[0].Title)
	}
}

func TestSave_MultipleTasksAccumulateInIndex(t *testing.T) {
	dir, store := setupTaskIndex(t)
	date := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)

	writeTaskToStore(t, store, "first task", "open", date)
	writeTaskToStore(t, store, "second task", "in_progress", date.Add(time.Hour))
	writeTaskToStore(t, store, "third task", "open", date.Add(2*time.Hour))

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

// --- UpdateFields rebuilds index ---------------------------------------------

func TestUpdateFields_RebuildsIndex(t *testing.T) {
	dir, store := setupTaskIndex(t)
	date := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)

	writeTaskToStore(t, store, "update me", "open", date)

	// Confirm index has the original status.
	before, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex before: %v", err)
	}
	if len(before) == 0 {
		t.Fatal("expected at least 1 entry before update")
	}

	// Update status.
	if err := store.UpdateFields("update-me", map[string]string{"status": "in_progress"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	after, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex after: %v", err)
	}
	if len(after) != 1 {
		t.Fatalf("expected 1 entry after update, got %d", len(after))
	}
	if after[0].Status != StatusInProgress {
		t.Errorf("Status after update = %q, want %q", after[0].Status, StatusInProgress)
	}
}

// --- Delete rebuilds index ---------------------------------------------------

func TestDelete_RebuildsIndex(t *testing.T) {
	dir, store := setupTaskIndex(t)
	date := time.Date(2025, 3, 1, 10, 0, 0, 0, time.UTC)

	writeTaskToStore(t, store, "keep me", "open", date)
	writeTaskToStore(t, store, "delete me", "open", date.Add(time.Hour))

	// Confirm 2 entries before delete.
	before, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex before: %v", err)
	}
	if len(before) != 2 {
		t.Fatalf("expected 2 entries before delete, got %d", len(before))
	}

	// Delete one task.
	if err := store.Delete("delete-me"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	after, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex after: %v", err)
	}
	if len(after) != 1 {
		t.Fatalf("expected 1 entry after delete, got %d", len(after))
	}
	if after[0].Title == "delete me" {
		t.Error("deleted task should not appear in index")
	}
	if after[0].Title != "keep me" {
		t.Errorf("remaining task title = %q, want 'keep me'", after[0].Title)
	}
}
