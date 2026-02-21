package cmd

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/senna-lang/logosyncx/pkg/session"
)

// --- helpers -----------------------------------------------------------------

func makeSearchSession(id, topic string, tags []string, excerpt string, date time.Time) session.Session {
	return session.Session{
		ID:      id,
		Date:    date,
		Topic:   topic,
		Tags:    tags,
		Agent:   "claude-code",
		Related: []string{},
		// Embed the desired excerpt text in the ## Summary section so that
		// session.Parse populates s.Excerpt with it.
		Body: "## Summary\n" + excerpt + "\n\n## Key Decisions\n- Decision A\n",
	}
}

// --- runSearch: not initialised ----------------------------------------------

func TestSearch_NotInitialized_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	if err := runSearch("anything", ""); err == nil {
		t.Fatal("expected error when project not initialised, got nil")
	}
}

// --- runSearch: no sessions --------------------------------------------------

func TestSearch_NoSessions_PrintsMessage(t *testing.T) {
	setupInitedProject(t)

	out := captureOutput(t, func() {
		if err := runSearch("anything", ""); err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
	})

	if !strings.Contains(out, "No sessions found") {
		t.Errorf("expected 'No sessions found', got: %q", out)
	}
}

// --- runSearch: keyword matching ---------------------------------------------

func TestSearch_MatchesTopic(t *testing.T) {
	s := makeSearchSession("id1", "jwt-authentication", []string{}, "Some summary.", time.Now())
	setupProjectWithSessions(t, []session.Session{s})

	out := captureOutput(t, func() {
		if err := runSearch("jwt", ""); err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
	})

	if !strings.Contains(out, "jwt-authentication") {
		t.Errorf("expected topic in output, got: %q", out)
	}
}

func TestSearch_MatchesTag(t *testing.T) {
	s := makeSearchSession("id1", "some-topic", []string{"security", "oauth"}, "Some summary.", time.Now())
	setupProjectWithSessions(t, []session.Session{s})

	out := captureOutput(t, func() {
		if err := runSearch("oauth", ""); err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
	})

	if !strings.Contains(out, "some-topic") {
		t.Errorf("expected session in output when keyword matches tag, got: %q", out)
	}
}

func TestSearch_MatchesExcerpt(t *testing.T) {
	s := makeSearchSession("id1", "refactor-session", []string{}, "We decided to migrate from REST to GraphQL.", time.Now())
	setupProjectWithSessions(t, []session.Session{s})

	out := captureOutput(t, func() {
		if err := runSearch("GraphQL", ""); err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
	})

	if !strings.Contains(out, "refactor-session") {
		t.Errorf("expected session in output when keyword matches excerpt, got: %q", out)
	}
}

func TestSearch_NoMatch_PrintsNoSessionsFound(t *testing.T) {
	s := makeSearchSession("id1", "cache-layer", []string{"redis"}, "Redis caching strategy.", time.Now())
	setupProjectWithSessions(t, []session.Session{s})

	out := captureOutput(t, func() {
		if err := runSearch("kubernetes", ""); err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
	})

	if !strings.Contains(out, "No sessions found") {
		t.Errorf("expected 'No sessions found', got: %q", out)
	}
}

// --- runSearch: case insensitivity -------------------------------------------

func TestSearch_CaseInsensitive_Topic(t *testing.T) {
	s := makeSearchSession("id1", "Database-Migration", []string{}, "Summary.", time.Now())
	setupProjectWithSessions(t, []session.Session{s})

	out := captureOutput(t, func() {
		if err := runSearch("DATABASE", ""); err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
	})

	if !strings.Contains(out, "Database-Migration") {
		t.Errorf("expected topic in output for case-insensitive match, got: %q", out)
	}
}

func TestSearch_CaseInsensitive_Tag(t *testing.T) {
	s := makeSearchSession("id1", "my-topic", []string{"GoLang"}, "Summary.", time.Now())
	setupProjectWithSessions(t, []session.Session{s})

	out := captureOutput(t, func() {
		if err := runSearch("golang", ""); err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
	})

	if !strings.Contains(out, "my-topic") {
		t.Errorf("expected topic in output for case-insensitive tag match, got: %q", out)
	}
}

func TestSearch_CaseInsensitive_Excerpt(t *testing.T) {
	s := makeSearchSession("id1", "api-design", []string{}, "Switched to OpenAPI specification.", time.Now())
	setupProjectWithSessions(t, []session.Session{s})

	out := captureOutput(t, func() {
		if err := runSearch("openapi", ""); err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
	})

	if !strings.Contains(out, "api-design") {
		t.Errorf("expected topic in output for case-insensitive excerpt match, got: %q", out)
	}
}

