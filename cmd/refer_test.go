package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/plan"
)

// --- helpers -----------------------------------------------------------------

func setupProjectWithPlan(t *testing.T, p plan.Plan) string {
	t.Helper()
	return setupProjectWithPlans(t, []plan.Plan{p})
}

func makeReferPlan(id, topic string, tags []string, date time.Time) plan.Plan {
	return plan.Plan{
		ID:      id,
		Date:    &date,
		Topic:   topic,
		Tags:    tags,
		Agent:   "claude-code",
		Related: []string{},
		Body: "## Background\nThis is the background for " + topic + ".\n\n" +
			"## Spec\n- Spec item A\n- Spec item B\n\n" +
			"## Notes\nSome extra detail that should not appear in --summary output.\n",
	}
}

// --- runRefer: no plans ------------------------------------------------------

func TestRefer_NoPlans_ReturnsError(t *testing.T) {
	setupInitedProject(t)

	err := runRefer("anything", false)
	if err == nil {
		t.Fatal("expected error when no plans exist, got nil")
	}
	if !strings.Contains(err.Error(), "no plan found") {
		t.Errorf("expected 'no plan found' in error, got: %v", err)
	}
}

// --- runRefer: no match ------------------------------------------------------

func TestRefer_NoMatch_ReturnsError(t *testing.T) {
	p := makeReferPlan("abc123", "auth-refactor", []string{"auth"}, time.Now())
	setupProjectWithPlan(t, p)

	err := runRefer("completely-unrelated", false)
	if err == nil {
		t.Fatal("expected error for non-matching name, got nil")
	}
	if !strings.Contains(err.Error(), "no plan found matching") {
		t.Errorf("expected 'no plan found matching' in error, got: %v", err)
	}
}

func TestRefer_NoMatch_ErrorContainsName(t *testing.T) {
	p := makeReferPlan("abc123", "auth-refactor", []string{}, time.Now())
	setupProjectWithPlan(t, p)

	err := runRefer("xyz-unknown", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "xyz-unknown") {
		t.Errorf("expected the search name in error message, got: %v", err)
	}
}

// --- runRefer: exact match ---------------------------------------------------

