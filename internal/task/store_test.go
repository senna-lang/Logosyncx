package task

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/senna-lang/logosyncx/pkg/config"
)

// --- helpers -----------------------------------------------------------------

func setupStore(t *testing.T) (string, *Store) {
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

func writeTaskFile(t *testing.T, store *Store, title, status, priority string, date time.Time) string {
	t.Helper()
	tk := &Task{
		Title:    title,
		Status:   Status(status),
		Priority: Priority(priority),
		Date:     date,
		Tags:     []string{},
		Body:     "## What\nTest task: " + title + ".\n",
	}
	path, err := store.Save(tk, tk.Body)
	if err != nil {
		t.Fatalf("Save %q: %v", title, err)
	}
	return path
}

func writeSessionFile(t *testing.T, dir, filename string) {
	t.Helper()
	content := "---\nid: s1\ntopic: test\n---\n\n## Summary\nTest session.\n"
	path := filepath.Join(dir, ".logosyncx", "sessions", filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write session %q: %v", filename, err)
	}
}

// --- NewStore ----------------------------------------------------------------

func TestNewStore_SetsCorrectPaths(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("proj")
	s := NewStore(dir, &cfg)
	if s.dir != filepath.Join(dir, ".logosyncx", "tasks") {
		t.Errorf("dir = %q, want .logosyncx/tasks", s.dir)
	}
	if s.sessionDir != filepath.Join(dir, ".logosyncx", "sessions") {
		t.Errorf("sessionDir = %q, want .logosyncx/sessions", s.sessionDir)
	}
}

// --- Save --------------------------------------------------------------------

func TestSave_CreatesFile(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{
		Title:    "Implement auth",
		Status:   StatusOpen,
		Priority: PriorityHigh,
		Body:     "## What\nImplement JWT.\n",
	}
	path, err := store.Save(tk, tk.Body)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file at %s, got: %v", path, err)
	}
}

func TestSave_FileInStatusSubdir(t *testing.T) {
	_, store := setupStore(t)
	path := writeTaskFile(t, store, "subdir-test", "in_progress", "medium", time.Now())

	expectedSubdir := filepath.Join(store.dir, "in_progress")
	if !strings.HasPrefix(path, expectedSubdir) {
		t.Errorf("expected path under tasks/in_progress/, got %q", path)
	}
}

func TestSave_AutoFillsID(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{Title: "autofill-id", Body: "## What\nTest.\n"}
	if _, err := store.Save(tk, tk.Body); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if tk.ID == "" {
		t.Error("expected ID to be auto-filled, got empty string")
	}
}

func TestSave_IDHasTPrefix(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{Title: "prefix-test", Body: "## What\nTest.\n"}
	if _, err := store.Save(tk, tk.Body); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !strings.HasPrefix(tk.ID, "t-") {
		t.Errorf("ID = %q, want 't-' prefix", tk.ID)
	}
}

func TestSave_PreservesExistingID(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{ID: "t-existing", Title: "preserve-id", Body: "## What\nTest.\n"}
	if _, err := store.Save(tk, tk.Body); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if tk.ID != "t-existing" {
		t.Errorf("ID = %q, want 't-existing'", tk.ID)
	}
}

func TestSave_AutoFillsDate(t *testing.T) {
	_, store := setupStore(t)
	before := time.Now().Add(-time.Second)
	tk := &Task{Title: "autofill-date", Body: "## What\nTest.\n"}
	if _, err := store.Save(tk, tk.Body); err != nil {
		t.Fatalf("Save: %v", err)
	}
	after := time.Now().Add(time.Second)
	if tk.Date.Before(before) || tk.Date.After(after) {
		t.Errorf("Date %v not in expected range", tk.Date)
	}
}

func TestSave_PreservesExistingDate(t *testing.T) {
	_, store := setupStore(t)
	existing := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tk := &Task{Date: existing, Title: "preserve-date", Body: "## What\nTest.\n"}
	if _, err := store.Save(tk, tk.Body); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !tk.Date.Equal(existing) {
		t.Errorf("Date = %v, want %v", tk.Date, existing)
	}
}

