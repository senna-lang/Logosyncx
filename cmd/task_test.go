package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/senna-lang/logosyncx/internal/task"
	"github.com/senna-lang/logosyncx/pkg/config"
)

// testPlan2 is a second plan slug used in plan-filter tests.
const testPlan2 = "20260202-other-plan"

// captureStdout redirects os.Stdout to a buffer for the duration of fn and
// returns the captured output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

// helperRebuildIndex rebuilds the task index for the test project so that
// derived fields such as Blocked are computed from the current task files.
func helperRebuildIndex(t *testing.T, root string) {
	t.Helper()
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	store := task.NewStore(root, &cfg)
	if _, err := store.RebuildTaskIndex(); err != nil {
		t.Fatalf("RebuildTaskIndex: %v", err)
	}
}

// --- task create -------------------------------------------------------------

func TestTaskCreate_AutoAssignsSeq(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "Alpha task", "medium", nil, nil); err != nil {
		t.Fatalf("create first: %v", err)
	}
	if err := runTaskCreate(dir, testPlan, "Beta task", "medium", nil, nil); err != nil {
		t.Fatalf("create second: %v", err)
	}

	tasks := loadAllTasks(t, dir)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	seqs := []int{tasks[0].Seq, tasks[1].Seq}
	sort.Ints(seqs)
	if seqs[0] != 1 || seqs[1] != 2 {
		t.Errorf("expected seqs [1 2], got %v", seqs)
	}
}

func TestTaskCreate_PrintsRelativePath(t *testing.T) {
	dir := setupInitedProject(t)

	out := captureStdout(t, func() {
		if err := runTaskCreate(dir, testPlan, "Path check", "medium", nil, nil); err != nil {
			t.Fatalf("create task: %v", err)
		}
	})

	if !strings.Contains(out, ".logosyncx/tasks/") {
		t.Errorf("expected relative path containing '.logosyncx/tasks/' in output, got:\n%s", out)
	}
}

// --- task update -------------------------------------------------------------

func TestTaskUpdate_Done_CreatesWalkthrough(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "Walkthrough task", "medium", nil, nil); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := runTaskUpdate("", "walkthrough-task", "done", "", ""); err != nil {
		t.Fatalf("update to done: %v", err)
	}

	tasks := loadAllTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	wtPath := filepath.Join(tasks[0].DirPath, "WALKTHROUGH.md")
	if _, err := os.Stat(wtPath); err != nil {
		t.Errorf("WALKTHROUGH.md not created after marking done: %v", err)
	}
}

func TestTaskUpdate_NoFileMove(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "Stable path task", "medium", nil, nil); err != nil {
		t.Fatalf("create task: %v", err)
	}

	tasks := loadAllTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	originalDir := tasks[0].DirPath

	if err := runTaskUpdate("", "stable-path", "in_progress", "", ""); err != nil {
		t.Fatalf("update to in_progress: %v", err)
	}

	if _, err := os.Stat(filepath.Join(originalDir, "TASK.md")); err != nil {
		t.Errorf("TASK.md not at original dir after update: %v", err)
	}
}

