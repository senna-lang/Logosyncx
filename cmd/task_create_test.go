package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/senna-lang/logosyncx/internal/task"
)

// --- flag-based task create --------------------------------------------------

func TestTaskCreate_TitleOnly(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate("", "My new task", "", "medium", nil); err != nil {
		t.Fatalf("runTaskCreate with --title failed: %v", err)
	}

	tasks := loadAllOpenTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Title != "My new task" {
		t.Errorf("title = %q, want 'My new task'", tasks[0].Title)
	}
}

func TestTaskCreate_AllFields(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate("", "Full flag task", "This task was created with all flags.", "high", []string{"go", "cli"}); err != nil {
		t.Fatalf("runTaskCreate with all flags failed: %v", err)
	}

	tasks := loadAllOpenTasks(t, dir)
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
	if !strings.Contains(tk.Body, "This task was created with all flags.") {
		t.Errorf("body does not contain description, got: %q", tk.Body)
	}
}

func TestTaskCreate_DescriptionInWhatSection(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate("", "Task with description", "Implement the thing.", "medium", nil); err != nil {
		t.Fatalf("runTaskCreate failed: %v", err)
	}

	tasks := loadAllOpenTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	body := tasks[0].Body
	if !strings.Contains(body, "## What") {
		t.Errorf("expected '## What' heading in body, got: %q", body)
	}
	if !strings.Contains(body, "Implement the thing.") {
		t.Errorf("expected description in body, got: %q", body)
	}
}

func TestTaskCreate_EmptyDescriptionProducesWhatSection(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate("", "No description task", "", "low", nil); err != nil {
		t.Fatalf("runTaskCreate failed: %v", err)
	}

	tasks := loadAllOpenTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if !strings.Contains(tasks[0].Body, "## What") {
		t.Errorf("expected '## What' heading even with no description, got: %q", tasks[0].Body)
	}
}

func TestTaskCreate_DefaultPriorityIsMedium(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate("", "Default priority task", "", "medium", nil); err != nil {
		t.Fatalf("runTaskCreate failed: %v", err)
	}

	tasks := loadAllOpenTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Priority != task.PriorityMedium {
		t.Errorf("priority = %q, want 'medium'", tasks[0].Priority)
	}
}

func TestTaskCreate_AutoFillsIDAndDate(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate("", "Autofill test task", "", "medium", nil); err != nil {
		t.Fatalf("runTaskCreate failed: %v", err)
	}

	tasks := loadAllOpenTasks(t, dir)
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

	if err := runTaskCreate("", "Status test task", "", "medium", nil); err != nil {
		t.Fatalf("runTaskCreate failed: %v", err)
	}

	tasks := loadAllOpenTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Status != task.StatusOpen {
		t.Errorf("status = %q, want 'open'", tasks[0].Status)
	}
}

func TestTaskCreate_ErrorOnInvalidPriority(t *testing.T) {
	setupInitedProject(t)

	err := runTaskCreate("", "Bad priority task", "", "urgent", nil)
	if err == nil {
		t.Fatal("expected error for invalid priority, got nil")
	}
	if !strings.Contains(err.Error(), "priority") {
		t.Errorf("expected 'priority' in error message, got: %v", err)
	}
}

func TestTaskCreate_ErrorWhenNoTitleProvided(t *testing.T) {
	err := runTaskCreate("", "", "", "medium", nil)
	if err == nil {
		t.Fatal("expected error when no title provided, got nil")
	}
	if !strings.Contains(err.Error(), "--title") {
		t.Errorf("expected '--title' in error message, got: %v", err)
	}
}

// --- helpers -----------------------------------------------------------------

// loadAllOpenTasks reads all task files from .logosyncx/tasks/open/.
func loadAllOpenTasks(t *testing.T, projectRoot string) []task.Task {
	t.Helper()
	dir := projectRoot + "/.logosyncx/tasks/open"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir tasks/open: %v", err)
	}

	var tasks []task.Task
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(dir + "/" + e.Name())
		if err != nil {
			t.Fatalf("ReadFile %s: %v", e.Name(), err)
		}
		tk, err := task.Parse(e.Name(), data)
		if err != nil {
			t.Fatalf("task.Parse %s: %v", e.Name(), err)
		}
		tasks = append(tasks, tk)
	}
	return tasks
}