func TestSave_AutoFillsStatusFromConfig(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{Title: "default-status", Body: "## What\nTest.\n"}
	if _, err := store.Save(tk, tk.Body); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if tk.Status == "" {
		t.Error("expected status to be auto-filled from config")
	}
	if tk.Status != Status(store.cfg.Tasks.DefaultStatus) {
		t.Errorf("Status = %q, want %q", tk.Status, store.cfg.Tasks.DefaultStatus)
	}
}

func TestSave_AutoFillsPriorityFromConfig(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{Title: "default-priority", Body: "## What\nTest.\n"}
	if _, err := store.Save(tk, tk.Body); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if tk.Priority != Priority(store.cfg.Tasks.DefaultPriority) {
		t.Errorf("Priority = %q, want %q", tk.Priority, store.cfg.Tasks.DefaultPriority)
	}
}

func TestSave_SetsFilenameAfterSave(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{Title: "filename-check", Body: "## What\nTest.\n"}
	if _, err := store.Save(tk, tk.Body); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if tk.Filename == "" {
		t.Error("expected Filename to be set after Save")
	}
	if !strings.Contains(tk.Filename, "filename-check") {
		t.Errorf("Filename = %q, want it to contain 'filename-check'", tk.Filename)
	}
}

func TestSave_FileNameFormat(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "test task", "open", "medium",
		time.Date(2025, 2, 20, 10, 0, 0, 0, time.UTC))

	// File should be in tasks/open/
	openDir := filepath.Join(store.dir, "open")
	entries, err := os.ReadDir(openDir)
	if err != nil {
		t.Fatalf("ReadDir tasks/open: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one file in tasks/open/")
	}
	name := entries[0].Name()
	if !strings.HasSuffix(name, ".md") {
		t.Errorf("filename %q should have .md suffix", name)
	}
	if len(name) < 11 || name[4] != '-' || name[7] != '-' || name[10] != '_' {
		t.Errorf("filename %q should start with YYYY-MM-DD_", name)
	}
}

func TestSave_SetsExcerptFromWhatSection(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{
		Title: "excerpt-test",
		Body:  "## What\nThe excerpt content from what section.\n",
	}
	if _, err := store.Save(tk, tk.Body); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !strings.Contains(tk.Excerpt, "excerpt content") {
		t.Errorf("Excerpt = %q, want it to contain 'excerpt content'", tk.Excerpt)
	}
}

