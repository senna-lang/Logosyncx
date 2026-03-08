package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/senna-lang/logosyncx/internal/task"
	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/knowledge"
	"github.com/senna-lang/logosyncx/pkg/plan"
)

// --- helpers -----------------------------------------------------------------

// setupDistillProject creates an inited project with one plan and tasks that
// are all marked done with WALKTHROUGH.md files, ready for distillation.
func setupDistillProject(t *testing.T, topic string) (root, planSlug string) {
	t.Helper()

	date := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	p := plan.Plan{
		ID:    "p-test01",
		Date:  &date,
		Topic: topic,
		Tags:  []string{"go"},
		Body:  "## Background\nThis is the plan background.\n",
	}
	root = setupProjectWithPlan(t, p)
	planSlug = strings.TrimSuffix(plan.FileName(p), ".md")

	// Create a task, mark it done (creates WALKTHROUGH.md), then fill it.
	if err := runTaskCreate(root, planSlug, "Test task one", "medium", nil, nil); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := runTaskUpdate("", "test-task-one", "done", "", ""); err != nil {
		t.Fatalf("update task to done: %v", err)
	}

	// Write real content into WALKTHROUGH.md.
	cfg, err := config.Load(root)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	store := task.NewStore(root, &cfg)
	tasks, err := store.List(task.Filter{Plan: planSlug})
	if err != nil || len(tasks) == 0 {
		t.Fatalf("list tasks: %v", err)
	}
	wtPath := filepath.Join(tasks[0].DirPath, "WALKTHROUGH.md")
	if err := os.WriteFile(wtPath, []byte("# What I did\nImplemented the feature.\n"), 0o644); err != nil {
		t.Fatalf("write WALKTHROUGH.md: %v", err)
	}

	return root, planSlug
}

// --- distill tests -----------------------------------------------------------

func TestDistill_CreatesKnowledgeFile(t *testing.T) {
	root, planSlug := setupDistillProject(t, "Auth Refactor")

	if err := runDistill(planSlug, false, false); err != nil {
		t.Fatalf("runDistill: %v", err)
	}

	knDir := knowledge.KnowledgeDir(root)
	entries, err := os.ReadDir(knDir)
	if err != nil {
		t.Fatalf("read knowledge dir: %v", err)
	}
	var kFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			kFiles = append(kFiles, e.Name())
		}
	}
	if len(kFiles) == 0 {
		t.Fatal("expected a knowledge .md file to be created")
	}
}

