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

	if err := runTaskCreate("", "My new task", "medium", nil, nil); err != nil {
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

func TestTaskCreate_AllFrontmatterFields(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate("", "Full flag task", "high", []string{"go", "cli"}, nil); err != nil {
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
}

func TestTaskCreate_DefaultPriorityIsMedium(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate("", "Default priority task", "medium", nil, nil); err != nil {
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

	if err := runTaskCreate("", "Autofill test task", "medium", nil, nil); err != nil {
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

	if err := runTaskCreate("", "Status test task", "medium", nil, nil); err != nil {
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

	err := runTaskCreate("", "Bad priority task", "urgent", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid priority, got nil")
	}
	if !strings.Contains(err.Error(), "priority") {
		t.Errorf("expected 'priority' in error message, got: %v", err)
	}
}

func TestTaskCreate_ErrorWhenNoTitleProvided(t *testing.T) {
	err := runTaskCreate("", "", "medium", nil, nil)
	if err == nil {
		t.Fatal("expected error when no title provided, got nil")
	}
	if !strings.Contains(err.Error(), "--title") {
		t.Errorf("expected '--title' in error message, got: %v", err)
	}
}

// --- --section flag: valid usage ---------------------------------------------

func TestTaskCreate_SectionFlag_WhatSection(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate("", "Task with what", "medium", nil, []string{"What=Implement the thing."}); err != nil {
		t.Fatalf("runTaskCreate with --section What failed: %v", err)
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
		t.Errorf("expected section content in body, got: %q", body)
	}
}

func TestTaskCreate_SectionFlag_MultipleSections(t *testing.T) {
	dir := setupInitedProject(t)

	sections := []string{"What=Do the thing.", "Why=Because it matters."}
	if err := runTaskCreate("", "Multi-section task", "medium", nil, sections); err != nil {
		t.Fatalf("runTaskCreate with multiple --section failed: %v", err)
	}

	tasks := loadAllOpenTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	body := tasks[0].Body
	if !strings.Contains(body, "## What") {
		t.Errorf("expected '## What' in body, got: %q", body)
	}
	if !strings.Contains(body, "## Why") {
		t.Errorf("expected '## Why' in body, got: %q", body)
	}
	if !strings.Contains(body, "Because it matters.") {
		t.Errorf("expected Why content in body, got: %q", body)
	}
}

func TestTaskCreate_SectionFlag_EmptySectionsProducesEmptyBody(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate("", "Empty body task", "medium", nil, nil); err != nil {
		t.Fatalf("runTaskCreate with no sections failed: %v", err)
	}

	tasks := loadAllOpenTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if strings.TrimSpace(tasks[0].Body) != "" {
		t.Errorf("expected empty body when no --section provided, got: %q", tasks[0].Body)
	}
}

func TestTaskCreate_SectionFlag_OutputOrderFollowsConfig(t *testing.T) {
	dir := setupInitedProject(t)

	// Provide sections in reverse config order (Why before What).
	// Output must follow config definition order.
	sections := []string{"Why=Because.", "What=Do it."}
	if err := runTaskCreate("", "Ordered task", "medium", nil, sections); err != nil {
		t.Fatalf("runTaskCreate failed: %v", err)
	}

	tasks := loadAllOpenTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	body := tasks[0].Body
	whatIdx := strings.Index(body, "## What")
	whyIdx := strings.Index(body, "## Why")
	if whatIdx == -1 || whyIdx == -1 {
		t.Fatalf("expected both sections in body, got: %q", body)
	}
	if whatIdx > whyIdx {
		t.Errorf("expected What before Why (config order), got body: %q", body)
	}
}

// --- --section flag: error cases ---------------------------------------------

func TestTaskCreate_SectionFlag_UnknownSection_ReturnsError(t *testing.T) {
	setupInitedProject(t)

	err := runTaskCreate("", "Bad section task", "medium", nil, []string{"UnknownSection=text"})
	if err == nil {
		t.Fatal("expected error for unknown --section name, got nil")
	}
	if !strings.Contains(err.Error(), "UnknownSection") {
		t.Errorf("expected unknown section name in error, got: %v", err)
	}
}

func TestTaskCreate_SectionFlag_UnknownSection_ListsAllowed(t *testing.T) {
	setupInitedProject(t)

	err := runTaskCreate("", "Bad section task", "medium", nil, []string{"BadSection=text"})
	if err == nil {
		t.Fatal("expected error for unknown --section name, got nil")
	}
	// Error message should list allowed section names.
	if !strings.Contains(err.Error(), "What") {
		t.Errorf("expected allowed section names in error, got: %v", err)
	}
}

func TestTaskCreate_SectionFlag_InvalidFormat_ReturnsError(t *testing.T) {
	setupInitedProject(t)

	err := runTaskCreate("", "Bad format task", "medium", nil, []string{"NoEqualsSign"})
	if err == nil {
		t.Fatal("expected error for bad --section format, got nil")
	}
	if !strings.Contains(err.Error(), "Name=content") {
		t.Errorf("expected format hint in error, got: %v", err)
	}
}

func TestTaskCreate_SectionFlag_DuplicateSection_ReturnsError(t *testing.T) {
	setupInitedProject(t)

	err := runTaskCreate("", "Dup section task", "medium", nil, []string{"What=first", "What=second"})
	if err == nil {
		t.Fatal("expected error for duplicate --section name, got nil")
	}
	if !strings.Contains(err.Error(), "more than once") {
		t.Errorf("expected 'more than once' in error, got: %v", err)
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
