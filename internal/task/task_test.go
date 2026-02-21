package task

import (
	"strings"
	"testing"
	"time"
)

// --- helpers -----------------------------------------------------------------

func taskMarkdown(id, title, status, priority, session string, tags []string, body string) string {
	tagYAML := "[]"
	if len(tags) > 0 {
		parts := make([]string, len(tags))
		for i, t := range tags {
			parts[i] = "  - " + t
		}
		tagYAML = "\n" + strings.Join(parts, "\n")
	}
	return "---\n" +
		"id: " + id + "\n" +
		"title: " + title + "\n" +
		"status: " + status + "\n" +
		"priority: " + priority + "\n" +
		"session: " + session + "\n" +
		"tags: " + tagYAML + "\n" +
		"assignee: \n" +
		"---\n\n" +
		body
}

// --- Parse -------------------------------------------------------------------

func TestParse_ValidFrontmatter(t *testing.T) {
	content := taskMarkdown("t-abc123", "Implement auth", "open", "high", "", nil,
		"## What\nImplement JWT auth.\n")
	got, err := Parse("2025-02-20_implement-auth.md", []byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if got.ID != "t-abc123" {
		t.Errorf("ID = %q, want 't-abc123'", got.ID)
	}
	if got.Title != "Implement auth" {
		t.Errorf("Title = %q, want 'Implement auth'", got.Title)
	}
	if got.Status != StatusOpen {
		t.Errorf("Status = %q, want %q", got.Status, StatusOpen)
	}
	if got.Priority != PriorityHigh {
		t.Errorf("Priority = %q, want %q", got.Priority, PriorityHigh)
	}
}

func TestParse_SetsFilename(t *testing.T) {
	content := taskMarkdown("t-1", "title", "open", "medium", "", nil, "## What\nbody\n")
	got, err := Parse("2025-01-01_title.md", []byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Filename != "2025-01-01_title.md" {
		t.Errorf("Filename = %q, want '2025-01-01_title.md'", got.Filename)
	}
}

func TestParse_SetsBody(t *testing.T) {
	body := "## What\nDo the thing.\n\n## Why\nBecause.\n"
	content := taskMarkdown("t-1", "title", "open", "medium", "", nil, body)
	got, err := Parse("test.md", []byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Body == "" {
		t.Error("expected non-empty Body")
	}
	if !strings.Contains(got.Body, "Do the thing") {
		t.Errorf("Body = %q, want it to contain 'Do the thing'", got.Body)
	}
}

func TestParse_ExtractsExcerpt(t *testing.T) {
	body := "## What\nImplement the JWT authentication flow.\n\n## Why\nSecurity.\n"
	content := taskMarkdown("t-1", "auth", "open", "medium", "", nil, body)
	got, err := Parse("test.md", []byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !strings.Contains(got.Excerpt, "JWT authentication") {
		t.Errorf("Excerpt = %q, want it to contain 'JWT authentication'", got.Excerpt)
	}
}

func TestParse_ParsesTags(t *testing.T) {
	content := taskMarkdown("t-1", "title", "open", "medium", "", []string{"auth", "go"}, "## What\nbody\n")
	got, err := Parse("test.md", []byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got.Tags) != 2 {
		t.Errorf("Tags = %v, want [auth go]", got.Tags)
	}
	if got.Tags[0] != "auth" || got.Tags[1] != "go" {
		t.Errorf("Tags = %v, want [auth go]", got.Tags)
	}
}

func TestParse_ParsesSession(t *testing.T) {
	content := taskMarkdown("t-1", "title", "open", "medium", "2025-02-20_auth.md", nil, "## What\nbody\n")
	got, err := Parse("test.md", []byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Session != "2025-02-20_auth.md" {
		t.Errorf("Session = %q, want '2025-02-20_auth.md'", got.Session)
	}
}

func TestParse_MissingFrontmatter_ReturnsError(t *testing.T) {
	_, err := Parse("bad.md", []byte("no frontmatter here"))
	if err == nil {
		t.Error("expected error for missing frontmatter, got nil")
	}
}

func TestParse_MissingClosingFrontmatter_ReturnsError(t *testing.T) {
	content := "---\ntitle: test\n"
	_, err := Parse("bad.md", []byte(content))
	if err == nil {
		t.Error("expected error for missing closing ---, got nil")
	}
}

func TestParse_AllStatusValues(t *testing.T) {
	for _, status := range []Status{StatusOpen, StatusInProgress, StatusDone, StatusCancelled} {
		content := taskMarkdown("t-1", "title", string(status), "medium", "", nil, "## What\nbody\n")
		got, err := Parse("test.md", []byte(content))
		if err != nil {
			t.Fatalf("Parse with status %q failed: %v", status, err)
		}
		if got.Status != status {
			t.Errorf("Status = %q, want %q", got.Status, status)
		}
	}
}

func TestParse_AllPriorityValues(t *testing.T) {
	for _, priority := range []Priority{PriorityHigh, PriorityMedium, PriorityLow} {
		content := taskMarkdown("t-1", "title", "open", string(priority), "", nil, "## What\nbody\n")
		got, err := Parse("test.md", []byte(content))
		if err != nil {
			t.Fatalf("Parse with priority %q failed: %v", priority, err)
		}
		if got.Priority != priority {
			t.Errorf("Priority = %q, want %q", got.Priority, priority)
		}
	}
}

// --- Marshal -----------------------------------------------------------------

func TestMarshal_ProducesFrontmatter(t *testing.T) {
	tk := Task{
		ID:       "t-abc",
		Title:    "My Task",
		Status:   StatusOpen,
		Priority: PriorityMedium,
		Tags:     []string{},
		Body:     "## What\nDo something.\n",
	}
	data, err := Marshal(tk)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(data)
	if !strings.HasPrefix(s, "---\n") {
		t.Error("expected output to start with '---'")
	}
	if !strings.Contains(s, "t-abc") {
		t.Error("expected id in output")
	}
	if !strings.Contains(s, "My Task") {
		t.Error("expected title in output")
	}
	if !strings.Contains(s, "Do something") {
		t.Error("expected body in output")
	}
}

func TestMarshal_RoundTrip(t *testing.T) {
	date := time.Date(2025, 2, 20, 10, 0, 0, 0, time.UTC)
	original := Task{
		ID:       "t-xyz",
		Date:     date,
		Title:    "Round-trip task",
		Status:   StatusInProgress,
		Priority: PriorityHigh,
		Session:  "2025-02-15_auth.md",
		Tags:     []string{"go", "testing"},
		Assignee: "alice",
		Body:     "## What\nRound trip test.\n",
	}

	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	parsed, err := Parse("test.md", data)
	if err != nil {
		t.Fatalf("Parse after Marshal: %v", err)
	}

	if parsed.ID != original.ID {
		t.Errorf("ID: got %q, want %q", parsed.ID, original.ID)
	}
	if parsed.Title != original.Title {
		t.Errorf("Title: got %q, want %q", parsed.Title, original.Title)
	}
	if parsed.Status != original.Status {
		t.Errorf("Status: got %q, want %q", parsed.Status, original.Status)
	}
	if parsed.Priority != original.Priority {
		t.Errorf("Priority: got %q, want %q", parsed.Priority, original.Priority)
	}
	if parsed.Session != original.Session {
		t.Errorf("Session: got %q, want %q", parsed.Session, original.Session)
	}
	if parsed.Assignee != original.Assignee {
		t.Errorf("Assignee: got %q, want %q", parsed.Assignee, original.Assignee)
	}
	if len(parsed.Tags) != len(original.Tags) {
		t.Errorf("Tags: got %v, want %v", parsed.Tags, original.Tags)
	}
	if !parsed.Date.Equal(original.Date) {
		t.Errorf("Date: got %v, want %v", parsed.Date, original.Date)
	}
}

// --- FileName ----------------------------------------------------------------

func TestFileName_BasicFormat(t *testing.T) {
	tk := Task{
		Date:  time.Date(2025, 2, 20, 10, 0, 0, 0, time.UTC),
		Title: "Implement Auth",
	}
	got := FileName(tk)
	if !strings.HasPrefix(got, "2025-02-20_") {
		t.Errorf("FileName = %q, want YYYY-MM-DD_ prefix", got)
	}
	if !strings.HasSuffix(got, ".md") {
		t.Errorf("FileName = %q, want .md suffix", got)
	}
	if !strings.Contains(got, "implement-auth") {
		t.Errorf("FileName = %q, want slug 'implement-auth'", got)
	}
}

func TestFileName_SpacesBecomeDashes(t *testing.T) {
	tk := Task{
		Date:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Title: "update db schema",
	}
	got := FileName(tk)
	if !strings.Contains(got, "update-db-schema") {
		t.Errorf("FileName = %q, want 'update-db-schema' slug", got)
	}
}

func TestFileName_EmptyTitleUsesUntitled(t *testing.T) {
	tk := Task{
		Date:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Title: "",
	}
	got := FileName(tk)
	if !strings.Contains(got, "untitled") {
		t.Errorf("FileName = %q, want 'untitled' for empty title", got)
	}
}

func TestFileName_SpecialCharsStripped(t *testing.T) {
	tk := Task{
		Date:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Title: "auth (v2): JWT!",
	}
	got := FileName(tk)
	for _, ch := range []string{"(", ")", "!", ":"} {
		if strings.Contains(got, ch) {
			t.Errorf("FileName = %q, want special character %q removed", got, ch)
		}
	}
}

// --- extractExcerpt ----------------------------------------------------------

func TestExtractExcerpt_FromWhatSection(t *testing.T) {
	body := []byte("## What\nThis is the what section.\n\n## Why\nThis is why.\n")
	got := extractExcerpt(body)
	if !strings.Contains(got, "what section") {
		t.Errorf("excerpt = %q, expected content from ## What", got)
	}
	if strings.Contains(got, "This is why") {
		t.Errorf("excerpt = %q, should not contain ## Why content", got)
	}
}

func TestExtractExcerpt_FallbackToBody(t *testing.T) {
	body := []byte("## Why\nNo What section here.\n")
	got := extractExcerpt(body)
	if got == "" {
		t.Error("expected non-empty fallback excerpt")
	}
}

func TestExtractExcerpt_EmptyWhatSection_FallsBack(t *testing.T) {
	body := []byte("## What\n\n## Why\nSome content.\n")
	got := extractExcerpt(body)
	if got == "" {
		t.Error("expected non-empty fallback excerpt")
	}
}

func TestExtractExcerpt_TruncatesLongContent(t *testing.T) {
	long := strings.Repeat("a", 400)
	body := []byte("## What\n" + long + "\n")
	got := extractExcerpt(body)
	// +1 accounts for the appended ellipsis rune
	if len([]rune(got)) > excerptMaxRunes+1 {
		t.Errorf("excerpt not truncated: rune length = %d", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncated excerpt should end with '…', got: %q", got)
	}
}

func TestExtractExcerpt_ShortContentNotTruncated(t *testing.T) {
	body := []byte("## What\nShort content.\n")
	got := extractExcerpt(body)
	if strings.HasSuffix(got, "…") {
		t.Errorf("short excerpt should not be truncated, got: %q", got)
	}
}

// --- IsValidStatus / IsValidPriority -----------------------------------------

func TestIsValidStatus_KnownValues(t *testing.T) {
	for _, s := range ValidStatuses {
		if !IsValidStatus(s) {
			t.Errorf("IsValidStatus(%q) = false, want true", s)
		}
	}
}

func TestIsValidStatus_UnknownValue(t *testing.T) {
	if IsValidStatus("unknown") {
		t.Error("IsValidStatus('unknown') = true, want false")
	}
}

func TestIsValidPriority_KnownValues(t *testing.T) {
	for _, p := range ValidPriorities {
		if !IsValidPriority(p) {
			t.Errorf("IsValidPriority(%q) = false, want true", p)
		}
	}
}

func TestIsValidPriority_UnknownValue(t *testing.T) {
	if IsValidPriority("urgent") {
		t.Error("IsValidPriority('urgent') = true, want false")
	}
}

// --- ToJSON ------------------------------------------------------------------

func TestToJSON_NilTagsBecomesEmpty(t *testing.T) {
	tk := &Task{ID: "t-1", Tags: nil}
	j := tk.ToJSON()
	if j.Tags == nil {
		t.Error("ToJSON: Tags should be [] not nil")
	}
}

func TestToJSON_CopiesAllFields(t *testing.T) {
	date := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	tk := &Task{
		ID:       "t-abc",
		Filename: "2025-03-01_test.md",
		Date:     date,
		Title:    "Test Task",
		Status:   StatusInProgress,
		Priority: PriorityHigh,
		Session:  "2025-02-20_auth.md",
		Tags:     []string{"go"},
		Assignee: "bob",
		Excerpt:  "Some excerpt.",
	}
	j := tk.ToJSON()
	if j.ID != tk.ID {
		t.Errorf("ID = %q, want %q", j.ID, tk.ID)
	}
	if j.Filename != tk.Filename {
		t.Errorf("Filename = %q, want %q", j.Filename, tk.Filename)
	}
	if !j.Date.Equal(tk.Date) {
		t.Errorf("Date = %v, want %v", j.Date, tk.Date)
	}
	if j.Title != tk.Title {
		t.Errorf("Title = %q, want %q", j.Title, tk.Title)
	}
	if j.Status != tk.Status {
		t.Errorf("Status = %q, want %q", j.Status, tk.Status)
	}
	if j.Priority != tk.Priority {
		t.Errorf("Priority = %q, want %q", j.Priority, tk.Priority)
	}
	if j.Session != tk.Session {
		t.Errorf("Session = %q, want %q", j.Session, tk.Session)
	}
	if j.Assignee != tk.Assignee {
		t.Errorf("Assignee = %q, want %q", j.Assignee, tk.Assignee)
	}
	if j.Excerpt != tk.Excerpt {
		t.Errorf("Excerpt = %q, want %q", j.Excerpt, tk.Excerpt)
	}
}

// --- slugify -----------------------------------------------------------------

func TestSlugify_LowerCase(t *testing.T) {
	got := slugify("My Task")
	if got != "my-task" {
		t.Errorf("slugify('My Task') = %q, want 'my-task'", got)
	}
}

func TestSlugify_RemovesSpecialChars(t *testing.T) {
	got := slugify("auth (v2)!")
	for _, r := range []string{"(", ")", "!"} {
		if strings.Contains(got, r) {
			t.Errorf("slugify result %q should not contain %q", got, r)
		}
	}
}

func TestSlugify_PreservesHyphens(t *testing.T) {
	got := slugify("jwt-auth")
	if got != "jwt-auth" {
		t.Errorf("slugify('jwt-auth') = %q, want 'jwt-auth'", got)
	}
}

func TestSlugify_PreservesUnderscores(t *testing.T) {
	got := slugify("my_task")
	if got != "my_task" {
		t.Errorf("slugify('my_task') = %q, want 'my_task'", got)
	}
}

func TestSlugify_EmptyString(t *testing.T) {
	got := slugify("")
	if got != "" {
		t.Errorf("slugify('') = %q, want ''", got)
	}
}

// --- ExtractSections ---------------------------------------------------------

func TestExtractSections_What(t *testing.T) {
	body := "## What\nThe what content.\n\n## Why\nThe why content.\n"
	got := ExtractSections(body, []string{"What"})
	if !strings.Contains(got, "The what content") {
		t.Errorf("got %q, want 'what content'", got)
	}
	if strings.Contains(got, "The why content") {
		t.Errorf("got %q, should not contain 'why content'", got)
	}
}

func TestExtractSections_Checklist(t *testing.T) {
	body := "## What\nDo it.\n\n## Checklist\n- [ ] Step one\n- [ ] Step two\n\n## Notes\nExtra.\n"
	got := ExtractSections(body, []string{"Checklist"})
	if !strings.Contains(got, "Step one") {
		t.Errorf("got %q, want checklist items", got)
	}
	if strings.Contains(got, "Extra") {
		t.Errorf("got %q, should not contain Notes content", got)
	}
}

func TestExtractSections_Multiple(t *testing.T) {
	body := "## What\nWhat content.\n\n## Checklist\n- [ ] item\n\n## Notes\nNotes content.\n"
	got := ExtractSections(body, []string{"What", "Checklist"})
	if !strings.Contains(got, "What content") {
		t.Errorf("expected What section in output, got: %q", got)
	}
	if !strings.Contains(got, "item") {
		t.Errorf("expected Checklist section in output, got: %q", got)
	}
	if strings.Contains(got, "Notes content") {
		t.Errorf("Notes section should be excluded, got: %q", got)
	}
}

func TestExtractSections_CaseInsensitive(t *testing.T) {
	body := "## WHAT\nContent here.\n"
	got := ExtractSections(body, []string{"what"})
	if !strings.Contains(got, "Content here") {
		t.Errorf("expected case-insensitive section match, got: %q", got)
	}
}

func TestExtractSections_EmptyList_ReturnsFullBody(t *testing.T) {
	body := "## What\nContent.\n"
	got := ExtractSections(body, []string{})
	if got != body {
		t.Errorf("empty sections list should return full body")
	}
}

// --- parseHeading ------------------------------------------------------------

func TestParseHeading_H2(t *testing.T) {
	text, level, ok := parseHeading("## What")
	if !ok {
		t.Fatal("expected ok=true for '## What'")
	}
	if level != 2 {
		t.Errorf("level = %d, want 2", level)
	}
	if text != "What" {
		t.Errorf("text = %q, want 'What'", text)
	}
}

func TestParseHeading_NotAHeading(t *testing.T) {
	_, _, ok := parseHeading("This is normal text")
	if ok {
		t.Error("expected ok=false for non-heading line")
	}
}

func TestParseHeading_HashWithNoSpace_NotAHeading(t *testing.T) {
	_, _, ok := parseHeading("##NoSpace")
	if ok {
		t.Error("expected ok=false for ## without space")
	}
}

// --- truncateRunes -----------------------------------------------------------

func TestTruncateRunes_ShortString(t *testing.T) {
	got := truncateRunes("hello", 10)
	if got != "hello" {
		t.Errorf("truncateRunes = %q, want 'hello'", got)
	}
}

func TestTruncateRunes_ExactLength(t *testing.T) {
	got := truncateRunes("hello", 5)
	if got != "hello" {
		t.Errorf("truncateRunes = %q, want 'hello'", got)
	}
}

func TestTruncateRunes_TooLong(t *testing.T) {
	got := truncateRunes("hello world", 5)
	if got != "hello…" {
		t.Errorf("truncateRunes = %q, want 'hello…'", got)
	}
}

func TestTruncateRunes_MultiByte(t *testing.T) {
	// Each Japanese character is 1 rune but 3 bytes.
	s := "あいうえお"
	got := truncateRunes(s, 3)
	if got != "あいう…" {
		t.Errorf("truncateRunes multibyte = %q, want 'あいう…'", got)
	}
}