func TestRefer_ExactTopicMatch_PrintsContent(t *testing.T) {
	p := makeReferPlan("abc123", "auth-refactor", []string{}, time.Now())
	setupProjectWithPlan(t, p)

	out := captureOutput(t, func() {
		if err := runRefer("auth-refactor", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	if !strings.Contains(out, "auth-refactor") {
		t.Errorf("expected topic in output, got: %q", out)
	}
	if !strings.Contains(out, "## Background") {
		t.Errorf("expected body content in output, got: %q", out)
	}
}

func TestRefer_ExactIDMatch_PrintsContent(t *testing.T) {
	p := makeReferPlan("deadbeef", "some-topic", []string{}, time.Now())
	setupProjectWithPlan(t, p)

	out := captureOutput(t, func() {
		if err := runRefer("deadbeef", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	if !strings.Contains(out, "some-topic") {
		t.Errorf("expected topic in output, got: %q", out)
	}
}

func TestRefer_ExactFilenameMatch_PrintsContent(t *testing.T) {
	now := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	p := makeReferPlan("abc123", "my-feature", []string{}, now)

	dir := setupInitedProject(t)
	plansDir := filepath.Join(dir, ".logosyncx", "plans")
	data, _ := plan.Marshal(p)
	data = append(data, []byte(p.Body)...)
	_ = os.WriteFile(filepath.Join(plansDir, "20240615-my-feature.md"), data, 0o644)

	out := captureOutput(t, func() {
		if err := runRefer("20240615-my-feature", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	if !strings.Contains(out, "my-feature") {
		t.Errorf("expected topic in output, got: %q", out)
	}
}

// --- runRefer: partial match -------------------------------------------------

func TestRefer_PartialTopicMatch_PrintsContent(t *testing.T) {
	p := makeReferPlan("abc123", "database-migration", []string{}, time.Now())
	setupProjectWithPlan(t, p)

	out := captureOutput(t, func() {
		if err := runRefer("migration", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	if !strings.Contains(out, "database-migration") {
		t.Errorf("expected topic in output, got: %q", out)
	}
}

func TestRefer_PartialFilenameMatch_PrintsContent(t *testing.T) {
	now := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	p := makeReferPlan("abc123", "cache-layer", []string{}, now)
	setupProjectWithPlan(t, p)

	out := captureOutput(t, func() {
		if err := runRefer("cache", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	if !strings.Contains(out, "cache-layer") {
		t.Errorf("expected topic in output, got: %q", out)
	}
}

// --- runRefer: case insensitivity --------------------------------------------

func TestRefer_CaseInsensitive_TopicMatch(t *testing.T) {
	p := makeReferPlan("abc123", "Auth-Refactor", []string{}, time.Now())
	setupProjectWithPlan(t, p)

	out := captureOutput(t, func() {
		if err := runRefer("auth-refactor", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	if !strings.Contains(out, "Auth-Refactor") {
		t.Errorf("expected topic in output, got: %q", out)
	}
}

func TestRefer_CaseInsensitive_PartialMatch(t *testing.T) {
	p := makeReferPlan("abc123", "Payment-Processing", []string{}, time.Now())
	setupProjectWithPlan(t, p)

	out := captureOutput(t, func() {
		if err := runRefer("PAYMENT", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	if !strings.Contains(out, "Payment-Processing") {
		t.Errorf("expected topic in output, got: %q", out)
	}
}

// --- runRefer: full content output -------------------------------------------

func TestRefer_FullContent_IncludesFrontmatter(t *testing.T) {
	p := makeReferPlan("front01", "frontmatter-check", []string{"go", "test"}, time.Now())
	setupProjectWithPlan(t, p)

	out := captureOutput(t, func() {
		if err := runRefer("frontmatter-check", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	if !strings.Contains(out, "---") {
		t.Errorf("expected YAML frontmatter delimiter in full output, got: %q", out)
	}
	if !strings.Contains(out, "front01") {
		t.Errorf("expected plan ID in frontmatter, got: %q", out)
	}
}

func TestRefer_FullContent_IncludesBody(t *testing.T) {
	p := makeReferPlan("abc123", "body-check", []string{}, time.Now())
	setupProjectWithPlan(t, p)

	out := captureOutput(t, func() {
		if err := runRefer("body-check", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	if !strings.Contains(out, "## Background") {
		t.Errorf("expected body section heading in output, got: %q", out)
	}
	if !strings.Contains(out, "## Notes") {
		t.Errorf("expected Notes section in output, got: %q", out)
	}
}

// --- runRefer: --summary flag ------------------------------------------------

func TestRefer_Summary_ReturnsOnlySummarySections(t *testing.T) {
	p := makeReferPlan("abc123", "summary-test", []string{}, time.Now())
	dir := setupProjectWithPlan(t, p)

	cfg, _ := config.Load(dir)
	cfg.Plans.SummarySections = []string{"Background", "Spec"}
	_ = config.Save(dir, cfg)

	out := captureOutput(t, func() {
		if err := runRefer("summary-test", true); err != nil {
			t.Fatalf("runRefer --summary failed: %v", err)
		}
	})

	if !strings.Contains(out, "## Background") {
		t.Errorf("expected Background section in output, got: %q", out)
	}
	if !strings.Contains(out, "## Spec") {
		t.Errorf("expected Spec section in output, got: %q", out)
	}
	if strings.Contains(out, "## Notes") {
		t.Errorf("expected Notes to be excluded, got: %q", out)
	}
}

func TestRefer_Summary_ExcludesBodyNotInSections(t *testing.T) {
	p := makeReferPlan("abc123", "exclude-test", []string{}, time.Now())
	dir := setupProjectWithPlan(t, p)

	cfg, _ := config.Load(dir)
	cfg.Plans.SummarySections = []string{"Background"}
	_ = config.Save(dir, cfg)

	out := captureOutput(t, func() {
		if err := runRefer("exclude-test", true); err != nil {
			t.Fatalf("runRefer --summary failed: %v", err)
		}
	})

	if !strings.Contains(out, "## Background") {
		t.Errorf("expected Background section, got: %q", out)
	}
	if strings.Contains(out, "## Spec") {
		t.Errorf("expected Spec to be excluded, got: %q", out)
	}
}

func TestRefer_Summary_DoesNotIncludeFrontmatter(t *testing.T) {
	p := makeReferPlan("frontcheck", "no-frontmatter", []string{}, time.Now())
	dir := setupProjectWithPlan(t, p)

	cfg, _ := config.Load(dir)
	cfg.Plans.SummarySections = []string{"Background"}
	_ = config.Save(dir, cfg)

	out := captureOutput(t, func() {
		if err := runRefer("no-frontmatter", true); err != nil {
			t.Fatalf("runRefer --summary failed: %v", err)
		}
	})

	if strings.Contains(out, "frontcheck") {
		t.Errorf("expected plan ID (from frontmatter) to be absent in --summary output, got: %q", out)
	}
}

// --- runRefer: multiple matches ----------------------------------------------

func TestRefer_MultipleMatches_ReturnsError(t *testing.T) {
	now := time.Now()
	plans := []plan.Plan{
		makeReferPlan("id1", "auth-login", []string{}, now.Add(-2*time.Hour)),
		makeReferPlan("id2", "auth-signup", []string{}, now.Add(-1*time.Hour)),
	}
	setupProjectWithPlans(t, plans)

	err := runRefer("auth", false)
	if err == nil {
		t.Fatal("expected error when multiple plans match, got nil")
	}
	if !strings.Contains(err.Error(), "more specific") {
		t.Errorf("expected hint to narrow search in error, got: %v", err)
	}
}

func TestRefer_MultipleMatches_DoesNotPrintContent(t *testing.T) {
	now := time.Now()
	plans := []plan.Plan{
		makeReferPlan("id1", "api-design", []string{}, now.Add(-2*time.Hour)),
		makeReferPlan("id2", "api-versioning", []string{}, now.Add(-1*time.Hour)),
	}
	setupProjectWithPlans(t, plans)

	out := captureOutput(t, func() {
		_ = runRefer("api", false)
	})

	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no stdout output for multiple matches, got: %q", out)
	}
}

// --- runRefer: exact match wins over partial ---------------------------------

func TestRefer_ExactMatchPreferredOverPartial(t *testing.T) {
	now := time.Now()
	plans := []plan.Plan{
		makeReferPlan("exact1", "auth", []string{}, now.Add(-2*time.Hour)),
		makeReferPlan("part1", "auth-middleware", []string{}, now.Add(-1*time.Hour)),
		makeReferPlan("part2", "oauth-setup", []string{}, now),
	}
	setupProjectWithPlans(t, plans)

	out := captureOutput(t, func() {
		if err := runRefer("auth", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	if !strings.Contains(out, "exact1") {
		t.Errorf("expected exact match plan ID in output, got: %q", out)
	}
}

// --- matchPlans unit tests ---------------------------------------------------

func TestMatchPlans_EmptyList(t *testing.T) {
	result := matchPlans(nil, "anything")
	if len(result) != 0 {
		t.Errorf("expected empty result for nil plans, got %d", len(result))
	}
}

func TestMatchPlans_ExactTopicMatch(t *testing.T) {
	plans := []plan.Plan{
		{ID: "a", Topic: "foo", Filename: "20240101-foo.md"},
		{ID: "b", Topic: "foobar", Filename: "20240102-foobar.md"},
	}
	result := matchPlans(plans, "foo")
	if len(result) != 1 {
		t.Fatalf("expected 1 exact match, got %d", len(result))
	}
	if result[0].ID != "a" {
		t.Errorf("expected plan 'a', got %q", result[0].ID)
	}
}

func TestMatchPlans_PartialTopicMatch(t *testing.T) {
	plans := []plan.Plan{
		{ID: "a", Topic: "database-migration", Filename: "20240101-database-migration.md"},
		{ID: "b", Topic: "cache-layer", Filename: "20240102-cache-layer.md"},
	}
	result := matchPlans(plans, "database")
	if len(result) != 1 {
		t.Fatalf("expected 1 partial match, got %d: %v", len(result), result)
	}
	if result[0].ID != "a" {
		t.Errorf("expected plan 'a', got %q", result[0].ID)
	}
}

func TestMatchPlans_IDMatch(t *testing.T) {
	plans := []plan.Plan{
		{ID: "abc123", Topic: "some-topic", Filename: "20240101-some-topic.md"},
		{ID: "def456", Topic: "other-topic", Filename: "20240102-other-topic.md"},
	}
	result := matchPlans(plans, "abc123")
	if len(result) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result))
	}
	if result[0].ID != "abc123" {
		t.Errorf("expected plan 'abc123', got %q", result[0].ID)
	}
}

func TestMatchPlans_NoMatch(t *testing.T) {
	plans := []plan.Plan{
		{ID: "a", Topic: "foo", Filename: "20240101-foo.md"},
	}
	result := matchPlans(plans, "zzz")
	if len(result) != 0 {
		t.Errorf("expected 0 matches, got %d", len(result))
	}
}

func TestMatchPlans_MultiplePartialMatches(t *testing.T) {
	plans := []plan.Plan{
		{ID: "a", Topic: "auth-login", Filename: "20240101-auth-login.md"},
		{ID: "b", Topic: "auth-signup", Filename: "20240102-auth-signup.md"},
		{ID: "c", Topic: "unrelated", Filename: "20240103-unrelated.md"},
	}
	result := matchPlans(plans, "auth")
	if len(result) != 2 {
		t.Errorf("expected 2 partial matches, got %d", len(result))
	}
}

func TestMatchPlans_MultipleExactMatches_ReturnsAll(t *testing.T) {
	plans := []plan.Plan{
		{ID: "a", Topic: "auth", Filename: "20240101-auth.md"},
		{ID: "b", Topic: "auth", Filename: "20240102-auth.md"},
	}
	result := matchPlans(plans, "auth")
	if len(result) != 2 {
		t.Errorf("expected 2 exact matches, got %d", len(result))
	}
}

func TestMatchPlans_CaseInsensitive(t *testing.T) {
	plans := []plan.Plan{
		{ID: "a", Topic: "Auth-Service", Filename: "20240101-auth-service.md"},
	}
	result := matchPlans(plans, "AUTH-SERVICE")
	if len(result) != 1 {
		t.Fatalf("expected 1 match for case-insensitive exact, got %d", len(result))
	}
}

// --- runRefer: not initialised -----------------------------------------------

func TestRefer_NotInitialized_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(orig) })

	err := runRefer("anything", false)
	if err == nil {
		t.Fatal("expected error when project not initialised, got nil")
	}
}
