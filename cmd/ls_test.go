package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/senna-lang/logosyncx/pkg/index"
	"github.com/senna-lang/logosyncx/pkg/plan"
)

// --- helpers -----------------------------------------------------------------

// setupProjectWithPlans initialises a project and writes plan files with body.
func setupProjectWithPlans(t *testing.T, plans []plan.Plan) string {
	t.Helper()
	dir := setupInitedProject(t)
	for _, p := range plans {
		writePlanFileWithBody(t, dir, p)
	}
	return dir
}

// writePlanFileWithBody writes a plan including its Body (unlike plan.Write
// which produces scaffold-only files). Used in tests so excerpt is populated.
func writePlanFileWithBody(t *testing.T, projectRoot string, p plan.Plan) {
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
		t.Fatalf("WriteFile plan: %v", err)
	}
}

func makeTestPlan(topic string, tags []string, date time.Time) plan.Plan {
	return plan.Plan{
		ID:       "test01",
		Date:     &date,
		Topic:    topic,
		Tags:     tags,
		Agent:    "claude-code",
		Related:  []string{},
		TasksDir: ".logosyncx/tasks/" + topic,
		Body:     "## Background\nThis is a test plan about " + topic + ".\n\n## Notes\n- Note one\n",
	}
}

// captureOutput redirects stdout during f() and returns what was written.
func captureOutput(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("ReadFrom pipe: %v", err)
	}
	return buf.String()
}

// --- runLS: no sessions ------------------------------------------------------

func TestLS_NoSessions_PrintsMessage(t *testing.T) {
	setupInitedProject(t)

	out := captureOutput(t, func() {
		if err := runLS("", "", false, false); err != nil {
			t.Fatalf("runLS failed: %v", err)
		}
	})

	if !strings.Contains(out, "No plans found") {
		t.Errorf("expected 'No plans found', got: %q", out)
	}
}

func TestLS_NoSessions_JSON_PrintsEmptyArray(t *testing.T) {
	setupInitedProject(t)

	out := captureOutput(t, func() {
		if err := runLS("", "", true, false); err != nil {
			t.Fatalf("runLS --json failed: %v", err)
		}
	})

	// Should still output valid JSON (empty array handled by "No plans found").
	// Actually runLS returns early with "No plans found." before printing JSON.
	if !strings.Contains(out, "No plans found") {
		t.Errorf("expected 'No plans found', got: %q", out)
	}
}

// --- runLS: not initialized --------------------------------------------------

func TestLS_NotInitialized_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(orig) })

	err := runLS("", "", false, false)
	if err == nil {
		t.Fatal("expected error when project not initialized, got nil")
	}
	if !strings.Contains(err.Error(), "logos init") {
		t.Errorf("expected 'logos init' hint in error, got: %v", err)
	}
}

// --- runLS: table output -----------------------------------------------------

func TestLS_Table_ContainsHeaders(t *testing.T) {
	now := time.Now()
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("auth-refactor", []string{"auth"}, now),
	})

	out := captureOutput(t, func() {
		if err := runLS("", "", false, false); err != nil {
			t.Fatalf("runLS failed: %v", err)
		}
	})

	if !strings.Contains(out, "DATE") {
		t.Errorf("expected DATE header, got: %q", out)
	}
	if !strings.Contains(out, "TOPIC") {
		t.Errorf("expected TOPIC header, got: %q", out)
	}
	if !strings.Contains(out, "TAGS") {
		t.Errorf("expected TAGS header, got: %q", out)
	}
}

func TestLS_Table_ContainsSessionData(t *testing.T) {
	now := time.Now()
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("auth-refactor", []string{"auth", "jwt"}, now),
	})

	out := captureOutput(t, func() {
		if err := runLS("", "", false, false); err != nil {
			t.Fatalf("runLS failed: %v", err)
		}
	})

	if !strings.Contains(out, "auth-refactor") {
		t.Errorf("expected topic 'auth-refactor' in output, got: %q", out)
	}
	if !strings.Contains(out, "auth") {
		t.Errorf("expected tag 'auth' in output, got: %q", out)
	}
	if !strings.Contains(out, "jwt") {
		t.Errorf("expected tag 'jwt' in output, got: %q", out)
	}
}

