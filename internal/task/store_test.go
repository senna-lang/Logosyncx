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

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func setupStore(t *testing.T) (string, *Store) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".logosyncx", "tasks"), 0o755); err != nil {
		t.Fatalf("mkdir tasks: %v", err)
	}
	cfg := config.Default("test-project")
	return dir, NewStore(dir, &cfg)
}

// createTask is a test helper that creates a task via store.Create and returns
// the resulting *Task (reloaded from disk).
func createTask(t *testing.T, store *Store, plan, title, status, priority string, dependsOn []int) *Task {
	t.Helper()
	tk := &Task{
		Title:     title,
		Plan:      plan,
		Status:    Status(status),
		Priority:  Priority(priority),
		Tags:      []string{},
		DependsOn: dependsOn,
	}
	path, err := store.Create(tk)
	if err != nil {
		t.Fatalf("Create %q: %v", title, err)
	}
	loaded, err := store.loadFile(path)
	if err != nil {
		t.Fatalf("loadFile after Create %q: %v", title, err)
	}
	return loaded
}

// writePlanTaskMD writes a raw TASK.md under tasks/<plan>/<taskDir>/ without
// going through store.Create. Useful for testing loadAll / RebuildTaskIndex
// with hand-crafted content.
func writePlanTaskMD(t *testing.T, dir, plan, taskDir, content string) string {
	t.Helper()
	p := filepath.Join(dir, ".logosyncx", "tasks", plan, taskDir)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", p, err)
	}
	path := filepath.Join(p, taskFileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	return path
}

// ---------------------------------------------------------------------------
// NewStore
// ---------------------------------------------------------------------------

func TestNewStore_SetsCorrectPaths(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("proj")
	s := NewStore(dir, &cfg)
	if s.dir != filepath.Join(dir, ".logosyncx", "tasks") {
		t.Errorf("dir = %q, want .logosyncx/tasks", s.dir)
	}
	if s.plansDir != filepath.Join(dir, ".logosyncx", "plans") {
		t.Errorf("plansDir = %q, want .logosyncx/plans", s.plansDir)
	}
}

// ---------------------------------------------------------------------------
// NextSeq
// ---------------------------------------------------------------------------

func TestStore_NextSeq_EmptyDir_Returns1(t *testing.T) {
	_, store := setupStore(t)
	planGroupDir := filepath.Join(store.dir, "20260304-auth-refactor")
	// Directory does not exist yet.
	seq, err := store.NextSeq(planGroupDir)
	if err != nil {
		t.Fatalf("NextSeq: %v", err)
	}
	if seq != 1 {
		t.Errorf("NextSeq = %d, want 1", seq)
	}
}

