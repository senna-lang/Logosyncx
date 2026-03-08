package plan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/senna-lang/logosyncx/internal/markdown"
)

// --- FileName ----------------------------------------------------------------

func TestFileName_Format(t *testing.T) {
	date := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	p := Plan{
		Topic: "auth refactor",
		Date:  &date,
	}
	name := FileName(p)

	// Must start with YYYYMMDD- prefix.
	if !strings.HasPrefix(name, "20260304-") {
		t.Errorf("FileName = %q, want prefix '20260304-'", name)
	}
	if !strings.HasSuffix(name, ".md") {
		t.Errorf("FileName = %q, want suffix '.md'", name)
	}
}

func TestFileName_SlugifiesTopic(t *testing.T) {
	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	p := Plan{Topic: "Auth Refactor & Setup", Date: &date}
	name := FileName(p)
	if name != "20260115-auth-refactor-setup.md" {
		t.Errorf("FileName = %q, want '20260115-auth-refactor-setup.md'", name)
	}
}

func TestFileName_NilDateUsesNow(t *testing.T) {
	p := Plan{Topic: "test"}
	name := FileName(p)
	// Just verify it has the YYYYMMDD- pattern (8 digits then dash).
	if len(name) < 9 || name[8] != '-' {
		t.Errorf("FileName with nil date = %q, want YYYYMMDD- prefix", name)
	}
}

// --- DefaultTasksDir ---------------------------------------------------------

func TestDefaultTasksDir(t *testing.T) {
	got := DefaultTasksDir("20260304-auth-refactor.md")
	want := filepath.Join(".logosyncx", "tasks", "20260304-auth-refactor")
	if got != want {
		t.Errorf("DefaultTasksDir = %q, want %q", got, want)
	}
}

func TestDefaultTasksDir_NoExtension(t *testing.T) {
	// Should not strip anything if there's no .md suffix.
	got := DefaultTasksDir("no-extension")
	want := filepath.Join(".logosyncx", "tasks", "no-extension")
	if got != want {
		t.Errorf("DefaultTasksDir = %q, want %q", got, want)
	}
}

// --- Parse / RoundTrip -------------------------------------------------------

