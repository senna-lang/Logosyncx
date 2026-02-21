package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/session"
)

// helpers

func setupInitedProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := runInit(); err != nil {
		t.Fatalf("runInit: %v", err)
	}
	return dir
}

func writeTempSession(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp session: %v", err)
	}
	return path
}

func validSessionContent(topic string) string {
	return "---\n" +
		"id: \n" +
		"date: \n" +
		"topic: " + topic + "\n" +
		"tags:\n  - test\n" +
		"agent: claude-code\n" +
		"related: []\n" +
		"---\n\n" +
		"## Summary\nThis is a test session about " + topic + ".\n\n" +
		"## Key Decisions\n- Decision one\n"
}

func minimalSessionContent(topic string) string {
	return "---\n" +
		"topic: " + topic + "\n" +
		"---\n\n" +
		"## Summary\nMinimal session.\n"
}

// --- flag validation ---------------------------------------------------------

func TestSave_ErrorWhenNoFlagsProvided(t *testing.T) {
	err := runSave("", false)
	if err == nil {
		t.Fatal("expected error when no flags provided, got nil")
	}
	if !strings.Contains(err.Error(), "--file") && !strings.Contains(err.Error(), "--stdin") {
		t.Errorf("expected error to mention --file or --stdin, got: %v", err)
	}
}

func TestSave_ErrorWhenBothFlagsProvided(t *testing.T) {
	err := runSave("somefile.md", true)
	if err == nil {
		t.Fatal("expected error when both --file and --stdin provided, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' error, got: %v", err)
	}
}

// --- --file ------------------------------------------------------------------

func TestSave_File_ErrorOnMissingFile(t *testing.T) {
	setupInitedProject(t)
	err := runSave("/nonexistent/path/session.md", false)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestSave_File_SavesSession(t *testing.T) {
	dir := setupInitedProject(t)

	content := validSessionContent("file-save-test")
	path := writeTempSession(t, dir, "session.md", content)

	if err := runSave(path, false); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}

	sessions, err := session.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Topic != "file-save-test" {
		t.Errorf("topic = %q, want 'file-save-test'", sessions[0].Topic)
	}
}

func TestSave_File_AutoFillsID(t *testing.T) {
	dir := setupInitedProject(t)

	// Session with empty id field.
	content := minimalSessionContent("autofill-id")
	path := writeTempSession(t, dir, "session.md", content)

	if err := runSave(path, false); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}

	sessions, err := session.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ID == "" {
		t.Error("expected ID to be auto-filled, got empty string")
	}
}

func TestSave_File_AutoFillsDate(t *testing.T) {
	dir := setupInitedProject(t)

	before := time.Now().Add(-time.Second)
	content := minimalSessionContent("autofill-date")
	path := writeTempSession(t, dir, "session.md", content)

	if err := runSave(path, false); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}
	after := time.Now().Add(time.Second)

	sessions, err := session.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	d := sessions[0].Date
	if d.Before(before) || d.After(after) {
		t.Errorf("date %v not within expected range [%v, %v]", d, before, after)
	}
}

func TestSave_File_PreservesExistingID(t *testing.T) {
	dir := setupInitedProject(t)

	content := "---\nid: myexistingid\ntopic: preserve-id\n---\n\n## Summary\nTest.\n"
	path := writeTempSession(t, dir, "session.md", content)

	if err := runSave(path, false); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}

	sessions, err := session.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ID != "myexistingid" {
		t.Errorf("ID = %q, want 'myexistingid'", sessions[0].ID)
	}
}

func TestSave_File_FileNameFormat(t *testing.T) {
	dir := setupInitedProject(t)

	content := minimalSessionContent("filename-format")
	path := writeTempSession(t, dir, "session.md", content)

	if err := runSave(path, false); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}

	sessionsDir := filepath.Join(dir, ".logosyncx", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file in sessions/, got %d", len(entries))
	}
	name := entries[0].Name()
	if !strings.HasSuffix(name, "_filename-format.md") {
		t.Errorf("filename %q should end with '_filename-format.md'", name)
	}
	// Should start with a date: YYYY-MM-DD_
	if len(name) < 11 || name[4] != '-' || name[7] != '-' || name[10] != '_' {
		t.Errorf("filename %q does not start with YYYY-MM-DD_ prefix", name)
	}
}

