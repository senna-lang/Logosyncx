// Package task provides filtering logic for task lists.
package task

import (
	"slices"
	"strings"
)

// Filter holds the criteria used to narrow down a list of tasks.
// Zero values mean "no constraint" — only non-zero fields are applied.
type Filter struct {
	// Plan is a substring matched against each task's Plan field.
	Plan string
	// Status is an exact match on task status (empty = any status).
	Status Status
	// Priority is an exact match on task priority (empty = any priority).
	Priority Priority
	// Tags requires the task to have at least one tag in this list.
	Tags []string
	// Keyword is a case-insensitive substring matched against title, tags,
	// and excerpt — used by logos task search.
	Keyword string
	// Blocked, when true, restricts results to tasks whose DependsOn seq
	// numbers contain at least one task that is not yet done.
	Blocked bool
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

// matchesJSONFilter reports whether e satisfies all active constraints in f.
func matchesJSONFilter(e TaskJSON, f Filter) bool {
	if f.Plan != "" {
		if !strings.Contains(strings.ToLower(e.Plan), strings.ToLower(f.Plan)) {
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
		tagMatch := slices.ContainsFunc(e.Tags, func(tag string) bool {
			return strings.Contains(strings.ToLower(tag), lower)
		})
		if !titleMatch && !excerptMatch && !tagMatch {
			return false
		}
	}
	if f.Blocked {
		if !e.Blocked {
			return false
		}
	}
	return true
}

// matchesFilter reports whether t satisfies all active constraints in f.
func matchesFilter(t *Task, f Filter) bool {
	if f.Plan != "" {
		if !strings.Contains(strings.ToLower(t.Plan), strings.ToLower(f.Plan)) {
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

	if f.Blocked {
		if !t.Blocked {
			return false
		}
	}

	return true
}

// hasAnyTag reports whether taskTags contains at least one tag from wantTags
// (case-insensitive comparison).
func hasAnyTag(taskTags, wantTags []string) bool {
	return slices.ContainsFunc(wantTags, func(want string) bool {
		lower := strings.ToLower(want)
		return slices.ContainsFunc(taskTags, func(have string) bool {
			return strings.ToLower(have) == lower
		})
	})
}

// matchesKeyword reports whether t's title, any tag, or excerpt contains
// lower (already lower-cased) as a substring.
func matchesKeyword(t *Task, lower string) bool {
	return strings.Contains(strings.ToLower(t.Title), lower) ||
		slices.ContainsFunc(t.Tags, func(tag string) bool {
			return strings.Contains(strings.ToLower(tag), lower)
		}) ||
		strings.Contains(strings.ToLower(t.Excerpt), lower)
}
