// Package plan provides types and functions for reading, writing, and
// parsing Logosyncx plan files — Markdown documents with YAML frontmatter
// stored under .logosyncx/plans/.
//
// Filename format: YYYYMMDD-<slug>.md (e.g. 20260304-auth-refactor.md).
// Write creates a frontmatter scaffold only; the agent fills the body using
// the Write tool guided by .logosyncx/templates/plan.md.
package plan

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const (
	plansDirName    = "plans"
	excerptMaxRunes = 300
	frontmatterSep  = "---"
)

// Plan represents a single plan file stored under .logosyncx/plans/.
type Plan struct {
	// Frontmatter fields.
	ID        string     `yaml:"id"`
	Date      *time.Time `yaml:"date,omitempty"`
	Topic     string     `yaml:"topic"`
	Tags      []string   `yaml:"tags"`
	Agent     string     `yaml:"agent"`
	Related   []string   `yaml:"related"`
	DependsOn []string   `yaml:"depends_on,omitempty"` // plan filenames this plan depends on
	TasksDir  string     `yaml:"tasks_dir"`
	Distilled bool       `yaml:"distilled"`

	// Derived fields (not written to frontmatter).
	Filename string `yaml:"-"`
	Excerpt  string `yaml:"-"`
	Body     string `yaml:"-"` // full markdown body (everything after frontmatter)
}

// PlansDir returns the path to the plans directory under a project root.
func PlansDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".logosyncx", plansDirName)
}

// ArchiveDir returns the path to the archive subdirectory under plans/.
func ArchiveDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".logosyncx", plansDirName, "archive")
}

// FileName returns the canonical filename for a plan: YYYYMMDD-<slug>.md.
// If Date is nil, the current time is used as a fallback.
func FileName(p Plan) string {
	t := time.Now()
	if p.Date != nil {
		t = *p.Date
	}
	return fmt.Sprintf("%s-%s.md", t.Format("20060102"), slugify(p.Topic))
}

// DefaultTasksDir returns the default tasks_dir for a plan given its filename.
// e.g. "20260304-auth-refactor.md" → ".logosyncx/tasks/20260304-auth-refactor"
func DefaultTasksDir(filename string) string {
	stem := strings.TrimSuffix(filename, ".md")
	return filepath.Join(".logosyncx", "tasks", stem)
}

// ParseOptions controls optional behaviour of Parse.
type ParseOptions struct {
	// ExcerptSection is the heading name used to extract the excerpt.
	// Defaults to "Background" when empty. Matched case-insensitively.
	ExcerptSection string
}

// Parse reads a plan markdown file from data.
func Parse(filename string, data []byte) (Plan, error) {
	return ParseWithOptions(filename, data, ParseOptions{})
}

// ParseWithOptions is like Parse but accepts options to customise excerpt
// extraction.
func ParseWithOptions(filename string, data []byte, opts ParseOptions) (Plan, error) {
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return Plan{}, fmt.Errorf("parse %s: %w", filename, err)
	}

	var p Plan
	if err := yaml.Unmarshal(fm, &p); err != nil {
		hint := ""
		if bytes.Contains(fm, []byte("{{")) {
			hint = " (hint: frontmatter contains '{{' — replace template placeholders before saving)"
		}
		return Plan{}, fmt.Errorf("parse frontmatter in %s: %w%s", filename, err, hint)
	}

	p.Filename = filename
	p.Body = string(body)
	p.Excerpt = extractExcerpt(body, opts.ExcerptSection)

	return p, nil
}

// LoadFile reads and parses a plan file at the given path.
func LoadFile(path string) (Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Plan{}, err
	}
	return Parse(filepath.Base(path), data)
}

// LoadAll reads every .md file from the plans directory under projectRoot
// and returns the parsed plans. Files that fail to parse are skipped and
// their errors collected (non-fatal).
func LoadAll(projectRoot string) ([]Plan, error) {
	return LoadAllWithOptions(projectRoot, ParseOptions{})
}

// LoadAllWithOptions is like LoadAll but parses each file with the given
// ParseOptions.
func LoadAllWithOptions(projectRoot string, opts ParseOptions) ([]Plan, error) {
	dir := PlansDir(projectRoot)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var plans []Plan
	var errs []string

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", entry.Name(), err))
			continue
		}
		p, err := ParseWithOptions(entry.Name(), data, opts)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", entry.Name(), err))
			continue
		}
		plans = append(plans, p)
	}

	if len(errs) > 0 {
		return plans, fmt.Errorf("some plan files could not be parsed:\n  %s",
			strings.Join(errs, "\n  "))
	}
	return plans, nil
}

