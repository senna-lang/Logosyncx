// Package index provides tests for the JSONL plan index.
package index

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/senna-lang/logosyncx/pkg/plan"
)

// --- helpers -----------------------------------------------------------------

// setupProject creates a temp directory with a .logosyncx/plans/ structure.
func setupProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".logosyncx", "plans"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return dir
}

// writePlanFile writes a plan to disk under projectRoot/plans/ including body.
// Unlike plan.Write (scaffold-only), this writes the Body field so that
// excerpt extraction works during Rebuild.
func writePlanFile(t *testing.T, projectRoot string, p plan.Plan) {
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
		t.Fatalf("WriteFile: %v", err)
	}
}

// makePlan returns a minimal plan.Plan for testing.
func makePlan(id, topic string, tags []string, date time.Time) plan.Plan {
	return plan.Plan{
		ID:       id,
		Date:     &date,
		Topic:    topic,
		Tags:     tags,
		Agent:    "claude-code",
		Related:  []string{},
		TasksDir: ".logosyncx/tasks/" + topic,
		Body:     "## Background\nThis is a test plan about " + topic + ".\n",
	}
}

// --- FilePath ----------------------------------------------------------------

func TestFilePath_ReturnsExpectedPath(t *testing.T) {
	got := FilePath("/project")
	want := "/project/.logosyncx/index.jsonl"
	if got != want {
		t.Errorf("FilePath = %q, want %q", got, want)
	}
}

// --- FromPlan ----------------------------------------------------------------

func TestFromPlan_CopiesAllFields(t *testing.T) {
	date := time.Date(2026, 3, 4, 10, 30, 0, 0, time.UTC)
	p := plan.Plan{
		ID:        "abc123",
		Filename:  "20260304-auth.md",
		Date:      &date,
		Topic:     "auth",
		Tags:      []string{"jwt", "security"},
		Agent:     "claude-code",
		Related:   []string{"20260101-prev.md"},
		TasksDir:  ".logosyncx/tasks/20260304-auth",
		Distilled: false,
		Excerpt:   "JWT authentication decisions.",
	}
	e := FromPlan(p, nil)

	if e.ID != p.ID {
		t.Errorf("ID = %q, want %q", e.ID, p.ID)
	}
	if e.Filename != p.Filename {
		t.Errorf("Filename = %q, want %q", e.Filename, p.Filename)
	}
	if !e.Date.Equal(*p.Date) {
		t.Errorf("Date = %v, want %v", e.Date, *p.Date)
	}
	if e.Topic != p.Topic {
		t.Errorf("Topic = %q, want %q", e.Topic, p.Topic)
	}
	if len(e.Tags) != 2 || e.Tags[0] != "jwt" {
		t.Errorf("Tags = %v, want [jwt security]", e.Tags)
	}
	if e.Agent != p.Agent {
		t.Errorf("Agent = %q, want %q", e.Agent, p.Agent)
	}
	if len(e.Related) != 1 || e.Related[0] != "20260101-prev.md" {
		t.Errorf("Related = %v, want [20260101-prev.md]", e.Related)
	}
	if e.TasksDir != p.TasksDir {
		t.Errorf("TasksDir = %q, want %q", e.TasksDir, p.TasksDir)
	}
	if e.Excerpt != p.Excerpt {
		t.Errorf("Excerpt = %q, want %q", e.Excerpt, p.Excerpt)
	}
}

func TestFromPlan_NilTagsBecomesEmpty(t *testing.T) {
	p := plan.Plan{ID: "x", Tags: nil, Related: []string{}}
	e := FromPlan(p, nil)
	if e.Tags == nil {
		t.Error("Tags should be [] not nil")
	}
}

func TestFromPlan_NilRelatedBecomesEmpty(t *testing.T) {
	p := plan.Plan{ID: "x", Tags: []string{}, Related: nil}
	e := FromPlan(p, nil)
	if e.Related == nil {
		t.Error("Related should be [] not nil")
	}
}

func TestFromPlan_NilDependsOnBecomesEmpty(t *testing.T) {
	p := plan.Plan{ID: "x", DependsOn: nil}
	e := FromPlan(p, nil)
	if e.DependsOn == nil {
		t.Error("DependsOn should be [] not nil")
	}
}

func TestFromPlan_NotBlocked_WhenNoDeps(t *testing.T) {
	p := plan.Plan{ID: "x", DependsOn: nil}
	e := FromPlan(p, nil)
	if e.Blocked {
		t.Error("expected Blocked = false when no DependsOn")
	}
}