// --- runSearch: --tag pre-filter ---------------------------------------------

func TestSearch_TagFilter_NarrowsResults(t *testing.T) {
	now := time.Now()
	sessions := []session.Session{
		makeSearchSession("id1", "auth-login", []string{"auth"}, "JWT tokens.", now.Add(-2*time.Hour)),
		makeSearchSession("id2", "payment-flow", []string{"billing"}, "JWT for payments.", now.Add(-1*time.Hour)),
	}
	setupProjectWithSessions(t, sessions)

	out := captureOutput(t, func() {
		if err := runSearch("jwt", "auth"); err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
	})

	// Only the session tagged "auth" should appear.
	if !strings.Contains(out, "auth-login") {
		t.Errorf("expected auth-login in output, got: %q", out)
	}
	if strings.Contains(out, "payment-flow") {
		t.Errorf("expected payment-flow to be excluded by --tag filter, got: %q", out)
	}
}

func TestSearch_TagFilter_NoKeywordMatchAfterTagFilter(t *testing.T) {
	s := makeSearchSession("id1", "auth-service", []string{"auth"}, "OAuth2 flow.", time.Now())
	setupProjectWithSessions(t, []session.Session{s})

	out := captureOutput(t, func() {
		if err := runSearch("kubernetes", "auth"); err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
	})

	if !strings.Contains(out, "No sessions found") {
		t.Errorf("expected 'No sessions found' when keyword has no match after tag filter, got: %q", out)
	}
}

func TestSearch_TagFilter_AllSessionsExcluded(t *testing.T) {
	s := makeSearchSession("id1", "auth-service", []string{"auth"}, "Summary.", time.Now())
	setupProjectWithSessions(t, []session.Session{s})

	out := captureOutput(t, func() {
		if err := runSearch("auth", "unrelated-tag"); err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
	})

	if !strings.Contains(out, "No sessions found") {
		t.Errorf("expected 'No sessions found' when tag filter matches nothing, got: %q", out)
	}
}

// --- runSearch: multiple matches ---------------------------------------------

func TestSearch_MultipleMatches_AllReturned(t *testing.T) {
	now := time.Now()
	sessions := []session.Session{
		makeSearchSession("id1", "auth-login", []string{"auth"}, "Login flow.", now.Add(-2*time.Hour)),
		makeSearchSession("id2", "auth-signup", []string{"auth"}, "Signup flow.", now.Add(-1*time.Hour)),
		makeSearchSession("id3", "cache-layer", []string{"redis"}, "Caching.", now),
	}
	setupProjectWithSessions(t, sessions)

	out := captureOutput(t, func() {
		if err := runSearch("auth", ""); err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
	})

	if !strings.Contains(out, "auth-login") {
		t.Errorf("expected auth-login in output, got: %q", out)
	}
	if !strings.Contains(out, "auth-signup") {
		t.Errorf("expected auth-signup in output, got: %q", out)
	}
	if strings.Contains(out, "cache-layer") {
		t.Errorf("expected cache-layer to be excluded, got: %q", out)
	}
}

// --- runSearch: table output format ------------------------------------------

func TestSearch_Output_ContainsHeaders(t *testing.T) {
	s := makeSearchSession("id1", "api-gateway", []string{"api"}, "Gateway design.", time.Now())
	setupProjectWithSessions(t, []session.Session{s})

	out := captureOutput(t, func() {
		if err := runSearch("api", ""); err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
	})

	if !strings.Contains(out, "DATE") {
		t.Errorf("expected DATE header in table output, got: %q", out)
	}
	if !strings.Contains(out, "TOPIC") {
		t.Errorf("expected TOPIC header in table output, got: %q", out)
	}
	if !strings.Contains(out, "TAGS") {
		t.Errorf("expected TAGS header in table output, got: %q", out)
	}
}

func TestSearch_Output_SortedNewestFirst(t *testing.T) {
	older := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	sessions := []session.Session{
		makeSearchSession("id1", "old-session", []string{"go"}, "Old stuff.", older),
		makeSearchSession("id2", "new-session", []string{"go"}, "New stuff.", newer),
	}
	setupProjectWithSessions(t, sessions)

	out := captureOutput(t, func() {
		if err := runSearch("go", ""); err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
	})

	oldIdx := strings.Index(out, "old-session")
	newIdx := strings.Index(out, "new-session")
	if oldIdx == -1 || newIdx == -1 {
		t.Fatalf("expected both sessions in output, got: %q", out)
	}
	if newIdx > oldIdx {
		t.Errorf("expected newer session to appear before older session in output")
	}
}

