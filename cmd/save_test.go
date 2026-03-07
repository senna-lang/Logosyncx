package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/senna-lang/logosyncx/pkg/plan"
)

// --- helpers -----------------------------------------------------------------

func setupInitedProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := runInit(); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	return dir
}

// --- flag validation ---------------------------------------------------------

func TestSave_ErrorWhenNoTopicProvided(t *testing.T) {
	err := runSave("", nil, "", nil, nil)
	if err == nil {
		t.Fatal("expected error when no topic provided, got nil")
	}
	if !strings.Contains(err.Error(), "--topic") {
		t.Errorf("expected error to mention --topic, got: %v", err)
	}
}

func TestSave_ErrorWhenNotInitialized(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(orig) })

	err := runSave("no-init", nil, "", nil, nil)
	if err == nil {
		t.Fatal("expected error when project not initialized, got nil")
	}
	if !strings.Contains(err.Error(), "logos init") {
		t.Errorf("expected 'logos init' hint in error, got: %v", err)
	}
}

// --- plan creation -----------------------------------------------------------

func TestSave_CreatesInPlansDir(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runSave("test topic", nil, "", nil, nil); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}

	plans, err := plan.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan in plans/, got %d", len(plans))
	}
}

func TestSave_FileNameFormat_YYYYMMDD(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runSave("filename format", nil, "", nil, nil); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}

	plansDir := filepath.Join(dir, ".logosyncx", "plans")
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	// Filter out archive/ subdir.
	var files []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e)
		}
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file in plans/, got %d", len(files))
	}
	name := files[0].Name()
	// Must start with YYYYMMDD- (8 digits + dash).
	if len(name) < 9 || name[8] != '-' {
		t.Errorf("filename %q does not start with YYYYMMDD- prefix", name)
	}
	if !strings.HasSuffix(name, ".md") {
		t.Errorf("filename %q does not end with .md", name)
	}
}

func TestSave_TasksDirSetInFrontmatter(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runSave("tasks dir test", nil, "", nil, nil); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}

	plans, err := plan.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if plans[0].TasksDir == "" {
		t.Error("expected tasks_dir to be set in plan frontmatter")
	}
	if !strings.Contains(plans[0].TasksDir, ".logosyncx/tasks/") {
		t.Errorf("tasks_dir %q should contain '.logosyncx/tasks/'", plans[0].TasksDir)
	}
}

func TestSave_ScaffoldOnly_NoBody(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runSave("scaffold only", nil, "", nil, nil); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}

	plans, err := plan.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	if strings.TrimSpace(plans[0].Body) != "" {
		t.Errorf("expected empty body (scaffold only), got: %q", plans[0].Body)
	}
}

func TestSave_AllFrontmatterFields(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runSave("all fields", []string{"go", "cli"}, "claude-code", []string{"old-plan.md"}, nil); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}

	plans, err := plan.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
	p := plans[0]
	if p.Topic != "all fields" {
		t.Errorf("topic = %q, want 'all fields'", p.Topic)
	}
	if p.Agent != "claude-code" {
		t.Errorf("agent = %q, want 'claude-code'", p.Agent)
	}
	if len(p.Tags) != 2 || p.Tags[0] != "go" || p.Tags[1] != "cli" {
		t.Errorf("tags = %v, want [go cli]", p.Tags)
	}
	if len(p.Related) != 1 || p.Related[0] != "old-plan.md" {
		t.Errorf("related = %v, want [old-plan.md]", p.Related)
	}
}

// --- --depends-on ------------------------------------------------------------

func TestSave_DependsOn_ResolvesPartialMatch(t *testing.T) {
	dir := setupInitedProject(t)

	// Create a first plan to depend on.
	if err := runSave("auth refactor", nil, "", nil, nil); err != nil {
		t.Fatalf("first runSave failed: %v", err)
	}

	// Create a second plan that depends on it via partial name.
	if err := runSave("jwt middleware", nil, "", nil, []string{"auth"}); err != nil {
		t.Fatalf("second runSave with --depends-on failed: %v", err)
	}

	plans, err := plan.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	// Find the jwt plan.
	var jwtPlan *plan.Plan
	for i := range plans {
		if strings.Contains(plans[i].Topic, "jwt") {
			jwtPlan = &plans[i]
			break
		}
	}
	if jwtPlan == nil {
		t.Fatal("jwt plan not found")
	}
	if len(jwtPlan.DependsOn) != 1 {
		t.Fatalf("expected 1 dependency, got %d: %v", len(jwtPlan.DependsOn), jwtPlan.DependsOn)
	}
	if !strings.Contains(jwtPlan.DependsOn[0], "auth") {
		t.Errorf("depends_on = %v, expected to contain 'auth'", jwtPlan.DependsOn)
	}
}