func TestFromPlan_NotBlocked_WhenDepsDistilled(t *testing.T) {
	parent := plan.Plan{
		ID:        "parent",
		Filename:  "20260101-parent.md",
		Topic:     "parent",
		Distilled: true,
	}
	child := plan.Plan{
		ID:        "child",
		Filename:  "20260301-child.md",
		Topic:     "child",
		DependsOn: []string{"20260101-parent.md"},
	}
	e := FromPlan(child, []plan.Plan{parent, child})
	if e.Blocked {
		t.Error("expected Blocked = false when all deps are distilled")
	}
}

func TestFromPlan_Blocked_WhenDepsNotDistilled(t *testing.T) {
	parent := plan.Plan{
		ID:        "parent",
		Filename:  "20260101-parent.md",
		Topic:     "parent",
		Distilled: false, // not distilled
	}
	child := plan.Plan{
		ID:        "child",
		Filename:  "20260301-child.md",
		Topic:     "child",
		DependsOn: []string{"20260101-parent.md"},
	}
	e := FromPlan(child, []plan.Plan{parent, child})
	if !e.Blocked {
		t.Error("expected Blocked = true when dep is not distilled")
	}
}

func TestFromPlan_Blocked_WhenOnlyOneDepsNotDistilled(t *testing.T) {
	done := plan.Plan{Filename: "20260101-done.md", Distilled: true}
	pending := plan.Plan{Filename: "20260201-pending.md", Distilled: false}
	child := plan.Plan{
		Filename:  "20260301-child.md",
		DependsOn: []string{"20260101-done.md", "20260201-pending.md"},
	}
	e := FromPlan(child, []plan.Plan{done, pending, child})
	if !e.Blocked {
		t.Error("expected Blocked = true when at least one dep is not distilled")
	}
}

func TestFromPlan_Distilled_PropagatedToEntry(t *testing.T) {
	p := plan.Plan{ID: "x", Distilled: true}
	e := FromPlan(p, nil)
	if !e.Distilled {
		t.Error("expected Distilled = true in entry")
	}
}

// --- ReadAll -----------------------------------------------------------------

func TestReadAll_FileNotExist_ReturnsErrNotExist(t *testing.T) {
	dir := setupProject(t)
	_, err := ReadAll(dir)
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}

