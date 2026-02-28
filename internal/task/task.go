// Package task provides types and functions for reading, writing, and
// parsing Logosyncx task files — Markdown documents with YAML frontmatter
// stored under .logosyncx/tasks/.
package task

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const (
	tasksDirName    = "tasks"
	excerptMaxRunes = 300
	frontmatterSep  = "---"
)

// Status represents the lifecycle state of a task.
type Status string

// Priority represents the urgency level of a task.
type Priority string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusCancelled  Status = "cancelled"
)

const (
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

// ValidStatuses lists every recognised Status value.
var ValidStatuses = []Status{StatusOpen, StatusInProgress, StatusDone, StatusCancelled}

// ValidPriorities lists every recognised Priority value.
var ValidPriorities = []Priority{PriorityHigh, PriorityMedium, PriorityLow}

// Task represents a single task file stored in .logosyncx/tasks/.
type Task struct {
	// Frontmatter fields (serialised to/from YAML).
	ID          string     `yaml:"id"`
	Date        time.Time  `yaml:"date"`
	Title       string     `yaml:"title"`
	Status      Status     `yaml:"status"`
	Priority    Priority   `yaml:"priority"`
	Session     string     `yaml:"session"`            // primary linked session filename (kept for backward compat)
	Sessions    []string   `yaml:"sessions,omitempty"` // task→session links (list of session filenames)
	Related     []string   `yaml:"related,omitempty"`  // task→task links (list of task filenames)
	Tags        []string   `yaml:"tags"`
	Assignee    string     `yaml:"assignee"`
	CompletedAt *time.Time `yaml:"completed_at,omitempty"` // set when status transitions to done or cancelled

	// Derived fields — not written to frontmatter.
	Filename string `yaml:"-"`
	Excerpt  string `yaml:"-"` // first excerptMaxRunes runes of ## What section
	Body     string `yaml:"-"` // full markdown body (everything after frontmatter)
}

// TaskJSON is the shape used for --json output.  It includes all frontmatter
// fields plus the derived Filename and Excerpt.
type TaskJSON struct {
	ID          string     `json:"id"`
	Filename    string     `json:"filename"`
	Date        time.Time  `json:"date"`
	Title       string     `json:"title"`
	Status      Status     `json:"status"`
	Priority    Priority   `json:"priority"`
	Session     string     `json:"session"`
	Sessions    []string   `json:"sessions"`
	Related     []string   `json:"related"`
	Tags        []string   `json:"tags"`
	Assignee    string     `json:"assignee"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Excerpt     string     `json:"excerpt"`
}

// ToJSON converts a Task to its JSON-output representation.
// Nil slice fields are normalised to empty slices.
func (t *Task) ToJSON() TaskJSON {
	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}
	sessions := t.Sessions
	if sessions == nil {
		sessions = []string{}
	}
	related := t.Related
	if related == nil {
		related = []string{}
	}
	return TaskJSON{
		ID:          t.ID,
		Filename:    t.Filename,
		Date:        t.Date,
		Title:       t.Title,
		Status:      t.Status,
		Priority:    t.Priority,
		Session:     t.Session,
		Sessions:    sessions,
		Related:     related,
		Tags:        tags,
		Assignee:    t.Assignee,
		CompletedAt: t.CompletedAt,
		Excerpt:     t.Excerpt,
	}
}

// IsValidStatus reports whether s is a recognised Status constant.
func IsValidStatus(s Status) bool {
	for _, v := range ValidStatuses {
		if s == v {
			return true
		}
	}
	return false
}

// IsValidPriority reports whether p is a recognised Priority constant.
func IsValidPriority(p Priority) bool {
	for _, v := range ValidPriorities {
		if p == v {
			return true
		}
	}
	return false
}

// ParseOptions controls optional behaviour of Parse.
type ParseOptions struct {
	// ExcerptSection is the heading name used to extract the excerpt.
	// Defaults to "What" when empty. Matched case-insensitively at any
	// heading level (h1–h6).
	ExcerptSection string
}

// Parse reads a task markdown file from data.
// The file must start with a YAML frontmatter block delimited by "---".
// filename is stored on the returned Task for display purposes.
func Parse(filename string, data []byte) (Task, error) {
	return ParseWithOptions(filename, data, ParseOptions{})
}

// ParseWithOptions is like Parse but accepts options to customise excerpt
// extraction. Use this when the project's tasks.excerpt_section differs from
// the default "What".
func ParseWithOptions(filename string, data []byte, opts ParseOptions) (Task, error) {
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return Task{}, fmt.Errorf("parse %s: %w", filename, err)
	}

	var t Task
	if err := yaml.Unmarshal(fm, &t); err != nil {
		return Task{}, fmt.Errorf("parse frontmatter in %s: %w", filename, err)
	}

	t.Filename = filename
	t.Body = string(body)
	t.Excerpt = extractExcerpt(body, opts.ExcerptSection)

	return t, nil
}

// Marshal serialises a Task back to its markdown representation
// (YAML frontmatter + body).
func Marshal(t Task) ([]byte, error) {
	fm, err := yaml.Marshal(t)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString(frontmatterSep + "\n")
	buf.Write(fm)
	buf.WriteString(frontmatterSep + "\n")
	if t.Body != "" {
		if !strings.HasPrefix(t.Body, "\n") {
			buf.WriteByte('\n')
		}
		buf.WriteString(t.Body)
	}
	return buf.Bytes(), nil
}

// FileName returns the canonical filename for a task: <date>_<slug>.md
// The slug is the task title converted to lower-case kebab-case.
func FileName(t Task) string {
	date := t.Date.Format("2006-01-02")
	slug := slugify(t.Title)
	if slug == "" {
		slug = "untitled"
	}
	return fmt.Sprintf("%s_%s.md", date, slug)
}

// slugify converts a string to a lower-case kebab-case filename segment.
// Spaces become hyphens; characters that are not alphanumeric, hyphens, or
// underscores are removed.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}

// ExtractSections returns only the markdown sections whose headings match
// the given list (case-insensitive). Used by logos task refer --summary.
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

// --- helpers -----------------------------------------------------------------

// splitFrontmatter separates YAML frontmatter from the markdown body.
// The file must begin with "---\n"; the closing "---" ends the frontmatter block.
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

// extractExcerpt returns the first excerptMaxRunes runes of the named excerpt
// section's content (stripped of the heading line itself and blank lines).
// The section is matched by name only — any heading level (h1–h6) is accepted.
// The section ends when a heading at the same or higher level is encountered.
// If excerptSection is empty it defaults to "What".
// Falls back to the beginning of the body if the named section is not found.
func extractExcerpt(body []byte, excerptSection string) string {
	if excerptSection == "" {
		excerptSection = "What"
	}
	text := string(body)
	lines := strings.Split(text, "\n")

	inSection := false
	currentLevel := 0
	var content strings.Builder

	for _, line := range lines {
		if heading, level, ok := parseHeading(line); ok {
			if inSection {
				if level <= currentLevel {
					break
				}
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
// line is a markdown ATX heading (e.g. "## What").
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