func TestDistill_SetsDistilledTrue(t *testing.T) {
	root, planSlug := setupDistillProject(t, "Distilled Flag Plan")

	if err := runDistill(planSlug, false, false); err != nil {
		t.Fatalf("runDistill: %v", err)
	}

	plans, err := plan.LoadAll(root)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	var found bool
	for _, p := range plans {
		if strings.Contains(p.Filename, "distilled-flag-plan") {
			if !p.Distilled {
				t.Errorf("plan.Distilled = false after distill")
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("plan not found after distill")
	}
}

func TestDistill_AlreadyDistilled_Error(t *testing.T) {
	_, planSlug := setupDistillProject(t, "Already Done Plan")

	// First distill succeeds.
	if err := runDistill(planSlug, false, false); err != nil {
		t.Fatalf("first runDistill: %v", err)
	}

	// Second distill without --force should fail.
	err := runDistill(planSlug, false, false)
	if err == nil {
		t.Fatal("expected error when distilling already-distilled plan without --force")
	}
	if !strings.Contains(err.Error(), "already distilled") {
		t.Errorf("expected 'already distilled' in error, got: %v", err)
	}
}

func TestDistill_Force_OverridesAlreadyDistilled(t *testing.T) {
	_, planSlug := setupDistillProject(t, "Force Re Distill Plan")

	if err := runDistill(planSlug, false, false); err != nil {
		t.Fatalf("first runDistill: %v", err)
	}

	// Second distill with --force should succeed.
	if err := runDistill(planSlug, true, false); err != nil {
		t.Errorf("runDistill --force on already-distilled plan: %v", err)
	}
}

func TestDistill_DryRun_NoWrite(t *testing.T) {
	root, planSlug := setupDistillProject(t, "Dry Run Plan")

	knDir := knowledge.KnowledgeDir(root)
	entriesBefore, _ := os.ReadDir(knDir)

	out := captureStdout(t, func() {
		if err := runDistill(planSlug, false, true); err != nil {
			t.Fatalf("runDistill --dry-run: %v", err)
		}
	})

	// Should print preview.
	if !strings.Contains(out, "DRY RUN") {
		t.Errorf("expected DRY RUN in output, got:\n%s", out)
	}

	// Knowledge file must NOT have been written.
	entriesAfter, _ := os.ReadDir(knDir)
	if len(entriesAfter) > len(entriesBefore) {
		t.Errorf("expected no new knowledge files in dry-run mode")
	}

	// Plan must NOT be marked distilled.
	plans, _ := plan.LoadAll(root)
	for _, p := range plans {
		if p.Distilled {
			t.Errorf("plan marked distilled despite --dry-run")
		}
	}
}

func TestDistill_IncompleteTasks_Error(t *testing.T) {
	date := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	p := plan.Plan{
		ID:    "p-inc01",
		Date:  &date,
		Topic: "Incomplete Tasks Plan",
		Tags:  []string{},
		Body:  "## Background\nTest.\n",
	}
	root := setupProjectWithPlan(t, p)
	planSlug := strings.TrimSuffix(plan.FileName(p), ".md")

	// Create a task but do NOT mark it done.
	if err := runTaskCreate(root, planSlug, "Open task", "medium", nil, nil); err != nil {
		t.Fatalf("create task: %v", err)
	}

	err := runDistill(planSlug, false, false)
	if err == nil {
		t.Fatal("expected error for incomplete tasks, got nil")
	}
	if !strings.Contains(err.Error(), "incomplete tasks") {
		t.Errorf("expected 'incomplete tasks' in error, got: %v", err)
	}
}

func TestDistill_NoWalkthroughs_Error(t *testing.T) {
	date := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	p := plan.Plan{
		ID:    "p-nowt01",
		Date:  &date,
		Topic: "No Walkthroughs Plan",
		Tags:  []string{},
		Body:  "## Background\nTest.\n",
	}
	root := setupProjectWithPlan(t, p)
	planSlug := strings.TrimSuffix(plan.FileName(p), ".md")

	// Create and mark done (creates WALKTHROUGH.md scaffold), then remove it.
	if err := runTaskCreate(root, planSlug, "Done task", "medium", nil, nil); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := runTaskUpdate("", "done-task", "done", "", ""); err != nil {
		t.Fatalf("update task to done: %v", err)
	}

	// Remove the WALKTHROUGH.md that was auto-created.
	cfg, _ := config.Load(root)
	store := task.NewStore(root, &cfg)
	tasks, _ := store.List(task.Filter{Plan: planSlug})
	if len(tasks) > 0 {
		_ = os.Remove(filepath.Join(tasks[0].DirPath, "WALKTHROUGH.md"))
	}

	err := runDistill(planSlug, false, false)
	if err == nil {
		t.Fatal("expected error when no walkthroughs exist, got nil")
	}
	if !strings.Contains(err.Error(), "no walkthroughs") {
		t.Errorf("expected 'no walkthroughs' in error, got: %v", err)
	}
}

// --- Source Walkthroughs paths (§10.5) ---------------------------------------

func TestDistill_KnowledgeFile_ContainsWalkthroughPaths(t *testing.T) {
	root, planSlug := setupDistillProject(t, "Walkthrough Paths Plan")

	if err := runDistill(planSlug, false, false); err != nil {
		t.Fatalf("runDistill: %v", err)
	}

	knDir := knowledge.KnowledgeDir(root)
	entries, err := os.ReadDir(knDir)
	if err != nil {
		t.Fatalf("read knowledge dir: %v", err)
	}
	var content []byte
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") {
			p := filepath.Join(knDir, e.Name())
			content, err = os.ReadFile(p)
			if err != nil {
				t.Fatalf("read knowledge file: %v", err)
			}
			break
		}
	}
	if len(content) == 0 {
		t.Fatal("knowledge file is empty")
	}
	body := string(content)
	if !strings.Contains(body, "WALKTHROUGH.md") {
		t.Errorf("expected WALKTHROUGH.md path in knowledge file, got:\n%s", body)
	}
	if !strings.Contains(body, ".logosyncx/tasks/") {
		t.Errorf("expected '.logosyncx/tasks/' path prefix in knowledge file, got:\n%s", body)
	}
}
