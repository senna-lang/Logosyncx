package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default("my-project")

	if cfg.Version != "1" {
		t.Errorf("expected version '1', got %q", cfg.Version)
	}
	if cfg.Project != "my-project" {
		t.Errorf("expected project 'my-project', got %q", cfg.Project)
	}
	if cfg.AgentsFile != "AGENTS.md" {
		t.Errorf("expected agents_file 'AGENTS.md', got %q", cfg.AgentsFile)
	}
	if len(cfg.Save.SummarySections) == 0 {
		t.Error("expected non-empty summary_sections")
	}
	if cfg.Privacy.FilterPatterns == nil {
		t.Error("expected filter_patterns to be non-nil slice")
	}
}

func TestConfigPath(t *testing.T) {
	got := ConfigPath("/home/user/myproject")
	want := filepath.Join("/home/user/myproject", DirName, ConfigFileName)
	if got != want {
		t.Errorf("ConfigPath = %q, want %q", got, want)
	}
}

func TestLoad_FileNotExist(t *testing.T) {
	dir := t.TempDir()

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("expected no error when config missing, got: %v", err)
	}
	if cfg.Version != "1" {
		t.Errorf("expected default version '1', got %q", cfg.Version)
	}
	if cfg.Project != filepath.Base(dir) {
		t.Errorf("expected project %q, got %q", filepath.Base(dir), cfg.Project)
	}
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()

	raw := `{
		"version": "1",
		"project": "test-proj",
		"agents_file": "CLAUDE.md",
		"save": {
			"summary_sections": ["Summary", "Decisions", "Action Items"]
		},
		"privacy": {
			"filter_patterns": ["sk-[a-zA-Z0-9]+"]
		}
	}`

	cfgDir := filepath.Join(dir, DirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, ConfigFileName), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Project != "test-proj" {
		t.Errorf("expected project 'test-proj', got %q", cfg.Project)
	}
	if cfg.AgentsFile != "CLAUDE.md" {
		t.Errorf("expected agents_file 'CLAUDE.md', got %q", cfg.AgentsFile)
	}
	if len(cfg.Save.SummarySections) != 3 {
		t.Errorf("expected 3 summary_sections, got %d", len(cfg.Save.SummarySections))
	}
	if len(cfg.Privacy.FilterPatterns) != 1 {
		t.Errorf("expected 1 filter_pattern, got %d", len(cfg.Privacy.FilterPatterns))
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()

	cfgDir := filepath.Join(dir, DirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, ConfigFileName), []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoad_AppliesDefaults(t *testing.T) {
	dir := t.TempDir()

	// Write a config with only the project field set — everything else missing.
	raw := `{"project": "partial-proj"}`

	cfgDir := filepath.Join(dir, DirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, ConfigFileName), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != "1" {
		t.Errorf("expected default version '1', got %q", cfg.Version)
	}
	if cfg.AgentsFile != "AGENTS.md" {
		t.Errorf("expected default agents_file 'AGENTS.md', got %q", cfg.AgentsFile)
	}
	if len(cfg.Save.SummarySections) == 0 {
		t.Error("expected default summary_sections to be applied")
	}
	if cfg.Privacy.FilterPatterns == nil {
		t.Error("expected filter_patterns to be non-nil after defaults")
	}
}

func TestSave_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	cfg := Default("save-test")

	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	path := ConfigPath(dir)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("expected config file to exist at %s", path)
	}
}

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	original := Config{
		Version:    "1",
		Project:    "roundtrip-proj",
		AgentsFile: "CLAUDE.md",
		Save: SaveConfig{
			SummarySections: []string{"Summary", "Key Decisions", "Action Items"},
		},
		Privacy: PrivacyConfig{
			FilterPatterns: []string{`sk-[a-zA-Z0-9]+`, `ghp_[a-zA-Z0-9]+`},
		},
	}

	if err := Save(dir, original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Project != original.Project {
		t.Errorf("project mismatch: got %q, want %q", loaded.Project, original.Project)
	}
	if loaded.AgentsFile != original.AgentsFile {
		t.Errorf("agents_file mismatch: got %q, want %q", loaded.AgentsFile, original.AgentsFile)
	}
	if len(loaded.Save.SummarySections) != len(original.Save.SummarySections) {
		t.Errorf("summary_sections length mismatch: got %d, want %d",
			len(loaded.Save.SummarySections), len(original.Save.SummarySections))
	}
	for i, s := range original.Save.SummarySections {
		if loaded.Save.SummarySections[i] != s {
			t.Errorf("summary_sections[%d]: got %q, want %q", i, loaded.Save.SummarySections[i], s)
		}
	}
	if len(loaded.Privacy.FilterPatterns) != len(original.Privacy.FilterPatterns) {
		t.Errorf("filter_patterns length mismatch: got %d, want %d",
			len(loaded.Privacy.FilterPatterns), len(original.Privacy.FilterPatterns))
	}
}

func TestSave_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	cfg := Default("json-check")

	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	data, err := os.ReadFile(ConfigPath(dir))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("saved file is not valid JSON: %v", err)
	}
}

func TestSave_CreatesDirectoryIfMissing(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "deep", "project")

	cfg := Default("nested")
	if err := Save(nested, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := os.Stat(ConfigPath(nested)); os.IsNotExist(err) {
		t.Fatal("expected config file to be created in nested directory")
	}
}