func TestTaskUpdate_InProgress_BlockedByDep(t *testing.T) {
	dir := setupInitedProject(t)

	// Create task 1 (no deps) — remains open.
	if err := runTaskCreate(dir, testPlan, "Prereq task", "medium", nil, nil); err != nil {
		t.Fatalf("create prereq: %v", err)
	}
	// Create task 2 that depends on task 1 (which is still open).
	if err := runTaskCreate(dir, testPlan, "Dependent task", "medium", nil, []int{1}); err != nil {
		t.Fatalf("create dependent: %v", err)
	}

	err := runTaskUpdate("", "dependent-task", "in_progress", "", "")
	if err == nil {
		t.Fatal("expected error when moving blocked task to in_progress, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "block") &&
		!strings.Contains(strings.ToLower(err.Error()), "depend") {
		t.Errorf("expected 'blocked' or 'depend' in error, got: %v", err)
	}
}

// --- task ls -----------------------------------------------------------------

func TestTaskLS_PlanFilter(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "Plan one task", "medium", nil, nil); err != nil {
		t.Fatalf("create plan1 task: %v", err)
	}
	if err := runTaskCreate(dir, testPlan2, "Plan two task", "medium", nil, nil); err != nil {
		t.Fatalf("create plan2 task: %v", err)
	}
	helperRebuildIndex(t, dir)

	out := captureStdout(t, func() {
		if err := runTaskLS(testPlan, "", "", "", false, false); err != nil {
			t.Fatalf("runTaskLS with plan filter: %v", err)
		}
	})

	if !strings.Contains(out, "Plan one task") {
		t.Errorf("expected 'Plan one task' in output, got:\n%s", out)
	}
	if strings.Contains(out, "Plan two task") {
		t.Errorf("unexpected 'Plan two task' in filtered output, got:\n%s", out)
	}
}

func TestTaskLS_Blocked(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "Unblocked task", "medium", nil, nil); err != nil {
		t.Fatalf("create unblocked: %v", err)
	}
	if err := runTaskCreate(dir, testPlan, "Blocked task", "medium", nil, []int{1}); err != nil {
		t.Fatalf("create blocked: %v", err)
	}
	// Rebuild so Blocked field is computed in the index.
	helperRebuildIndex(t, dir)

	out := captureStdout(t, func() {
		if err := runTaskLS("", "", "", "", false, true); err != nil {
			t.Fatalf("runTaskLS --blocked: %v", err)
		}
	})

	if !strings.Contains(out, "Blocked task") {
		t.Errorf("expected 'Blocked task' in --blocked output, got:\n%s", out)
	}
	if strings.Contains(out, "Unblocked task") {
		t.Errorf("unexpected 'Unblocked task' in --blocked output, got:\n%s", out)
	}
}

func TestTaskLS_JSON_IncludesBlockedField(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "JSON field task", "medium", nil, nil); err != nil {
		t.Fatalf("create task: %v", err)
	}
	helperRebuildIndex(t, dir)

	out := captureStdout(t, func() {
		if err := runTaskLS("", "", "", "", true, false); err != nil {
			t.Fatalf("runTaskLS --json: %v", err)
		}
	})

	var entries []map[string]any
	if err := json.Unmarshal([]byte(out), &entries); err != nil {
		t.Fatalf("unmarshal JSON output: %v\noutput: %s", err, out)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least 1 JSON entry")
	}
	if _, ok := entries[0]["blocked"]; !ok {
		t.Errorf("JSON output missing 'blocked' field; got keys: %v", entries[0])
	}
}

// --- task refer --------------------------------------------------------------

func TestTaskRefer_Disambiguate_WithPlan(t *testing.T) {
	dir := setupInitedProject(t)

	// Create tasks with the same title stem in two different plans.
	if err := runTaskCreate(dir, testPlan, "Shared name task", "medium", nil, nil); err != nil {
		t.Fatalf("create plan1 task: %v", err)
	}
	if err := runTaskCreate(dir, testPlan2, "Shared name task", "medium", nil, nil); err != nil {
		t.Fatalf("create plan2 task: %v", err)
	}

	// Without --plan filter: ambiguous → error.
	err := runTaskRefer("shared-name", "", false)
	if err == nil {
		t.Fatal("expected ambiguity error when two tasks match without --plan filter")
	}

	// With --plan filter: resolves to exactly one.
	err = runTaskRefer("shared-name", testPlan, false)
	if err != nil {
		t.Errorf("expected no error with --plan filter, got: %v", err)
	}
}

// --- task delete -------------------------------------------------------------

func TestTaskDelete_RemovesDir(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "Delete me task", "medium", nil, nil); err != nil {
		t.Fatalf("create task: %v", err)
	}

	tasks := loadAllTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task before delete, got %d", len(tasks))
	}
	taskDir := tasks[0].DirPath

	if err := runTaskDelete("", "delete-me", true); err != nil {
		t.Fatalf("delete --force: %v", err)
	}

	if _, err := os.Stat(taskDir); !os.IsNotExist(err) {
		t.Errorf("expected task dir to be removed, stat err: %v", err)
	}
}