// Write creates a frontmatter scaffold for p under projectRoot/plans/.
// The plans directory is created if it does not exist.
// Body is intentionally left empty — the agent fills it using the Write tool.
// Returns the full path of the written file.
func Write(projectRoot string, p Plan) (string, error) {
	dir := PlansDir(projectRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	data, err := Marshal(p)
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, FileName(p))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Marshal serialises a Plan to its markdown representation (YAML frontmatter
// followed by the body when non-empty). Write calls Marshal to produce scaffold
// files (body empty), while other callers such as logos distill use it to
// rewrite an existing plan preserving its body.
func Marshal(p Plan) ([]byte, error) {
	fm, err := yaml.Marshal(p)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString(frontmatterSep + "\n")
	buf.Write(fm)
	buf.WriteString(frontmatterSep + "\n")
	if p.Body != "" {
		if !strings.HasPrefix(p.Body, "\n") {
			buf.WriteByte('\n')
		}
		buf.WriteString(p.Body)
	}
	return buf.Bytes(), nil
}

// Archive moves the plan file identified by filename from plans/ to
// plans/archive/. Returns the new absolute path of the archived file.
func Archive(projectRoot, filename string) (string, error) {
	src := filepath.Join(PlansDir(projectRoot), filename)
	dst := filepath.Join(ArchiveDir(projectRoot), filename)

	if err := os.MkdirAll(ArchiveDir(projectRoot), 0o755); err != nil {
		return "", fmt.Errorf("create archive dir: %w", err)
	}

	if err := os.Rename(src, dst); err != nil {
		return "", fmt.Errorf("archive %s: %w", filename, err)
	}
	return dst, nil
}

// ExtractSections returns only the markdown sections whose headings match
// the given list (case-insensitive). Used by `logos refer --summary`.
func ExtractSections(body string, sectionNames []string) string {
	if len(sectionNames) == 0 {
		return body
	}

	wanted := make(map[string]bool, len(sectionNames))
	for _, name := range sectionNames {
		wanted[strings.ToLower(strings.TrimSpace(name))] = true
	}

	lines := strings.Split(body, "\n")
	var result strings.Builder
	inWanted := false
	currentLevel := 0

	for _, line := range lines {
		if heading, level, ok := parseHeading(line); ok {
			if inWanted && level <= currentLevel {
				inWanted = false
			}
			if wanted[strings.ToLower(strings.TrimSpace(heading))] {
				inWanted = true
				currentLevel = level
			}
		}
		if inWanted {
			result.WriteString(line)
			result.WriteByte('\n')
		}
	}

	return strings.TrimRight(result.String(), "\n")
}

// GenerateID returns a new random 6-character lowercase hex string.
func GenerateID() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// --- helpers -----------------------------------------------------------------

// slugify converts a string to a URL-safe kebab-case slug.
// Consecutive hyphens are collapsed to one; leading/trailing hyphens removed.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevHyphen := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
			prevHyphen = false
		case r == '-', r == ' ':
			if !prevHyphen {
				b.WriteRune('-')
				prevHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// splitFrontmatter separates YAML frontmatter from the markdown body.
// The file must begin with "---\n"; the closing "---" ends the frontmatter.
func splitFrontmatter(data []byte) (frontmatter, body []byte, err error) {
	text := string(data)
	if !strings.HasPrefix(text, frontmatterSep) {
		return nil, nil, errors.New("missing frontmatter: file must begin with '---'")
	}

	rest := text[len(frontmatterSep):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	} else if len(rest) > 0 && rest[0] == '\r' && len(rest) > 1 && rest[1] == '\n' {
		rest = rest[2:]
	}

	idx := strings.Index(rest, "\n"+frontmatterSep)
	if idx == -1 {
		return nil, nil, errors.New("missing closing '---' for frontmatter")
	}

	fm := rest[:idx]
	remainder := rest[idx+1+len(frontmatterSep):]
	if len(remainder) > 0 && remainder[0] == '\n' {
		remainder = remainder[1:]
	}

	return []byte(fm), []byte(remainder), nil
}

// extractExcerpt returns the first excerptMaxRunes runes of the named section's
// content. Falls back to the beginning of the body if the section is not found.
func extractExcerpt(body []byte, excerptSection string) string {
	if excerptSection == "" {
		excerptSection = "Background"
	}
	text := string(body)
	lines := strings.Split(text, "\n")

	inSection := false
	currentLevel := 0
	var content strings.Builder

	for _, line := range lines {
		if heading, level, ok := parseHeading(line); ok {
			if inSection && level <= currentLevel {
				break
			}
			if strings.EqualFold(strings.TrimSpace(heading), excerptSection) {
				inSection = true
				currentLevel = level
				continue
			}
		}
		if inSection {
			content.WriteString(line)
			content.WriteByte('\n')
		}
	}

	excerpt := strings.TrimSpace(content.String())
	if excerpt == "" {
		excerpt = strings.TrimSpace(text)
	}
	return truncateRunes(excerpt, excerptMaxRunes)
}

// parseHeading returns the heading text, its level (1–6), and true if the
// line is a markdown ATX heading (e.g. "## Background").
func parseHeading(line string) (text string, level int, ok bool) {
	trimmed := strings.TrimRight(line, " \t")
	i := 0
	for i < len(trimmed) && trimmed[i] == '#' {
		i++
	}
	if i == 0 || i > 6 {
		return "", 0, false
	}
	if i >= len(trimmed) || trimmed[i] != ' ' {
		return "", 0, false
	}
	return strings.TrimSpace(trimmed[i+1:]), i, true
}

// truncateRunes truncates s to at most n runes, appending "…" if truncated.
func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n]) + "…"
}