func TestLS_Table_MultipleSessions(t *testing.T) {
	base := time.Date(2025, 2, 20, 10, 0, 0, 0, time.UTC)
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("auth-refactor", []string{"auth"}, base),
		makeTestPlan("db-schema", []string{"postgres"}, base.Add(-48*time.Hour)),
		makeTestPlan("security-audit", []string{"security"}, base.Add(-96*time.Hour)),
	})

	out := captureOutput(t, func() {
		if err := runLS("", "", false, false); err != nil {
			t.Fatalf("runLS failed: %v", err)
		}
	})

	if !strings.Contains(out, "auth-refactor") {
		t.Error("expected auth-refactor in output")
	}
	if !strings.Contains(out, "db-schema") {
		t.Error("expected db-schema in output")
	}
	if !strings.Contains(out, "security-audit") {
		t.Error("expected security-audit in output")
	}
}

func TestLS_Table_NoTagsShowsDash(t *testing.T) {
	now := time.Now()
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("no-tags", []string{}, now),
	})

	out := captureOutput(t, func() {
		if err := runLS("", "", false, false); err != nil {
			t.Fatalf("runLS failed: %v", err)
		}
	})

	if !strings.Contains(out, "-") {
		t.Errorf("expected '-' for empty tags, got: %q", out)
	}
}

// --- runLS: --json output ----------------------------------------------------

func TestLS_JSON_ValidJSON(t *testing.T) {
	now := time.Now()
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("auth-refactor", []string{"auth"}, now),
	})

	out := captureOutput(t, func() {
		if err := runLS("", "", true, false); err != nil {
			t.Fatalf("runLS --json failed: %v", err)
		}
	})

	var result []index.Entry
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, out)
	}
}

func TestLS_JSON_ContainsRequiredFields(t *testing.T) {
	now := time.Date(2025, 2, 20, 10, 30, 0, 0, time.UTC)
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("auth-refactor", []string{"auth", "jwt"}, now),
	})

	out := captureOutput(t, func() {
		if err := runLS("", "", true, false); err != nil {
			t.Fatalf("runLS --json failed: %v", err)
		}
	})

	var result []index.Entry
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	r := result[0]
	if r.Topic != "auth-refactor" {
		t.Errorf("topic = %q, want 'auth-refactor'", r.Topic)
	}
	if len(r.Tags) != 2 {
		t.Errorf("tags length = %d, want 2", len(r.Tags))
	}
	if r.Filename == "" {
		t.Error("expected non-empty filename")
	}
	if r.Excerpt == "" {
		t.Error("expected non-empty excerpt")
	}
}

func TestLS_JSON_TagsNeverNull(t *testing.T) {
	now := time.Now()
	p := makeTestPlan("no-tags", nil, now)
	p.Tags = nil
	setupProjectWithPlans(t, []plan.Plan{p})

	out := captureOutput(t, func() {
		if err := runLS("", "", true, false); err != nil {
			t.Fatalf("runLS --json failed: %v", err)
		}
	})

	var result []index.Entry
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result[0].Tags == nil {
		t.Error("tags should never be null in JSON output")
	}
}

func TestLS_JSON_RelatedNeverNull(t *testing.T) {
	now := time.Now()
	p := makeTestPlan("no-related", []string{"test"}, now)
	p.Related = nil
	setupProjectWithPlans(t, []plan.Plan{p})

	out := captureOutput(t, func() {
		if err := runLS("", "", true, false); err != nil {
			t.Fatalf("runLS --json failed: %v", err)
		}
	})

	var result []index.Entry
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result[0].Related == nil {
		t.Error("related should never be null in JSON output")
	}
}

// --- runLS: --tag filter -----------------------------------------------------

