package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- FileName ----------------------------------------------------------------

func TestKnowledge_FileName_Format(t *testing.T) {
	date := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	k := Knowledge{
		Topic: "auth refactor",
		Date:  &date,
	}
	name := FileName(k)

	if !strings.HasPrefix(name, "20260610-") {
		t.Errorf("FileName = %q, want prefix '20260610-'", name)
	}
	if !strings.HasSuffix(name, ".md") {
		t.Errorf("FileName = %q, want suffix '.md'", name)
	}
	if name != "20260610-auth-refactor.md" {
		t.Errorf("FileName = %q, want '20260610-auth-refactor.md'", name)
	}
}

func TestKnowledge_FileName_NilDateUsesNow(t *testing.T) {
	k := Knowledge{Topic: "test"}
	name := FileName(k)
	if len(name) < 9 || name[8] != '-' {
		t.Errorf("FileName with nil date = %q, want YYYYMMDD- prefix", name)
	}
}

// --- Write -------------------------------------------------------------------

func TestWrite_CreatesFile(t *testing.T) {
	root := t.TempDir()
	date := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	k := Knowledge{
		Topic: "auth refactor",
		Plan:  "20260601-auth-refactor.md",
		Tags:  []string{"auth", "go"},
		Date:  &date,
	}

	rel, err := Write(root, k, "plan body content", "## Summary\n## Key Learnings\n")
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	path := filepath.Join(root, rel)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", path)
	}
}

func TestWrite_ContainsSourceBlock(t *testing.T) {
	root := t.TempDir()
	date := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	k := Knowledge{
		Topic: "test topic",
		Plan:  "20260601-test.md",
		Date:  &date,
	}
	sourceBlock := "## Plan: test\n\nSome plan content.\n\n## Walkthrough: 001 task\nDid stuff."

	rel, err := Write(root, k, sourceBlock, "")
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "<!-- SOURCE MATERIAL") {
		t.Error("expected SOURCE MATERIAL comment header")
	}
	if !strings.Contains(content, sourceBlock) {
		t.Error("expected source block content in file")
	}
	if !strings.Contains(content, "-->") {
		t.Error("expected closing --> for source block")
	}
}

func TestWrite_ContainsSectionHeadings(t *testing.T) {
	root := t.TempDir()
	date := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	k := Knowledge{
		Topic: "test topic",
		Plan:  "20260601-test.md",
		Date:  &date,
	}
	templateSections := "## Summary\n\nSome description.\n\n## Key Learnings\n\n## Reusable Patterns\n"

	rel, err := Write(root, k, "source", templateSections)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	for _, heading := range []string{"## Summary", "## Key Learnings", "## Reusable Patterns"} {
		if !strings.Contains(content, heading) {
			t.Errorf("expected heading %q in file", heading)
		}
	}
}

func TestWrite_AutoGeneratesID(t *testing.T) {
	root := t.TempDir()
	date := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	k := Knowledge{
		Topic: "no id",
		Plan:  "plan.md",
		Date:  &date,
	}

	rel, err := Write(root, k, "", "")
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if !strings.Contains(string(data), "id: k-") {
		t.Error("expected auto-generated id starting with 'k-'")
	}
}

func TestWrite_RelativePathUnderKnowledgeDir(t *testing.T) {
	root := t.TempDir()
	date := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	k := Knowledge{
		Topic: "path test",
		Plan:  "plan.md",
		Date:  &date,
	}

	rel, err := Write(root, k, "", "")
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if !strings.HasPrefix(rel, ".logosyncx/knowledge/") {
		t.Errorf("returned path %q should start with '.logosyncx/knowledge/'", rel)
	}
}