func TestParse_RoundTrip(t *testing.T) {
	date := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	original := Plan{
		ID:       "abc123",
		Date:     &date,
		Topic:    "round-trip-test",
		Tags:     []string{"go", "test"},
		Agent:    "claude-code",
		Related:  []string{"20260101-old-plan.md"},
		TasksDir: ".logosyncx/tasks/20260304-round-trip-test",
	}

	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	parsed, err := Parse("20260304-round-trip-test.md", data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.ID != original.ID {
		t.Errorf("ID = %q, want %q", parsed.ID, original.ID)
	}
	if parsed.Topic != original.Topic {
		t.Errorf("Topic = %q, want %q", parsed.Topic, original.Topic)
	}
	if parsed.Agent != original.Agent {
		t.Errorf("Agent = %q, want %q", parsed.Agent, original.Agent)
	}
	if len(parsed.Tags) != 2 || parsed.Tags[0] != "go" || parsed.Tags[1] != "test" {
		t.Errorf("Tags = %v, want [go test]", parsed.Tags)
	}
	if len(parsed.Related) != 1 || parsed.Related[0] != "20260101-old-plan.md" {
		t.Errorf("Related = %v, want [20260101-old-plan.md]", parsed.Related)
	}
	if parsed.TasksDir != original.TasksDir {
		t.Errorf("TasksDir = %q, want %q", parsed.TasksDir, original.TasksDir)
	}
	if parsed.Filename != "20260304-round-trip-test.md" {
		t.Errorf("Filename = %q, want '20260304-round-trip-test.md'", parsed.Filename)
	}
}

func TestParse_DependsOn(t *testing.T) {
	raw := `---
id: abc123
topic: child-plan
depends_on:
  - 20260101-parent-plan.md
  - 20260201-other-plan.md
tasks_dir: .logosyncx/tasks/20260304-child-plan
---
`
	p, err := Parse("20260304-child-plan.md", []byte(raw))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(p.DependsOn) != 2 {
		t.Fatalf("DependsOn len = %d, want 2", len(p.DependsOn))
	}
	if p.DependsOn[0] != "20260101-parent-plan.md" {
		t.Errorf("DependsOn[0] = %q, want '20260101-parent-plan.md'", p.DependsOn[0])
	}
	if p.DependsOn[1] != "20260201-other-plan.md" {
		t.Errorf("DependsOn[1] = %q, want '20260201-other-plan.md'", p.DependsOn[1])
	}
}

func TestParse_Distilled(t *testing.T) {
	raw := `---
id: abc123
topic: distilled-plan
distilled: true
tasks_dir: .logosyncx/tasks/20260304-distilled-plan
---
`
	p, err := Parse("20260304-distilled-plan.md", []byte(raw))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !p.Distilled {
		t.Error("expected Distilled = true")
	}
}

func TestParse_DistilledFalseByDefault(t *testing.T) {
	raw := `---
id: abc123
topic: fresh-plan
tasks_dir: .logosyncx/tasks/20260304-fresh-plan
---
`
	p, err := Parse("20260304-fresh-plan.md", []byte(raw))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if p.Distilled {
		t.Error("expected Distilled = false by default")
	}
}

func TestMarshal_PreservesBodyContent(t *testing.T) {
	date := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	body := "\n## Background\n\nThis is the background section.\n\n## Spec\n\nDo the thing.\n"
	original := Plan{
		ID:       "abc123",
		Date:     &date,
		Topic:    "body-round-trip-test",
		Tags:     []string{"go", "test"},
		Agent:    "claude-code",
		TasksDir: ".logosyncx/tasks/20260304-body-round-trip-test",
		Body:     body,
	}

	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	parsed, err := Parse("20260304-body-round-trip-test.md", data)
	if err != nil {
		t.Fatalf("Parse after Marshal failed: %v", err)
	}

	if parsed.Body != body {
		t.Errorf("Body not preserved.\ngot:  %q\nwant: %q", parsed.Body, body)
	}
}

func TestMarshal_EmptyBody_NoTrailingContent(t *testing.T) {
	date := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	p := Plan{
		ID:       "abc123",
		Date:     &date,
		Topic:    "scaffold-only",
		TasksDir: ".logosyncx/tasks/20260304-scaffold-only",
		Body:     "",
	}

	data, err := Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	content := string(data)
	// With an empty body, the file must end with the closing "---\n".
	if !strings.HasSuffix(strings.TrimRight(content, "\n"), "---") {
		t.Errorf("expected scaffold to end with '---', got: %q", content)
	}
}

func TestMarshal_BodyPreservedAfterDistilledUpdate(t *testing.T) {
	// Regression test: logos distill calls Marshal after setting Distilled=true.
	// The body must survive that rewrite.
	date := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	body := "\n## Background\n\nOriginal body content that must not be lost.\n"
	p := Plan{
		ID:        "def456",
		Date:      &date,
		Topic:     "distill-test",
		TasksDir:  ".logosyncx/tasks/20260304-distill-test",
		Distilled: false,
		Body:      body,
	}

	// Simulate what logos distill does: set Distilled=true and rewrite.
	p.Distilled = true
	data, err := Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	parsed, err := Parse("20260304-distill-test.md", data)
	if err != nil {
		t.Fatalf("Parse after distill Marshal failed: %v", err)
	}

	if !parsed.Distilled {
		t.Error("expected Distilled = true after rewrite")
	}
	if parsed.Body != body {
		t.Errorf("Body destroyed by distill rewrite.\ngot:  %q\nwant: %q", parsed.Body, body)
	}
}

func TestParse_MissingFrontmatter_ReturnsError(t *testing.T) {
	_, err := Parse("bad.md", []byte("no frontmatter here"))
	if err == nil {
		t.Fatal("expected error for missing frontmatter, got nil")
	}
}

func TestParse_ExcerptExtractedFromBackground(t *testing.T) {
	raw := `---
id: abc123
topic: excerpt-test
tasks_dir: .logosyncx/tasks/20260304-excerpt-test
---

## Background

This is the background section content.

## Spec

This should not appear in the excerpt.
`
	p, err := ParseWithOptions("20260304-excerpt-test.md", []byte(raw), ParseOptions{ExcerptSection: "Background"})
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if !strings.Contains(p.Excerpt, "background section content") {
		t.Errorf("Excerpt = %q, expected background content", p.Excerpt)
	}
	if strings.Contains(p.Excerpt, "Spec") {
		t.Errorf("Excerpt should not contain Spec section content, got: %q", p.Excerpt)
	}
}

// --- LoadAll -----------------------------------------------------------------

func TestLoadAll_ScansPlansDir(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, ".logosyncx", "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write two plan files.
	for _, name := range []string{"20260101-plan-alpha.md", "20260201-plan-beta.md"} {
		raw := "---\nid: test\ntopic: " + strings.TrimSuffix(name, ".md") + "\ntasks_dir: .logosyncx/tasks/x\n---\n"
		if err := os.WriteFile(filepath.Join(plansDir, name), []byte(raw), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A non-.md file should be ignored.
	_ = os.WriteFile(filepath.Join(plansDir, "readme.txt"), []byte("ignore me"), 0o644)

	plans, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(plans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(plans))
	}
}

func TestLoadAll_EmptyDir_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".logosyncx", "plans"), 0o755); err != nil {
		t.Fatal(err)
	}
	plans, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("expected 0 plans, got %d", len(plans))
	}
}

