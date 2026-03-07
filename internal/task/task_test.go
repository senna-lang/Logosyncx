package task

import (
	"strings"
	"testing"
	"time"
)

// --- helpers -----------------------------------------------------------------

func taskMarkdown(id, title, status, priority, plan string, seq int, tags []string, body string) string {
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
		"seq: " + string(rune('0'+seq)) + "\n" +
		"status: " + status + "\n" +
		"priority: " + priority + "\n" +
		"plan: " + plan + "\n" +
		"tags: " + tagYAML + "\n" +
		"assignee: \n" +
		"---\n\n" +
		body
}

// --- Parse -------------------------------------------------------------------

func TestParse_ValidFrontmatter(t *testing.T) {
	content := taskMarkdown("t-abc123", "Implement auth", "open", "high", "", 1, nil,
		"## What\nImplement JWT auth.\n")
	got, err := Parse("TASK.md", []byte(content))
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

func TestParse_SetsBody(t *testing.T) {
	content := taskMarkdown("t-1", "title", "open", "medium", "", 0, nil,
		"## What\nThis is the body.\n")
	got, err := Parse("TASK.md", []byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !strings.Contains(got.Body, "This is the body") {
		t.Errorf("Body = %q, expected body content", got.Body)
	}
}

func TestParse_ExtractsExcerpt(t *testing.T) {
	content := taskMarkdown("t-1", "title", "open", "medium", "", 0, nil,
		"## What\nThis is the excerpt content.\n")
	got, err := Parse("TASK.md", []byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !strings.Contains(got.Excerpt, "excerpt content") {
		t.Errorf("Excerpt = %q, expected excerpt content", got.Excerpt)
	}
}

func TestParse_ParsesTags(t *testing.T) {
	content := taskMarkdown("t-1", "title", "open", "medium", "", 0, []string{"auth", "jwt"},
		"## What\nbody\n")
	got, err := Parse("TASK.md", []byte(content))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("Tags len = %d, want 2", len(got.Tags))
	}
	if got.Tags[0] != "auth" || got.Tags[1] != "jwt" {
		t.Errorf("Tags = %v, want [auth jwt]", got.Tags)
	}
}

func TestParse_ParsesPlan(t *testing.T) {
	raw := "---\nid: t-1\ntitle: test\nstatus: open\npriority: medium\nplan: 20260304-auth-refactor\ntags: []\nassignee: \n---\n\n## What\nbody\n"
	got, err := Parse("TASK.md", []byte(raw))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Plan != "20260304-auth-refactor" {
		t.Errorf("Plan = %q, want '20260304-auth-refactor'", got.Plan)
	}
}

func TestParse_ParsesDependsOn(t *testing.T) {
	raw := "---\nid: t-1\ntitle: test\nstatus: open\npriority: medium\nplan: myplan\ndepends_on:\n  - 1\n  - 2\ntags: []\nassignee: \n---\n\n## What\nbody\n"
	got, err := Parse("TASK.md", []byte(raw))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got.DependsOn) != 2 {
		t.Fatalf("DependsOn len = %d, want 2", len(got.DependsOn))
	}
	if got.DependsOn[0] != 1 || got.DependsOn[1] != 2 {
		t.Errorf("DependsOn = %v, want [1 2]", got.DependsOn)
	}
}

func TestParse_ParsesSeq(t *testing.T) {
	raw := "---\nid: t-1\ntitle: test\nseq: 3\nstatus: open\npriority: medium\nplan: myplan\ntags: []\nassignee: \n---\n\n## What\nbody\n"
	got, err := Parse("TASK.md", []byte(raw))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.Seq != 3 {
		t.Errorf("Seq = %d, want 3", got.Seq)
	}
}

func TestParse_MissingFrontmatter_ReturnsError(t *testing.T) {
	_, err := Parse("TASK.md", []byte("no frontmatter here"))
	if err == nil {
		t.Error("expected error for missing frontmatter, got nil")
	}
}

func TestParse_MissingClosingFrontmatter_ReturnsError(t *testing.T) {
	content := "---\ntitle: test\n"
	_, err := Parse("TASK.md", []byte(content))
	if err == nil {
		t.Error("expected error for missing closing ---, got nil")
	}
}

func TestParse_AllStatusValues(t *testing.T) {
	for _, status := range []Status{StatusOpen, StatusInProgress, StatusDone} {
		content := taskMarkdown("t-1", "title", string(status), "medium", "", 0, nil, "## What\nbody\n")
		got, err := Parse("TASK.md", []byte(content))
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
		content := taskMarkdown("t-1", "title", "open", string(priority), "", 0, nil, "## What\nbody\n")
		got, err := Parse("TASK.md", []byte(content))
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
		ID:        "t-xyz",
		Date:      date,
		Title:     "Round-trip task",
		Seq:       2,
		Status:    StatusInProgress,
		Priority:  PriorityHigh,
		Plan:      "20260304-auth-refactor",
		DependsOn: []int{1},
		Tags:      []string{"go", "testing"},
		Assignee:  "alice",
		Body:      "## What\nRound trip test.\n",
	}

	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	parsed, err := Parse("TASK.md", data)
	if err != nil {
		t.Fatalf("Parse after Marshal: %v", err)
	}

	if parsed.ID != original.ID {
		t.Errorf("ID: got %q, want %q", parsed.ID, original.ID)
	}
	if parsed.Title != original.Title {
		t.Errorf("Title: got %q, want %q", parsed.Title, original.Title)
	}
	if parsed.Seq != original.Seq {
		t.Errorf("Seq: got %d, want %d", parsed.Seq, original.Seq)
	}
	if parsed.Status != original.Status {
		t.Errorf("Status: got %q, want %q", parsed.Status, original.Status)
	}
	if parsed.Priority != original.Priority {
		t.Errorf("Priority: got %q, want %q", parsed.Priority, original.Priority)
	}
	if parsed.Plan != original.Plan {
		t.Errorf("Plan: got %q, want %q", parsed.Plan, original.Plan)
	}
	if len(parsed.DependsOn) != 1 || parsed.DependsOn[0] != 1 {
		t.Errorf("DependsOn: got %v, want [1]", parsed.DependsOn)
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

// --- TaskDirName -------------------------------------------------------------

func TestTask_TaskDirName_Format(t *testing.T) {
	cases := []struct {
		seq   int
		title string
		want  string
	}{
		{1, "Add JWT middleware", "001-add-jwt-middleware"},
		{2, "Setup RS256 keys", "002-setup-rs256-keys"},
		{10, "Refactor auth module", "010-refactor-auth-module"},
		{100, "Big task", "100-big-task"},
		{1, "Auth & Setup!", "001-auth-setup"},
	}
	for _, tc := range cases {
		got := TaskDirName(tc.seq, tc.title)
		if got != tc.want {
			t.Errorf("TaskDirName(%d, %q) = %q, want %q", tc.seq, tc.title, got, tc.want)
		}
	}
}

// --- extractExcerpt ----------------------------------------------------------

func TestExtractExcerpt_FromWhatSection(t *testing.T) {
	body := []byte("## What\nThis is the what section.\n\n## Why\nThis is why.\n")
	got := extractExcerpt(body, "")
	if !strings.Contains(got, "what section") {
		t.Errorf("excerpt = %q, expected content from ## What", got)
	}
}

func TestExtractExcerpt_FallbackToBody(t *testing.T) {
	body := []byte("No headings here, just plain text.")
	got := extractExcerpt(body, "")
	if got == "" {
		t.Error("expected non-empty excerpt via fallback")
	}
}

func TestExtractExcerpt_EmptyWhatSection_FallsBack(t *testing.T) {
	body := []byte("## What\n\n## Why\nThis is why.\n")
	got := extractExcerpt(body, "")
	if got == "" {
		t.Error("expected non-empty excerpt via fallback when ## What is empty")
	}
}

func TestExtractExcerpt_TruncatesLongContent(t *testing.T) {
	long := strings.Repeat("a", 400)
	body := []byte("## What\n" + long + "\n")
	got := extractExcerpt(body, "")
	if len([]rune(got)) > excerptMaxRunes+1 { // +1 for the ellipsis rune
		t.Errorf("excerpt length = %d runes, expected at most %d", len([]rune(got)), excerptMaxRunes+1)
	}
}

func TestExtractExcerpt_ShortContentNotTruncated(t *testing.T) {
	body := []byte("## What\nShort content.\n")
	got := extractExcerpt(body, "")
	if strings.HasSuffix(got, "…") {
		t.Error("short content should not be truncated")
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

func TestTask_NoStatusCancelled(t *testing.T) {
	if IsValidStatus("cancelled") {
		t.Error("cancelled should not be a valid status in v2")
	}
	for _, s := range ValidStatuses {
		if s == "cancelled" {
			t.Error("cancelled found in ValidStatuses, should have been removed")
		}
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

func TestToJSON_NilDependsOnBecomesEmpty(t *testing.T) {
	tk := &Task{ID: "t-1", DependsOn: nil}
	j := tk.ToJSON()
	if j.DependsOn == nil {
		t.Error("ToJSON: DependsOn should be [] not nil")
	}
}

func TestFromTask_NilSlicesNormalized(t *testing.T) {
	tk := &Task{
		ID:        "t-abc",
		Tags:      nil,
		DependsOn: nil,
	}
	j := FromTask(tk)
	if j.Tags == nil {
		t.Error("FromTask: Tags should be [] not nil")
	}
	if j.DependsOn == nil {
		t.Error("FromTask: DependsOn should be [] not nil")
	}
}

func TestToJSON_CopiesAllFields(t *testing.T) {
	date := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	tk := &Task{
		ID:        "t-abc",
		DirPath:   ".logosyncx/tasks/20260304-auth-refactor/001-add-jwt",
		Date:      date,
		Title:     "Test Task",
		Seq:       1,
		Status:    StatusInProgress,
		Priority:  PriorityHigh,
		Plan:      "20260304-auth-refactor",
		DependsOn: []int{},
		Tags:      []string{"go"},
		Assignee:  "bob",
		Excerpt:   "Some excerpt.",
	}
	j := tk.ToJSON()
	if j.ID != tk.ID {
		t.Errorf("ID = %q, want %q", j.ID, tk.ID)
	}
	if j.DirPath != tk.DirPath {
		t.Errorf("DirPath = %q, want %q", j.DirPath, tk.DirPath)
	}
	if !j.Date.Equal(tk.Date) {
		t.Errorf("Date = %v, want %v", j.Date, tk.Date)
	}
	if j.Title != tk.Title {
		t.Errorf("Title = %q, want %q", j.Title, tk.Title)
	}
	if j.Seq != tk.Seq {
		t.Errorf("Seq = %d, want %d", j.Seq, tk.Seq)
	}
	if j.Status != tk.Status {
		t.Errorf("Status = %q, want %q", j.Status, tk.Status)
	}
	if j.Priority != tk.Priority {
		t.Errorf("Priority = %q, want %q", j.Priority, tk.Priority)
	}
	if j.Plan != tk.Plan {
		t.Errorf("Plan = %q, want %q", j.Plan, tk.Plan)
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
		t.Errorf("slugify = %q, want 'my-task'", got)
	}
}

func TestSlugify_RemovesSpecialChars(t *testing.T) {
	got := slugify("auth (v2): JWT!")
	for _, ch := range []string{"(", ")", "!", ":"} {
		if strings.Contains(got, ch) {
			t.Errorf("slugify(%q) = %q, special char %q should be removed", "auth (v2): JWT!", got, ch)
		}
	}
}

func TestSlugify_PreservesHyphens(t *testing.T) {
	got := slugify("already-kebab")
	if got != "already-kebab" {
		t.Errorf("slugify = %q, want 'already-kebab'", got)
	}
}

func TestSlugify_PreservesUnderscores(t *testing.T) {
	got := slugify("with_underscore")
	if got != "with_underscore" {
		t.Errorf("slugify = %q, want 'with_underscore'", got)
	}
}

func TestSlugify_EmptyString(t *testing.T) {
	got := slugify("")
	if got != "" {
		t.Errorf("slugify('') = %q, want ''", got)
	}
}

func TestSlugify_CollapsesConsecutiveHyphens(t *testing.T) {
	got := slugify("auth  refactor")
	if strings.Contains(got, "--") {
		t.Errorf("slugify = %q, consecutive hyphens should be collapsed", got)
	}
}

// --- ExtractSections ---------------------------------------------------------

func TestExtractSections_What(t *testing.T) {
	body := "## What\nDo the thing.\n\n## Why\nBecause.\n"
	got := ExtractSections(body, []string{"What"})
	if !strings.Contains(got, "Do the thing") {
		t.Errorf("ExtractSections = %q, expected What content", got)
	}
	if strings.Contains(got, "Because") {
		t.Error("ExtractSections should not include Why content")
	}
}

func TestExtractSections_Checklist(t *testing.T) {
	body := "## What\nDo the thing.\n\n## Checklist\n- [ ] step one\n"
	got := ExtractSections(body, []string{"Checklist"})
	if !strings.Contains(got, "step one") {
		t.Errorf("ExtractSections = %q, expected Checklist content", got)
	}
}

func TestExtractSections_Multiple(t *testing.T) {
	body := "## What\nDo the thing.\n\n## Why\nBecause.\n\n## Notes\nExtra info.\n"
	got := ExtractSections(body, []string{"What", "Notes"})
	if !strings.Contains(got, "Do the thing") {
		t.Error("expected What content")
	}
	if !strings.Contains(got, "Extra info") {
		t.Error("expected Notes content")
	}
	if strings.Contains(got, "Because") {
		t.Error("should not include Why content")
	}
}

func TestExtractSections_CaseInsensitive(t *testing.T) {
	body := "## WHAT\nCase insensitive content.\n"
	got := ExtractSections(body, []string{"what"})
	if !strings.Contains(got, "Case insensitive") {
		t.Errorf("ExtractSections should match case-insensitively, got %q", got)
	}
}

func TestExtractSections_EmptyList_ReturnsFullBody(t *testing.T) {
	body := "## What\nContent.\n"
	got := ExtractSections(body, []string{})
	if got != body {
		t.Errorf("empty section list should return full body")
	}
}

// --- parseHeading ------------------------------------------------------------

func TestParseHeading_H2(t *testing.T) {
	text, level, ok := parseHeading("## Summary")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if level != 2 {
		t.Errorf("level = %d, want 2", level)
	}
	if text != "Summary" {
		t.Errorf("text = %q, want 'Summary'", text)
	}
}

func TestParseHeading_NotAHeading(t *testing.T) {
	_, _, ok := parseHeading("plain text")
	if ok {
		t.Error("expected ok=false for plain text")
	}
}

func TestParseHeading_HashWithNoSpace_NotAHeading(t *testing.T) {
	_, _, ok := parseHeading("##NoSpace")
	if ok {
		t.Error("expected ok=false when no space after #")
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
	// Each Japanese character is 1 rune.
	got := truncateRunes("日本語テスト", 3)
	if got != "日本語…" {
		t.Errorf("truncateRunes = %q, want '日本語…'", got)
	}
}
