package session

import (
	"bytes"
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
	sessionsDirName = "sessions"
	excerptMaxRunes = 300
	frontmatterSep  = "---"
)

// Session represents a single saved conversation context file.
type Session struct {
	// Frontmatter fields
	ID      string    `yaml:"id"`
	Date    time.Time `yaml:"date"`
	Topic   string    `yaml:"topic"`
	Tags    []string  `yaml:"tags"`
	Agent   string    `yaml:"agent"`
	Related []string  `yaml:"related"`

	// Derived fields (not written to frontmatter)
	Filename string `yaml:"-"`
	Excerpt  string `yaml:"-"`
	Body     string `yaml:"-"` // full markdown body (everything after frontmatter)
}

// SessionsDir returns the path to the sessions directory under a project root.
func SessionsDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".logosyncx", sessionsDirName)
}

// FilePath returns the canonical file path for a session inside the given project root.
// The filename is derived from the session's Date and Topic: <date>_<topic>.md
func FilePath(projectRoot string, s Session) string {
	return filepath.Join(SessionsDir(projectRoot), FileName(s))
}

// FileName returns the canonical filename for a session: <date>_<topic>.md
func FileName(s Session) string {
	date := s.Date.Format("2006-01-02")
	topic := sanitizeTopic(s.Topic)
	return fmt.Sprintf("%s_%s.md", date, topic)
}

// sanitizeTopic converts a topic string into a safe filename segment.
// Spaces are replaced with hyphens; characters that are not alphanumeric,
// hyphens, or underscores are removed.
func sanitizeTopic(topic string) string {
	topic = strings.ToLower(strings.TrimSpace(topic))
	var b strings.Builder
	for _, r := range topic {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}

// Parse reads a session markdown file from data.
// The file must start with a YAML frontmatter block delimited by "---".
// The filename is stored as-is for display purposes.
func Parse(filename string, data []byte) (Session, error) {
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return Session{}, fmt.Errorf("parse %s: %w", filename, err)
	}

	var s Session
	if err := yaml.Unmarshal(fm, &s); err != nil {
		return Session{}, fmt.Errorf("parse frontmatter in %s: %w", filename, err)
	}

	s.Filename = filename
	s.Body = string(body)
	s.Excerpt = extractExcerpt(body)

	return s, nil
}

// LoadFile reads and parses a session file at the given path.
func LoadFile(path string) (Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Session{}, err
	}
	return Parse(filepath.Base(path), data)
}

// LoadAll reads every .md file from the sessions directory under projectRoot
// and returns the parsed sessions. Files that fail to parse are skipped and
// their errors collected into a combined error (non-fatal).
func LoadAll(projectRoot string) ([]Session, error) {
	dir := SessionsDir(projectRoot)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Session
	var errs []string

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		s, err := LoadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", entry.Name(), err))
			continue
		}
		sessions = append(sessions, s)
	}

	if len(errs) > 0 {
		return sessions, fmt.Errorf("some session files could not be parsed:\n  %s",
			strings.Join(errs, "\n  "))
	}
	return sessions, nil
}

// Write serialises s to a markdown file under projectRoot/sessions/.
// The sessions directory is created if it does not exist.
// The returned string is the full path of the written file.
func Write(projectRoot string, s Session) (string, error) {
	dir := SessionsDir(projectRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	data, err := Marshal(s)
	if err != nil {
		return "", err
	}

	path := FilePath(projectRoot, s)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// Marshal serialises a Session back to its markdown representation
// (YAML frontmatter + body).
func Marshal(s Session) ([]byte, error) {
	fm, err := yaml.Marshal(s)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	buf.WriteString(frontmatterSep + "\n")
	buf.Write(fm)
	buf.WriteString(frontmatterSep + "\n")
	if s.Body != "" {
		if !strings.HasPrefix(s.Body, "\n") {
			buf.WriteByte('\n')
		}
		buf.WriteString(s.Body)
	}
	return buf.Bytes(), nil
}

// ExtractSections returns only the markdown sections whose headings match
// the given list (case-insensitive). This is used by `logos refer --summary`.
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
			// If we're inside a wanted section and we encounter a heading at
			// the same or higher level, the section has ended.
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

	// Strip the opening "---" line.
	rest := text[len(frontmatterSep):]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	} else if len(rest) > 0 && rest[0] == '\r' && len(rest) > 1 && rest[1] == '\n' {
		rest = rest[2:]
	}

	// Find the closing "---".
	idx := strings.Index(rest, "\n"+frontmatterSep)
	if idx == -1 {
		return nil, nil, errors.New("missing closing '---' for frontmatter")
	}

	fm := rest[:idx]
	remainder := rest[idx+1+len(frontmatterSep):]
	// Skip the newline after the closing "---".
	if len(remainder) > 0 && remainder[0] == '\n' {
		remainder = remainder[1:]
	}

	return []byte(fm), []byte(remainder), nil
}

// extractExcerpt returns the first excerptMaxRunes runes of the ## Summary
// section content (stripped of the heading line itself and blank lines).
// If no Summary section is found, it falls back to the beginning of the body.
func extractExcerpt(body []byte) string {
	text := string(body)
	lines := strings.Split(text, "\n")

	inSummary := false
	var content strings.Builder

	for _, line := range lines {
		if heading, level, ok := parseHeading(line); ok {
			if inSummary {
				// A new heading at the same or higher level ends the section.
				if level <= 2 {
					break
				}
			}
			if level == 2 && strings.EqualFold(strings.TrimSpace(heading), "summary") {
				inSummary = true
				continue
			}
		}
		if inSummary {
			content.WriteString(line)
			content.WriteByte('\n')
		}
	}

	excerpt := strings.TrimSpace(content.String())

	// Fallback: use the beginning of the body.
	if excerpt == "" {
		excerpt = strings.TrimSpace(text)
	}

	return truncateRunes(excerpt, excerptMaxRunes)
}

// parseHeading returns the heading text, its level (1–6), and true if the
// line is a markdown ATX heading (e.g. "## Summary").
func parseHeading(line string) (text string, level int, ok bool) {
	trimmed := strings.TrimRight(line, " \t")
	i := 0
	for i < len(trimmed) && trimmed[i] == '#' {
		i++
	}
	if i == 0 || i > 6 {
		return "", 0, false
	}
	// Must be followed by a space.
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
