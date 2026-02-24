package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- helpers -----------------------------------------------------------------

func makeSession(topic string, tags []string) Session {
	d := time.Date(2025, 2, 20, 10, 30, 0, 0, time.UTC)
	return Session{
		ID:      "abc123",
		Date:    &d,
		Topic:   topic,
		Tags:    tags,
		Agent:   "claude-code",
		Related: []string{},
		Body:    "## Summary\nThis is a test session.\n\n## Key Decisions\n- Decision one\n",
	}
}

func sampleMarkdown(summaryContent string) []byte {
	return []byte("---\n" +
		"id: abc123\n" +
		"date: 2025-02-20T10:30:00Z\n" +
		"topic: auth-refactor\n" +
		"tags:\n  - auth\n  - jwt\n" +
		"agent: claude-code\n" +
		"related: []\n" +
		"---\n" +
		"\n## Summary\n" + summaryContent + "\n" +
		"\n## Key Decisions\n- Use httpOnly cookies\n")
}

// --- FileName ----------------------------------------------------------------

func TestFileName_Basic(t *testing.T) {
	s := makeSession("auth-refactor", nil)
	got := FileName(s)
	want := "2025-02-20_auth-refactor.md"
	if got != want {
		t.Errorf("FileName = %q, want %q", got, want)
	}
}

func TestFileName_SpacesConvertedToHyphens(t *testing.T) {
	s := makeSession("db schema migration", nil)
	got := FileName(s)
	if !strings.Contains(got, "db-schema-migration") {
		t.Errorf("expected hyphens in filename, got %q", got)
	}
}

func TestFileName_UpperCaseLowered(t *testing.T) {
	s := makeSession("Auth Refactor", nil)
	got := FileName(s)
	if strings.Contains(got, "A") || strings.Contains(got, "R") {
		t.Errorf("expected lowercase filename, got %q", got)
	}
}

func TestFileName_SpecialCharsRemoved(t *testing.T) {
	s := makeSession("auth/refactor & review!", nil)
	got := FileName(s)
	for _, r := range []string{"/", "&", "!", " "} {
		if strings.Contains(got, r) {
			t.Errorf("filename contains unexpected char %q: %q", r, got)
		}
	}
}

func TestFileName_DateFormat(t *testing.T) {
	s := makeSession("topic", nil)
	got := FileName(s)
	// Must start with YYYY-MM-DD_
	if len(got) < 11 || got[4] != '-' || got[7] != '-' || got[10] != '_' {
		t.Errorf("filename does not start with YYYY-MM-DD_: %q", got)
	}
}

// --- SessionsDir / FilePath --------------------------------------------------

func TestSessionsDir(t *testing.T) {
	got := SessionsDir("/home/user/project")
	want := filepath.Join("/home/user/project", ".logosyncx", "sessions")
	if got != want {
		t.Errorf("SessionsDir = %q, want %q", got, want)
	}
}

func TestFilePath(t *testing.T) {
	s := makeSession("auth-refactor", nil)
	got := FilePath("/project", s)
	if !strings.HasPrefix(got, filepath.Join("/project", ".logosyncx", "sessions")) {
		t.Errorf("FilePath %q does not have expected prefix", got)
	}
	if !strings.HasSuffix(got, ".md") {
		t.Errorf("FilePath %q does not end with .md", got)
	}
}

// --- splitFrontmatter --------------------------------------------------------

func TestSplitFrontmatter_Valid(t *testing.T) {
	data := []byte("---\ntopic: test\n---\n\n## Body\n")
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(fm), "topic: test") {
		t.Errorf("frontmatter missing expected content, got: %q", fm)
	}
	if !strings.Contains(string(body), "## Body") {
		t.Errorf("body missing expected content, got: %q", body)
	}
}

func TestSplitFrontmatter_MissingOpening(t *testing.T) {
	data := []byte("topic: test\n---\n\n## Body\n")
	_, _, err := splitFrontmatter(data)
	if err == nil {
		t.Fatal("expected error for missing opening ---, got nil")
	}
}

func TestSplitFrontmatter_MissingClosing(t *testing.T) {
	data := []byte("---\ntopic: test\n\n## Body\n")
	_, _, err := splitFrontmatter(data)
	if err == nil {
		t.Fatal("expected error for missing closing ---, got nil")
	}
}

