// Package config provides types and functions for loading, saving, and
// applying defaults to the .logosyncx/config.json project configuration file.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// requiredSessionKeys lists the JSON keys that must be present in the
// "sessions" object for a config.json to be considered up-to-date.
var requiredSessionKeys = []string{
	"summary_sections",
	"excerpt_section",
	"sections",
}

// requiredTaskKeys lists the JSON keys that must be present in the
// "tasks" object for a config.json to be considered up-to-date.
var requiredTaskKeys = []string{
	"default_status",
	"default_priority",
	"summary_sections",
	"excerpt_section",
	"sections",
}

// requiredGcKeys lists the JSON keys that must be present in the
// "gc" object for a config.json to be considered up-to-date.
var requiredGcKeys = []string{
	"linked_task_done_days",
	"orphan_session_days",
}

const (
	DirName        = ".logosyncx"
	ConfigFileName = "config.json"
)

// SectionConfig defines a single section in a session or task body template.
// Level is the markdown heading level (1–6). Required, when true, causes
// logos save / logos task create to warn if the section is absent from the body.
type SectionConfig struct {
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Required bool   `json:"required"`
}

// SessionsConfig holds settings related to session files.
type SessionsConfig struct {
	// SummarySections lists the section headings returned by logos refer --summary.
	SummarySections []string `json:"summary_sections"`
	// ExcerptSection is the section whose content is used as the session excerpt
	// stored in the index. This must match the first Required section and should
	// not be renamed after sessions have been saved (doing so breaks index consistency).
	ExcerptSection string `json:"excerpt_section"`
	// Sections defines the ordered list of body sections for session files.
	// The first Required section is treated as the excerpt source and must remain
	// named "Summary" (or whatever ExcerptSection is set to) throughout a project's
	// lifetime. Other sections may be freely added, removed, or renamed.
	Sections []SectionConfig `json:"sections"`
}

// TasksConfig holds settings related to task management.
type TasksConfig struct {
	DefaultStatus   string   `json:"default_status"`
	DefaultPriority string   `json:"default_priority"`
	SummarySections []string `json:"summary_sections"`
	// ExcerptSection is the section whose content is used as the task excerpt
	// stored in the task index. Defaults to "What". Renaming this after tasks
	// have been saved breaks index consistency (same caveat as sessions).
	ExcerptSection string `json:"excerpt_section"`
	// Sections defines the ordered list of body sections for task files.
	// Sections with Required:true will trigger a warning if absent when creating
	// a task. These sections are not used for indexing, so all sections are freely
	// editable without breaking existing tasks.
	Sections []SectionConfig `json:"sections"`
}

// GcConfig holds settings that control the session garbage-collection behaviour
// of `logos gc`. Thresholds here serve as project-wide defaults; they can be
// overridden on the command line with --linked-days and --orphan-days.
type GcConfig struct {
	// LinkedTaskDoneDays is the number of days that must have elapsed since
	// the latest linked task was completed before a session becomes a strong
	// GC candidate. Falls back to session creation date when completed_at is
	// not recorded. Default: 30.
	LinkedTaskDoneDays int `json:"linked_task_done_days"`
	// OrphanSessionDays is the number of days since a session was created
	// before it becomes a weak GC candidate (no linked tasks). Default: 90.
	OrphanSessionDays int `json:"orphan_session_days"`
}

// GitConfig holds settings related to git automation behaviour.
type GitConfig struct {
	// AutoPush, when true, makes logos save automatically run git commit and
	// git push after staging the session file.  Defaults to false so that
	// humans (and agents that prefer manual control) can review before pushing.
	AutoPush bool `json:"auto_push"`
}

// PrivacyConfig holds settings related to privacy filtering.
type PrivacyConfig struct {
	FilterPatterns []string `json:"filter_patterns"`
}

// Config represents the contents of .logosyncx/config.json.
type Config struct {
	Version    string         `json:"version"`
	Project    string         `json:"project"`
	AgentsFile string         `json:"agents_file"`
	Sessions   SessionsConfig `json:"sessions"`
	Tasks      TasksConfig    `json:"tasks"`
	Privacy    PrivacyConfig  `json:"privacy"`
	Git        GitConfig      `json:"git"`
	GC         GcConfig       `json:"gc"`
}

// defaultSessionSections mirrors the sections that were previously defined in
// .logosyncx/template.md. Summary is required (used for excerpt extraction);
// the remaining sections are optional user-managed content.
var defaultSessionSections = []SectionConfig{
	{Name: "Summary", Level: 2, Required: true},
	{Name: "Key Decisions", Level: 2, Required: false},
	{Name: "Context Used", Level: 2, Required: false},
	{Name: "Notes", Level: 2, Required: false},
	{Name: "Raw Conversation", Level: 2, Required: false},
}

// defaultTaskSections mirrors the sections that were previously defined in
// .logosyncx/task-template.md. What is required; the rest are optional.
var defaultTaskSections = []SectionConfig{
	{Name: "What", Level: 2, Required: true},
	{Name: "Why", Level: 2, Required: false},
	{Name: "Scope", Level: 2, Required: false},
	{Name: "Checklist", Level: 2, Required: false},
	{Name: "Notes", Level: 2, Required: false},
}