func TestSave_CreatesDirIfNotExist(t *testing.T) {
	dir := t.TempDir()
	// Do NOT create tasks/ dir — Save should create it (including the status subdir).
	if err := os.MkdirAll(filepath.Join(dir, ".logosyncx"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfg := config.Default("proj")
	store := NewStore(dir, &cfg)

	tk := &Task{Title: "create-dir", Body: "## What\nTest.\n"}
	if _, err := store.Save(tk, tk.Body); err != nil {
		t.Fatalf("Save should create tasks/open/ dir automatically: %v", err)
	}
	// Default status is "open", so tasks/open/ should have been created.
	openDir := filepath.Join(store.dir, string(StatusOpen))
	if _, err := os.Stat(openDir); err != nil {
		t.Errorf("expected tasks/open/ dir to be created, got: %v", err)
	}
}

// --- List --------------------------------------------------------------------

func TestList_EmptyDir_ReturnsEmpty(t *testing.T) {
	_, store := setupStore(t)
	tasks, err := store.List(Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestList_NoDirExists_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("proj")
	store := NewStore(dir, &cfg)
	tasks, err := store.List(Filter{})
	if err != nil {
		t.Fatalf("List on missing dir should not error: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestList_ReturnsAllTasks(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "task-one", "open", "high", time.Now())
	writeTaskFile(t, store, "task-two", "open", "medium", time.Now())

	tasks, err := store.List(Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestList_ReturnsTasksAcrossAllStatusSubdirs(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "open-task", "open", "medium", time.Now())
	writeTaskFile(t, store, "wip-task", "in_progress", "medium", time.Now())
	writeTaskFile(t, store, "done-task", "done", "medium", time.Now())
	writeTaskFile(t, store, "cancelled-task", "cancelled", "medium", time.Now())

	tasks, err := store.List(Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 4 {
		t.Errorf("expected 4 tasks across all subdirs, got %d", len(tasks))
	}
}

func TestList_SortedNewestFirst(t *testing.T) {
	_, store := setupStore(t)
	older := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	writeTaskFile(t, store, "old-task", "open", "medium", older)
	writeTaskFile(t, store, "new-task", "open", "medium", newer)

	tasks, err := store.List(Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].Title != "new-task" {
		t.Errorf("expected newest first, got %q", tasks[0].Title)
	}
}

func TestList_AppliesStatusFilter(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "open-task", "open", "medium", time.Now())
	writeTaskFile(t, store, "wip-task", "in_progress", "medium", time.Now())

	tasks, err := store.List(Filter{Status: StatusOpen})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "open-task" {
		t.Errorf("expected only 'open-task', got %v", tasks)
	}
}

func TestList_AppliesPriorityFilter(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "high-task", "open", "high", time.Now())
	writeTaskFile(t, store, "low-task", "open", "low", time.Now())

	tasks, err := store.List(Filter{Priority: PriorityHigh})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "high-task" {
		t.Errorf("expected only 'high-task', got %v", tasks)
	}
}

// --- Get ---------------------------------------------------------------------

func TestGet_ExactFilenameMatch(t *testing.T) {
	_, store := setupStore(t)
	path := writeTaskFile(t, store, "auth-task", "open", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))
	filename := filepath.Base(path)

	got, err := store.Get(filename)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "auth-task" {
		t.Errorf("Title = %q, want 'auth-task'", got.Title)
	}
}

func TestGet_PartialMatch(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "auth-task", "open", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))

	got, err := store.Get("auth-task")
	if err != nil {
		t.Fatalf("Get partial match: %v", err)
	}
	if got.Title != "auth-task" {
		t.Errorf("Title = %q, want 'auth-task'", got.Title)
	}
}

func TestGet_FindsAcrossStatusSubdirs(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "open-task", "open", "medium", time.Now())
	writeTaskFile(t, store, "done-task", "done", "medium", time.Now())

	got, err := store.Get("done-task")
	if err != nil {
		t.Fatalf("Get across subdirs: %v", err)
	}
	if got.Title != "done-task" {
		t.Errorf("Title = %q, want 'done-task'", got.Title)
	}
	if got.Status != StatusDone {
		t.Errorf("Status = %q, want 'done'", got.Status)
	}
}

func TestGet_NotFound_ReturnsErrNotFound(t *testing.T) {
	_, store := setupStore(t)

	_, err := store.Get("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGet_AmbiguousMatch_ReturnsErrAmbiguous(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "auth-login", "open", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))
	writeTaskFile(t, store, "auth-signup", "open", "medium", time.Date(2025, 2, 21, 0, 0, 0, 0, time.UTC))

	_, err := store.Get("auth")
	if !errors.Is(err, ErrAmbiguous) {
		t.Errorf("expected ErrAmbiguous, got %v", err)
	}
}

func TestGet_AmbiguousAcrossSubdirs(t *testing.T) {
	_, store := setupStore(t)
	// Same slug in different status dirs would be ambiguous.
	writeTaskFile(t, store, "auth-login", "open", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))
	writeTaskFile(t, store, "auth-logout", "done", "medium", time.Date(2025, 2, 21, 0, 0, 0, 0, time.UTC))

	_, err := store.Get("auth")
	if !errors.Is(err, ErrAmbiguous) {
		t.Errorf("expected ErrAmbiguous across subdirs, got %v", err)
	}
}

func TestGet_CaseInsensitive(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "auth-task", "open", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))

	got, err := store.Get("AUTH-TASK")
	if err != nil {
		t.Fatalf("Get case-insensitive: %v", err)
	}
	if got.Title != "auth-task" {
		t.Errorf("Title = %q, want 'auth-task'", got.Title)
	}
}