func TestReadAll_EmptyFile_ReturnsNoEntries(t *testing.T) {
	dir := setupProject(t)
	if err := os.WriteFile(FilePath(dir), []byte{}, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	entries, err := ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestReadAll_OneEntry(t *testing.T) {
	dir := setupProject(t)
	date := time.Date(2026, 3, 4, 10, 30, 0, 0, time.UTC)
	p := makePlan("a1b2c3", "auth-refactor", []string{"auth", "jwt"}, date)
	e := FromPlan(p, nil)
	e.Filename = "20260304-auth-refactor.md"

	if err := Append(dir, e); err != nil {
		t.Fatalf("Append: %v", err)
	}

	entries, err := ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	got := entries[0]
	if got.ID != "a1b2c3" {
		t.Errorf("ID = %q, want 'a1b2c3'", got.ID)
	}
	if got.Topic != "auth-refactor" {
		t.Errorf("Topic = %q, want 'auth-refactor'", got.Topic)
	}
	if len(got.Tags) != 2 {
		t.Errorf("Tags = %v, want [auth jwt]", got.Tags)
	}
	if !got.Date.Equal(date) {
		t.Errorf("Date = %v, want %v", got.Date, date)
	}
}

func TestReadAll_MultipleEntries(t *testing.T) {
	dir := setupProject(t)
	for i, topic := range []string{"topic-a", "topic-b", "topic-c"} {
		e := Entry{
			ID:        []string{"id1", "id2", "id3"}[i],
			Topic:     topic,
			Tags:      []string{},
			Related:   []string{},
			DependsOn: []string{},
			Date:      time.Now(),
		}
		if err := Append(dir, e); err != nil {
			t.Fatalf("Append %s: %v", topic, err)
		}
	}

	entries, err := ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestReadAll_SkipsBlankLines(t *testing.T) {
	dir := setupProject(t)
	e := Entry{ID: "x1", Topic: "t", Tags: []string{}, Related: []string{}, DependsOn: []string{}, Date: time.Now()}
	if err := Append(dir, e); err != nil {
		t.Fatalf("Append: %v", err)
	}

	f, err := os.OpenFile(FilePath(dir), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	_, _ = f.WriteString("\n\n")
	f.Close()

	entries, readErr := ReadAll(dir)
	if readErr != nil {
		t.Fatalf("ReadAll: %v", readErr)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry (blank lines skipped), got %d", len(entries))
	}
}

func TestReadAll_MalformedLine_ReturnsError(t *testing.T) {
	dir := setupProject(t)
	if err := os.WriteFile(FilePath(dir), []byte("not valid json\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := ReadAll(dir)
	if err == nil {
		t.Error("expected error for malformed JSON line, got nil")
	}
}

// --- Append ------------------------------------------------------------------

func TestAppend_CreatesFileIfNotExists(t *testing.T) {
	dir := setupProject(t)
	e := Entry{ID: "new1", Topic: "test", Tags: []string{}, Related: []string{}, DependsOn: []string{}, Date: time.Now()}

	if err := Append(dir, e); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if _, err := os.Stat(FilePath(dir)); err != nil {
		t.Errorf("expected index.jsonl to exist, got: %v", err)
	}
}

func TestAppend_MultipleCallsAccumulate(t *testing.T) {
	dir := setupProject(t)
	for i, topic := range []string{"a", "b", "c"} {
		e := Entry{ID: []string{"i1", "i2", "i3"}[i], Topic: topic, Tags: []string{}, Related: []string{}, DependsOn: []string{}, Date: time.Now()}
		if err := Append(dir, e); err != nil {
			t.Fatalf("Append %q: %v", topic, err)
		}
	}

	entries, err := ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries after 3 appends, got %d", len(entries))
	}
}

func TestAppend_PreservesExistingEntries(t *testing.T) {
	dir := setupProject(t)
	e1 := Entry{ID: "first", Topic: "first-topic", Tags: []string{}, Related: []string{}, DependsOn: []string{}, Date: time.Now()}
	e2 := Entry{ID: "second", Topic: "second-topic", Tags: []string{}, Related: []string{}, DependsOn: []string{}, Date: time.Now()}

	if err := Append(dir, e1); err != nil {
		t.Fatalf("first Append: %v", err)
	}
	if err := Append(dir, e2); err != nil {
		t.Fatalf("second Append: %v", err)
	}

	entries, err := ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if entries[0].ID != "first" {
		t.Errorf("first entry ID = %q, want 'first'", entries[0].ID)
	}
	if entries[1].ID != "second" {
		t.Errorf("second entry ID = %q, want 'second'", entries[1].ID)
	}
}

// --- Rebuild -----------------------------------------------------------------

func TestRebuild_EmptyPlans_CreatesEmptyIndex(t *testing.T) {
	dir := setupProject(t)
	n, err := Rebuild(dir, "")
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 plans indexed, got %d", n)
	}
	if _, statErr := os.Stat(FilePath(dir)); statErr != nil {
		t.Errorf("index.jsonl should exist after Rebuild, got: %v", statErr)
	}
}

func TestRebuild_ScansPlansDir(t *testing.T) {
	dir := setupProject(t)
	date := time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)

	writePlanFile(t, dir, makePlan("id1", "auth-flow", []string{"auth"}, date))
	writePlanFile(t, dir, makePlan("id2", "db-schema", []string{"postgres"}, date.Add(-24*time.Hour)))

	n, err := Rebuild(dir, "")
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 plans indexed, got %d", n)
	}

	entries, err := ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries in index, got %d", len(entries))
	}
}

func TestRebuild_OverwritesExistingIndex(t *testing.T) {
	dir := setupProject(t)
	date := time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)

	stale := Entry{ID: "stale", Topic: "old-topic", Tags: []string{}, Related: []string{}, DependsOn: []string{}, Date: date}
	if err := Append(dir, stale); err != nil {
		t.Fatalf("Append stale: %v", err)
	}

	writePlanFile(t, dir, makePlan("fresh", "new-topic", []string{}, date))
	n, err := Rebuild(dir, "")
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 plan indexed, got %d", n)
	}

	entries, err := ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (stale overwritten), got %d", len(entries))
	}
	if entries[0].Topic == "old-topic" {
		t.Error("stale entry should have been removed by Rebuild")
	}
	if entries[0].Topic != "new-topic" {
		t.Errorf("expected 'new-topic', got %q", entries[0].Topic)
	}
}

func TestRebuild_PopulatesExcerpt(t *testing.T) {
	dir := setupProject(t)
	date := time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)
	p := plan.Plan{
		ID:       "exc1",
		Date:     &date,
		Topic:    "excerpt-test",
		Tags:     []string{},
		Related:  []string{},
		TasksDir: ".logosyncx/tasks/20260304-excerpt-test",
		Body:     "## Background\nThis excerpt should appear in the index.\n",
	}
	writePlanFile(t, dir, p)

	if _, err := Rebuild(dir, "Background"); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	entries, err := ReadAll(dir)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Excerpt == "" {
		t.Error("expected non-empty excerpt after Rebuild")
	}
}

func TestRebuild_NoPlansDir_ReturnsZero(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".logosyncx"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	n, err := Rebuild(dir, "")
	if err != nil {
		t.Fatalf("Rebuild with no plans dir: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 plans, got %d", n)
	}
}
