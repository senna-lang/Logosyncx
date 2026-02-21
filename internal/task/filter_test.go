package task

import (
	"testing"
	"time"
)

// --- helpers -----------------------------------------------------------------

func makeFilterTask(id, title string, status Status, priority Priority, session string, tags []string, excerpt string) *Task {
	return &Task{
		ID:       id,
		Date:     time.Now(),
		Title:    title,
		Status:   status,
		Priority: priority,
		Session:  session,
		Tags:     tags,
		Excerpt:  excerpt,
	}
}

// --- Apply: no filter --------------------------------------------------------

func TestApply_NoFilter_ReturnsAll(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "task-a", StatusOpen, PriorityHigh, "", nil, ""),
		makeFilterTask("t-2", "task-b", StatusInProgress, PriorityMedium, "", nil, ""),
	}
	got := Apply(tasks, Filter{})
	if len(got) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(got))
	}
}

func TestApply_NilInput_ReturnsNil(t *testing.T) {
	got := Apply(nil, Filter{})
	if len(got) != 0 {
		t.Errorf("expected empty result for nil input, got %d", len(got))
	}
}

func TestApply_EmptyInput_ReturnsEmpty(t *testing.T) {
	got := Apply([]*Task{}, Filter{})
	if len(got) != 0 {
		t.Errorf("expected empty result, got %d", len(got))
	}
}

// --- Apply: Session filter ---------------------------------------------------

func TestApply_SessionFilter_MatchesSubstring(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "auth-task", StatusOpen, PriorityMedium, "2025-02-20_auth-refactor.md", nil, ""),
		makeFilterTask("t-2", "db-task", StatusOpen, PriorityMedium, "2025-02-18_db-schema.md", nil, ""),
	}
	got := Apply(tasks, Filter{Session: "auth"})
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %d", len(got))
	}
	if got[0].Title != "auth-task" {
		t.Errorf("expected 'auth-task', got %q", got[0].Title)
	}
}

func TestApply_SessionFilter_CaseInsensitive(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "task", StatusOpen, PriorityMedium, "2025-02-20_AUTH-refactor.md", nil, ""),
	}
	got := Apply(tasks, Filter{Session: "auth"})
	if len(got) != 1 {
		t.Errorf("expected 1 match for case-insensitive session filter, got %d", len(got))
	}
}

func TestApply_SessionFilter_NoMatch(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "task", StatusOpen, PriorityMedium, "2025-02-20_auth.md", nil, ""),
	}
	got := Apply(tasks, Filter{Session: "postgres"})
	if len(got) != 0 {
		t.Errorf("expected 0 matches, got %d", len(got))
	}
}

func TestApply_SessionFilter_EmptySession_NotFiltered(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "task", StatusOpen, PriorityMedium, "", nil, ""),
	}
	// Empty filter.Session means "no constraint", so the task should pass.
	got := Apply(tasks, Filter{})
	if len(got) != 1 {
		t.Errorf("expected 1 task when no session filter, got %d", len(got))
	}
}

// --- Apply: Status filter ----------------------------------------------------

func TestApply_StatusFilter_ExactMatch(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "open-task", StatusOpen, PriorityMedium, "", nil, ""),
		makeFilterTask("t-2", "wip-task", StatusInProgress, PriorityMedium, "", nil, ""),
		makeFilterTask("t-3", "done-task", StatusDone, PriorityMedium, "", nil, ""),
	}
	got := Apply(tasks, Filter{Status: StatusOpen})
	if len(got) != 1 {
		t.Fatalf("expected 1 open task, got %d", len(got))
	}
	if got[0].Status != StatusOpen {
		t.Errorf("got status %q, want %q", got[0].Status, StatusOpen)
	}
}

func TestApply_StatusFilter_InProgress(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "open-task", StatusOpen, PriorityMedium, "", nil, ""),
		makeFilterTask("t-2", "wip-task", StatusInProgress, PriorityMedium, "", nil, ""),
	}
	got := Apply(tasks, Filter{Status: StatusInProgress})
	if len(got) != 1 || got[0].Title != "wip-task" {
		t.Errorf("expected 'wip-task', got %v", got)
	}
}

func TestApply_StatusFilter_NoMatch(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "task", StatusOpen, PriorityMedium, "", nil, ""),
	}
	got := Apply(tasks, Filter{Status: StatusCancelled})
	if len(got) != 0 {
		t.Errorf("expected 0 matches, got %d", len(got))
	}
}