// --- UpdateFields ------------------------------------------------------------

func TestUpdateFields_Status(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "update-status", "open", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))

	if err := store.UpdateFields("update-status", map[string]string{"status": "in_progress"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	got, err := store.Get("update-status")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Status != StatusInProgress {
		t.Errorf("Status = %q, want 'in_progress'", got.Status)
	}
}

func TestUpdateFields_Status_MovesFile(t *testing.T) {
	_, store := setupStore(t)
	path := writeTaskFile(t, store, "move-me", "open", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))

	// Confirm the file starts in tasks/open/.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("original file should exist at %s", path)
	}

	if err := store.UpdateFields("move-me", map[string]string{"status": "in_progress"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	// Old path should no longer exist.
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("old file should have been removed from %s", path)
	}

	// File should now be in tasks/in_progress/.
	filename := filepath.Base(path)
	newPath := filepath.Join(store.dir, "in_progress", filename)
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("file should have been moved to %s: %v", newPath, err)
	}
}

func TestUpdateFields_NonStatusField_DoesNotMoveFile(t *testing.T) {
	_, store := setupStore(t)
	path := writeTaskFile(t, store, "no-move", "open", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))

	if err := store.UpdateFields("no-move", map[string]string{"priority": "high"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	// File should still be in tasks/open/.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should remain at %s after non-status update: %v", path, err)
	}
}

func TestUpdateFields_Priority(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "update-priority", "open", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))

	if err := store.UpdateFields("update-priority", map[string]string{"priority": "high"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	got, err := store.Get("update-priority")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Priority != PriorityHigh {
		t.Errorf("Priority = %q, want 'high'", got.Priority)
	}
}

func TestUpdateFields_Assignee(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "update-assignee", "open", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))

	if err := store.UpdateFields("update-assignee", map[string]string{"assignee": "alice"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	got, err := store.Get("update-assignee")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Assignee != "alice" {
		t.Errorf("Assignee = %q, want 'alice'", got.Assignee)
	}
}

func TestUpdateFields_Session(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "update-session", "open", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))

	if err := store.UpdateFields("update-session", map[string]string{"session": "2025-02-15_auth.md"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	got, err := store.Get("update-session")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Session != "2025-02-15_auth.md" {
		t.Errorf("Session = %q, want '2025-02-15_auth.md'", got.Session)
	}
}

func TestUpdateFields_MultipleFields(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "multi-update", "open", "low", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))

	if err := store.UpdateFields("multi-update", map[string]string{
		"status":   "in_progress",
		"priority": "high",
		"assignee": "bob",
	}); err != nil {
		t.Fatalf("UpdateFields multiple: %v", err)
	}

	got, err := store.Get("multi-update")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Status != StatusInProgress {
		t.Errorf("Status = %q, want 'in_progress'", got.Status)
	}
	if got.Priority != PriorityHigh {
		t.Errorf("Priority = %q, want 'high'", got.Priority)
	}
	if got.Assignee != "bob" {
		t.Errorf("Assignee = %q, want 'bob'", got.Assignee)
	}
}

func TestUpdateFields_UnknownField_ReturnsError(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "unknown-field", "open", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))

	err := store.UpdateFields("unknown-field", map[string]string{"nonexistent": "value"})
	if err == nil {
		t.Error("expected error for unknown field, got nil")
	}
}

func TestUpdateFields_NotFound_ReturnsErrNotFound(t *testing.T) {
	_, store := setupStore(t)
	err := store.UpdateFields("nonexistent", map[string]string{"status": "open"})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateFields_BodyPreserved(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{
		Title:    "body-preserved",
		Status:   StatusOpen,
		Priority: PriorityMedium,
		Body:     "## What\nThis body must survive the update.\n\n## Why\nBecause.\n",
	}
	if _, err := store.Save(tk, tk.Body); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.UpdateFields("body-preserved", map[string]string{"status": "in_progress"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	got, err := store.Get("body-preserved")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if !strings.Contains(got.Body, "This body must survive") {
		t.Errorf("Body = %q, expected 'This body must survive'", got.Body)
	}
}

// --- Delete ------------------------------------------------------------------

func TestDelete_RemovesFile(t *testing.T) {
	_, store := setupStore(t)
	path := writeTaskFile(t, store, "to-delete", "open", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))

	if err := store.Delete("to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected file to be deleted, got: %v", err)
	}
}

func TestDelete_RemovesFileFromAnySubdir(t *testing.T) {
	_, store := setupStore(t)
	path := writeTaskFile(t, store, "done-delete", "done", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))

	if err := store.Delete("done-delete"); err != nil {
		t.Fatalf("Delete from done/: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected file to be deleted from tasks/done/, got: %v", err)
	}
}