func TestSave_File_MissingTopicUsesUntitled(t *testing.T) {
	dir := setupInitedProject(t)

	// No topic in frontmatter.
	content := "---\nid: notopic\n---\n\n## Summary\nNo topic here.\n"
	path := writeTempSession(t, dir, "session.md", content)

	// Should not error, just warn.
	if err := runSave(path, false); err != nil {
		t.Fatalf("runSave should not error on missing topic, got: %v", err)
	}

	sessionsDir := filepath.Join(dir, ".logosyncx", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file in sessions/, got %d", len(entries))
	}
	if !strings.Contains(entries[0].Name(), "untitled") {
		t.Errorf("expected 'untitled' in filename, got %q", entries[0].Name())
	}
}

func TestSave_File_ErrorOnInvalidMarkdown(t *testing.T) {
	dir := setupInitedProject(t)

	// No frontmatter at all.
	content := "This is just plain text, no frontmatter."
	path := writeTempSession(t, dir, "bad.md", content)

	err := runSave(path, false)
	if err == nil {
		t.Fatal("expected error for invalid markdown, got nil")
	}
}

func TestSave_File_ErrorWhenNotInitialized(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(orig) })

	// No logos init run — no .logosyncx/ directory.
	content := minimalSessionContent("no-init")
	path := writeTempSession(t, dir, "session.md", content)

	err := runSave(path, false)
	if err == nil {
		t.Fatal("expected error when project not initialized, got nil")
	}
	if !strings.Contains(err.Error(), "logos init") {
		t.Errorf("expected 'logos init' hint in error, got: %v", err)
	}
}

// --- --stdin -----------------------------------------------------------------

func TestSave_Stdin_SavesSession(t *testing.T) {
	dir := setupInitedProject(t)

	content := minimalSessionContent("stdin-test")

	// Redirect stdin from a pipe.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	_, _ = w.WriteString(content)
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	if err := runSave("", true); err != nil {
		t.Fatalf("runSave --stdin failed: %v", err)
	}

	sessions, err := session.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Topic != "stdin-test" {
		t.Errorf("topic = %q, want 'stdin-test'", sessions[0].Topic)
	}
}

// --- generateID --------------------------------------------------------------

func TestGenerateID_NotEmpty(t *testing.T) {
	id, err := generateID()
	if err != nil {
		t.Fatalf("generateID failed: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty ID")
	}
}

func TestGenerateID_Length(t *testing.T) {
	id, err := generateID()
	if err != nil {
		t.Fatalf("generateID failed: %v", err)
	}
	if len(id) != 6 {
		t.Errorf("expected ID length 6, got %d: %q", len(id), id)
	}
}

func TestGenerateID_IsHex(t *testing.T) {
	id, err := generateID()
	if err != nil {
		t.Fatalf("generateID failed: %v", err)
	}
	for _, r := range id {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			t.Errorf("ID %q contains non-hex character %q", id, r)
		}
	}
}

func TestGenerateID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 20; i++ {
		id, err := generateID()
		if err != nil {
			t.Fatalf("generateID failed: %v", err)
		}
		if ids[id] {
			t.Errorf("duplicate ID generated: %q", id)
		}
		ids[id] = true
	}
}

// --- warnPrivacy -------------------------------------------------------------

func TestWarnPrivacy_NoPatterns(t *testing.T) {
	// Should not panic with empty pattern list.
	warnPrivacy("some content with sk-abc123", []string{})
}

func TestWarnPrivacy_InvalidPattern(t *testing.T) {
	// Should not panic with an invalid regex.
	warnPrivacy("content", []string{"[invalid"})
}

func TestWarnPrivacy_NoMatch(t *testing.T) {
	// Smoke test: no panic when pattern doesn't match.
	warnPrivacy("clean content", []string{`sk-[a-zA-Z0-9]+`})
}

// --- config integration ------------------------------------------------------

func TestSave_UsesConfigPrivacyPatterns(t *testing.T) {
	dir := setupInitedProject(t)

	// Write a config with a privacy filter.
	cfg, _ := config.Load(dir)
	cfg.Privacy.FilterPatterns = []string{`sk-[a-zA-Z0-9]+`}
	_ = config.Save(dir, cfg)

	// Write a session containing a fake API key.
	content := "---\ntopic: privacy-test\n---\n\n## Summary\nUsed sk-abc123 for auth.\n"
	path := writeTempSession(t, dir, "session.md", content)

	// Should succeed (warning only, not a hard error).
	if err := runSave(path, false); err != nil {
		t.Fatalf("runSave should succeed even with privacy match, got: %v", err)
	}
}