func TestSave_DependsOn_NotFound_HardError(t *testing.T) {
	setupInitedProject(t)

	err := runSave("some plan", nil, "", nil, []string{"nonexistent-plan"})
	if err == nil {
		t.Fatal("expected error for nonexistent plan, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestSave_DependsOn_Ambiguous_HardError(t *testing.T) {
	setupInitedProject(t)

	// Create two plans with "api" in their names.
	if err := runSave("api auth", nil, "", nil, nil); err != nil {
		t.Fatalf("runSave api-auth failed: %v", err)
	}
	if err := runSave("api gateway", nil, "", nil, nil); err != nil {
		t.Fatalf("runSave api-gateway failed: %v", err)
	}

	err := runSave("new plan", nil, "", nil, []string{"api"})
	if err == nil {
		t.Fatal("expected error for ambiguous plan name, got nil")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected 'ambiguous' in error, got: %v", err)
	}
}

// --- circular dependency detection (§8.4) ------------------------------------

func TestDetectCircular_DirectSelf(t *testing.T) {
	plans := []plan.Plan{
		{Filename: "20260601-a.md", DependsOn: []string{}},
	}
	// A depends on itself directly.
	err := detectCircular("20260601-a.md", []string{"20260601-a.md"}, plans)
	if err == nil {
		t.Fatal("expected circular dependency error, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("expected 'circular' in error, got: %v", err)
	}
}

func TestDetectCircular_TransitiveCycle(t *testing.T) {
	// A → B → A
	plans := []plan.Plan{
		{Filename: "20260601-a.md", DependsOn: []string{"20260601-b.md"}},
		{Filename: "20260601-b.md", DependsOn: []string{"20260601-a.md"}},
	}
	err := detectCircular("20260601-a.md", []string{"20260601-b.md"}, plans)
	if err == nil {
		t.Fatal("expected transitive circular dependency error, got nil")
	}
}

func TestDetectCircular_NoCycle(t *testing.T) {
	// A → B → C (linear, no cycle)
	plans := []plan.Plan{
		{Filename: "20260601-a.md", DependsOn: []string{"20260601-b.md"}},
		{Filename: "20260601-b.md", DependsOn: []string{"20260601-c.md"}},
		{Filename: "20260601-c.md", DependsOn: []string{}},
	}
	if err := detectCircular("20260601-a.md", []string{"20260601-b.md"}, plans); err != nil {
		t.Errorf("expected no error for acyclic deps, got: %v", err)
	}
}

// --- blocked plan check (§8.2) -----------------------------------------------

func TestBlockedByDep_NotDistilled(t *testing.T) {
	dep := plan.Plan{Filename: "20260601-infra.md", Distilled: false}
	p := plan.Plan{Filename: "20260602-auth.md", DependsOn: []string{"20260601-infra.md"}}
	allPlans := []plan.Plan{dep, p}

	blocker := blockedByDep(p, allPlans)
	if blocker != "20260601-infra.md" {
		t.Errorf("blockedByDep = %q, want '20260601-infra.md'", blocker)
	}
}

func TestBlockedByDep_Distilled(t *testing.T) {
	dep := plan.Plan{Filename: "20260601-infra.md", Distilled: true}
	p := plan.Plan{Filename: "20260602-auth.md", DependsOn: []string{"20260601-infra.md"}}
	allPlans := []plan.Plan{dep, p}

	if blocker := blockedByDep(p, allPlans); blocker != "" {
		t.Errorf("expected empty blocker for distilled dep, got %q", blocker)
	}
}

func TestBlockedByDep_NoDeps(t *testing.T) {
	p := plan.Plan{Filename: "20260602-auth.md", DependsOn: nil}
	if blocker := blockedByDep(p, nil); blocker != "" {
		t.Errorf("expected empty blocker for plan with no deps, got %q", blocker)
	}
}
