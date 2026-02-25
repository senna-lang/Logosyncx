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

// --- flag validation ---------------------------------------------------------

func TestSave_ErrorWhenNoTopicProvided(t *testing.T) {
	err := runSave("", nil, "", nil, nil)
	if err == nil {
		t.Fatal("expected error when no topic provided, got nil")
	}
	if !strings.Contains(err.Error(), "--topic") {
		t.Errorf("expected error to mention --topic, got: %v", err)
	}
}

// --- flag-based save ---------------------------------------------------------

func TestSave_TopicOnly(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runSave("topic-only", nil, "", nil, nil); err != nil {
		t.Fatalf("runSave with --topic failed: %v", err)
	}

	sessions, err := session.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Topic != "topic-only" {
		t.Errorf("topic = %q, want 'topic-only'", sessions[0].Topic)
	}
}

func TestSave_AllFrontmatterFields(t *testing.T) {
	dir := setupInitedProject(t)

	sections := []string{"Summary=This is a full flag-based session."}
	if err := runSave("all-fields", []string{"go", "cli"}, "claude-code", []string{"2026-01-01_previous.md"}, sections); err != nil {
		t.Fatalf("runSave with all flags failed: %v", err)
	}

	sessions, err := session.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	s := sessions[0]
	if s.Topic != "all-fields" {
		t.Errorf("topic = %q, want 'all-fields'", s.Topic)
	}
	if s.Agent != "claude-code" {
		t.Errorf("agent = %q, want 'claude-code'", s.Agent)
	}
	if len(s.Tags) != 2 || s.Tags[0] != "go" || s.Tags[1] != "cli" {
		t.Errorf("tags = %v, want [go cli]", s.Tags)
	}
	if len(s.Related) != 1 || s.Related[0] != "2026-01-01_previous.md" {
		t.Errorf("related = %v, want [2026-01-01_previous.md]", s.Related)
	}
	if !strings.Contains(s.Body, "full flag-based session") {
		t.Errorf("body does not contain expected text, got: %q", s.Body)
	}
}

func TestSave_AutoFillsIDAndDate(t *testing.T) {
	dir := setupInitedProject(t)

	before := time.Now().Add(-time.Second)
	if err := runSave("autofill-flags", nil, "", nil, nil); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}
	after := time.Now().Add(time.Second)

	sessions, err := session.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	s := sessions[0]
	if s.ID == "" {
		t.Error("expected ID to be auto-filled, got empty string")
	}
	if s.Date.Before(before) || s.Date.After(after) {
		t.Errorf("date %v not within expected range [%v, %v]", s.Date, before, after)
	}
}

func TestSave_FileNameFormat(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runSave("filename-format", nil, "", nil, nil); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}

	sessionsDir := dir + "/.logosyncx/sessions"
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

func TestSave_ErrorWhenNotInitialized(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(orig) })

	err := runSave("no-init", nil, "", nil, nil)
	if err == nil {
		t.Fatal("expected error when project not initialized, got nil")
	}
	if !strings.Contains(err.Error(), "logos init") {
		t.Errorf("expected 'logos init' hint in error, got: %v", err)
	}
}

// --- --section flag: valid usage ---------------------------------------------

func TestSave_SectionFlag_ValidSection(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runSave("section-valid", nil, "", nil, []string{"Summary=This is a summary."}); err != nil {
		t.Fatalf("runSave with valid --section failed: %v", err)
	}

	sessions, err := session.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	body := sessions[0].Body
	if !strings.Contains(body, "## Summary") {
		t.Errorf("expected '## Summary' heading in body, got: %q", body)
	}
	if !strings.Contains(body, "This is a summary.") {
		t.Errorf("expected section content in body, got: %q", body)
	}
}

func TestSave_SectionFlag_MultipleSections(t *testing.T) {
	dir := setupInitedProject(t)

	sections := []string{"Summary=My summary.", "Key Decisions=- Decision A"}
	if err := runSave("multi-section", nil, "", nil, sections); err != nil {
		t.Fatalf("runSave with multiple --section failed: %v", err)
	}

	sessions, err := session.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	body := sessions[0].Body
	if !strings.Contains(body, "## Summary") {
		t.Errorf("expected '## Summary' in body, got: %q", body)
	}
	if !strings.Contains(body, "## Key Decisions") {
		t.Errorf("expected '## Key Decisions' in body, got: %q", body)
	}
}

func TestSave_SectionFlag_ContentPreserved(t *testing.T) {
	dir := setupInitedProject(t)

	content := "Decided to use Go for the CLI tool."
	if err := runSave("content-preserved", nil, "", nil, []string{"Summary=" + content}); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}

	sessions, err := session.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if !strings.Contains(sessions[0].Body, content) {
		t.Errorf("expected content %q in body, got: %q", content, sessions[0].Body)
	}
}