func TestDelete_NotFound_ReturnsErrNotFound(t *testing.T) {
	_, store := setupStore(t)
	err := store.Delete("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete_AmbiguousMatch_ReturnsErrAmbiguous(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "auth-login", "open", "medium", time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC))
	writeTaskFile(t, store, "auth-signup", "open", "medium", time.Date(2025, 2, 21, 0, 0, 0, 0, time.UTC))

	err := store.Delete("auth")
	if !errors.Is(err, ErrAmbiguous) {
		t.Errorf("expected ErrAmbiguous, got %v", err)
	}
}

func TestDelete_TaskGoneFromList(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "delete-me", "open", "medium", time.Now())
	writeTaskFile(t, store, "keep-me", "open", "medium", time.Now())

	if err := store.Delete("delete-me"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	tasks, err := store.List(Filter{})
	if err != nil {
		t.Fatalf("List after delete: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task after delete, got %d", len(tasks))
	}
	if tasks[0].Title == "delete-me" {
		t.Error("deleted task should not appear in List")
	}
}

// --- Purge -------------------------------------------------------------------

func TestPurge_DeletesAllWithStatus(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "done-one", "done", "medium", time.Now())
	writeTaskFile(t, store, "done-two", "done", "medium", time.Now())
	writeTaskFile(t, store, "keep-open", "open", "medium", time.Now())

	n, err := store.Purge(StatusDone)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 deleted, got %d", n)
	}

	remaining, err := store.List(Filter{})
	if err != nil {
		t.Fatalf("List after Purge: %v", err)
	}
	if len(remaining) != 1 || remaining[0].Title != "keep-open" {
		t.Errorf("expected only 'keep-open' to remain, got %d tasks", len(remaining))
	}
}

func TestPurge_EmptyStatus_ReturnsZero(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "open-task", "open", "medium", time.Now())

	n, err := store.Purge(StatusDone) // no done tasks exist
	if err != nil {
		t.Fatalf("Purge on empty status: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 deleted, got %d", n)
	}

	// open task should be unaffected
	tasks, _ := store.List(Filter{})
	if len(tasks) != 1 {
		t.Errorf("expected 1 remaining task, got %d", len(tasks))
	}
}

