package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helpers

func runInitInDir(t *testing.T, dir string) error {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return runInit()
}

// --- runInit -----------------------------------------------------------------

func TestInit_CreatesLogosyncxDir(t *testing.T) {
	dir := t.TempDir()
	if err := runInitInDir(t, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".logosyncx")); os.IsNotExist(err) {
		t.Error("expected .logosyncx/ to be created")
	}
}

func TestInit_CreatesSessionsDir(t *testing.T) {
	dir := t.TempDir()
	if err := runInitInDir(t, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".logosyncx", "sessions")); os.IsNotExist(err) {
		t.Error("expected .logosyncx/sessions/ to be created")
	}
}

func TestInit_CreatesConfigJSON(t *testing.T) {
	dir := t.TempDir()
	if err := runInitInDir(t, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := filepath.Join(dir, ".logosyncx", "config.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected config.json to be created")
	}
}

func TestInit_ConfigJSON_ContainsProjectName(t *testing.T) {
	dir := t.TempDir()
	if err := runInitInDir(t, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".logosyncx", "config.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	projectName := filepath.Base(dir)
	if !strings.Contains(string(data), projectName) {
		t.Errorf("config.json missing project name %q, got: %s", projectName, data)
	}
}

func TestInit_CreatesUSAGEMD(t *testing.T) {
	dir := t.TempDir()
	if err := runInitInDir(t, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := filepath.Join(dir, ".logosyncx", "USAGE.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("USAGE.md not created: %v", err)
	}
	if !strings.Contains(string(data), "logos ls") {
		t.Error("USAGE.md should contain logos ls reference")
	}
}

func TestInit_CreatesTemplateMD(t *testing.T) {
	dir := t.TempDir()
	if err := runInitInDir(t, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	path := filepath.Join(dir, ".logosyncx", "template.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("template.md not created: %v", err)
	}
	if !strings.Contains(string(data), "{{topic}}") {
		t.Error("template.md should contain {{topic}} placeholder")
	}
	if !strings.Contains(string(data), "## Summary") {
		t.Error("template.md should contain ## Summary section")
	}
}

func TestInit_ErrorIfAlreadyInitialized(t *testing.T) {
	dir := t.TempDir()
	if err := runInitInDir(t, dir); err != nil {
		t.Fatalf("first init failed: %v", err)
	}
	err := runInitInDir(t, dir)
	if err == nil {
		t.Fatal("expected error on second init, got nil")
	}
	if !strings.Contains(err.Error(), "already initialized") {
		t.Errorf("expected 'already initialized' in error, got: %v", err)
	}
}

// --- detectAgentsFile --------------------------------------------------------

func TestDetectAgentsFile_DefaultsToAgentsMD(t *testing.T) {
	dir := t.TempDir()
	got := detectAgentsFile(dir)
	if got != "AGENTS.md" {
		t.Errorf("expected AGENTS.md, got %q", got)
	}
}

func TestDetectAgentsFile_PrefersCLAUDEMD(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte("# Claude\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := detectAgentsFile(dir)
	if got != "CLAUDE.md" {
		t.Errorf("expected CLAUDE.md, got %q", got)
	}
}

func TestDetectAgentsFile_IgnoresCLAUDEMDIfAbsent(t *testing.T) {
	dir := t.TempDir()
	// Only AGENTS.md exists.
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Agents\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := detectAgentsFile(dir)
	if got != "AGENTS.md" {
		t.Errorf("expected AGENTS.md, got %q", got)
	}
}

// --- appendAgentsLine --------------------------------------------------------

func TestAppendAgentsLine_CreatesFileIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if err := appendAgentsLine(path); err != nil {
		t.Fatalf("appendAgentsLine failed: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected AGENTS.md to be created")
	}
}

func TestAppendAgentsLine_ContainsUSAGEReference(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	if err := appendAgentsLine(path); err != nil {
		t.Fatalf("appendAgentsLine failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "logosyncx/USAGE.md") {
		t.Errorf("expected USAGE.md reference, got: %s", data)
	}
	if !strings.Contains(string(data), "logos") {
		t.Errorf("expected logos reference, got: %s", data)
	}
}

func TestAppendAgentsLine_AppendsToExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")
	existing := "# Agent Instructions\n\nDo stuff.\n"
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := appendAgentsLine(path); err != nil {
		t.Fatalf("appendAgentsLine failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Do stuff.") {
		t.Error("existing content should be preserved")
	}
	if !strings.Contains(content, "logosyncx/USAGE.md") {
		t.Error("USAGE.md reference should be appended")
	}
}

func TestAppendAgentsLine_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "AGENTS.md")

	if err := appendAgentsLine(path); err != nil {
		t.Fatalf("first append failed: %v", err)
	}
	if err := appendAgentsLine(path); err != nil {
		t.Fatalf("second append failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	count := strings.Count(string(data), "logosyncx/USAGE.md")
	if count != 1 {
		t.Errorf("expected exactly 1 USAGE.md reference, got %d", count)
	}
}

// --- init command detects correct agents file --------------------------------

func TestInit_UsesAgentsMDByDefault(t *testing.T) {
	dir := t.TempDir()
	if err := runInitInDir(t, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	agentsPath := filepath.Join(dir, "AGENTS.md")
	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("AGENTS.md not created: %v", err)
	}
	if !strings.Contains(string(data), "logosyncx/USAGE.md") {
		t.Error("AGENTS.md should contain USAGE.md reference")
	}
}

func TestInit_UsesCLAUDEMDIfPresent(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte("# Claude\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runInitInDir(t, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "logosyncx/USAGE.md") {
		t.Error("CLAUDE.md should contain USAGE.md reference")
	}
	// AGENTS.md should NOT have been created.
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
		t.Error("AGENTS.md should not be created when CLAUDE.md is present")
	}
}

func TestInit_ConfigRecordsAgentsFile(t *testing.T) {
	dir := t.TempDir()
	claudePath := filepath.Join(dir, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte("# Claude\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runInitInDir(t, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".logosyncx", "config.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "CLAUDE.md") {
		t.Errorf("config.json should record agents_file as CLAUDE.md, got: %s", data)
	}
}
