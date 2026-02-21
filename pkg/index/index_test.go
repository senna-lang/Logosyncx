// Package index provides tests for the JSONL session index.
package index

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/senna-lang/logosyncx/pkg/session"
)

// --- helpers -----------------------------------------------------------------

// setupProject creates a temp directory with a .logosyncx/ structure and
// returns the project root. It does NOT create index.jsonl.
func setupProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".logosyncx", "sessions"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return dir
}

// writeSessionFile writes a session to disk under projectRoot/sessions/.
func writeSessionFile(t *testing.T, projectRoot string, s session.Session) {
	t.Helper()
	if _, err := session.Write(projectRoot, s); err != nil {
		t.Fatalf("session.Write: %v", err)
	}
}

// makeSession returns a minimal session.Session for testing.
func makeSession(id, topic string, tags []string, date time.Time) session.Session {
	return session.Session{
		ID:      id,
		Date:    date,
		Topic:   topic,
		Tags:    tags,
		Agent:   "claude-code",
		Related: []string{},
		Body:    "## Summary\nThis is a test session about " + topic + ".\n",
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

// --- FromSession -------------------------------------------------------------

func TestFromSession_CopiesAllFields(t *testing.T) {
	date := time.Date(2025, 2, 20, 10, 30, 0, 0, time.UTC)
	s := session.Session{
		ID:       "abc123",
		Filename: "2025-02-20_auth.md",
		Date:     date,
		Topic:    "auth",
		Tags:     []string{"jwt", "security"},
		Agent:    "claude-code",
		Related:  []string{"2025-02-15_audit.md"},
		Excerpt:  "JWT authentication decisions.",
	}
	e := FromSession(s)

	if e.ID != s.ID {
		t.Errorf("ID = %q, want %q", e.ID, s.ID)
	}
	if e.Filename != s.Filename {
		t.Errorf("Filename = %q, want %q", e.Filename, s.Filename)
	}
	if !e.Date.Equal(s.Date) {
		t.Errorf("Date = %v, want %v", e.Date, s.Date)
	}
	if e.Topic != s.Topic {
		t.Errorf("Topic = %q, want %q", e.Topic, s.Topic)
	}
	if len(e.Tags) != len(s.Tags) || e.Tags[0] != s.Tags[0] {
		t.Errorf("Tags = %v, want %v", e.Tags, s.Tags)
	}
	if e.Agent != s.Agent {
		t.Errorf("Agent = %q, want %q", e.Agent, s.Agent)
	}
	if len(e.Related) != len(s.Related) {
		t.Errorf("Related = %v, want %v", e.Related, s.Related)
	}
	if e.Excerpt != s.Excerpt {
		t.Errorf("Excerpt = %q, want %q", e.Excerpt, s.Excerpt)
	}
}

func TestFromSession_NilTagsBecomesEmpty(t *testing.T) {
	s := session.Session{ID: "x", Tags: nil, Related: []string{}}
	e := FromSession(s)
	if e.Tags == nil {
		t.Error("Tags should be [] not nil")
	}
	if len(e.Tags) != 0 {
		t.Errorf("Tags should be empty, got %v", e.Tags)
	}
}

func TestFromSession_NilRelatedBecomesEmpty(t *testing.T) {
	s := session.Session{ID: "x", Tags: []string{}, Related: nil}
	e := FromSession(s)
	if e.Related == nil {
		t.Error("Related should be [] not nil")
	}
	if len(e.Related) != 0 {
		t.Errorf("Related should be empty, got %v", e.Related)
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
	// Create empty index file.
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
	date := time.Date(2025, 2, 20, 10, 30, 0, 0, time.UTC)
	s := makeSession("a1b2c3", "auth-refactor", []string{"auth", "jwt"}, date)
	e := FromSession(s)
	e.Filename = "2025-02-20_auth-refactor.md"

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
			ID:      []string{"id1", "id2", "id3"}[i],
			Topic:   topic,
			Tags:    []string{},
			Related: []string{},
			Date:    time.Now(),
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
	e := Entry{ID: "x1", Topic: "t", Tags: []string{}, Related: []string{}, Date: time.Now()}
	if err := Append(dir, e); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Append blank lines manually.
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
	e := Entry{ID: "new1", Topic: "test", Tags: []string{}, Related: []string{}, Date: time.Now()}

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
		e := Entry{ID: []string{"i1", "i2", "i3"}[i], Topic: topic, Tags: []string{}, Related: []string{}, Date: time.Now()}
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
	e1 := Entry{ID: "first", Topic: "first-topic", Tags: []string{}, Related: []string{}, Date: time.Now()}
	e2 := Entry{ID: "second", Topic: "second-topic", Tags: []string{}, Related: []string{}, Date: time.Now()}

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

func TestRebuild_EmptySessions_CreatesEmptyIndex(t *testing.T) {
	dir := setupProject(t)
	n, err := Rebuild(dir)
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 sessions indexed, got %d", n)
	}
	// Index file should exist even when empty.
	if _, statErr := os.Stat(FilePath(dir)); statErr != nil {
		t.Errorf("index.jsonl should exist after Rebuild, got: %v", statErr)
	}
}

func TestRebuild_IndexesAllSessions(t *testing.T) {
	dir := setupProject(t)
	date := time.Date(2025, 2, 20, 10, 0, 0, 0, time.UTC)

	writeSessionFile(t, dir, makeSession("id1", "auth-flow", []string{"auth"}, date))
	writeSessionFile(t, dir, makeSession("id2", "db-schema", []string{"postgres"}, date.Add(-24*time.Hour)))

	n, err := Rebuild(dir)
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 sessions indexed, got %d", n)
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
	date := time.Date(2025, 2, 20, 10, 0, 0, 0, time.UTC)

	// Write a stale entry directly.
	stale := Entry{ID: "stale", Topic: "old-topic", Tags: []string{}, Related: []string{}, Date: date}
	if err := Append(dir, stale); err != nil {
		t.Fatalf("Append stale: %v", err)
	}

	// Now write one real session and rebuild.
	writeSessionFile(t, dir, makeSession("fresh", "new-topic", []string{}, date))
	n, err := Rebuild(dir)
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 session indexed, got %d", n)
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
	date := time.Date(2025, 2, 20, 10, 0, 0, 0, time.UTC)
	s := session.Session{
		ID:      "exc1",
		Date:    date,
		Topic:   "excerpt-test",
		Tags:    []string{},
		Related: []string{},
		Body:    "## Summary\nThis excerpt should appear in the index.\n",
	}
	writeSessionFile(t, dir, s)

	if _, err := Rebuild(dir); err != nil {
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

func TestRebuild_NoSessionsDir_ReturnsZero(t *testing.T) {
	// Project root has .logosyncx/ but no sessions/ subdir.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".logosyncx"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	n, err := Rebuild(dir)
	if err != nil {
		t.Fatalf("Rebuild with no sessions dir: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 sessions, got %d", n)
	}
}
