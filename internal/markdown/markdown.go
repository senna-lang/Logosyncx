// Package markdown provides shared helpers for parsing Markdown documents
// with YAML frontmatter, used by both the plan and task packages.
package markdown

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	// ExcerptMaxRunes is the maximum number of runes in an extracted excerpt.
	ExcerptMaxRunes = 300
	frontmatterSep  = "---"
)

// Slugify converts s into a lowercase, hyphen-separated identifier suitable
// for use in file and directory names.
func Slugify(s string) string {
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

// SplitFrontmatter separates YAML frontmatter from the Markdown body.
// The file must begin with "---"; the closing "---" ends the frontmatter.
func SplitFrontmatter(data []byte) (frontmatter, body []byte, err error) {
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

// ExtractExcerpt returns the first ExcerptMaxRunes runes of the named
// section's content. Falls back to the beginning of the body if the section
// is not found or excerptSection is empty.
func ExtractExcerpt(body []byte, excerptSection string) string {
	text := string(body)

	if excerptSection != "" {
		lines := strings.Split(text, "\n")
		inSection := false
		currentLevel := 0
		var content strings.Builder

		for _, line := range lines {
			if heading, level, ok := ParseHeading(line); ok {
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
		if excerpt != "" {
			return TruncateRunes(excerpt, ExcerptMaxRunes)
		}
	}

	return TruncateRunes(strings.TrimSpace(text), ExcerptMaxRunes)
}

// ParseHeading returns the heading text, its level (1–6), and true if the
// line is a Markdown ATX heading (e.g. "## Background").
func ParseHeading(line string) (text string, level int, ok bool) {
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

// TruncateRunes truncates s to at most n runes, appending "…" if truncated.
func TruncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n]) + "…"
}