func TestPurge_NoDirExists_ReturnsZero(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("proj")
	store := NewStore(dir, &cfg)

	n, err := store.Purge(StatusCancelled)
	if err != nil {
		t.Fatalf("Purge on missing dir should not error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestPurge_OnlyDeletesMatchingStatus(t *testing.T) {
	_, store := setupStore(t)
	writeTaskFile(t, store, "cancelled-task", "cancelled", "medium", time.Now())
	writeTaskFile(t, store, "open-task", "open", "medium", time.Now())
	writeTaskFile(t, store, "wip-task", "in_progress", "medium", time.Now())

	n, err := store.Purge(StatusCancelled)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 deleted, got %d", n)
	}

	remaining, _ := store.List(Filter{})
	if len(remaining) != 2 {
		t.Errorf("expected 2 remaining tasks, got %d", len(remaining))
	}
	for _, task := range remaining {
		if task.Status == StatusCancelled {
			t.Errorf("cancelled task should have been purged, found %q", task.Title)
		}
	}
}

// --- ResolveSession ----------------------------------------------------------

func TestResolveSession_ExactFilename(t *testing.T) {
	dir, store := setupStore(t)
	writeSessionFile(t, dir, "2025-02-20_auth-refactor.md")

	got, err := store.ResolveSession("2025-02-20_auth-refactor.md")
	if err != nil {
		t.Fatalf("ResolveSession: %v", err)
	}
	if got != "2025-02-20_auth-refactor.md" {
		t.Errorf("got %q, want '2025-02-20_auth-refactor.md'", got)
	}
}

func TestResolveSession_PartialMatch(t *testing.T) {
	dir, store := setupStore(t)
	writeSessionFile(t, dir, "2025-02-20_auth-refactor.md")

	got, err := store.ResolveSession("auth-refactor")
	if err != nil {
		t.Fatalf("ResolveSession partial: %v", err)
	}
	if got != "2025-02-20_auth-refactor.md" {
		t.Errorf("got %q, want '2025-02-20_auth-refactor.md'", got)
	}
}

func TestResolveSession_NotFound_ReturnsErrNotFound(t *testing.T) {
	_, store := setupStore(t)

	_, err := store.ResolveSession("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestResolveSession_AmbiguousMatch_ReturnsErrAmbiguous(t *testing.T) {
	dir, store := setupStore(t)
	writeSessionFile(t, dir, "2025-02-20_auth-login.md")
	writeSessionFile(t, dir, "2025-02-21_auth-signup.md")

	_, err := store.ResolveSession("auth")
	if !errors.Is(err, ErrAmbiguous) {
		t.Errorf("expected ErrAmbiguous, got %v", err)
	}
}

func TestResolveSession_NoSessionsDir_ReturnsErrNotFound(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("proj")
	store := NewStore(dir, &cfg)

	_, err := store.ResolveSession("anything")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound for missing sessions dir, got %v", err)
	}
}

func TestResolveSession_CaseInsensitive(t *testing.T) {
	dir, store := setupStore(t)
	writeSessionFile(t, dir, "2025-02-20_AUTH-refactor.md")

	got, err := store.ResolveSession("auth-refactor")
	if err != nil {
		t.Fatalf("ResolveSession case-insensitive: %v", err)
	}
	if got != "2025-02-20_AUTH-refactor.md" {
		t.Errorf("got %q, want '2025-02-20_AUTH-refactor.md'", got)
	}
}

// --- generateID --------------------------------------------------------------

func TestGenerateTaskID_HasTPrefix(t *testing.T) {
	id, err := generateID()
	if err != nil {
		t.Fatalf("generateID: %v", err)
	}
	if !strings.HasPrefix(id, "t-") {
		t.Errorf("ID = %q, want 't-' prefix", id)
	}
}

func TestGenerateTaskID_CorrectLength(t *testing.T) {
	id, err := generateID()
	if err != nil {
		t.Fatalf("generateID: %v", err)
	}
	// "t-" + 6 hex chars = 8 chars total
	if len(id) != 8 {
		t.Errorf("ID = %q, want length 8 ('t-' + 6 hex chars)", id)
	}
}

func TestGenerateTaskID_IsUnique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 20; i++ {
		id, err := generateID()
		if err != nil {
			t.Fatalf("generateID: %v", err)
		}
		if ids[id] {
			t.Errorf("duplicate ID generated: %q", id)
		}
		ids[id] = true
	}
}

// --- sortByDateDesc ----------------------------------------------------------

func TestSortByDateDesc_Tasks(t *testing.T) {
	tasks := []*Task{
		{Title: "a", Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "c", Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
		{Title: "b", Date: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
	}
	sortByDateDesc(tasks)
	if tasks[0].Title != "c" || tasks[1].Title != "b" || tasks[2].Title != "a" {
		t.Errorf("unexpected order: %v %v %v", tasks[0].Title, tasks[1].Title, tasks[2].Title)
	}
}

func TestSortByDateDesc_SingleElement(t *testing.T) {
	tasks := []*Task{{Title: "only", Date: time.Now()}}
	sortByDateDesc(tasks) // should not panic
	if tasks[0].Title != "only" {
		t.Errorf("single element lost after sort")
	}
}