func TestSplitFrontmatter_EmptyBody(t *testing.T) {
	data := []byte("---\ntopic: test\n---\n")
	_, body, err := splitFrontmatter(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(strings.TrimSpace(string(body))) != 0 {
		t.Errorf("expected empty body, got %q", body)
	}
}

// --- Parse -------------------------------------------------------------------

func TestParse_ValidFile(t *testing.T) {
	data := sampleMarkdown("JWT authentication discussion.")
	s, err := Parse("2025-02-20_auth-refactor.md", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.ID != "abc123" {
		t.Errorf("ID = %q, want 'abc123'", s.ID)
	}
	if s.Topic != "auth-refactor" {
		t.Errorf("Topic = %q, want 'auth-refactor'", s.Topic)
	}
	if len(s.Tags) != 2 {
		t.Errorf("Tags length = %d, want 2", len(s.Tags))
	}
	if s.Tags[0] != "auth" {
		t.Errorf("Tags[0] = %q, want 'auth'", s.Tags[0])
	}
	if s.Agent != "claude-code" {
		t.Errorf("Agent = %q, want 'claude-code'", s.Agent)
	}
	if s.Filename != "2025-02-20_auth-refactor.md" {
		t.Errorf("Filename = %q, want '2025-02-20_auth-refactor.md'", s.Filename)
	}
}

func TestParse_SetsExcerpt(t *testing.T) {
	data := sampleMarkdown("This is the summary content.")
	s, err := Parse("test.md", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Excerpt == "" {
		t.Error("expected non-empty excerpt")
	}
	if !strings.Contains(s.Excerpt, "summary content") {
		t.Errorf("excerpt does not contain expected text, got: %q", s.Excerpt)
	}
}

func TestParse_SetsBody(t *testing.T) {
	data := sampleMarkdown("Summary text.")
	s, err := Parse("test.md", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Body == "" {
		t.Error("expected non-empty body")
	}
	if !strings.Contains(s.Body, "## Summary") {
		t.Errorf("body does not contain ## Summary, got: %q", s.Body)
	}
}

func TestParse_MissingFrontmatter(t *testing.T) {
	data := []byte("## Summary\nNo frontmatter here.\n")
	_, err := Parse("bad.md", data)
	if err == nil {
		t.Fatal("expected error for missing frontmatter, got nil")
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	data := []byte("---\n: invalid: yaml: [\n---\n\n## Body\n")
	_, err := Parse("bad.md", data)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

// --- extractExcerpt ----------------------------------------------------------

func TestExtractExcerpt_FromSummarySection(t *testing.T) {
	body := []byte("\n## Summary\nThis is the summary.\n\n## Key Decisions\n- decision\n")
	got := extractExcerpt(body, "Summary")
	if !strings.Contains(got, "This is the summary.") {
		t.Errorf("excerpt = %q, expected summary content", got)
	}
	if strings.Contains(got, "Key Decisions") {
		t.Errorf("excerpt should not contain Key Decisions section, got: %q", got)
	}
}

func TestExtractExcerpt_FallbackToBody(t *testing.T) {
	body := []byte("\n## Key Decisions\n- decision\n")
	got := extractExcerpt(body, "Summary")
	if got == "" {
		t.Error("expected non-empty excerpt fallback")
	}
}

func TestExtractExcerpt_TruncatesLongContent(t *testing.T) {
	long := strings.Repeat("a", 500)
	body := []byte("\n## Summary\n" + long + "\n")
	got := extractExcerpt(body, "Summary")
	if len([]rune(got)) > excerptMaxRunes+1 { // +1 for ellipsis
		t.Errorf("excerpt length %d exceeds max %d", len([]rune(got)), excerptMaxRunes+1)
	}
}

func TestExtractExcerpt_ShortContentNotTruncated(t *testing.T) {
	body := []byte("\n## Summary\nShort summary.\n")
	got := extractExcerpt(body, "Summary")
	if strings.HasSuffix(got, "…") {
		t.Errorf("short excerpt should not be truncated, got: %q", got)
	}
}

// --- parseHeading ------------------------------------------------------------

func TestParseHeading_H1(t *testing.T) {
	text, level, ok := parseHeading("# Title")
	if !ok || level != 1 || text != "Title" {
		t.Errorf("parseHeading('# Title') = (%q, %d, %v), want ('Title', 1, true)", text, level, ok)
	}
}

func TestParseHeading_H2(t *testing.T) {
	text, level, ok := parseHeading("## Summary")
	if !ok || level != 2 || text != "Summary" {
		t.Errorf("parseHeading('## Summary') = (%q, %d, %v), want ('Summary', 2, true)", text, level, ok)
	}
}

func TestParseHeading_H6(t *testing.T) {
	_, level, ok := parseHeading("###### Deep")
	if !ok || level != 6 {
		t.Errorf("expected level 6, got %d, ok=%v", level, ok)
	}
}

func TestParseHeading_NotAHeading(t *testing.T) {
	_, _, ok := parseHeading("This is not a heading")
	if ok {
		t.Error("expected ok=false for non-heading line")
	}
}

func TestParseHeading_HashWithNoSpace(t *testing.T) {
	_, _, ok := parseHeading("##NoSpace")
	if ok {
		t.Error("expected ok=false when no space after hashes")
	}
}

func TestParseHeading_TooManyHashes(t *testing.T) {
	_, _, ok := parseHeading("####### Too deep")
	if ok {
		t.Error("expected ok=false for 7 hashes")
	}
}

// --- ExtractSections ---------------------------------------------------------

func TestExtractSections_SingleSection(t *testing.T) {
	body := "## Summary\nSummary content.\n\n## Key Decisions\n- decision one\n"
	got := ExtractSections(body, []string{"Summary"})
	if !strings.Contains(got, "Summary content.") {
		t.Errorf("expected summary content, got: %q", got)
	}
	if strings.Contains(got, "Key Decisions") {
		t.Errorf("should not include Key Decisions, got: %q", got)
	}
}

func TestExtractSections_MultipleSections(t *testing.T) {
	body := "## Summary\nSummary content.\n\n## Key Decisions\n- decision\n\n## Notes\nSome notes.\n"
	got := ExtractSections(body, []string{"Summary", "Key Decisions"})
	if !strings.Contains(got, "Summary content.") {
		t.Errorf("expected summary content, got: %q", got)
	}
	if !strings.Contains(got, "Key Decisions") {
		t.Errorf("expected Key Decisions section, got: %q", got)
	}
	if strings.Contains(got, "Some notes.") {
		t.Errorf("should not include Notes section, got: %q", got)
	}
}

func TestExtractSections_CaseInsensitive(t *testing.T) {
	body := "## Summary\nContent here.\n"
	got := ExtractSections(body, []string{"summary"})
	if !strings.Contains(got, "Content here.") {
		t.Errorf("expected content with case-insensitive match, got: %q", got)
	}
}

func TestExtractSections_EmptySections_ReturnsFullBody(t *testing.T) {
	body := "## Summary\nContent.\n"
	got := ExtractSections(body, []string{})
	if got != body {
		t.Errorf("empty sections should return full body")
	}
}

func TestExtractSections_NonexistentSection(t *testing.T) {
	body := "## Summary\nContent.\n"
	got := ExtractSections(body, []string{"NonExistent"})
	if got != "" {
		t.Errorf("expected empty string for nonexistent section, got: %q", got)
	}
}

// --- Marshal -----------------------------------------------------------------

func TestMarshal_ProducesFrontmatter(t *testing.T) {
	s := makeSession("auth-refactor", []string{"auth", "jwt"})
	data, err := Marshal(s)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		t.Errorf("marshalled output should start with ---\\n, got: %q", text[:min(20, len(text))])
	}
	if !strings.Contains(text, "topic: auth-refactor") {
		t.Errorf("marshalled output missing topic field, got: %q", text)
	}
}

func TestMarshal_RoundTrip(t *testing.T) {
	original := makeSession("round-trip", []string{"test"})
	original.ID = "rt001"

	data, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	parsed, err := Parse("round-trip.md", data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.ID != original.ID {
		t.Errorf("ID mismatch: got %q, want %q", parsed.ID, original.ID)
	}
	if parsed.Topic != original.Topic {
		t.Errorf("Topic mismatch: got %q, want %q", parsed.Topic, original.Topic)
	}
	if len(parsed.Tags) != len(original.Tags) {
		t.Errorf("Tags length mismatch: got %d, want %d", len(parsed.Tags), len(original.Tags))
	}
}

// --- Write / LoadFile / LoadAll ----------------------------------------------

func TestWrite_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	s := makeSession("write-test", []string{"test"})

	path, err := Write(dir, s)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to exist at %s", path)
	}
}

func TestWrite_FileNameFormat(t *testing.T) {
	dir := t.TempDir()
	s := makeSession("write-test", nil)

	path, err := Write(dir, s)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	base := filepath.Base(path)
	if !strings.HasPrefix(base, "2025-02-20_") {
		t.Errorf("filename %q should start with date prefix", base)
	}
	if !strings.HasSuffix(base, ".md") {
		t.Errorf("filename %q should end with .md", base)
	}
}

func TestLoadFile_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := makeSession("loadfile-test", []string{"go", "cli"})
	s.ID = "lf001"

	path, err := Write(dir, s)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}

	if loaded.ID != s.ID {
		t.Errorf("ID mismatch: got %q, want %q", loaded.ID, s.ID)
	}
	if loaded.Topic != s.Topic {
		t.Errorf("Topic mismatch: got %q, want %q", loaded.Topic, s.Topic)
	}
	if len(loaded.Tags) != len(s.Tags) {
		t.Errorf("Tags length: got %d, want %d", len(loaded.Tags), len(s.Tags))
	}
}

func TestLoadFile_NotExist(t *testing.T) {
	_, err := LoadFile("/nonexistent/path/session.md")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestLoadAll_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	sessions, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("unexpected error for empty sessions dir: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestLoadAll_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	topics := []string{"first-topic", "second-topic", "third-topic"}
	for _, topic := range topics {
		s := makeSession(topic, []string{"test"})
		if _, err := Write(dir, s); err != nil {
			t.Fatalf("Write failed for %s: %v", topic, err)
		}
	}

	sessions, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(sessions) != len(topics) {
		t.Errorf("expected %d sessions, got %d", len(topics), len(sessions))
	}
}

func TestLoadAll_SkipsNonMarkdown(t *testing.T) {
	dir := t.TempDir()

	// Write a valid session.
	s := makeSession("valid-session", nil)
	if _, err := Write(dir, s); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Write a non-.md file into the sessions directory.
	sessDir := SessionsDir(dir)
	if err := os.WriteFile(filepath.Join(sessDir, "notes.txt"), []byte("not a session"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	sessions, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("expected 1 session (skipping .txt), got %d", len(sessions))
	}
}

func TestLoadAll_DirNotExist(t *testing.T) {
	dir := t.TempDir()
	// Do NOT create .logosyncx/sessions — LoadAll should return nil, nil.
	sessions, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("expected no error when sessions dir missing, got: %v", err)
	}
	if sessions != nil {
		t.Errorf("expected nil sessions, got %v", sessions)
	}
}

// --- truncateRunes -----------------------------------------------------------

func TestTruncateRunes_ShortString(t *testing.T) {
	got := truncateRunes("hello", 10)
	if got != "hello" {
		t.Errorf("truncateRunes short = %q, want 'hello'", got)
	}
}

func TestTruncateRunes_ExactLength(t *testing.T) {
	got := truncateRunes("hello", 5)
	if got != "hello" {
		t.Errorf("truncateRunes exact = %q, want 'hello'", got)
	}
}

func TestTruncateRunes_LongString(t *testing.T) {
	long := strings.Repeat("a", 100)
	got := truncateRunes(long, 50)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncated string should end with ellipsis, got: %q", got)
	}
	if len([]rune(got)) != 51 { // 50 runes + ellipsis
		t.Errorf("truncated length = %d, want 51", len([]rune(got)))
	}
}

func TestTruncateRunes_MultiByte(t *testing.T) {
	// Japanese characters are multi-byte but single rune each.
	s := strings.Repeat("あ", 100)
	got := truncateRunes(s, 10)
	if len([]rune(got)) != 11 { // 10 runes + ellipsis
		t.Errorf("multibyte truncation: got %d runes, want 11", len([]rune(got)))
	}
}

// min is a local helper for Go versions before 1.21.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