func TestSave_SectionFlag_EmptySectionsProducesEmptyBody(t *testing.T) {
	dir := setupInitedProject(t)

	if err := runSave("empty-body", nil, "", nil, nil); err != nil {
		t.Fatalf("runSave with no sections failed: %v", err)
	}

	sessions, err := session.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if strings.TrimSpace(sessions[0].Body) != "" {
		t.Errorf("expected empty body when no --section provided, got: %q", sessions[0].Body)
	}
}

// --- --section flag: error cases ---------------------------------------------

func TestSave_SectionFlag_UnknownSection_ReturnsError(t *testing.T) {
	setupInitedProject(t)

	err := runSave("unknown-section", nil, "", nil, []string{"UnknownSection=text"})
	if err == nil {
		t.Fatal("expected error for unknown --section name, got nil")
	}
	if !strings.Contains(err.Error(), "UnknownSection") {
		t.Errorf("expected unknown section name in error, got: %v", err)
	}
}

func TestSave_SectionFlag_UnknownSection_ListsAllowed(t *testing.T) {
	setupInitedProject(t)

	err := runSave("unknown-section", nil, "", nil, []string{"BadSection=text"})
	if err == nil {
		t.Fatal("expected error for unknown --section name, got nil")
	}
	// Error message should list allowed section names.
	if !strings.Contains(err.Error(), "Summary") {
		t.Errorf("expected allowed section names in error, got: %v", err)
	}
}

func TestSave_SectionFlag_InvalidFormat_ReturnsError(t *testing.T) {
	setupInitedProject(t)

	err := runSave("bad-format", nil, "", nil, []string{"NoEqualsSign"})
	if err == nil {
		t.Fatal("expected error for bad --section format, got nil")
	}
	if !strings.Contains(err.Error(), "Name=content") {
		t.Errorf("expected format hint in error, got: %v", err)
	}
}

func TestSave_SectionFlag_EmptyName_ReturnsError(t *testing.T) {
	setupInitedProject(t)

	err := runSave("empty-name", nil, "", nil, []string{"=some content"})
	if err == nil {
		t.Fatal("expected error for empty section name, got nil")
	}
}

func TestSave_SectionFlag_DuplicateSection_ReturnsError(t *testing.T) {
	setupInitedProject(t)

	err := runSave("dup-section", nil, "", nil, []string{"Summary=first", "Summary=second"})
	if err == nil {
		t.Fatal("expected error for duplicate --section name, got nil")
	}
	if !strings.Contains(err.Error(), "more than once") {
		t.Errorf("expected 'more than once' in error, got: %v", err)
	}
}

// --- --section flag: output ordering -----------------------------------------

func TestSave_SectionFlag_OutputOrderFollowsConfig(t *testing.T) {
	dir := setupInitedProject(t)

	// Provide sections in reverse config order; output must follow config order.
	sections := []string{"Key Decisions=- Decision A", "Summary=My summary."}
	if err := runSave("section-order", nil, "", nil, sections); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}

	sessions, err := session.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	body := sessions[0].Body
	summaryIdx := strings.Index(body, "## Summary")
	decisionsIdx := strings.Index(body, "## Key Decisions")
	if summaryIdx == -1 || decisionsIdx == -1 {
		t.Fatalf("expected both sections in body, got: %q", body)
	}
	if summaryIdx > decisionsIdx {
		t.Errorf("expected Summary before Key Decisions (config order), got body: %q", body)
	}
}

func TestSave_SectionFlag_HeadingLevelFromConfig(t *testing.T) {
	dir := setupInitedProject(t)

	// Default config sets sections at level 2 (##).
	if err := runSave("heading-level", nil, "", nil, []string{"Summary=text"}); err != nil {
		t.Fatalf("runSave failed: %v", err)
	}

	sessions, err := session.LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if !strings.Contains(sessions[0].Body, "## Summary") {
		t.Errorf("expected level-2 heading '## Summary', got: %q", sessions[0].Body)
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
	warnPrivacy("some content with sk-abc123", []string{})
}

func TestWarnPrivacy_InvalidPattern(t *testing.T) {
	warnPrivacy("content", []string{"[invalid"})
}

func TestWarnPrivacy_NoMatch(t *testing.T) {
	warnPrivacy("clean content", []string{`sk-[a-zA-Z0-9]+`})
}

// --- config integration ------------------------------------------------------

func TestSave_UsesConfigPrivacyPatterns(t *testing.T) {
	dir := setupInitedProject(t)

	cfg, _ := config.Load(dir)
	cfg.Privacy.FilterPatterns = []string{`sk-[a-zA-Z0-9]+`}
	_ = config.Save(dir, cfg)

	sections := []string{"Summary=Used sk-abc123 for auth."}
	if err := runSave("privacy-test", nil, "", nil, sections); err != nil {
		t.Fatalf("runSave should succeed even with privacy match, got: %v", err)
	}
}