func TestApply_StatusFilter_Empty_MatchesAll(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "a", StatusOpen, PriorityMedium, "", nil, ""),
		makeFilterTask("t-2", "b", StatusInProgress, PriorityMedium, "", nil, ""),
	}
	got := Apply(tasks, Filter{Status: ""})
	if len(got) != 2 {
		t.Errorf("empty status filter should match all, got %d", len(got))
	}
}

// --- Apply: Priority filter --------------------------------------------------

func TestApply_PriorityFilter_High(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "high-task", StatusOpen, PriorityHigh, "", nil, ""),
		makeFilterTask("t-2", "low-task", StatusOpen, PriorityLow, "", nil, ""),
	}
	got := Apply(tasks, Filter{Priority: PriorityHigh})
	if len(got) != 1 || got[0].Title != "high-task" {
		t.Errorf("expected 'high-task', got %v", got)
	}
}

func TestApply_PriorityFilter_Medium(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "med-task", StatusOpen, PriorityMedium, "", nil, ""),
		makeFilterTask("t-2", "high-task", StatusOpen, PriorityHigh, "", nil, ""),
	}
	got := Apply(tasks, Filter{Priority: PriorityMedium})
	if len(got) != 1 || got[0].Title != "med-task" {
		t.Errorf("expected 'med-task', got %v", got)
	}
}

func TestApply_PriorityFilter_Empty_MatchesAll(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "a", StatusOpen, PriorityHigh, "", nil, ""),
		makeFilterTask("t-2", "b", StatusOpen, PriorityLow, "", nil, ""),
	}
	got := Apply(tasks, Filter{Priority: ""})
	if len(got) != 2 {
		t.Errorf("empty priority filter should match all, got %d", len(got))
	}
}

// --- Apply: Tags filter ------------------------------------------------------

func TestApply_TagsFilter_SingleTag(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "auth-task", StatusOpen, PriorityMedium, "", []string{"auth", "jwt"}, ""),
		makeFilterTask("t-2", "db-task", StatusOpen, PriorityMedium, "", []string{"postgres"}, ""),
	}
	got := Apply(tasks, Filter{Tags: []string{"auth"}})
	if len(got) != 1 || got[0].Title != "auth-task" {
		t.Errorf("expected 'auth-task', got %v", got)
	}
}

func TestApply_TagsFilter_AnyTagMatches(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "task-a", StatusOpen, PriorityMedium, "", []string{"auth"}, ""),
		makeFilterTask("t-2", "task-b", StatusOpen, PriorityMedium, "", []string{"security"}, ""),
		makeFilterTask("t-3", "task-c", StatusOpen, PriorityMedium, "", []string{"postgres"}, ""),
	}
	// Filter with two tags — task must match at least one.
	got := Apply(tasks, Filter{Tags: []string{"auth", "security"}})
	if len(got) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(got))
	}
}

func TestApply_TagsFilter_CaseInsensitive(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "task", StatusOpen, PriorityMedium, "", []string{"GoLang"}, ""),
	}
	got := Apply(tasks, Filter{Tags: []string{"golang"}})
	if len(got) != 1 {
		t.Errorf("expected case-insensitive tag match, got %d", len(got))
	}
}

func TestApply_TagsFilter_NoMatch(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "task", StatusOpen, PriorityMedium, "", []string{"auth"}, ""),
	}
	got := Apply(tasks, Filter{Tags: []string{"redis"}})
	if len(got) != 0 {
		t.Errorf("expected 0 matches, got %d", len(got))
	}
}

func TestApply_TagsFilter_Empty_MatchesAll(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "a", StatusOpen, PriorityMedium, "", []string{"auth"}, ""),
		makeFilterTask("t-2", "b", StatusOpen, PriorityMedium, "", []string{"postgres"}, ""),
	}
	got := Apply(tasks, Filter{Tags: nil})
	if len(got) != 2 {
		t.Errorf("empty tags filter should match all, got %d", len(got))
	}
}

func TestApply_TagsFilter_TaskWithNoTags_NoMatch(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "task", StatusOpen, PriorityMedium, "", nil, ""),
	}
	got := Apply(tasks, Filter{Tags: []string{"auth"}})
	if len(got) != 0 {
		t.Errorf("task with no tags should not match tag filter, got %d", len(got))
	}
}