func TestLoadAll_MissingDir_ReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	// No .logosyncx/plans directory created.
	plans, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll with missing dir: %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("expected 0 plans, got %d", len(plans))
	}
}

// --- Write -------------------------------------------------------------------

func TestWrite_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	p := Plan{
		ID:       "abc123",
		Date:     &date,
		Topic:    "write-test",
		TasksDir: ".logosyncx/tasks/20260304-write-test",
	}

	path, err := Write(dir, p)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("expected file to exist at %s", path)
	}
	if !strings.HasSuffix(path, "20260304-write-test.md") {
		t.Errorf("path = %q, want suffix '20260304-write-test.md'", path)
	}
}

func TestWrite_ScaffoldOnly_NoBody(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	p := Plan{
		ID:       "abc123",
		Date:     &date,
		Topic:    "scaffold-test",
		TasksDir: ".logosyncx/tasks/20260304-scaffold-test",
	}

	path, err := Write(dir, p)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	// File must end with the closing "---\n" (scaffold only — no body appended).
	if !strings.HasSuffix(strings.TrimRight(content, "\n"), "---") {
		t.Errorf("expected scaffold to end with '---', got: %q", content)
	}
}

// --- Archive -----------------------------------------------------------------

func TestArchive_MovesFile(t *testing.T) {
	dir := t.TempDir()
	plansDir := filepath.Join(dir, ".logosyncx", "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		t.Fatal(err)
	}
	filename := "20260304-archive-me.md"
	src := filepath.Join(plansDir, filename)
	if err := os.WriteFile(src, []byte("---\nid: x\ntopic: t\ntasks_dir: t\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	dst, err := Archive(dir, filename)
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("original file should have been moved")
	}
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		t.Errorf("expected archived file at %s", dst)
	}
}

// --- GenerateID --------------------------------------------------------------

func TestGenerateID_Length(t *testing.T) {
	id, err := GenerateID()
	if err != nil {
		t.Fatalf("GenerateID failed: %v", err)
	}
	if len(id) != 6 {
		t.Errorf("expected length 6, got %d: %q", len(id), id)
	}
}

func TestGenerateID_IsHex(t *testing.T) {
	id, err := GenerateID()
	if err != nil {
		t.Fatalf("GenerateID failed: %v", err)
	}
	for _, r := range id {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			t.Errorf("ID %q contains non-hex character %q", id, r)
		}
	}
}

// --- slugify -----------------------------------------------------------------

func TestSlugify_Basic(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Auth Refactor", "auth-refactor"},
		{"auth-refactor", "auth-refactor"},
		{"Auth & Setup!", "auth-setup"},
		{"  leading spaces  ", "leading-spaces"},
		{"UPPER CASE", "upper-case"},
		{"mixed123Numbers", "mixed123numbers"},
	}
	for _, tc := range cases {
		got := markdown.Slugify(tc.input)
		if got != tc.want {
			t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