func TestStore_NextSeq_ExistingTasks(t *testing.T) {
	_, store := setupStore(t)
	planGroupDir := filepath.Join(store.dir, "20260304-auth-refactor")
	// Create two task directories manually.
	for _, name := range []string{"001-setup-keys", "002-add-middleware"} {
		if err := os.MkdirAll(filepath.Join(planGroupDir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	seq, err := store.NextSeq(planGroupDir)
	if err != nil {
		t.Fatalf("NextSeq: %v", err)
	}
	if seq != 3 {
		t.Errorf("NextSeq = %d, want 3", seq)
	}
}

func TestStore_NextSeq_NonContiguousSeqs(t *testing.T) {
	_, store := setupStore(t)
	planGroupDir := filepath.Join(store.dir, "20260304-auth")
	for _, name := range []string{"001-alpha", "005-gamma"} {
		if err := os.MkdirAll(filepath.Join(planGroupDir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	seq, err := store.NextSeq(planGroupDir)
	if err != nil {
		t.Fatalf("NextSeq: %v", err)
	}
	if seq != 6 {
		t.Errorf("NextSeq = %d, want 6", seq)
	}
}

func TestStore_NextSeq_IgnoresFiles(t *testing.T) {
	_, store := setupStore(t)
	planGroupDir := filepath.Join(store.dir, "20260304-auth")
	if err := os.MkdirAll(planGroupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A file (not dir) should be ignored even if it has a numeric prefix.
	_ = os.WriteFile(filepath.Join(planGroupDir, "003-readme.md"), []byte("x"), 0o644)
	seq, err := store.NextSeq(planGroupDir)
	if err != nil {
		t.Fatalf("NextSeq: %v", err)
	}
	if seq != 1 {
		t.Errorf("NextSeq = %d, want 1 (files ignored)", seq)
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestStore_Create_WritesTaskMDInPlanGroupDir(t *testing.T) {
	dir, store := setupStore(t)
	tk := &Task{Title: "Add JWT middleware", Plan: "20260304-auth-refactor", Tags: []string{}}

	path, err := store.Create(tk)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// TASK.md must exist at the returned path.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("TASK.md not found at %s: %v", path, err)
	}

	// The file must be inside tasks/20260304-auth-refactor/.
	rel, err := filepath.Rel(filepath.Join(dir, ".logosyncx", "tasks", "20260304-auth-refactor"), path)
	if err != nil || strings.HasPrefix(rel, "..") {
		t.Errorf("TASK.md %q is not inside plan group dir", path)
	}
}

func TestStore_Create_AutoAssignsSeq(t *testing.T) {
	_, store := setupStore(t)

	tk1 := &Task{Title: "First task", Plan: "20260304-auth", Tags: []string{}}
	if _, err := store.Create(tk1); err != nil {
		t.Fatalf("Create first: %v", err)
	}

	tk2 := &Task{Title: "Second task", Plan: "20260304-auth", Tags: []string{}}
	if _, err := store.Create(tk2); err != nil {
		t.Fatalf("Create second: %v", err)
	}

	if tk1.Seq != 1 {
		t.Errorf("first task Seq = %d, want 1", tk1.Seq)
	}
	if tk2.Seq != 2 {
		t.Errorf("second task Seq = %d, want 2", tk2.Seq)
	}
}

func TestStore_Create_DirNameFormat(t *testing.T) {
	dir, store := setupStore(t)
	tk := &Task{Title: "Setup RS256 keys", Plan: "20260304-auth", Tags: []string{}}
	if _, err := store.Create(tk); err != nil {
		t.Fatalf("Create: %v", err)
	}

	planGroupDir := filepath.Join(dir, ".logosyncx", "tasks", "20260304-auth")
	entries, err := os.ReadDir(planGroupDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 task dir, got %d", len(entries))
	}
	name := entries[0].Name()
	if !strings.HasPrefix(name, "001-") {
		t.Errorf("task dir %q should start with '001-'", name)
	}
	if !strings.Contains(name, "setup-rs256-keys") {
		t.Errorf("task dir %q should contain slug 'setup-rs256-keys'", name)
	}
}

func TestStore_Create_AutoFillsID(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{Title: "task", Plan: "20260304-auth", Tags: []string{}}
	if _, err := store.Create(tk); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if tk.ID == "" {
		t.Error("expected ID to be auto-filled")
	}
	if !strings.HasPrefix(tk.ID, "t-") {
		t.Errorf("ID = %q, want 't-' prefix", tk.ID)
	}
}

func TestStore_Create_AutoFillsDate(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{Title: "task", Plan: "20260304-auth", Tags: []string{}}
	before := time.Now()
	if _, err := store.Create(tk); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if tk.Date.IsZero() {
		t.Error("expected Date to be auto-filled")
	}
	if tk.Date.Before(before) {
		t.Errorf("Date %v is before the test started", tk.Date)
	}
}

func TestStore_Create_AutoFillsStatusFromConfig(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{Title: "task", Plan: "20260304-auth", Tags: []string{}}
	if _, err := store.Create(tk); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if tk.Status != StatusOpen {
		t.Errorf("Status = %q, want 'open' (default)", tk.Status)
	}
}

func TestStore_Create_SetsTaskDirPath(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{Title: "dirpath test", Plan: "20260304-auth", Tags: []string{}}
	if _, err := store.Create(tk); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if tk.DirPath == "" {
		t.Error("expected DirPath to be set after Create")
	}
	if _, err := os.Stat(tk.DirPath); err != nil {
		t.Errorf("DirPath %q does not exist: %v", tk.DirPath, err)
	}
}

func TestStore_Create_RequiresPlan(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{Title: "task", Plan: "", Tags: []string{}}
	_, err := store.Create(tk)
	if err == nil {
		t.Fatal("expected error when plan is empty, got nil")
	}
}

func TestStore_Create_RequiresTitle(t *testing.T) {
	_, store := setupStore(t)
	tk := &Task{Title: "", Plan: "20260304-auth", Tags: []string{}}
	_, err := store.Create(tk)
	if err == nil {
		t.Fatal("expected error when title is empty, got nil")
	}
}

func TestStore_Create_AppendsBothTasksInPlanGroup(t *testing.T) {
	dir, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Task A", "open", "medium", nil)
	createTask(t, store, "20260304-auth", "Task B", "open", "medium", nil)

	planGroupDir := filepath.Join(dir, ".logosyncx", "tasks", "20260304-auth")
	entries, err := os.ReadDir(planGroupDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	dirs := 0
	for _, e := range entries {
		if e.IsDir() {
			dirs++
		}
	}
	if dirs != 2 {
		t.Errorf("expected 2 task dirs, got %d", dirs)
	}
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestStore_Get_FindsByPartialName(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Add JWT middleware", "open", "medium", nil)

	got, err := store.Get("", "jwt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Add JWT middleware" {
		t.Errorf("Title = %q, want 'Add JWT middleware'", got.Title)
	}
}

func TestStore_Get_FindsByPlanPartial(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth-refactor", "Task A", "open", "medium", nil)
	createTask(t, store, "20260305-db-schema", "Task A", "open", "medium", nil)

	// "task-a" matches both, but "auth" plan partial narrows it to one.
	got, err := store.Get("auth", "task-a")
	if err != nil {
		t.Fatalf("Get with planPartial: %v", err)
	}
	if got.Plan != "20260304-auth-refactor" {
		t.Errorf("Plan = %q, want '20260304-auth-refactor'", got.Plan)
	}
}

func TestStore_Get_NotFound_ReturnsErrNotFound(t *testing.T) {
	_, store := setupStore(t)
	_, err := store.Get("", "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_Get_AmbiguousAcrossPlans(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Add middleware", "open", "medium", nil)
	createTask(t, store, "20260305-db", "Add middleware", "open", "medium", nil)

	// Both plans have a task matching "add-middleware" and no plan filter.
	_, err := store.Get("", "add-middleware")
	if !errors.Is(err, ErrAmbiguous) {
		t.Errorf("expected ErrAmbiguous, got %v", err)
	}
}

func TestStore_Get_CaseInsensitive(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Add JWT Middleware", "open", "medium", nil)

	got, err := store.Get("", "ADD-JWT")
	if err != nil {
		t.Fatalf("Get case-insensitive: %v", err)
	}
	if got.Title != "Add JWT Middleware" {
		t.Errorf("Title = %q, want 'Add JWT Middleware'", got.Title)
	}
}

func TestStore_GetByName_SearchesAllPlans(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Unique task name", "open", "medium", nil)

	got, err := store.GetByName("unique-task")
	if err != nil {
		t.Fatalf("GetByName: %v", err)
	}
	if got.Title != "Unique task name" {
		t.Errorf("Title = %q, want 'Unique task name'", got.Title)
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestStore_List_EmptyTasks_ReturnsEmpty(t *testing.T) {
	_, store := setupStore(t)
	tasks, err := store.List(Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(tasks))
	}
}

func TestStore_List_ReturnsAllTasks(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Task A", "open", "medium", nil)
	createTask(t, store, "20260304-auth", "Task B", "open", "medium", nil)
	createTask(t, store, "20260305-db", "Task C", "in_progress", "high", nil)

	tasks, err := store.List(Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestStore_List_FilterByStatus(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Open task", "open", "medium", nil)
	createTask(t, store, "20260304-auth", "Done task", "done", "medium", nil)

	tasks, err := store.List(Filter{Status: StatusOpen})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 open task, got %d", len(tasks))
	}
	if tasks[0].Status != StatusOpen {
		t.Errorf("Status = %q, want 'open'", tasks[0].Status)
	}
}

func TestStore_List_FilterByPlan(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Auth task", "open", "medium", nil)
	createTask(t, store, "20260305-db", "DB task", "open", "medium", nil)

	tasks, err := store.List(Filter{Plan: "auth"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 auth task, got %d", len(tasks))
	}
	if tasks[0].Title != "Auth task" {
		t.Errorf("Title = %q, want 'Auth task'", tasks[0].Title)
	}
}

func TestStore_List_SortedNewestFirst(t *testing.T) {
	_, store := setupStore(t)

	// Create tasks and manually set dates via raw write to control ordering.
	older := "---\nid: t-001\ntitle: old\nseq: 1\ndate: 2025-01-01T00:00:00Z\nstatus: open\npriority: medium\nplan: 20260304-auth\ntags: []\nassignee: \n---\n\n## What\nOld task.\n"
	newer := "---\nid: t-002\ntitle: new\nseq: 2\ndate: 2025-06-01T00:00:00Z\nstatus: open\npriority: medium\nplan: 20260304-auth\ntags: []\nassignee: \n---\n\n## What\nNew task.\n"
	writePlanTaskMD(t, store.projectRoot, "20260304-auth", "001-old", older)
	writePlanTaskMD(t, store.projectRoot, "20260304-auth", "002-new", newer)

	tasks, err := store.List(Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(tasks) < 2 {
		t.Fatalf("expected at least 2 tasks, got %d", len(tasks))
	}
	if !tasks[0].Date.After(tasks[1].Date) {
		t.Errorf("expected newest first: tasks[0].Date=%v tasks[1].Date=%v",
			tasks[0].Date, tasks[1].Date)
	}
}

// ---------------------------------------------------------------------------
// UpdateFields
// ---------------------------------------------------------------------------

func TestStore_UpdateFields_NoFileMoves(t *testing.T) {
	dir, store := setupStore(t)
	tk := createTask(t, store, "20260304-auth", "Update me", "open", "medium", nil)
	originalDir := tk.DirPath

	if err := store.UpdateFields("", "update-me", map[string]string{"status": "in_progress"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	// DirPath must not change.
	if _, err := os.Stat(filepath.Join(originalDir, taskFileName)); err != nil {
		t.Errorf("TASK.md should still be at original path %s: %v", originalDir, err)
	}
	_ = dir
}

func TestStore_UpdateFields_UpdatesStatus(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Status test", "open", "medium", nil)

	if err := store.UpdateFields("", "status-test", map[string]string{"status": "in_progress"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	got, err := store.GetByName("status-test")
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Status != StatusInProgress {
		t.Errorf("Status = %q, want 'in_progress'", got.Status)
	}
}

func TestStore_UpdateFields_UpdatesPriority(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Priority test", "open", "medium", nil)

	if err := store.UpdateFields("", "priority-test", map[string]string{"priority": "high"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	got, err := store.GetByName("priority-test")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Priority != PriorityHigh {
		t.Errorf("Priority = %q, want 'high'", got.Priority)
	}
}

func TestStore_UpdateFields_UpdatesAssignee(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Assignee test", "open", "medium", nil)

	if err := store.UpdateFields("", "assignee-test", map[string]string{"assignee": "alice"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	got, err := store.GetByName("assignee-test")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Assignee != "alice" {
		t.Errorf("Assignee = %q, want 'alice'", got.Assignee)
	}
}

func TestStore_UpdateFields_Done_SetsCompletedAt(t *testing.T) {
	_, store := setupStore(t)
	tk := createTask(t, store, "20260304-auth", "Completed task", "open", "medium", nil)

	// Pre-write WALKTHROUGH.md with real content.
	wtPath := filepath.Join(tk.DirPath, walkthroughFileName)
	if err := os.WriteFile(wtPath, []byte("# Walkthrough\n\nContent.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	before := time.Now()
	if err := store.UpdateFields("", "completed-task", map[string]string{"status": "done"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	got, err := store.GetByName("completed-task")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set when status transitions to done")
	}
	if got.CompletedAt.Before(before) {
		t.Errorf("CompletedAt %v is before the test started", got.CompletedAt)
	}
}

func TestStore_UpdateFields_Done_RequiresWalkthroughContent(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Walkthrough task", "open", "medium", nil)

	// No WALKTHROUGH.md yet — must be rejected.
	err := store.UpdateFields("", "walkthrough-task", map[string]string{"status": "done"})
	if err == nil {
		t.Fatal("expected error when WALKTHROUGH.md has no content, got nil")
	}
	if !strings.Contains(err.Error(), "WALKTHROUGH.md") {
		t.Errorf("expected 'WALKTHROUGH.md' in error, got: %v", err)
	}
}

func TestStore_UpdateFields_Done_ScaffoldOnly_Rejected(t *testing.T) {
	_, store := setupStore(t)
	tk := createTask(t, store, "20260304-auth", "Scaffold only task", "open", "medium", nil)

	// Write scaffold-only content (all HTML comment lines).
	wtPath := filepath.Join(tk.DirPath, walkthroughFileName)
	scaffold := "<!-- fill in this section -->\n<!-- another comment -->\n"
	if err := os.WriteFile(wtPath, []byte(scaffold), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	err := store.UpdateFields("", "scaffold-only-task", map[string]string{"status": "done"})
	if err == nil {
		t.Fatal("expected error for scaffold-only WALKTHROUGH.md, got nil")
	}
}

func TestStore_UpdateFields_Done_WithContent_Succeeds(t *testing.T) {
	_, store := setupStore(t)
	tk := createTask(t, store, "20260304-auth", "Content filled task", "open", "medium", nil)

	// Write real content into WALKTHROUGH.md.
	wtPath := filepath.Join(tk.DirPath, walkthroughFileName)
	if err := os.WriteFile(wtPath, []byte("# Walkthrough\n\nActual content here.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := store.UpdateFields("", "content-filled-task", map[string]string{"status": "done"}); err != nil {
		t.Fatalf("UpdateFields with content-filled WALKTHROUGH.md: %v", err)
	}
}

func TestStore_UpdateFields_Done_WalkthroughNotOverwritten(t *testing.T) {
	_, store := setupStore(t)
	tk := createTask(t, store, "20260304-auth", "Idempotent walkthrough", "open", "medium", nil)

	// Pre-create WALKTHROUGH.md with custom content.
	wtPath := filepath.Join(tk.DirPath, walkthroughFileName)
	customContent := "# Custom walkthrough content"
	if err := os.WriteFile(wtPath, []byte(customContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Mark done — should NOT overwrite existing WALKTHROUGH.md.
	if err := store.UpdateFields("", "idempotent-walkthrough", map[string]string{"status": "done"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	data, err := os.ReadFile(wtPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != customContent {
		t.Errorf("WALKTHROUGH.md was overwritten; got %q, want %q", string(data), customContent)
	}
}

func TestStore_UpdateFields_InProgress_BlockedByDep_HardError(t *testing.T) {
	_, store := setupStore(t)
	// Create dep task (seq 1, open).
	createTask(t, store, "20260304-auth", "Dep task", "open", "medium", nil)
	// Create blocked task (seq 2, depends on seq 1).
	createTask(t, store, "20260304-auth", "Blocked task", "open", "medium", []int{1})

	err := store.UpdateFields("auth", "blocked-task", map[string]string{"status": "in_progress"})
	if err == nil {
		t.Fatal("expected ErrBlocked error, got nil")
	}
	if !errors.Is(err, ErrBlocked) {
		t.Errorf("expected ErrBlocked, got %v", err)
	}
}

func TestStore_UpdateFields_InProgress_NotBlocked_WhenDepDone(t *testing.T) {
	_, store := setupStore(t)
	// Create dep task and mark it done (requires WALKTHROUGH.md with content).
	depTask := createTask(t, store, "20260304-auth", "Dep task", "open", "medium", nil)
	wtPath := filepath.Join(depTask.DirPath, walkthroughFileName)
	if err := os.WriteFile(wtPath, []byte("# Walkthrough\n\nDone.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile walkthrough: %v", err)
	}
	if err := store.UpdateFields("auth", "dep-task", map[string]string{"status": "done"}); err != nil {
		t.Fatalf("mark dep done: %v", err)
	}
	// Create task that depends on seq 1.
	createTask(t, store, "20260304-auth", "Unblocked task", "open", "medium", []int{1})

	// Should succeed: dependency is done.
	if err := store.UpdateFields("auth", "unblocked-task", map[string]string{"status": "in_progress"}); err != nil {
		t.Fatalf("expected no error for unblocked task, got: %v", err)
	}
}

func TestStore_UpdateFields_UnknownField_ReturnsError(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Field test", "open", "medium", nil)

	err := store.UpdateFields("", "field-test", map[string]string{"unknown": "value"})
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
}

func TestStore_UpdateFields_InvalidStatus_ReturnsError(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Status validation", "open", "medium", nil)

	err := store.UpdateFields("", "status-validation", map[string]string{"status": "typo"})
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if !strings.Contains(err.Error(), "invalid status") {
		t.Errorf("expected 'invalid status' in error, got: %v", err)
	}

	// File must be unchanged: status must still be open.
	reloaded, getErr := store.GetByName("status-validation")
	if getErr != nil {
		t.Fatalf("GetByName after failed update: %v", getErr)
	}
	if reloaded.Status != StatusOpen {
		t.Errorf("status was mutated: got %q, want %q", reloaded.Status, StatusOpen)
	}
}

func TestStore_UpdateFields_InvalidPriority_ReturnsError(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Priority validation", "open", "medium", nil)

	err := store.UpdateFields("", "priority-validation", map[string]string{"priority": "urgent"})
	if err == nil {
		t.Fatal("expected error for invalid priority, got nil")
	}
	if !strings.Contains(err.Error(), "invalid priority") {
		t.Errorf("expected 'invalid priority' in error, got: %v", err)
	}

	// File must be unchanged: priority must still be medium.
	reloaded, getErr := store.GetByName("priority-validation")
	if getErr != nil {
		t.Fatalf("GetByName after failed update: %v", getErr)
	}
	if reloaded.Priority != PriorityMedium {
		t.Errorf("priority was mutated: got %q, want %q", reloaded.Priority, PriorityMedium)
	}
}

func TestStore_UpdateFields_NotFound_ReturnsErrNotFound(t *testing.T) {
	_, store := setupStore(t)
	err := store.UpdateFields("", "nonexistent-task", map[string]string{"status": "done"})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_UpdateFields_BodyPreserved(t *testing.T) {
	_, store := setupStore(t)
	tk := createTask(t, store, "20260304-auth", "Body preserved", "open", "medium", nil)

	// Write a body into TASK.md manually.
	taskPath := filepath.Join(tk.DirPath, taskFileName)
	data, err := os.ReadFile(taskPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	withBody := string(data) + "\n## What\n\nThis body must survive the update.\n"
	if err := os.WriteFile(taskPath, []byte(withBody), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := store.UpdateFields("", "body-preserved", map[string]string{"priority": "high"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	updated, err := os.ReadFile(taskPath)
	if err != nil {
		t.Fatalf("ReadFile after update: %v", err)
	}
	if !strings.Contains(string(updated), "This body must survive the update") {
		t.Error("body was lost after UpdateFields")
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestStore_Delete_RemovesTaskDir(t *testing.T) {
	_, store := setupStore(t)
	tk := createTask(t, store, "20260304-auth", "Delete me", "open", "medium", nil)
	dirPath := tk.DirPath

	deleted, err := store.Delete("", "delete-me")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted.Title != "Delete me" {
		t.Errorf("deleted.Title = %q, want 'Delete me'", deleted.Title)
	}

	// Task directory must no longer exist.
	if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
		t.Errorf("task dir %s should have been removed", dirPath)
	}
}

func TestStore_Delete_TaskGoneFromList(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Keep me", "open", "medium", nil)
	createTask(t, store, "20260304-auth", "Delete me", "open", "medium", nil)

	if _, err := store.Delete("", "delete-me"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	tasks, err := store.List(Filter{})
	if err != nil {
		t.Fatalf("List after delete: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task remaining, got %d", len(tasks))
	}
	if tasks[0].Title == "Delete me" {
		t.Error("deleted task still appears in List")
	}
}

func TestStore_Delete_NotFound_ReturnsErrNotFound(t *testing.T) {
	_, store := setupStore(t)
	_, err := store.Delete("", "nonexistent-task")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_Delete_AmbiguousMatch_ReturnsErrAmbiguous(t *testing.T) {
	_, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Ambiguous task", "open", "medium", nil)
	createTask(t, store, "20260305-db", "Ambiguous task", "open", "medium", nil)

	_, err := store.Delete("", "ambiguous-task")
	if !errors.Is(err, ErrAmbiguous) {
		t.Errorf("expected ErrAmbiguous, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// IsBlocked
// ---------------------------------------------------------------------------

func TestIsBlocked_NoDeps_ReturnsFalse(t *testing.T) {
	t1 := &Task{Seq: 1, Status: StatusOpen, DependsOn: nil}
	planTasks := []*Task{t1}
	if IsBlocked(t1, planTasks) {
		t.Error("task with no deps should not be blocked")
	}
}

func TestIsBlocked_DepDone_ReturnsFalse(t *testing.T) {
	dep := &Task{Seq: 1, Status: StatusDone}
	blocked := &Task{Seq: 2, Status: StatusOpen, DependsOn: []int{1}}
	planTasks := []*Task{dep, blocked}
	if IsBlocked(blocked, planTasks) {
		t.Error("task whose dep is done should not be blocked")
	}
}

func TestIsBlocked_DepOpen_ReturnsTrue(t *testing.T) {
	dep := &Task{Seq: 1, Status: StatusOpen}
	blocked := &Task{Seq: 2, Status: StatusOpen, DependsOn: []int{1}}
	planTasks := []*Task{dep, blocked}
	if !IsBlocked(blocked, planTasks) {
		t.Error("task whose dep is open should be blocked")
	}
}

func TestIsBlocked_DepInProgress_ReturnsTrue(t *testing.T) {
	dep := &Task{Seq: 1, Status: StatusInProgress}
	blocked := &Task{Seq: 2, Status: StatusOpen, DependsOn: []int{1}}
	planTasks := []*Task{dep, blocked}
	if !IsBlocked(blocked, planTasks) {
		t.Error("task whose dep is in_progress should be blocked")
	}
}

func TestIsBlocked_UnknownDep_ReturnsTrue(t *testing.T) {
	// Dep seq 99 does not exist in planTasks — treated as blocked.
	blocked := &Task{Seq: 2, Status: StatusOpen, DependsOn: []int{99}}
	if !IsBlocked(blocked, []*Task{blocked}) {
		t.Error("task with unknown dep seq should be blocked")
	}
}

func TestIsBlocked_MultipleDeps_AllDone_ReturnsFalse(t *testing.T) {
	d1 := &Task{Seq: 1, Status: StatusDone}
	d2 := &Task{Seq: 2, Status: StatusDone}
	t3 := &Task{Seq: 3, Status: StatusOpen, DependsOn: []int{1, 2}}
	if IsBlocked(t3, []*Task{d1, d2, t3}) {
		t.Error("all deps done — should not be blocked")
	}
}

func TestIsBlocked_MultipleDeps_OnePending_ReturnsTrue(t *testing.T) {
	d1 := &Task{Seq: 1, Status: StatusDone}
	d2 := &Task{Seq: 2, Status: StatusOpen}
	t3 := &Task{Seq: 3, Status: StatusOpen, DependsOn: []int{1, 2}}
	if !IsBlocked(t3, []*Task{d1, d2, t3}) {
		t.Error("one dep still open — should be blocked")
	}
}

// ---------------------------------------------------------------------------
// CreateWalkthroughScaffold
// ---------------------------------------------------------------------------

func TestCreateWalkthroughScaffold_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("test")
	store := NewStore(dir, &cfg)

	taskDir := filepath.Join(dir, "task-dir")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tk := &Task{Title: "My Task", DirPath: taskDir}
	if err := store.CreateWalkthroughScaffold(tk); err != nil {
		t.Fatalf("CreateWalkthroughScaffold: %v", err)
	}

	path := filepath.Join(taskDir, walkthroughFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "# Walkthrough: My Task") {
		t.Errorf("expected title in scaffold, got: %q", content)
	}
	if !strings.Contains(content, "## What Was Done") {
		t.Error("expected '## What Was Done' section")
	}
	if !strings.Contains(content, "## How It Was Done") {
		t.Error("expected '## How It Was Done' section")
	}
	if !strings.Contains(content, "## Gotchas & Lessons Learned") {
		t.Error("expected '## Gotchas & Lessons Learned' section")
	}
	if !strings.Contains(content, "## Reusable Patterns") {
		t.Error("expected '## Reusable Patterns' section")
	}
}

func TestCreateWalkthroughScaffold_Idempotent(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("test")
	store := NewStore(dir, &cfg)

	taskDir := filepath.Join(dir, "task-dir")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Pre-create with custom content.
	path := filepath.Join(taskDir, walkthroughFileName)
	custom := "# Custom content"
	if err := os.WriteFile(path, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	tk := &Task{Title: "Any Task", DirPath: taskDir}
	if err := store.CreateWalkthroughScaffold(tk); err != nil {
		t.Fatalf("CreateWalkthroughScaffold: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != custom {
		t.Errorf("existing WALKTHROUGH.md was overwritten; got %q", string(data))
	}
}

func TestCreateWalkthroughScaffold_UsesTemplate(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("test")
	store := NewStore(dir, &cfg)

	// Create .logosyncx/templates/walkthrough.md with custom content.
	templatesDir := filepath.Join(dir, ".logosyncx", "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	customTemplate := "## Custom Section\n\n<!-- Custom instructions. -->\n"
	templatePath := filepath.Join(templatesDir, "walkthrough.md")
	if err := os.WriteFile(templatePath, []byte(customTemplate), 0o644); err != nil {
		t.Fatal(err)
	}

	taskDir := filepath.Join(dir, "task-dir")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tk := &Task{Title: "Template Task", DirPath: taskDir}
	if err := store.CreateWalkthroughScaffold(tk); err != nil {
		t.Fatalf("CreateWalkthroughScaffold: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(taskDir, walkthroughFileName))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "# Walkthrough: Template Task") {
		t.Errorf("expected title in scaffold, got: %q", content)
	}
	if !strings.Contains(content, "## Custom Section") {
		t.Errorf("expected custom template section, got: %q", content)
	}
	if strings.Contains(content, "## What Was Done") {
		t.Error("expected default sections to be absent when template exists")
	}
}

func TestCreateWalkthroughScaffold_FallsBackWithoutTemplate(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default("test")
	store := NewStore(dir, &cfg)

	taskDir := filepath.Join(dir, "task-dir")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tk := &Task{Title: "Fallback Task", DirPath: taskDir}
	if err := store.CreateWalkthroughScaffold(tk); err != nil {
		t.Fatalf("CreateWalkthroughScaffold: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(taskDir, walkthroughFileName))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "## What Was Done") {
		t.Error("expected fallback '## What Was Done' section")
	}
	if !strings.Contains(content, "## How It Was Done") {
		t.Error("expected fallback '## How It Was Done' section")
	}
	if !strings.Contains(content, "## Gotchas & Lessons Learned") {
		t.Error("expected fallback '## Gotchas & Lessons Learned' section")
	}
	if !strings.Contains(content, "## Reusable Patterns") {
		t.Error("expected fallback '## Reusable Patterns' section")
	}
}

// ---------------------------------------------------------------------------
// RebuildTaskIndex
// ---------------------------------------------------------------------------

func TestStore_RebuildTaskIndex_EmptyTasks_CreatesEmptyIndex(t *testing.T) {
	dir, store := setupStore(t)
	n, err := store.RebuildTaskIndex()
	if err != nil {
		t.Fatalf("RebuildTaskIndex: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
	if _, err := os.Stat(TaskIndexFilePath(dir)); err != nil {
		t.Errorf("index file should exist after RebuildTaskIndex: %v", err)
	}
}

func TestStore_RebuildTaskIndex_IndexesAllTasks(t *testing.T) {
	dir, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Task A", "open", "medium", nil)
	createTask(t, store, "20260304-auth", "Task B", "done", "high", nil)
	createTask(t, store, "20260305-db", "Task C", "in_progress", "low", nil)

	// Truncate to force full rebuild.
	if err := os.WriteFile(TaskIndexFilePath(dir), []byte{}, 0o644); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	n, err := store.RebuildTaskIndex()
	if err != nil {
		t.Fatalf("RebuildTaskIndex: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 tasks indexed, got %d", n)
	}

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries in index, got %d", len(entries))
	}
}

func TestStore_RebuildTaskIndex_OverwritesExistingIndex(t *testing.T) {
	dir, store := setupStore(t)

	// Write a stale entry directly.
	stale := TaskJSON{ID: "t-stale", Title: "stale task", Tags: []string{}, DependsOn: []int{}}
	if err := AppendTaskIndex(dir, stale); err != nil {
		t.Fatalf("AppendTaskIndex stale: %v", err)
	}

	// Create one real task.
	createTask(t, store, "20260304-auth", "Real task", "open", "medium", nil)

	n, err := store.RebuildTaskIndex()
	if err != nil {
		t.Fatalf("RebuildTaskIndex: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 real task indexed, got %d", n)
	}

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (stale overwritten), got %d", len(entries))
	}
	if entries[0].Title == "stale task" {
		t.Error("stale entry should have been removed")
	}
}

func TestStore_Create_AppendsToIndex(t *testing.T) {
	dir, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Index task", "open", "medium", nil)

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after Create, got %d", len(entries))
	}
	if entries[0].Title != "Index task" {
		t.Errorf("Title = %q, want 'Index task'", entries[0].Title)
	}
}

func TestStore_UpdateFields_RebuildsIndex(t *testing.T) {
	dir, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Update index", "open", "medium", nil)

	if err := store.UpdateFields("", "update-index", map[string]string{"status": "in_progress"}); err != nil {
		t.Fatalf("UpdateFields: %v", err)
	}

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Status != StatusInProgress {
		t.Errorf("Status = %q, want 'in_progress'", entries[0].Status)
	}
}

func TestStore_Delete_RebuildsIndex(t *testing.T) {
	dir, store := setupStore(t)
	createTask(t, store, "20260304-auth", "Keep me", "open", "medium", nil)
	createTask(t, store, "20260304-auth", "Delete me", "open", "medium", nil)

	if _, err := store.Delete("", "delete-me"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	entries, err := ReadAllTaskIndex(dir)
	if err != nil {
		t.Fatalf("ReadAllTaskIndex: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after delete, got %d", len(entries))
	}
	if entries[0].Title == "Delete me" {
		t.Error("deleted task should not appear in index")
	}
}

// ---------------------------------------------------------------------------
// generateID / sortByDateDesc
// ---------------------------------------------------------------------------

func TestGenerateTaskID_HasTPrefix(t *testing.T) {
	id, err := generateID()
	if err != nil {
		t.Fatalf("generateID: %v", err)
	}
	if !strings.HasPrefix(id, "t-") {
		t.Errorf("ID %q should have 't-' prefix", id)
	}
}

func TestGenerateTaskID_CorrectLength(t *testing.T) {
	id, err := generateID()
	if err != nil {
		t.Fatalf("generateID: %v", err)
	}
	// "t-" (2) + 6 hex chars = 8
	if len(id) != 8 {
		t.Errorf("ID length = %d, want 8 (got %q)", len(id), id)
	}
}

func TestGenerateTaskID_IsUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		id, err := generateID()
		if err != nil {
			t.Fatalf("generateID: %v", err)
		}
		if seen[id] {
			t.Errorf("duplicate ID generated: %q", id)
		}
		seen[id] = true
	}
}

func TestSortByDateDesc_Tasks(t *testing.T) {
	older := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	tasks := []*Task{
		{Title: "old", Date: older},
		{Title: "new", Date: newer},
	}
	sortByDateDesc(tasks)
	if tasks[0].Title != "new" {
		t.Errorf("expected 'new' first, got %q", tasks[0].Title)
	}
}

func TestSortByDateDesc_SingleElement(t *testing.T) {
	tasks := []*Task{{Title: "only", Date: time.Now()}}
	sortByDateDesc(tasks) // should not panic
	if len(tasks) != 1 {
		t.Error("single element should survive sort")
	}
}

// ---------------------------------------------------------------------------
// parseSeqPrefix
// ---------------------------------------------------------------------------

func TestParseSeqPrefix_Valid(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"001-add-jwt", 1},
		{"010-refactor", 10},
		{"100-big-task", 100},
		{"002-setup", 2},
	}
	for _, tc := range cases {
		got := parseSeqPrefix(tc.input)
		if got != tc.want {
			t.Errorf("parseSeqPrefix(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestParseSeqPrefix_Invalid(t *testing.T) {
	cases := []string{"no-prefix", "-no-leading", "abc-task", ""}
	for _, s := range cases {
		if n := parseSeqPrefix(s); n != 0 {
			t.Errorf("parseSeqPrefix(%q) = %d, want 0", s, n)
		}
	}
}

// --- depends_on seq validation (§8.4) ----------------------------------------

func TestCreate_DependsOn_NonExistentSeq_Error(t *testing.T) {
	dir, store := setupStore(t)
	_ = dir

	// No tasks exist yet; seq 1 does not exist.
	tk := &Task{
		Title:     "Dependent task",
		Plan:      "test-plan",
		DependsOn: []int{1},
	}
	_, err := store.Create(tk)
	if err == nil {
		t.Fatal("expected error for non-existent depends_on seq, got nil")
	}
	if !strings.Contains(err.Error(), "seq 1") {
		t.Errorf("expected 'seq 1' in error message, got: %v", err)
	}
}

func TestCreate_DependsOn_ValidSeq_Succeeds(t *testing.T) {
	dir, store := setupStore(t)
	_ = dir

	// Create task with seq 1.
	createTask(t, store, "test-plan", "First task", "open", "medium", nil)

	// Now create task depending on seq 1 — should succeed.
	tk := &Task{
		Title:     "Second task",
		Plan:      "test-plan",
		DependsOn: []int{1},
	}
	_, err := store.Create(tk)
	if err != nil {
		t.Errorf("expected no error for valid depends_on seq, got: %v", err)
	}
}