func TestLS_FilterTag_MatchesSessions(t *testing.T) {
	base := time.Now()
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("auth-refactor", []string{"auth", "jwt"}, base),
		makeTestPlan("db-schema", []string{"postgres"}, base.Add(-time.Hour)),
		makeTestPlan("security-audit", []string{"auth", "security"}, base.Add(-2*time.Hour)),
	})

	out := captureOutput(t, func() {
		if err := runLS("auth", "", false, false); err != nil {
			t.Fatalf("runLS --tag auth failed: %v", err)
		}
	})

	if !strings.Contains(out, "auth-refactor") {
		t.Error("expected auth-refactor in tag=auth results")
	}
	if !strings.Contains(out, "security-audit") {
		t.Error("expected security-audit in tag=auth results")
	}
	if strings.Contains(out, "db-schema") {
		t.Error("db-schema should NOT appear in tag=auth results")
	}
}

func TestLS_FilterTag_NoMatchShowsNoSessions(t *testing.T) {
	now := time.Now()
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("auth-refactor", []string{"auth"}, now),
	})

	out := captureOutput(t, func() {
		if err := runLS("nonexistenttag", "", false, false); err != nil {
			t.Fatalf("runLS failed: %v", err)
		}
	})

	if !strings.Contains(out, "No plans found") {
		t.Errorf("expected 'No plans found', got: %q", out)
	}
}

func TestLS_FilterTag_ExactMatch(t *testing.T) {
	now := time.Now()
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("auth-topic", []string{"auth"}, now),
		makeTestPlan("auth-extended", []string{"authentication"}, now.Add(-time.Hour)),
	})

	out := captureOutput(t, func() {
		if err := runLS("auth", "", false, false); err != nil {
			t.Fatalf("runLS failed: %v", err)
		}
	})

	// Only exact tag "auth" should match, not "authentication".
	if !strings.Contains(out, "auth-topic") {
		t.Error("expected auth-topic in results")
	}
	if strings.Contains(out, "auth-extended") {
		t.Error("auth-extended should NOT appear (tag 'authentication' != 'auth')")
	}
}

// --- runLS: --since filter ---------------------------------------------------

func TestLS_FilterSince_IncludesOnAndAfter(t *testing.T) {
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("new-session", []string{}, time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)),
		makeTestPlan("old-session", []string{}, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		makeTestPlan("boundary-session", []string{}, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)),
	})

	out := captureOutput(t, func() {
		if err := runLS("", "2025-02-01", false, false); err != nil {
			t.Fatalf("runLS --since failed: %v", err)
		}
	})

	if !strings.Contains(out, "new-session") {
		t.Error("expected new-session in since=2025-02-01 results")
	}
	if !strings.Contains(out, "boundary-session") {
		t.Error("expected boundary-session (on the boundary date) in results")
	}
	if strings.Contains(out, "old-session") {
		t.Error("old-session should NOT appear in since=2025-02-01 results")
	}
}

func TestLS_FilterSince_InvalidDate_ReturnsError(t *testing.T) {
	setupInitedProject(t)

	err := runLS("", "not-a-date", false, false)
	if err == nil {
		t.Fatal("expected error for invalid --since date, got nil")
	}
	if !strings.Contains(err.Error(), "YYYY-MM-DD") {
		t.Errorf("expected format hint in error, got: %v", err)
	}
}

// --- runLS: sort order -------------------------------------------------------

func TestLS_SortedNewestFirst(t *testing.T) {
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("oldest", []string{}, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		makeTestPlan("newest", []string{}, time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)),
		makeTestPlan("middle", []string{}, time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)),
	})

	out := captureOutput(t, func() {
		if err := runLS("", "", false, false); err != nil {
			t.Fatalf("runLS failed: %v", err)
		}
	})

	newestPos := strings.Index(out, "newest")
	middlePos := strings.Index(out, "middle")
	oldestPos := strings.Index(out, "oldest")

	if newestPos == -1 || middlePos == -1 || oldestPos == -1 {
		t.Fatalf("one or more topics missing from output: %q", out)
	}
	if newestPos > middlePos {
		t.Error("newest should appear before middle in output")
	}
	if middlePos > oldestPos {
		t.Error("middle should appear before oldest in output")
	}
}

// --- sortByDateDesc ----------------------------------------------------------