// --- filterKeyword unit tests ------------------------------------------------

func TestFilterKeyword_MatchesTopic(t *testing.T) {
	sessions := []session.Session{
		{Topic: "jwt-auth", Tags: []string{}, Excerpt: ""},
		{Topic: "cache-layer", Tags: []string{}, Excerpt: ""},
	}
	result := filterKeyword(sessions, "jwt")
	if len(result) != 1 || result[0].Topic != "jwt-auth" {
		t.Errorf("expected jwt-auth, got %v", result)
	}
}

func TestFilterKeyword_MatchesTag(t *testing.T) {
	sessions := []session.Session{
		{Topic: "topic-a", Tags: []string{"security", "auth"}, Excerpt: ""},
		{Topic: "topic-b", Tags: []string{"redis"}, Excerpt: ""},
	}
	result := filterKeyword(sessions, "auth")
	if len(result) != 1 || result[0].Topic != "topic-a" {
		t.Errorf("expected topic-a, got %v", result)
	}
}

func TestFilterKeyword_MatchesExcerpt(t *testing.T) {
	sessions := []session.Session{
		{Topic: "topic-a", Tags: []string{}, Excerpt: "We adopted event sourcing."},
		{Topic: "topic-b", Tags: []string{}, Excerpt: "Standard REST approach."},
	}
	result := filterKeyword(sessions, "event sourcing")
	if len(result) != 1 || result[0].Topic != "topic-a" {
		t.Errorf("expected topic-a, got %v", result)
	}
}

func TestFilterKeyword_NoMatch(t *testing.T) {
	sessions := []session.Session{
		{Topic: "foo", Tags: []string{"bar"}, Excerpt: "baz"},
	}
	result := filterKeyword(sessions, "zzz")
	if len(result) != 0 {
		t.Errorf("expected no matches, got %d", len(result))
	}
}

func TestFilterKeyword_EmptySessions(t *testing.T) {
	result := filterKeyword(nil, "anything")
	if len(result) != 0 {
		t.Errorf("expected empty result for nil sessions, got %d", len(result))
	}
}

func TestFilterKeyword_CaseInsensitive(t *testing.T) {
	sessions := []session.Session{
		{Topic: "GraphQL-Migration", Tags: []string{}, Excerpt: ""},
	}
	result := filterKeyword(sessions, "GRAPHQL")
	if len(result) != 1 {
		t.Errorf("expected 1 case-insensitive match, got %d", len(result))
	}
}

func TestFilterKeyword_MultipleMatches(t *testing.T) {
	sessions := []session.Session{
		{Topic: "auth-login", Tags: []string{"auth"}, Excerpt: "Login."},
		{Topic: "auth-signup", Tags: []string{"auth"}, Excerpt: "Signup."},
		{Topic: "payments", Tags: []string{"billing"}, Excerpt: "Stripe."},
	}
	result := filterKeyword(sessions, "auth")
	if len(result) != 2 {
		t.Errorf("expected 2 matches, got %d", len(result))
	}
}

// --- sessionMatchesKeyword unit tests ----------------------------------------

func TestSessionMatchesKeyword_TopicOnly(t *testing.T) {
	s := session.Session{Topic: "database-migration", Tags: []string{}, Excerpt: ""}
	if !sessionMatchesKeyword(s, "database") {
		t.Error("expected match on topic substring")
	}
}

func TestSessionMatchesKeyword_TagOnly(t *testing.T) {
	s := session.Session{Topic: "unrelated", Tags: []string{"golang", "testing"}, Excerpt: ""}
	if !sessionMatchesKeyword(s, "testing") {
		t.Error("expected match on tag")
	}
}

func TestSessionMatchesKeyword_ExcerptOnly(t *testing.T) {
	s := session.Session{Topic: "unrelated", Tags: []string{}, Excerpt: "Decided to use Postgres."}
	if !sessionMatchesKeyword(s, "postgres") {
		t.Error("expected case-insensitive match on excerpt")
	}
}

func TestSessionMatchesKeyword_NoMatch(t *testing.T) {
	s := session.Session{Topic: "foo", Tags: []string{"bar"}, Excerpt: "baz"}
	if sessionMatchesKeyword(s, "zzz") {
		t.Error("expected no match")
	}
}

func TestSessionMatchesKeyword_EmptyKeyword_MatchesAll(t *testing.T) {
	s := session.Session{Topic: "foo", Tags: []string{}, Excerpt: ""}
	// Empty string is a substring of everything.
	if !sessionMatchesKeyword(s, "") {
		t.Error("expected empty keyword to match all sessions")
	}
}
