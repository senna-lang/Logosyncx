package cmd

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/session"
)

// --- helpers -----------------------------------------------------------------

func setupProjectWithSession(t *testing.T, s session.Session) string {
	t.Helper()
	return setupProjectWithSessions(t, []session.Session{s})
}

func makeReferSession(id, topic string, tags []string, date time.Time) session.Session {
	return session.Session{
		ID:      id,
		Date:    &date,
		Topic:   topic,
		Tags:    tags,
		Agent:   "claude-code",
		Related: []string{},
		Body: "## Summary\nThis is the summary for " + topic + ".\n\n" +
			"## Key Decisions\n- Decision A\n- Decision B\n\n" +
			"## Implementation Details\nSome extra detail that should not appear in --summary output.\n",
	}
}

// --- runRefer: no sessions ---------------------------------------------------

func TestRefer_NoSessions_ReturnsError(t *testing.T) {
	setupInitedProject(t)

	err := runRefer("anything", false)
	if err == nil {
		t.Fatal("expected error when no sessions exist, got nil")
	}
	if !strings.Contains(err.Error(), "no session found") {
		t.Errorf("expected 'no session found' in error, got: %v", err)
	}
}

// --- runRefer: no match ------------------------------------------------------

func TestRefer_NoMatch_ReturnsError(t *testing.T) {
	s := makeReferSession("abc123", "auth-refactor", []string{"auth"}, time.Now())
	setupProjectWithSession(t, s)

	err := runRefer("completely-unrelated", false)
	if err == nil {
		t.Fatal("expected error for non-matching name, got nil")
	}
	if !strings.Contains(err.Error(), "no session found matching") {
		t.Errorf("expected 'no session found matching' in error, got: %v", err)
	}
}

