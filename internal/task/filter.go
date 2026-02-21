// Package task provides filtering logic for task lists.
package task

import "strings"

// Filter holds the criteria used to narrow down a list of tasks.
// Zero values mean "no constraint" — only non-zero fields are applied.
type Filter struct {
	// Session is a substring matched against each task's Session field.
	Session string
	// Status is an exact match on task status (empty = any status).
	Status Status
	// Priority is an exact match on task priority (empty = any priority).
	Priority Priority
	// Tags requires the task to have at least one tag in this list.
	Tags []string
	// Keyword is a case-insensitive substring matched against title, tags,
	// and excerpt — used by logos task search.
	Keyword string
}

// Apply returns the subset of tasks that satisfy every non-zero field of f.
// The original slice is not modified; a new slice is returned.
func Apply(tasks []*Task, f Filter) []*Task {
	var out []*Task
	for _, t := range tasks {
		if !matchesFilter(t, f) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// ApplyToJSON returns the subset of TaskJSON entries that satisfy every
// non-zero field of f.  The original slice is not modified; a new slice is
// returned.  This is the index-based counterpart of Apply.
func ApplyToJSON(entries []TaskJSON, f Filter) []TaskJSON {
	var out []TaskJSON
	for _, e := range entries {
		if !matchesJSONFilter(e, f) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// SortJSONByDateDesc sorts TaskJSON entries newest-first in-place.
func SortJSONByDateDesc(entries []TaskJSON) {
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].Date.After(entries[j-1].Date); j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
}

// matchesJSONFilter reports whether e satisfies all active constraints in f.
func matchesJSONFilter(e TaskJSON, f Filter) bool {
	if f.Session != "" {
		if !strings.Contains(strings.ToLower(e.Session), strings.ToLower(f.Session)) {
			return false
		}
	}
	if f.Status != "" {
		if e.Status != f.Status {
			return false
		}
	}
	if f.Priority != "" {
		if e.Priority != f.Priority {
			return false
		}
	}
	if len(f.Tags) > 0 {
		if !hasAnyTag(e.Tags, f.Tags) {
			return false
		}
	}
	if f.Keyword != "" {
		lower := strings.ToLower(f.Keyword)
		titleMatch := strings.Contains(strings.ToLower(e.Title), lower)
		excerptMatch := strings.Contains(strings.ToLower(e.Excerpt), lower)
		tagMatch := false
		for _, tag := range e.Tags {
			if strings.Contains(strings.ToLower(tag), lower) {
				tagMatch = true
				break
			}
		}
		if !titleMatch && !excerptMatch && !tagMatch {
			return false
		}
	}
	return true
}

// matchesFilter reports whether t satisfies all active constraints in f.
func matchesFilter(t *Task, f Filter) bool {
	if f.Session != "" {
		if !strings.Contains(strings.ToLower(t.Session), strings.ToLower(f.Session)) {
			return false
		}
	}

	if f.Status != "" {
		if t.Status != f.Status {
			return false
		}
	}

	if f.Priority != "" {
		if t.Priority != f.Priority {
			return false
		}
	}

	if len(f.Tags) > 0 {
		if !hasAnyTag(t.Tags, f.Tags) {
			return false
		}
	}

	if f.Keyword != "" {
		if !matchesKeyword(t, strings.ToLower(f.Keyword)) {
			return false
		}
	}

	return true
}

// hasAnyTag reports whether taskTags contains at least one tag from wantTags
// (case-insensitive comparison).
func hasAnyTag(taskTags, wantTags []string) bool {
	for _, want := range wantTags {
		lower := strings.ToLower(want)
		for _, have := range taskTags {
			if strings.ToLower(have) == lower {
				return true
			}
		}
	}
	return false
}

// matchesKeyword reports whether t's title, any tag, or excerpt contains
// lower (already lower-cased) as a substring.
func matchesKeyword(t *Task, lower string) bool {
	if strings.Contains(strings.ToLower(t.Title), lower) {
		return true
	}
	for _, tag := range t.Tags {
		if strings.Contains(strings.ToLower(tag), lower) {
			return true
		}
	}
	if strings.Contains(strings.ToLower(t.Excerpt), lower) {
		return true
	}
	return false
}
