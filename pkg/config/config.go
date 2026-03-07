// Package config provides types and functions for loading, saving, and
// applying defaults to the .logosyncx/config.json project configuration file.
// Version "2" schema: sessions renamed to plans, sections arrays removed,
// knowledge section added, orphan_plan_days replaces orphan_session_days.
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

// PlansConfig holds settings related to plan files.
type PlansConfig struct {
	// SummarySections lists the section headings returned by logos refer --summary.
	SummarySections []string `json:"summary_sections"`
	// ExcerptSection is the section whose content is used as the plan excerpt
	// stored in the index.
	ExcerptSection string `json:"excerpt_section"`
}

// TasksConfig holds settings related to task management.
type TasksConfig struct {
	DefaultStatus   string   `json:"default_status"`
	DefaultPriority string   `json:"default_priority"`
	SummarySections []string `json:"summary_sections"`
	// ExcerptSection is the section whose content is used as the task excerpt
	// stored in the task index.
	ExcerptSection string `json:"excerpt_section"`
}

// KnowledgeConfig holds settings related to knowledge files.
type KnowledgeConfig struct {
	// SummarySections lists the section headings returned by logos refer --summary
	// on a knowledge file.
	SummarySections []string `json:"summary_sections"`
	// ExcerptSection is the section whose content is used as the knowledge excerpt.
	ExcerptSection string `json:"excerpt_section"`
}

// GcConfig holds settings that control the plan garbage-collection behaviour
// of `logos gc`. Thresholds here serve as project-wide defaults; they can be
// overridden on the command line with --linked-days and --orphan-days.
type GcConfig struct {
	// LinkedTaskDoneDays is the number of days that must have elapsed since
	// the latest linked task was completed before a plan becomes a strong
	// GC candidate. Default: 30.
	LinkedTaskDoneDays int `json:"linked_task_done_days"`
	// OrphanPlanDays is the number of days since a plan was created before it
	// becomes a weak GC candidate (no tasks). Default: 90.
	OrphanPlanDays int `json:"orphan_plan_days"`
}

// GitConfig holds settings related to git automation behaviour.
type GitConfig struct {
	// AutoPush, when true, makes logos save automatically run git commit and
	// git push after staging the plan file. Defaults to false.
	AutoPush bool `json:"auto_push"`
}

// PrivacyConfig holds settings related to privacy filtering.
type PrivacyConfig struct {
	FilterPatterns []string `json:"filter_patterns"`
}

// Config represents the contents of .logosyncx/config.json.
type Config struct {
	Version    string          `json:"version"`
	Project    string          `json:"project"`
	AgentsFile string          `json:"agents_file"`
	Plans      PlansConfig     `json:"plans"`
	Tasks      TasksConfig     `json:"tasks"`
	Knowledge  KnowledgeConfig `json:"knowledge"`
	Privacy    PrivacyConfig   `json:"privacy"`
	Git        GitConfig       `json:"git"`
	GC         GcConfig        `json:"gc"`
}

// Default returns a Config populated with sensible default values.
func Default(projectName string) Config {
	return Config{
		Version:    "2",
		Project:    projectName,
		AgentsFile: "AGENTS.md",
		Plans: PlansConfig{
			SummarySections: []string{"Background", "Spec"},
			ExcerptSection:  "Background",
		},
		Tasks: TasksConfig{
			DefaultStatus:   "open",
			DefaultPriority: "medium",
			SummarySections: []string{"What", "Checklist"},
			ExcerptSection:  "What",
		},
		Knowledge: KnowledgeConfig{
			SummarySections: []string{"Summary", "Key Learnings"},
			ExcerptSection:  "Summary",
		},
		Privacy: PrivacyConfig{
			FilterPatterns: []string{},
		},
		GC: GcConfig{
			LinkedTaskDoneDays: 30,
			OrphanPlanDays:     90,
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
		cfg.Version = "2"
	}
	if cfg.Project == "" {
		cfg.Project = filepath.Base(projectRoot)
	}
	if cfg.AgentsFile == "" {
		cfg.AgentsFile = "AGENTS.md"
	}
	if len(cfg.Plans.SummarySections) == 0 {
		cfg.Plans.SummarySections = []string{"Background", "Spec"}
	}
	if cfg.Plans.ExcerptSection == "" {
		cfg.Plans.ExcerptSection = "Background"
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
	if len(cfg.Knowledge.SummarySections) == 0 {
		cfg.Knowledge.SummarySections = []string{"Summary", "Key Learnings"}
	}
	if cfg.Knowledge.ExcerptSection == "" {
		cfg.Knowledge.ExcerptSection = "Summary"
	}
	if cfg.Privacy.FilterPatterns == nil {
		cfg.Privacy.FilterPatterns = []string{}
	}
	if cfg.GC.LinkedTaskDoneDays == 0 {
		cfg.GC.LinkedTaskDoneDays = 30
	}
	if cfg.GC.OrphanPlanDays == 0 {
		cfg.GC.OrphanPlanDays = 90
	}
}