func TestRefer_NoMatch_ErrorContainsName(t *testing.T) {
	s := makeReferSession("abc123", "auth-refactor", []string{}, time.Now())
	setupProjectWithSession(t, s)

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
	s := makeReferSession("abc123", "auth-refactor", []string{}, time.Now())
	setupProjectWithSession(t, s)

	out := captureOutput(t, func() {
		if err := runRefer("auth-refactor", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	if !strings.Contains(out, "auth-refactor") {
		t.Errorf("expected topic in output, got: %q", out)
	}
	if !strings.Contains(out, "## Summary") {
		t.Errorf("expected body content in output, got: %q", out)
	}
}

func TestRefer_ExactIDMatch_PrintsContent(t *testing.T) {
	s := makeReferSession("deadbeef", "some-topic", []string{}, time.Now())
	setupProjectWithSession(t, s)

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
	s := makeReferSession("abc123", "my-feature", []string{}, now)
	setupProjectWithSession(t, s)

	// The canonical filename is "2024-06-15_my-feature.md"; match on the stem.
	out := captureOutput(t, func() {
		if err := runRefer("2024-06-15_my-feature", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	if !strings.Contains(out, "my-feature") {
		t.Errorf("expected topic in output, got: %q", out)
	}
}

// --- runRefer: partial match -------------------------------------------------

func TestRefer_PartialTopicMatch_PrintsContent(t *testing.T) {
	s := makeReferSession("abc123", "database-migration", []string{}, time.Now())
	setupProjectWithSession(t, s)

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
	s := makeReferSession("abc123", "cache-layer", []string{}, now)
	setupProjectWithSession(t, s)

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
	s := makeReferSession("abc123", "Auth-Refactor", []string{}, time.Now())
	setupProjectWithSession(t, s)

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
	s := makeReferSession("abc123", "Payment-Processing", []string{}, time.Now())
	setupProjectWithSession(t, s)

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
	s := makeReferSession("front01", "frontmatter-check", []string{"go", "test"}, time.Now())
	setupProjectWithSession(t, s)

	out := captureOutput(t, func() {
		if err := runRefer("frontmatter-check", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	// Full output must include the YAML frontmatter delimiter.
	if !strings.Contains(out, "---") {
		t.Errorf("expected YAML frontmatter delimiter in full output, got: %q", out)
	}
	// Must include the ID field.
	if !strings.Contains(out, "front01") {
		t.Errorf("expected session ID in frontmatter, got: %q", out)
	}
}

func TestRefer_FullContent_IncludesBody(t *testing.T) {
	s := makeReferSession("abc123", "body-check", []string{}, time.Now())
	setupProjectWithSession(t, s)

	out := captureOutput(t, func() {
		if err := runRefer("body-check", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	if !strings.Contains(out, "## Summary") {
		t.Errorf("expected body section heading in output, got: %q", out)
	}
	if !strings.Contains(out, "## Implementation Details") {
		t.Errorf("expected Implementation Details section in output, got: %q", out)
	}
}

// --- runRefer: --summary flag ------------------------------------------------

func TestRefer_Summary_ReturnsOnlySummarySections(t *testing.T) {
	s := makeReferSession("abc123", "summary-test", []string{}, time.Now())
	dir := setupProjectWithSession(t, s)

	// Ensure config has the default summary sections.
	cfg, _ := config.Load(dir)
	cfg.Save.SummarySections = []string{"Summary", "Key Decisions"}
	_ = config.Save(dir, cfg)

	out := captureOutput(t, func() {
		if err := runRefer("summary-test", true); err != nil {
			t.Fatalf("runRefer --summary failed: %v", err)
		}
	})

	if !strings.Contains(out, "## Summary") {
		t.Errorf("expected Summary section in output, got: %q", out)
	}
	if !strings.Contains(out, "## Key Decisions") {
		t.Errorf("expected Key Decisions section in output, got: %q", out)
	}
	// The Implementation Details section must NOT appear.
	if strings.Contains(out, "## Implementation Details") {
		t.Errorf("expected Implementation Details to be excluded, got: %q", out)
	}
}

func TestRefer_Summary_ExcludesBodyNotInSections(t *testing.T) {
	s := makeReferSession("abc123", "exclude-test", []string{}, time.Now())
	dir := setupProjectWithSession(t, s)

	cfg, _ := config.Load(dir)
	cfg.Save.SummarySections = []string{"Summary"}
	_ = config.Save(dir, cfg)

	out := captureOutput(t, func() {
		if err := runRefer("exclude-test", true); err != nil {
			t.Fatalf("runRefer --summary failed: %v", err)
		}
	})

	if !strings.Contains(out, "## Summary") {
		t.Errorf("expected Summary section, got: %q", out)
	}
	if strings.Contains(out, "## Key Decisions") {
		t.Errorf("expected Key Decisions to be excluded, got: %q", out)
	}
}

func TestRefer_Summary_DoesNotIncludeFrontmatter(t *testing.T) {
	s := makeReferSession("frontcheck", "no-frontmatter", []string{}, time.Now())
	dir := setupProjectWithSession(t, s)

	cfg, _ := config.Load(dir)
	cfg.Save.SummarySections = []string{"Summary"}
	_ = config.Save(dir, cfg)

	out := captureOutput(t, func() {
		if err := runRefer("no-frontmatter", true); err != nil {
			t.Fatalf("runRefer --summary failed: %v", err)
		}
	})

	// --summary prints only body sections, not the YAML frontmatter block.
	if strings.Contains(out, "frontcheck") {
		t.Errorf("expected session ID (from frontmatter) to be absent in --summary output, got: %q", out)
	}
}

// --- runRefer: multiple matches ----------------------------------------------

func TestRefer_MultipleMatches_ReturnsError(t *testing.T) {
	now := time.Now()
	sessions := []session.Session{
		makeReferSession("id1", "auth-login", []string{}, now.Add(-2*time.Hour)),
		makeReferSession("id2", "auth-signup", []string{}, now.Add(-1*time.Hour)),
	}
	setupProjectWithSessions(t, sessions)

	err := runRefer("auth", false)
	if err == nil {
		t.Fatal("expected error when multiple sessions match, got nil")
	}
	if !strings.Contains(err.Error(), "more specific") {
		t.Errorf("expected hint to narrow search in error, got: %v", err)
	}
}

func TestRefer_MultipleMatches_DoesNotPrintContent(t *testing.T) {
	now := time.Now()
	sessions := []session.Session{
		makeReferSession("id1", "api-design", []string{}, now.Add(-2*time.Hour)),
		makeReferSession("id2", "api-versioning", []string{}, now.Add(-1*time.Hour)),
	}
	setupProjectWithSessions(t, sessions)

	out := captureOutput(t, func() {
		_ = runRefer("api", false)
	})

	// stdout must be empty; candidate list goes to stderr.
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no stdout output for multiple matches, got: %q", out)
	}
}

// --- runRefer: exact match wins over partial ---------------------------------

func TestRefer_ExactMatchPreferredOverPartial(t *testing.T) {
	now := time.Now()
	sessions := []session.Session{
		// Exact topic match.
		makeReferSession("exact1", "auth", []string{}, now.Add(-2*time.Hour)),
		// Partial topic match.
		makeReferSession("part1", "auth-middleware", []string{}, now.Add(-1*time.Hour)),
		makeReferSession("part2", "oauth-setup", []string{}, now),
	}
	setupProjectWithSessions(t, sessions)

	out := captureOutput(t, func() {
		if err := runRefer("auth", false); err != nil {
			t.Fatalf("runRefer failed: %v", err)
		}
	})

	// Only the exact match ("auth") should be printed.
	if !strings.Contains(out, "exact1") {
		t.Errorf("expected exact match session ID in output, got: %q", out)
	}
}

// --- matchSessions unit tests ------------------------------------------------

func TestMatchSessions_EmptyList(t *testing.T) {
	result := matchSessions(nil, "anything")
	if len(result) != 0 {
		t.Errorf("expected empty result for nil sessions, got %d", len(result))
	}
}

func TestMatchSessions_ExactTopicMatch(t *testing.T) {
	sessions := []session.Session{
		{ID: "a", Topic: "foo", Filename: "2024-01-01_foo.md"},
		{ID: "b", Topic: "foobar", Filename: "2024-01-02_foobar.md"},
	}
	result := matchSessions(sessions, "foo")
	if len(result) != 1 {
		t.Fatalf("expected 1 exact match, got %d", len(result))
	}
	if result[0].ID != "a" {
		t.Errorf("expected session 'a', got %q", result[0].ID)
	}
}

func TestMatchSessions_PartialTopicMatch(t *testing.T) {
	sessions := []session.Session{
		{ID: "a", Topic: "database-migration", Filename: "2024-01-01_database-migration.md"},
		{ID: "b", Topic: "cache-layer", Filename: "2024-01-02_cache-layer.md"},
	}
	result := matchSessions(sessions, "database")
	if len(result) != 1 {
		t.Fatalf("expected 1 partial match, got %d: %v", len(result), result)
	}
	if result[0].ID != "a" {
		t.Errorf("expected session 'a', got %q", result[0].ID)
	}
}

func TestMatchSessions_IDMatch(t *testing.T) {
	sessions := []session.Session{
		{ID: "abc123", Topic: "some-topic", Filename: "2024-01-01_some-topic.md"},
		{ID: "def456", Topic: "other-topic", Filename: "2024-01-02_other-topic.md"},
	}
	result := matchSessions(sessions, "abc123")
	if len(result) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result))
	}
	if result[0].ID != "abc123" {
		t.Errorf("expected session 'abc123', got %q", result[0].ID)
	}
}

func TestMatchSessions_NoMatch(t *testing.T) {
	sessions := []session.Session{
		{ID: "a", Topic: "foo", Filename: "2024-01-01_foo.md"},
	}
	result := matchSessions(sessions, "zzz")
	if len(result) != 0 {
		t.Errorf("expected 0 matches, got %d", len(result))
	}
}

func TestMatchSessions_MultiplePartialMatches(t *testing.T) {
	sessions := []session.Session{
		{ID: "a", Topic: "auth-login", Filename: "2024-01-01_auth-login.md"},
		{ID: "b", Topic: "auth-signup", Filename: "2024-01-02_auth-signup.md"},
		{ID: "c", Topic: "unrelated", Filename: "2024-01-03_unrelated.md"},
	}
	result := matchSessions(sessions, "auth")
	if len(result) != 2 {
		t.Errorf("expected 2 partial matches, got %d", len(result))
	}
}

func TestMatchSessions_MultipleExactMatches_ReturnsAll(t *testing.T) {
	// Two sessions with the same topic (edge case: duplicates).
	sessions := []session.Session{
		{ID: "a", Topic: "auth", Filename: "2024-01-01_auth.md"},
		{ID: "b", Topic: "auth", Filename: "2024-01-02_auth.md"},
	}
	result := matchSessions(sessions, "auth")
	// Both are exact matches; since there is more than one, we expect both.
	if len(result) != 2 {
		t.Errorf("expected 2 exact matches, got %d", len(result))
	}
}

func TestMatchSessions_CaseInsensitive(t *testing.T) {
	sessions := []session.Session{
		{ID: "a", Topic: "Auth-Service", Filename: "2024-01-01_auth-service.md"},
	}
	result := matchSessions(sessions, "AUTH-SERVICE")
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
