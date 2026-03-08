// Package task provides types and functions for reading, writing, and
// parsing Logosyncx task files — Markdown documents with YAML frontmatter
// stored under .logosyncx/tasks/<plan-slug>/.
package task

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/senna-lang/logosyncx/internal/markdown"
	"gopkg.in/yaml.v3"
)

const frontmatterSep = "---"

// Status represents the lifecycle state of a task.
type Status string

// Priority represents the urgency level of a task.
type Priority string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

const (
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

// ValidStatuses lists every recognised Status value.
var ValidStatuses = []Status{StatusOpen, StatusInProgress, StatusDone}

// ValidPriorities lists every recognised Priority value.
var ValidPriorities = []Priority{PriorityHigh, PriorityMedium, PriorityLow}

// Task represents a single task file stored under .logosyncx/tasks/<plan-slug>/.
type Task struct {
	// Frontmatter fields (serialised to/from YAML).
	ID          string     `yaml:"id"`
	Date        time.Time  `yaml:"date"`
	Title       string     `yaml:"title"`
	Seq         int        `yaml:"seq"`
	Status      Status     `yaml:"status"`
	Priority    Priority   `yaml:"priority"`
	Plan        string     `yaml:"plan"`
	DependsOn   []int      `yaml:"depends_on,omitempty"`
	Tags        []string   `yaml:"tags"`
	Assignee    string     `yaml:"assignee"`
	CompletedAt *time.Time `yaml:"completed_at,omitempty"`

	// Derived fields — not written to frontmatter.
	DirPath string `yaml:"-"` // absolute path to the task's directory (set by store)
	Blocked bool   `yaml:"-"` // true when at least one depends_on seq is not yet done
	Excerpt string `yaml:"-"` // first excerptMaxRunes runes of the excerpt section
	Body    string `yaml:"-"` // full markdown body (everything after frontmatter)
}

// TaskJSON is the shape used for --json output and the task-index.jsonl.
// It includes all frontmatter fields plus the derived DirPath, Blocked, and Excerpt.
type TaskJSON struct {
	ID          string     `json:"id"`
	DirPath     string     `json:"dir_path"`
	Date        time.Time  `json:"date"`
	Title       string     `json:"title"`
	Seq         int        `json:"seq"`
	Status      Status     `json:"status"`
	Priority    Priority   `json:"priority"`
	Plan        string     `json:"plan"`
	DependsOn   []int      `json:"depends_on"`
	Tags        []string   `json:"tags"`
	Assignee    string     `json:"assignee"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Blocked     bool       `json:"blocked"`
	Excerpt     string     `json:"excerpt"`
}

// ToJSON converts a Task to its JSON-output representation.
// Nil slice fields are normalised to empty slices.
func (t *Task) ToJSON() TaskJSON {
	return TaskJSON{
		ID:          t.ID,
		DirPath:     t.DirPath,
		Date:        t.Date,
		Title:       t.Title,
		Seq:         t.Seq,
		Status:      t.Status,
		Priority:    t.Priority,
		Plan:        t.Plan,
		DependsOn:   normalizeInts(t.DependsOn),
		Tags:        normalizeStrings(t.Tags),
		Assignee:    t.Assignee,
		CompletedAt: t.CompletedAt,
		Blocked:     false, // store sets this during loadAll
		Excerpt:     t.Excerpt,
	}
}

// FromTask converts a *Task to TaskJSON (package-level function form of ToJSON).
// Nil slices are normalised to empty slices. Blocked is always false here;
// the store sets it during loadAll after evaluating depends_on.
func FromTask(t *Task) TaskJSON {
	return t.ToJSON()
}

// IsValidStatus reports whether s is a recognised Status constant.
func IsValidStatus(s Status) bool {
	return slices.Contains(ValidStatuses, s)
}

// IsValidPriority reports whether p is a recognised Priority constant.
func IsValidPriority(p Priority) bool {
	return slices.Contains(ValidPriorities, p)
}

// TaskDirName returns the directory name for a task given its seq number and
// title: e.g. seq=1, title="Add JWT middleware" → "001-add-jwt-middleware".
func TaskDirName(seq int, title string) string {
	return fmt.Sprintf("%03d-%s", seq, markdown.Slugify(title))
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
	fm, body, err := markdown.SplitFrontmatter(data)
	if err != nil {
		return Task{}, fmt.Errorf("parse %s: %w", filename, err)
	}

	var t Task
	if err := yaml.Unmarshal(fm, &t); err != nil {
		return Task{}, fmt.Errorf("parse frontmatter in %s: %w", filename, err)
	}

	t.Body = string(body)
	section := opts.ExcerptSection
	if section == "" {
		section = "What"
	}
	t.Excerpt = markdown.ExtractExcerpt(body, section)

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
// NOTE: The flat TASK.md layout is planned for Task 005 (store rewrite).
func FileName(t Task) string {
	date := t.Date.Format("2006-01-02")
	slug := markdown.Slugify(t.Title)
	if slug == "" {
		slug = "untitled"
	}
	return fmt.Sprintf("%s_%s.md", date, slug)
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
		if heading, level, ok := markdown.ParseHeading(line); ok {
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

// normalizeInts returns a non-nil empty slice when s is nil.
func normalizeInts(s []int) []int {
	if s == nil {
		return []int{}
	}
	return s
}

// normalizeStrings returns a non-nil empty slice when s is nil.
func normalizeStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
