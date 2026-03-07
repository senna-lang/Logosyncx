package cmd

import (
	"strings"
	"testing"

	"github.com/senna-lang/logosyncx/internal/task"
	"github.com/senna-lang/logosyncx/pkg/config"
)

// --- helpers -----------------------------------------------------------------

const testPlan = "20260101-test-plan"

// loadAllTasks loads all tasks via Store.List (works with the flat layout).
func loadAllTasks(t *testing.T, projectRoot string) []*task.Task {
	t.Helper()
	cfg, err := config.Load(projectRoot)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	store := task.NewStore(projectRoot, &cfg)
	tasks, err := store.List(task.Filter{})
	if err != nil {
		t.Fatalf("store.List: %v", err)
	}
	return tasks
}

// --- flag-based task create --------------------------------------------------

func TestTaskCreate_TitleOnly(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "My new task", "medium", nil, nil); err != nil {
		t.Fatalf("runTaskCreate with --title failed: %v", err)
	}

	tasks := loadAllTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Title != "My new task" {
		t.Errorf("title = %q, want 'My new task'", tasks[0].Title)
	}
}

func TestTaskCreate_AllFrontmatterFields(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "Full flag task", "high", []string{"go", "cli"}, nil); err != nil {
		t.Fatalf("runTaskCreate with all flags failed: %v", err)
	}

	tasks := loadAllTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	tk := tasks[0]
	if tk.Title != "Full flag task" {
		t.Errorf("title = %q, want 'Full flag task'", tk.Title)
	}
	if tk.Priority != task.PriorityHigh {
		t.Errorf("priority = %q, want 'high'", tk.Priority)
	}
	if len(tk.Tags) != 2 || tk.Tags[0] != "go" || tk.Tags[1] != "cli" {
		t.Errorf("tags = %v, want [go cli]", tk.Tags)
	}
}

func TestTaskCreate_DefaultPriorityIsMedium(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "Default priority task", "medium", nil, nil); err != nil {
		t.Fatalf("runTaskCreate failed: %v", err)
	}

	tasks := loadAllTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Priority != task.PriorityMedium {
		t.Errorf("priority = %q, want 'medium'", tasks[0].Priority)
	}
}

func TestTaskCreate_AutoFillsIDAndDate(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "Autofill test task", "medium", nil, nil); err != nil {
		t.Fatalf("runTaskCreate failed: %v", err)
	}

	tasks := loadAllTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	tk := tasks[0]
	if tk.ID == "" {
		t.Error("expected ID to be auto-filled, got empty string")
	}
	if tk.Date.IsZero() {
		t.Error("expected Date to be auto-filled, got zero value")
	}
}

func TestTaskCreate_DefaultStatusIsOpen(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "Status test task", "medium", nil, nil); err != nil {
		t.Fatalf("runTaskCreate failed: %v", err)
	}

	tasks := loadAllTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Status != task.StatusOpen {
		t.Errorf("status = %q, want 'open'", tasks[0].Status)
	}
}

func TestTaskCreate_ErrorOnInvalidPriority(t *testing.T) {
	dir := setupInitedProject(t)

	err := runTaskCreate(dir, testPlan, "Bad priority task", "urgent", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid priority, got nil")
	}
	if !strings.Contains(err.Error(), "priority") {
		t.Errorf("expected 'priority' in error message, got: %v", err)
	}
}

func TestTaskCreate_ErrorWhenNoTitleProvided(t *testing.T) {
	dir := setupInitedProject(t)

	// runTaskCreate bypasses cobra flag validation, so store returns its own
	// error. We check for the word "title" (not the cobra flag name "--title").
	err := runTaskCreate(dir, testPlan, "", "medium", nil, nil)
	if err == nil {
		t.Fatal("expected error when no title provided, got nil")
	}
	if !strings.Contains(err.Error(), "title") {
		t.Errorf("expected 'title' in error message, got: %v", err)
	}
}

func TestTaskCreate_ErrorWhenNoPlanProvided(t *testing.T) {
	dir := setupInitedProject(t)

	err := runTaskCreate(dir, "", "Some task", "medium", nil, nil)
	if err == nil {
		t.Fatal("expected error when no plan provided, got nil")
	}
	if !strings.Contains(err.Error(), "plan") {
		t.Errorf("expected 'plan' in error message, got: %v", err)
	}
}

func TestTaskCreate_PlanGroupDirIsCreated(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "Dir check task", "medium", nil, nil); err != nil {
		t.Fatalf("runTaskCreate failed: %v", err)
	}

	tasks := loadAllTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Plan != testPlan {
		t.Errorf("plan = %q, want %q", tasks[0].Plan, testPlan)
	}
}
