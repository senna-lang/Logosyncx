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
	if len(cfg.Sessions.SummarySections) == 0 {
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
		"sessions": {
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
	if len(cfg.Sessions.SummarySections) != 3 {
		t.Errorf("expected 3 summary_sections, got %d", len(cfg.Sessions.SummarySections))
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
	if len(cfg.Sessions.SummarySections) == 0 {
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

func TestDefault_GitAutoPushIsFalse(t *testing.T) {
	cfg := Default("auto-push-default")
	if cfg.Git.AutoPush {
		t.Error("expected Git.AutoPush to default to false")
	}
}

func TestLoad_AppliesDefaults_GitAutoPushIsFalse(t *testing.T) {
	dir := t.TempDir()

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
	if cfg.Git.AutoPush {
		t.Error("expected Git.AutoPush to be false when not set in config")
	}
}

func TestLoad_GitAutoPushTrue(t *testing.T) {
	dir := t.TempDir()

	raw := `{
		"project": "push-proj",
		"git": {
			"auto_push": true
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
	if !cfg.Git.AutoPush {
		t.Error("expected Git.AutoPush to be true when set in config")
	}
}

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	original := Config{
		Version:    "1",
		Project:    "roundtrip-proj",
		AgentsFile: "CLAUDE.md",
		Sessions: SessionsConfig{
			SummarySections: []string{"Summary", "Key Decisions", "Action Items"},
		},
		Privacy: PrivacyConfig{
			FilterPatterns: []string{`sk-[a-zA-Z0-9]+`, `ghp_[a-zA-Z0-9]+`},
		},
		Git: GitConfig{
			AutoPush: true,
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
	if len(loaded.Sessions.SummarySections) != len(original.Sessions.SummarySections) {
		t.Errorf("summary_sections length mismatch: got %d, want %d",
			len(loaded.Sessions.SummarySections), len(original.Sessions.SummarySections))
	}
	for i, s := range original.Sessions.SummarySections {
		if loaded.Sessions.SummarySections[i] != s {
			t.Errorf("summary_sections[%d]: got %q, want %q", i, loaded.Sessions.SummarySections[i], s)
		}
	}
	if len(loaded.Privacy.FilterPatterns) != len(original.Privacy.FilterPatterns) {
		t.Errorf("filter_patterns length mismatch: got %d, want %d",
			len(loaded.Privacy.FilterPatterns), len(original.Privacy.FilterPatterns))
	}
	if loaded.Git.AutoPush != original.Git.AutoPush {
		t.Errorf("git.auto_push mismatch: got %v, want %v", loaded.Git.AutoPush, original.Git.AutoPush)
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

// --- Migrate -----------------------------------------------------------------

func TestMigrate_NoFile(t *testing.T) {
	dir := t.TempDir()

	changed, err := Migrate(dir)
	if err != nil {
		t.Fatalf("expected no error when config missing, got: %v", err)
	}
	if changed {
		t.Error("expected changed=false when config.json does not exist")
	}
}

func TestMigrate_CompleteConfig(t *testing.T) {
	dir := t.TempDir()

	// Write a fully-populated config.json (all expected keys present).
	cfg := Default("complete-proj")
	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	changed, err := Migrate(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Error("expected changed=false for a complete config.json")
	}
}

func TestMigrate_MissingSessionsKey(t *testing.T) {
	dir := t.TempDir()

	// Old-style config using "save" instead of "sessions" — "sessions" key absent.
	raw := `{
		"version": "1",
		"project": "old-proj",
		"agents_file": "AGENTS.md",
		"save": {
			"summary_sections": ["Summary"],
			"excerpt_section": "Summary"
		},
		"tasks": {
			"default_status": "open",
			"default_priority": "medium",
			"summary_sections": ["What"],
			"excerpt_section": "What",
			"sections": []
		}
	}`

	cfgDir := filepath.Join(dir, DirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, ConfigFileName), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := Migrate(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true when 'sessions' key is absent")
	}

	// Verify the migrated file now contains the "sessions" key.
	reloaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after migration failed: %v", err)
	}
	if reloaded.Sessions.ExcerptSection == "" {
		t.Error("expected sessions.excerpt_section to be set after migration")
	}
	if len(reloaded.Sessions.Sections) == 0 {
		t.Error("expected sessions.sections to be non-empty after migration")
	}
}

func TestMigrate_MissingExcerptSection(t *testing.T) {
	dir := t.TempDir()

	// Config with "sessions" key but missing "excerpt_section" inside it.
	raw := `{
		"version": "1",
		"project": "partial-proj",
		"agents_file": "AGENTS.md",
		"sessions": {
			"summary_sections": ["Summary", "Key Decisions"],
			"sections": []
		},
		"tasks": {
			"default_status": "open",
			"default_priority": "medium",
			"summary_sections": ["What"],
			"excerpt_section": "What",
			"sections": []
		}
	}`

	cfgDir := filepath.Join(dir, DirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, ConfigFileName), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := Migrate(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true when sessions.excerpt_section is absent")
	}

	reloaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after migration failed: %v", err)
	}
	if reloaded.Sessions.ExcerptSection != "Summary" {
		t.Errorf("expected sessions.excerpt_section='Summary', got %q", reloaded.Sessions.ExcerptSection)
	}
}

func TestMigrate_MissingTasksExcerptSection(t *testing.T) {
	dir := t.TempDir()

	// Config with "tasks" key but missing "excerpt_section" inside it.
	raw := `{
		"version": "1",
		"project": "task-partial",
		"agents_file": "AGENTS.md",
		"sessions": {
			"summary_sections": ["Summary"],
			"excerpt_section": "Summary",
			"sections": []
		},
		"tasks": {
			"default_status": "open",
			"default_priority": "medium",
			"summary_sections": ["What"],
			"sections": []
		}
	}`

	cfgDir := filepath.Join(dir, DirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, ConfigFileName), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := Migrate(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true when tasks.excerpt_section is absent")
	}

	reloaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after migration failed: %v", err)
	}
	if reloaded.Tasks.ExcerptSection != "What" {
		t.Errorf("expected tasks.excerpt_section='What', got %q", reloaded.Tasks.ExcerptSection)
	}
}

func TestMigrate_PreservesExistingValues(t *testing.T) {
	dir := t.TempDir()

	// Config with custom values — Migrate should not overwrite them.
	raw := `{
		"version": "1",
		"project": "custom-proj",
		"agents_file": "CLAUDE.md",
		"sessions": {
			"summary_sections": ["Overview", "Decisions"],
			"excerpt_section": "Overview",
			"sections": []
		},
		"tasks": {
			"default_status": "open",
			"default_priority": "high",
			"summary_sections": ["What"],
			"excerpt_section": "What",
			"sections": []
		},
		"gc": {
			"linked_task_done_days": 30,
			"orphan_session_days": 90
		}
	}`

	cfgDir := filepath.Join(dir, DirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, ConfigFileName), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := Migrate(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Error("expected changed=false when all keys are already present")
	}

	// Double-check the custom values survived.
	reloaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if reloaded.Sessions.ExcerptSection != "Overview" {
		t.Errorf("excerpt_section overwritten: got %q, want 'Overview'", reloaded.Sessions.ExcerptSection)
	}
	if reloaded.AgentsFile != "CLAUDE.md" {
		t.Errorf("agents_file overwritten: got %q, want 'CLAUDE.md'", reloaded.AgentsFile)
	}
	if reloaded.Tasks.DefaultPriority != "high" {
		t.Errorf("tasks.default_priority overwritten: got %q, want 'high'", reloaded.Tasks.DefaultPriority)
	}
}

func TestMigrate_InvalidJSON(t *testing.T) {
	dir := t.TempDir()

	cfgDir := filepath.Join(dir, DirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, ConfigFileName), []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Malformed JSON should not return an error — Migrate leaves the file alone.
	changed, err := Migrate(dir)
	if err != nil {
		t.Fatalf("expected no error for invalid JSON, got: %v", err)
	}
	if changed {
		t.Error("expected changed=false for invalid JSON (file should be left alone)")
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	dir := t.TempDir()

	// Start with a config that needs migration.
	raw := `{"project": "idempotent-proj"}`

	cfgDir := filepath.Join(dir, DirName)
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, ConfigFileName), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	changed1, err := Migrate(dir)
	if err != nil {
		t.Fatalf("first Migrate failed: %v", err)
	}
	if !changed1 {
		t.Fatal("expected changed=true on first migration")
	}

	// Running Migrate a second time must be a no-op.
	changed2, err := Migrate(dir)
	if err != nil {
		t.Fatalf("second Migrate failed: %v", err)
	}
	if changed2 {
		t.Error("expected changed=false on second migration (idempotent)")
	}
}
