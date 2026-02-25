// Package cmd — sections.go provides helpers for validating and building
// markdown section bodies from --section flags and project config.
package cmd

import (
	"fmt"
	"strings"

	"github.com/senna-lang/logosyncx/pkg/config"
)

// parseSectionFlag parses a single "--section" flag value of the form
// "Name=content" into its name and content parts.
//
// The first '=' is the delimiter; everything after it (including any
// additional '=' characters) is treated as content.
func parseSectionFlag(s string) (name, content string, err error) {
	idx := strings.IndexByte(s, '=')
	if idx < 0 {
		return "", "", fmt.Errorf(
			"invalid --section value %q: expected format 'Name=content' (e.g. --section \"Summary=my text\")",
			s,
		)
	}
	name = strings.TrimSpace(s[:idx])
	content = s[idx+1:]
	if name == "" {
		return "", "", fmt.Errorf(
			"invalid --section value %q: section name must not be empty",
			s,
		)
	}
	return name, content, nil
}

// allowedSectionSet builds a case-insensitive lookup map from a slice of
// SectionConfig values.
func allowedSectionSet(sections []config.SectionConfig) map[string]bool {
	m := make(map[string]bool, len(sections))
	for _, s := range sections {
		m[strings.ToLower(strings.TrimSpace(s.Name))] = true
	}
	return m
}

// allowedSectionNames returns the display names of all configured sections
// as a comma-separated string, for use in error messages.
func allowedSectionNames(sections []config.SectionConfig) string {
	names := make([]string, 0, len(sections))
	for _, s := range sections {
		names = append(names, s.Name)
	}
	return strings.Join(names, ", ")
}

// buildBodyFromSections builds a markdown body string from a list of
// "--section Name=content" flag values.
//
// Sections are emitted in the order they appear in configSections; any
// flagSections entry whose name does not appear in configSections causes an
// error to be returned immediately.
//
// Each section is rendered as:
//
//	<hashes> <Name>\n\n<content>\n\n
//
// where the number of hashes is taken from SectionConfig.Level (defaulting
// to 2 if Level is zero).
func buildBodyFromSections(flagSections []string, configSections []config.SectionConfig) (string, error) {
	if len(flagSections) == 0 {
		return "", nil
	}

	// Build a canonical-name map keyed by lowercase name.
	type sectionEntry struct {
		cfg     config.SectionConfig
		content string
		present bool
	}
	byLower := make(map[string]*sectionEntry, len(configSections))
	// Keep insertion order for output.
	ordered := make([]*sectionEntry, 0, len(configSections))
	for i := range configSections {
		e := &sectionEntry{cfg: configSections[i]}
		key := strings.ToLower(strings.TrimSpace(configSections[i].Name))
		byLower[key] = e
		ordered = append(ordered, e)
	}

	// Parse and validate each flag value.
	for _, fs := range flagSections {
		name, content, err := parseSectionFlag(fs)
		if err != nil {
			return "", err
		}
		lower := strings.ToLower(name)
		entry, ok := byLower[lower]
		if !ok {
			return "", fmt.Errorf(
				"section %q is not defined in config\nallowed sections: %s\n(edit .logosyncx/config.json to add new sections)",
				name,
				allowedSectionNames(configSections),
			)
		}
		if entry.present {
			return "", fmt.Errorf("section %q was specified more than once in --section flags", name)
		}
		entry.content = content
		entry.present = true
	}

	// Render sections in config order, skipping any that were not supplied.
	var buf strings.Builder
	for _, e := range ordered {
		if !e.present {
			continue
		}
		level := e.cfg.Level
		if level < 1 || level > 6 {
			level = 2
		}
		hashes := strings.Repeat("#", level)
		fmt.Fprintf(&buf, "%s %s\n\n", hashes, e.cfg.Name)
		content := strings.TrimRight(e.content, "\n")
		if content != "" {
			buf.WriteString(content)
			buf.WriteString("\n")
		}
		buf.WriteString("\n")
	}

	return buf.String(), nil
}

// parseBodyHeading returns the heading text, its ATX level (1–6), and true
// when line is a valid ATX markdown heading ("# …" through "###### …").
// Leading and trailing whitespace on the heading text is stripped.
func parseBodyHeading(line string) (text string, level int, ok bool) {
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i == 0 || i > 6 {
		return "", 0, false
	}
	// Must be followed by exactly one space.
	if i >= len(line) || line[i] != ' ' {
		return "", 0, false
	}
	return strings.TrimSpace(line[i+1:]), i, true
}
