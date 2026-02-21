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

// SaveConfig holds settings related to session saving.
type SaveConfig struct {
	SummarySections []string `json:"summary_sections"`
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
	Privacy    PrivacyConfig `json:"privacy"`
}

// Default returns a Config populated with sensible default values.
func Default(projectName string) Config {
	return Config{
		Version:    "1",
		Project:    projectName,
		AgentsFile: "AGENTS.md",
		Save: SaveConfig{
			SummarySections: []string{"Summary", "Key Decisions"},
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

	data, err := json.MarshalIndent(cfg, "", "  ")
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
	if cfg.Privacy.FilterPatterns == nil {
		cfg.Privacy.FilterPatterns = []string{}
	}
}