// --- Apply: Keyword filter ---------------------------------------------------

func TestApply_KeywordFilter_MatchesTitle(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "jwt-authentication", StatusOpen, PriorityMedium, "", nil, ""),
		makeFilterTask("t-2", "cache-layer", StatusOpen, PriorityMedium, "", nil, ""),
	}
	got := Apply(tasks, Filter{Keyword: "jwt"})
	if len(got) != 1 || got[0].Title != "jwt-authentication" {
		t.Errorf("expected 'jwt-authentication', got %v", got)
	}
}

func TestApply_KeywordFilter_MatchesTag(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "some-task", StatusOpen, PriorityMedium, "", []string{"security", "auth"}, ""),
		makeFilterTask("t-2", "other-task", StatusOpen, PriorityMedium, "", []string{"redis"}, ""),
	}
	got := Apply(tasks, Filter{Keyword: "security"})
	if len(got) != 1 || got[0].Title != "some-task" {
		t.Errorf("expected 'some-task', got %v", got)
	}
}

func TestApply_KeywordFilter_MatchesExcerpt(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "refactor", StatusOpen, PriorityMedium, "", nil, "Migrate from REST to GraphQL."),
		makeFilterTask("t-2", "unrelated", StatusOpen, PriorityMedium, "", nil, "Standard CRUD operations."),
	}
	got := Apply(tasks, Filter{Keyword: "graphql"})
	if len(got) != 1 || got[0].Title != "refactor" {
		t.Errorf("expected 'refactor', got %v", got)
	}
}

func TestApply_KeywordFilter_CaseInsensitive(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "Database-Migration", StatusOpen, PriorityMedium, "", nil, ""),
	}
	got := Apply(tasks, Filter{Keyword: "DATABASE"})
	if len(got) != 1 {
		t.Errorf("expected case-insensitive keyword match, got %d", len(got))
	}
}

func TestApply_KeywordFilter_NoMatch(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "auth-task", StatusOpen, PriorityMedium, "", []string{"auth"}, "Login flow."),
	}
	got := Apply(tasks, Filter{Keyword: "kubernetes"})
	if len(got) != 0 {
		t.Errorf("expected 0 matches, got %d", len(got))
	}
}

func TestApply_KeywordFilter_Empty_MatchesAll(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "a", StatusOpen, PriorityMedium, "", nil, ""),
		makeFilterTask("t-2", "b", StatusOpen, PriorityMedium, "", nil, ""),
	}
	got := Apply(tasks, Filter{Keyword: ""})
	if len(got) != 2 {
		t.Errorf("empty keyword filter should match all, got %d", len(got))
	}
}

// --- Apply: combined filters -------------------------------------------------

func TestApply_CombinedStatusAndPriority(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "high-open", StatusOpen, PriorityHigh, "", nil, ""),
		makeFilterTask("t-2", "med-open", StatusOpen, PriorityMedium, "", nil, ""),
		makeFilterTask("t-3", "high-wip", StatusInProgress, PriorityHigh, "", nil, ""),
	}
	got := Apply(tasks, Filter{Status: StatusOpen, Priority: PriorityHigh})
	if len(got) != 1 || got[0].Title != "high-open" {
		t.Errorf("expected 'high-open', got %v", got)
	}
}

func TestApply_CombinedKeywordAndStatus(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "auth-open", StatusOpen, PriorityMedium, "", []string{"auth"}, ""),
		makeFilterTask("t-2", "auth-wip", StatusInProgress, PriorityMedium, "", []string{"auth"}, ""),
		makeFilterTask("t-3", "cache-open", StatusOpen, PriorityMedium, "", []string{"redis"}, ""),
	}
	got := Apply(tasks, Filter{Status: StatusOpen, Keyword: "auth"})
	if len(got) != 1 || got[0].Title != "auth-open" {
		t.Errorf("expected 'auth-open', got %v", got)
	}
}

func TestApply_CombinedTagAndKeyword(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "jwt-login", StatusOpen, PriorityMedium, "", []string{"auth"}, "JWT tokens used."),
		makeFilterTask("t-2", "jwt-payment", StatusOpen, PriorityMedium, "", []string{"billing"}, "JWT for payments."),
	}
	// Only auth-tagged tasks containing "jwt".
	got := Apply(tasks, Filter{Tags: []string{"auth"}, Keyword: "jwt"})
	if len(got) != 1 || got[0].Title != "jwt-login" {
		t.Errorf("expected 'jwt-login', got %v", got)
	}
}

