// Package knowledge provides types and functions for creating Logosyncx
// knowledge files — distilled records stored under .logosyncx/knowledge/.
//
// Filename format: YYYYMMDD-<slug>.md (e.g. 20260610-auth-refactor.md).
// Write creates a frontmatter + source-material block + empty section
// headings scaffold; the agent fills in the sections.
package knowledge

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	knowledgeDirName = "knowledge"
	frontmatterSep   = "---"
)

// Knowledge represents a single knowledge file stored under .logosyncx/knowledge/.
type Knowledge struct {
	ID    string     `yaml:"id"`
	Date  *time.Time `yaml:"date,omitempty"`
	Topic string     `yaml:"topic"`
	Plan  string     `yaml:"plan"` // source plan filename
	Tasks []string   `yaml:"tasks,omitempty"`
	Tags  []string   `yaml:"tags"`
	Body  string     `yaml:"-"`
}

// KnowledgeDir returns the path to the knowledge directory under a project root.
func KnowledgeDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".logosyncx", knowledgeDirName)
}

// FileName returns the canonical filename for a knowledge entry: YYYYMMDD-<slug>.md.
// If Date is nil, the current time is used.
func FileName(k Knowledge) string {
	t := time.Now()
	if k.Date != nil {
		t = *k.Date
	}
	return fmt.Sprintf("%s-%s.md", t.Format("20060102"), slugify(k.Topic))
}

// Write creates a knowledge scaffold file under projectRoot/knowledge/.
// The file contains:
//   - YAML frontmatter (ID auto-generated if empty, Date defaulted to now)
//   - An HTML comment block with sourceBlock (plan body + walkthroughs)
//   - Empty section headings extracted from templateSections
//
// Returns the path of the written file relative to projectRoot.
func Write(projectRoot string, k Knowledge, sourceBlock string, templateSections string) (string, error) {
	dir := KnowledgeDir(projectRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create knowledge dir: %w", err)
	}

	if k.ID == "" {
		id, err := generateID()
		if err != nil {
			return "", fmt.Errorf("generate id: %w", err)
		}
		k.ID = "k-" + id
	}

	if k.Date == nil {
		now := time.Now().UTC()
		k.Date = &now
	}

	fm, err := yaml.Marshal(k)
	if err != nil {
		return "", fmt.Errorf("marshal frontmatter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(frontmatterSep + "\n")
	buf.Write(fm)
	buf.WriteString(frontmatterSep + "\n")
	buf.WriteByte('\n')

	// SOURCE MATERIAL HTML comment block.
	buf.WriteString("<!-- SOURCE MATERIAL — read this, fill in the sections below, then remove this block. -->\n")
	buf.WriteString("<!--\n")
	buf.WriteString(sourceBlock)
	if !strings.HasSuffix(sourceBlock, "\n") {
		buf.WriteByte('\n')
	}
	buf.WriteString("-->\n")

	// Empty section headings from the template.
	headings := extractHeadings(templateSections)
	for _, h := range headings {
		buf.WriteByte('\n')
		buf.WriteString(h)
		buf.WriteByte('\n')
	}

	filename := FileName(k)
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return "", fmt.Errorf("write knowledge file: %w", err)
	}

	rel, err := filepath.Rel(projectRoot, path)
	if err != nil {
		return path, nil
	}
	return rel, nil
}

// generateID returns a new random 6-character lowercase hex string.
func generateID() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// extractHeadings returns all ATX heading lines (## Foo) from s.
func extractHeadings(s string) []string {
	var headings []string
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimRight(line, " \t")
		i := 0
		for i < len(trimmed) && trimmed[i] == '#' {
			i++
		}
		if i == 0 || i > 6 {
			continue
		}
		if i < len(trimmed) && trimmed[i] == ' ' {
			headings = append(headings, trimmed)
		}
	}
	return headings
}

// slugify converts a string to a URL-safe kebab-case slug.
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
