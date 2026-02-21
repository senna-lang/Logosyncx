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