func TestTaskDelete_Force_SkipsPrompt(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "Force delete task", "medium", nil, nil); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// --force should not read from stdin, so no stdin setup needed.
	if err := runTaskDelete("", "force-delete", true); err != nil {
		t.Fatalf("expected no error with --force, got: %v", err)
	}

	remaining := loadAllTasks(t, dir)
	if len(remaining) != 0 {
		t.Errorf("expected 0 tasks after forced delete, got %d", len(remaining))
	}
}

// --- task search -------------------------------------------------------------

func TestTaskSearch_PlanFilter(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "Auth refactor task", "medium", nil, nil); err != nil {
		t.Fatalf("create plan1 task: %v", err)
	}
	if err := runTaskCreate(dir, testPlan2, "Auth review task", "medium", nil, nil); err != nil {
		t.Fatalf("create plan2 task: %v", err)
	}

	out := captureStdout(t, func() {
		if err := runTaskSearch("auth", testPlan, "", ""); err != nil {
			t.Fatalf("runTaskSearch with plan filter: %v", err)
		}
	})

	if !strings.Contains(out, "Auth refactor task") {
		t.Errorf("expected 'Auth refactor task' in output, got:\n%s", out)
	}
	if strings.Contains(out, "Auth review task") {
		t.Errorf("unexpected 'Auth review task' in filtered output, got:\n%s", out)
	}
}

// --- task walkthrough --------------------------------------------------------

func TestTaskWalkthrough_FillStatusDetection(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name     string
		content  string
		wantStat string
	}{
		{
			name:     "no file",
			content:  "",
			wantStat: "-",
		},
		{
			name:     "scaffold only — headings and comments",
			content:  "# Section\n<!-- comment -->\n",
			wantStat: "[scaffold only]",
		},
		{
			name:     "filled — real content",
			content:  "# Section\nSome real content here.\n",
			wantStat: "[filled]",
		},
		{
			name:     "filled — content after multi-line comment",
			content:  "<!--\nmulti\nline\n-->\nActual content.\n",
			wantStat: "[filled]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".md")
			if tc.content != "" {
				if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
					t.Fatalf("write file: %v", err)
				}
			}
			got := walkthroughFillStatus(path)
			if got != tc.wantStat {
				t.Errorf("walkthroughFillStatus(%q) = %q, want %q", tc.name, got, tc.wantStat)
			}
		})
	}
}

func TestTaskWalkthrough_ListMode(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "List walk task", "medium", nil, nil); err != nil {
		t.Fatalf("create task: %v", err)
	}

	out := captureStdout(t, func() {
		if err := runTaskWalkthrough(testPlan, ""); err != nil {
			t.Fatalf("runTaskWalkthrough list mode: %v", err)
		}
	})

	if !strings.Contains(out, "List walk task") {
		t.Errorf("expected task title in walkthrough list, got:\n%s", out)
	}
	if !strings.Contains(out, "WALKTHROUGH") {
		t.Errorf("expected WALKTHROUGH header in list output, got:\n%s", out)
	}
	// No WALKTHROUGH.md yet → status should be "-".
	if !strings.Contains(out, "-") {
		t.Errorf("expected '-' status for task without WALKTHROUGH.md, got:\n%s", out)
	}
}

func TestTaskWalkthrough_PrintContent(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runTaskCreate(dir, testPlan, "Print walk task", "medium", nil, nil); err != nil {
		t.Fatalf("create task: %v", err)
	}
	// Mark done so WALKTHROUGH.md is created.
	if err := runTaskUpdate("", "print-walk-task", "done", "", ""); err != nil {
		t.Fatalf("update to done: %v", err)
	}

	// Write real content to WALKTHROUGH.md so it can be printed.
	tasks := loadAllTasks(t, dir)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	wtPath := filepath.Join(tasks[0].DirPath, "WALKTHROUGH.md")
	content := "# What I did\nFixed the bug by refactoring.\n"
	if err := os.WriteFile(wtPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write WALKTHROUGH.md: %v", err)
	}

	out := captureStdout(t, func() {
		if err := runTaskWalkthrough(testPlan, "print-walk"); err != nil {
			t.Fatalf("runTaskWalkthrough print mode: %v", err)
		}
	})

	if !strings.Contains(out, "Fixed the bug") {
		t.Errorf("expected walkthrough content in output, got:\n%s", out)
	}
}