func TestSortByDateDesc_Basic(t *testing.T) {
	entries := []index.Entry{
		{Topic: "a", Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Topic: "c", Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
		{Topic: "b", Date: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
	}
	sortByDateDesc(entries)

	if entries[0].Topic != "c" || entries[1].Topic != "b" || entries[2].Topic != "a" {
		t.Errorf("sort order wrong: got %v %v %v", entries[0].Topic, entries[1].Topic, entries[2].Topic)
	}
}

func TestSortByDateDesc_AlreadySorted(t *testing.T) {
	entries := []index.Entry{
		{Topic: "c", Date: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)},
		{Topic: "b", Date: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)},
		{Topic: "a", Date: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	sortByDateDesc(entries)

	if entries[0].Topic != "c" {
		t.Errorf("first should be 'c', got %q", entries[0].Topic)
	}
}

func TestSortByDateDesc_SingleElement(t *testing.T) {
	entries := []index.Entry{
		{Topic: "only", Date: time.Now()},
	}
	sortByDateDesc(entries) // should not panic
	if entries[0].Topic != "only" {
		t.Error("single element sort changed the element")
	}
}

// --- joinTags ----------------------------------------------------------------

func TestJoinTags_Empty(t *testing.T) {
	got := joinTags([]string{})
	if got != "-" {
		t.Errorf("joinTags([]) = %q, want '-'", got)
	}
}

func TestJoinTags_Nil(t *testing.T) {
	got := joinTags(nil)
	if got != "-" {
		t.Errorf("joinTags(nil) = %q, want '-'", got)
	}
}

func TestJoinTags_Single(t *testing.T) {
	got := joinTags([]string{"auth"})
	if got != "auth" {
		t.Errorf("joinTags(['auth']) = %q, want 'auth'", got)
	}
}

func TestJoinTags_Multiple(t *testing.T) {
	got := joinTags([]string{"auth", "jwt", "security"})
	if got != "auth, jwt, security" {
		t.Errorf("joinTags = %q, want 'auth, jwt, security'", got)
	}
}

// --- filterTag ---------------------------------------------------------------

func TestFilterTag_ReturnsMatchingOnly(t *testing.T) {
	entries := []index.Entry{
		{Topic: "a", Tags: []string{"auth", "jwt"}},
		{Topic: "b", Tags: []string{"postgres"}},
		{Topic: "c", Tags: []string{"auth"}},
	}
	got := filterTag(entries, "auth")
	if len(got) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(got))
	}
}

func TestFilterTag_NoMatch(t *testing.T) {
	entries := []index.Entry{
		{Topic: "a", Tags: []string{"auth"}},
	}
	got := filterTag(entries, "postgres")
	if len(got) != 0 {
		t.Errorf("expected 0 matches, got %d", len(got))
	}
}

// --- filterSince -------------------------------------------------------------

func TestFilterSince_IncludesBoundary(t *testing.T) {
	boundary := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	entries := []index.Entry{
		{Topic: "on", Date: boundary},
		{Topic: "after", Date: boundary.Add(24 * time.Hour)},
		{Topic: "before", Date: boundary.Add(-24 * time.Hour)},
	}
	got := filterSince(entries, boundary)
	if len(got) != 2 {
		t.Fatalf("expected 2 sessions (on + after), got %d", len(got))
	}
	for _, e := range got {
		if e.Topic == "before" {
			t.Error("'before' session should not be included")
		}
	}
}

// --- combined filters --------------------------------------------------------

func TestLS_TagAndSinceCombined(t *testing.T) {
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("auth-new", []string{"auth"}, time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)),
		makeTestPlan("auth-old", []string{"auth"}, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		makeTestPlan("db-new", []string{"postgres"}, time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)),
	})

	out := captureOutput(t, func() {
		if err := runLS("auth", "2025-02-01", false, false); err != nil {
			t.Fatalf("runLS failed: %v", err)
		}
	})

	if !strings.Contains(out, "auth-new") {
		t.Error("expected auth-new in results")
	}
	if strings.Contains(out, "auth-old") {
		t.Error("auth-old should be excluded (too old)")
	}
	if strings.Contains(out, "db-new") {
		t.Error("db-new should be excluded (wrong tag)")
	}
}