// Default returns a Config populated with sensible default values.
func Default(projectName string) Config {
	return Config{
		Version:    "1",
		Project:    projectName,
		AgentsFile: "AGENTS.md",
		Sessions: SessionsConfig{
			SummarySections: []string{"Summary", "Key Decisions"},
			ExcerptSection:  "Summary",
			Sections:        defaultSessionSections,
		},
		Tasks: TasksConfig{
			DefaultStatus:   "open",
			DefaultPriority: "medium",
			SummarySections: []string{"What", "Checklist"},
			ExcerptSection:  "What",
			Sections:        defaultTaskSections,
		},
		Privacy: PrivacyConfig{
			FilterPatterns: []string{},
		},
		GC: GcConfig{
			LinkedTaskDoneDays: 30,
			OrphanSessionDays:  90,
		},
	}
}

// ConfigPath returns the path to config.json given the project root.
func ConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, DirName, ConfigFileName)
}

// Load reads and parses config.json from the given project root.
// If the file does not exist, it returns a default Config and no error.
// Missing fields are filled with defaults after parsing.
func Load(projectRoot string) (Config, error) {
	path := ConfigPath(projectRoot)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(filepath.Base(projectRoot)), nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	applyDefaults(&cfg, projectRoot)
	return cfg, nil
}

// Save serialises cfg and writes it to config.json under the given project root.
// The .logosyncx directory is created if it does not exist.
func Save(projectRoot string, cfg Config) error {
	dir := filepath.Join(projectRoot, DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	return os.WriteFile(ConfigPath(projectRoot), data, 0o644)
}

// Migrate checks whether any expected fields are absent from the on-disk
// config.json and, if so, re-writes the file with all default values applied.
// It returns (true, nil) when the file was updated, and (false, nil) when it
// was already complete or when config.json does not exist (logos init creates
// it from scratch, so there is nothing to migrate).
//
// Migrate is intentionally conservative: it only adds missing fields; it never
// removes or overrides fields that are already present.
func Migrate(projectRoot string) (bool, error) {
	path := ConfigPath(projectRoot)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	// Parse the raw JSON to detect absent keys — this distinguishes a truly
	// absent key from one that is present but set to a zero value.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		// Malformed JSON: leave the file alone and let Load surface the error.
		return false, nil
	}

	if !isMigrationNeeded(raw) {
		return false, nil
	}

	// Load applies applyDefaults in memory; Save writes the result back.
	cfg, err := Load(projectRoot)
	if err != nil {
		return false, err
	}
	if err := Save(projectRoot, cfg); err != nil {
		return false, err
	}
	return true, nil
}

// isMigrationNeeded returns true when any expected key is absent from the
// parsed top-level or nested JSON objects.
func isMigrationNeeded(raw map[string]json.RawMessage) bool {
	// Top-level scalar fields.
	for _, key := range []string{"version", "agents_file"} {
		if _, ok := raw[key]; !ok {
			return true
		}
	}

	// sessions sub-keys.
	sessRaw, ok := raw["sessions"]
	if !ok {
		return true
	}
	var sessMap map[string]json.RawMessage
	if err := json.Unmarshal(sessRaw, &sessMap); err != nil {
		return true
	}
	for _, key := range requiredSessionKeys {
		if _, ok := sessMap[key]; !ok {
			return true
		}
	}

	// tasks sub-keys.
	tasksRaw, ok := raw["tasks"]
	if !ok {
		return true
	}
	var tasksMap map[string]json.RawMessage
	if err := json.Unmarshal(tasksRaw, &tasksMap); err != nil {
		return true
	}
	for _, key := range requiredTaskKeys {
		if _, ok := tasksMap[key]; !ok {
			return true
		}
	}

	// gc sub-keys.
	gcRaw, ok := raw["gc"]
	if !ok {
		return true
	}
	var gcMap map[string]json.RawMessage
	if err := json.Unmarshal(gcRaw, &gcMap); err != nil {
		return true
	}
	for _, key := range requiredGcKeys {
		if _, ok := gcMap[key]; !ok {
			return true
		}
	}

	return false
}

// applyDefaults fills in zero-value fields with sensible defaults.
func applyDefaults(cfg *Config, projectRoot string) {
	if cfg.Version == "" {
		cfg.Version = "1"
	}
	if cfg.Project == "" {
		cfg.Project = filepath.Base(projectRoot)
	}
	if cfg.AgentsFile == "" {
		cfg.AgentsFile = "AGENTS.md"
	}
	if len(cfg.Sessions.SummarySections) == 0 {
		cfg.Sessions.SummarySections = []string{"Summary", "Key Decisions"}
	}
	if cfg.Sessions.ExcerptSection == "" {
		cfg.Sessions.ExcerptSection = "Summary"
	}
	if len(cfg.Sessions.Sections) == 0 {
		cfg.Sessions.Sections = defaultSessionSections
	}
	if cfg.Tasks.DefaultStatus == "" {
		cfg.Tasks.DefaultStatus = "open"
	}
	if cfg.Tasks.DefaultPriority == "" {
		cfg.Tasks.DefaultPriority = "medium"
	}
	if len(cfg.Tasks.SummarySections) == 0 {
		cfg.Tasks.SummarySections = []string{"What", "Checklist"}
	}
	if cfg.Tasks.ExcerptSection == "" {
		cfg.Tasks.ExcerptSection = "What"
	}
	if len(cfg.Tasks.Sections) == 0 {
		cfg.Tasks.Sections = defaultTaskSections
	}
	if cfg.Privacy.FilterPatterns == nil {
		cfg.Privacy.FilterPatterns = []string{}
	}
	if cfg.GC.LinkedTaskDoneDays == 0 {
		cfg.GC.LinkedTaskDoneDays = 30
	}
	if cfg.GC.OrphanSessionDays == 0 {
		cfg.GC.OrphanSessionDays = 90
	}
}
