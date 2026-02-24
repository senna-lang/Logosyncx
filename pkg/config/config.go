// Package config provides types and functions for loading, saving, and
// applying defaults to the .logosyncx/config.json project configuration file.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

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

// SaveConfig holds settings related to session saving.
type SaveConfig struct {
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
	// Sections defines the ordered list of body sections for task files.
	// Sections with Required:true will trigger a warning if absent when creating
	// a task. These sections are not used for indexing, so all sections are freely
	// editable without breaking existing tasks.
	Sections []SectionConfig `json:"sections"`
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
	Version    string        `json:"version"`
	Project    string        `json:"project"`
	AgentsFile string        `json:"agents_file"`
	Save       SaveConfig    `json:"save"`
	Tasks      TasksConfig   `json:"tasks"`
	Privacy    PrivacyConfig `json:"privacy"`
	Git        GitConfig     `json:"git"`
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
		Save: SaveConfig{
			SummarySections: []string{"Summary", "Key Decisions"},
			ExcerptSection:  "Summary",
			Sections:        defaultSessionSections,
		},
		Tasks: TasksConfig{
			DefaultStatus:   "open",
			DefaultPriority: "medium",
			SummarySections: []string{"What", "Checklist"},
			Sections:        defaultTaskSections,
		},
		Privacy: PrivacyConfig{
			FilterPatterns: []string{},
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
	if len(cfg.Save.SummarySections) == 0 {
		cfg.Save.SummarySections = []string{"Summary", "Key Decisions"}
	}
	if cfg.Save.ExcerptSection == "" {
		cfg.Save.ExcerptSection = "Summary"
	}
	if len(cfg.Save.Sections) == 0 {
		cfg.Save.Sections = defaultSessionSections
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
	if len(cfg.Tasks.Sections) == 0 {
		cfg.Tasks.Sections = defaultTaskSections
	}
	if cfg.Privacy.FilterPatterns == nil {
		cfg.Privacy.FilterPatterns = []string{}
	}
}