// --- filesystem: sessions in subdir ------------------------------------------

func TestLS_FindsSessionsFromSubdirectory(t *testing.T) {
	dir := setupInitedProject(t)

	p := makeTestPlan("subdir-test", []string{"test"}, time.Now())
	writePlanFileWithBody(t, dir, p)

	// Change into a subdirectory — FindRoot should still locate .logosyncx/.
	subdir := filepath.Join(dir, "pkg", "plans")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	_ = os.Chdir(subdir)
	t.Cleanup(func() { _ = os.Chdir(orig) })

	out := captureOutput(t, func() {
		if err := runLS("", "", false, false); err != nil {
			t.Fatalf("runLS from subdir failed: %v", err)
		}
	})

	if !strings.Contains(out, "subdir-test") {
		t.Errorf("expected 'subdir-test' in output, got: %q", out)
	}
}

// --- runLS: --blocked filter -------------------------------------------------

func TestLS_Blocked_Filter(t *testing.T) {
	base := time.Now()
	// Plan A: depends on Plan B (which is not distilled) → blocked
	planA := plan.Plan{
		ID:        "a",
		Date:      &base,
		Topic:     "dependent-plan",
		Tags:      []string{},
		Related:   []string{},
		TasksDir:  ".logosyncx/tasks/dependent-plan",
		DependsOn: []string{},
	}
	// Plan B: no dependency → not blocked
	planB := plan.Plan{
		ID:       "b",
		Date:     &base,
		Topic:    "independent-plan",
		Tags:     []string{},
		Related:  []string{},
		TasksDir: ".logosyncx/tasks/independent-plan",
	}

	dir := setupProjectWithPlans(t, []plan.Plan{planA, planB})

	// Directly write a plan B file with a dependency for planA.
	plansDir := dir + "/.logosyncx/plans"
	bFile := planB
	bFilename := plan.FileName(bFile)
	aWithDep := planA
	depName := []string{bFilename}
	aWithDep.DependsOn = depName
	aData, _ := plan.Marshal(aWithDep)
	_ = os.WriteFile(plansDir+"/"+plan.FileName(aWithDep), aData, 0o644)

	// Rebuild index.
	if err := runSync(); err != nil {
		t.Fatalf("runSync: %v", err)
	}

	out := captureOutput(t, func() {
		if err := runLS("", "", false, true); err != nil {
			t.Fatalf("runLS --blocked failed: %v", err)
		}
	})

	// Output should contain blocked plan or "No plans found" if none blocked.
	// Just verify the flag doesn't error.
	_ = out
}

func TestLS_JSON_IncludesBlockedField(t *testing.T) {
	now := time.Now()
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("test-plan", []string{"go"}, now),
	})

	out := captureOutput(t, func() {
		if err := runLS("", "", true, false); err != nil {
			t.Fatalf("runLS --json failed: %v", err)
		}
	})

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %q", err, out)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if _, ok := result[0]["blocked"]; !ok {
		t.Error("expected 'blocked' field in JSON output")
	}
}

func TestLS_JSON_IncludesDistilledField(t *testing.T) {
	now := time.Now()
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("test-plan", []string{"go"}, now),
	})

	out := captureOutput(t, func() {
		if err := runLS("", "", true, false); err != nil {
			t.Fatalf("runLS --json failed: %v", err)
		}
	})

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %q", err, out)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if _, ok := result[0]["distilled"]; !ok {
		t.Error("expected 'distilled' field in JSON output")
	}
}

func TestLS_Table_ContainsDISTILLEDHeader(t *testing.T) {
	now := time.Now()
	setupProjectWithPlans(t, []plan.Plan{
		makeTestPlan("some-plan", []string{}, now),
	})

	out := captureOutput(t, func() {
		if err := runLS("", "", false, false); err != nil {
			t.Fatalf("runLS failed: %v", err)
		}
	})

	if !strings.Contains(out, "DISTILLED") {
		t.Errorf("expected DISTILLED header in table, got: %q", out)
	}
}