func TestApply_CombinedSessionAndStatus(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "task-a", StatusOpen, PriorityMedium, "auth-refactor.md", nil, ""),
		makeFilterTask("t-2", "task-b", StatusInProgress, PriorityMedium, "auth-refactor.md", nil, ""),
		makeFilterTask("t-3", "task-c", StatusOpen, PriorityMedium, "db-schema.md", nil, ""),
	}
	got := Apply(tasks, Filter{Session: "auth", Status: StatusOpen})
	if len(got) != 1 || got[0].Title != "task-a" {
		t.Errorf("expected 'task-a', got %v", got)
	}
}

func TestApply_AllFiltersActive_NarrowsToOne(t *testing.T) {
	tasks := []*Task{
		makeFilterTask("t-1", "auth-login", StatusOpen, PriorityHigh, "auth-session.md", []string{"auth", "jwt"}, "Implement login flow."),
		makeFilterTask("t-2", "auth-signup", StatusOpen, PriorityMedium, "auth-session.md", []string{"auth"}, "Implement signup flow."),
		makeFilterTask("t-3", "cache-layer", StatusInProgress, PriorityHigh, "cache-session.md", []string{"redis"}, "Redis caching."),
	}
	got := Apply(tasks, Filter{
		Session:  "auth",
		Status:   StatusOpen,
		Priority: PriorityHigh,
		Tags:     []string{"jwt"},
		Keyword:  "login",
	})
	if len(got) != 1 || got[0].Title != "auth-login" {
		t.Errorf("expected only 'auth-login', got %v", got)
	}
}

// --- matchesKeyword unit tests -----------------------------------------------

func TestMatchesKeyword_MatchesTitle(t *testing.T) {
	tk := &Task{Title: "database-migration", Tags: nil, Excerpt: ""}
	if !matchesKeyword(tk, "database") {
		t.Error("expected match on title substring")
	}
}

func TestMatchesKeyword_MatchesTag(t *testing.T) {
	tk := &Task{Title: "unrelated", Tags: []string{"golang", "testing"}, Excerpt: ""}
	if !matchesKeyword(tk, "testing") {
		t.Error("expected match on tag")
	}
}

func TestMatchesKeyword_MatchesExcerpt(t *testing.T) {
	tk := &Task{Title: "unrelated", Tags: nil, Excerpt: "Decided to use Postgres."}
	if !matchesKeyword(tk, "postgres") {
		t.Error("expected case-insensitive match on excerpt")
	}
}

func TestMatchesKeyword_NoMatch(t *testing.T) {
	tk := &Task{Title: "foo", Tags: []string{"bar"}, Excerpt: "baz"}
	if matchesKeyword(tk, "zzz") {
		t.Error("expected no match")
	}
}

func TestMatchesKeyword_EmptyKeyword_MatchesAll(t *testing.T) {
	tk := &Task{Title: "foo", Tags: nil, Excerpt: ""}
	if !matchesKeyword(tk, "") {
		t.Error("expected empty keyword to match all tasks")
	}
}

// --- hasAnyTag unit tests ----------------------------------------------------

func TestHasAnyTag_Match(t *testing.T) {
	if !hasAnyTag([]string{"auth", "jwt"}, []string{"jwt"}) {
		t.Error("expected match on 'jwt'")
	}
}

func TestHasAnyTag_NoMatch(t *testing.T) {
	if hasAnyTag([]string{"auth"}, []string{"redis"}) {
		t.Error("expected no match")
	}
}

func TestHasAnyTag_EmptyWant(t *testing.T) {
	if hasAnyTag([]string{"auth"}, []string{}) {
		t.Error("empty want list should never match")
	}
}

func TestHasAnyTag_EmptyHave(t *testing.T) {
	if hasAnyTag(nil, []string{"auth"}) {
		t.Error("empty have list should never match")
	}
}

func TestHasAnyTag_CaseInsensitive(t *testing.T) {
	if !hasAnyTag([]string{"GoLang"}, []string{"golang"}) {
		t.Error("expected case-insensitive tag match")
	}
}
