// Package task provides Store — the read/write layer for .logosyncx/tasks/.
// Layout: .logosyncx/tasks/<plan-slug>/NNN-<title>/TASK.md
package task

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/senna-lang/logosyncx/internal/gitutil"
	"github.com/senna-lang/logosyncx/pkg/config"
)

// idPrefix is prepended to every auto-generated task ID.
const idPrefix = "t-"

// taskFileName is the canonical filename for every task file.
const taskFileName = "TASK.md"

// walkthroughFileName is created when a task is marked done.
const walkthroughFileName = "WALKTHROUGH.md"

// ErrNotFound is returned by Get when no match is found.
var ErrNotFound = errors.New("not found")

// ErrAmbiguous is returned by Get when more than one match is found.
var ErrAmbiguous = errors.New("ambiguous: multiple matches")

// ErrBlocked is returned by UpdateFields when a task cannot be moved to
// in_progress because one or more of its depends_on tasks are not yet done.
var ErrBlocked = errors.New("task is blocked by unfinished dependencies")

// Store is the read/write gateway for task files under .logosyncx/tasks/.
//
// Directory layout:
//
//	.logosyncx/tasks/
//	└── <plan-slug>/          ← one directory per plan
//	    └── NNN-<title>/      ← one directory per task (zero-padded seq)
//	        ├── TASK.md       ← frontmatter + body
//	        └── WALKTHROUGH.md (created when task is marked done)
type Store struct {
	projectRoot string
	dir         string // absolute path to .logosyncx/tasks/
	plansDir    string // absolute path to .logosyncx/plans/
	cfg         *config.Config
}

// NewStore creates a Store rooted at projectRoot using the provided config.
func NewStore(projectRoot string, cfg *config.Config) *Store {
	return &Store{
		projectRoot: projectRoot,
		dir:         filepath.Join(projectRoot, ".logosyncx", "tasks"),
		plansDir:    filepath.Join(projectRoot, ".logosyncx", "plans"),
		cfg:         cfg,
	}
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// NextSeq returns the next available sequential number for tasks inside
// planGroupDir.  It scans for subdirectories whose names start with a
// zero-padded decimal prefix (e.g. "001-", "002-") and returns max+1.
// Returns 1 when planGroupDir does not exist or contains no numbered entries.
func (s *Store) NextSeq(planGroupDir string) (int, error) {
	entries, err := os.ReadDir(planGroupDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 1, nil
		}
		return 0, fmt.Errorf("read plan group dir %s: %w", planGroupDir, err)
	}

	max := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		seq := parseSeqPrefix(e.Name())
		if seq > max {
			max = seq
		}
	}
	return max + 1, nil
}

// Create creates a new task directory and TASK.md scaffold under
// .logosyncx/tasks/<plan-slug>/NNN-<title>/.
//
// Auto-fills: ID, Date, Seq, Status, Priority.
// t.Plan must be set by the caller (slug of the parent plan).
// Returns the absolute path to the created TASK.md.
func (s *Store) Create(t *Task) (string, error) {
	if strings.TrimSpace(t.Title) == "" {
		return "", fmt.Errorf("task title is required")
	}
	if strings.TrimSpace(t.Plan) == "" {
		return "", fmt.Errorf("task plan is required")
	}

	// Auto-fill ID.
	if t.ID == "" {
		id, err := generateID()
		if err != nil {
			return "", fmt.Errorf("generate task id: %w", err)
		}
		t.ID = id
	}

	// Auto-fill Date.
	if t.Date.IsZero() {
		t.Date = time.Now()
	}

	// Auto-fill Status.
	if t.Status == "" {
		t.Status = Status(s.cfg.Tasks.DefaultStatus)
		if !IsValidStatus(t.Status) {
			t.Status = StatusOpen
		}
	}

	// Auto-fill Priority.
	if t.Priority == "" {
		t.Priority = Priority(s.cfg.Tasks.DefaultPriority)
		if !IsValidPriority(t.Priority) {
			t.Priority = PriorityMedium
		}
	}

	// Resolve plan group directory.
	planGroupDir := filepath.Join(s.dir, t.Plan)

	// Validate that all depends_on seq numbers exist in the plan group (§8.4).
	if len(t.DependsOn) > 0 {
		existing, _ := s.loadPlanTasks(planGroupDir)
		existingSeqs := make(map[int]bool, len(existing))
		for _, et := range existing {
			existingSeqs[et.Seq] = true
		}
		for _, dep := range t.DependsOn {
			if !existingSeqs[dep] {
				return "", fmt.Errorf("depends_on seq %d does not exist in plan %q", dep, t.Plan)
			}
		}
	}

	// Auto-assign Seq.
	seq, err := s.NextSeq(planGroupDir)
	if err != nil {
		return "", err
	}
	t.Seq = seq

	// Create task directory: NNN-<slug>.
	taskDirName := TaskDirName(t.Seq, t.Title)
	taskDir := filepath.Join(planGroupDir, taskDirName)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return "", fmt.Errorf("create task dir %s: %w", taskDir, err)
	}

	t.DirPath = taskDir

	// Write TASK.md scaffold (frontmatter only).
	data, err := Marshal(*t)
	if err != nil {
		return "", fmt.Errorf("marshal task: %w", err)
	}

	taskPath := filepath.Join(taskDir, taskFileName)
	if err := os.WriteFile(taskPath, data, 0o644); err != nil {
		return "", fmt.Errorf("write TASK.md: %w", err)
	}

	// Reload so Excerpt is populated.
	if loaded, err := s.loadFile(taskPath); err == nil {
		*t = *loaded
	}

	// Best-effort git add.
	if s.cfg.Git.AutoPush {
		_ = gitutil.Add(s.projectRoot, taskPath)
	}

	// Best-effort index append.
	_ = AppendTaskIndex(s.projectRoot, FromTask(t))
	if s.cfg.Git.AutoPush {
		_ = gitutil.Add(s.projectRoot, TaskIndexFilePath(s.projectRoot))
	}

	return taskPath, nil
}

// Get finds the single task matching the given partial strings.
//
//   - planPartial: case-insensitive substring matched against plan group
//     directory names; empty means search all plans.
//   - nameOrPartial: case-insensitive substring matched against task directory
//     names (e.g. "001-add-jwt").
//
// Returns ErrNotFound (0 matches) or ErrAmbiguous (2+ matches).
func (s *Store) Get(planPartial, nameOrPartial string) (*Task, error) {
	taskPaths, err := s.findTaskPaths(planPartial, nameOrPartial)
	if err != nil {
		return nil, err
	}

	switch len(taskPaths) {
	case 0:
		return nil, fmt.Errorf("%w: %q in tasks/", ErrNotFound, nameOrPartial)
	case 1:
		return s.loadFile(taskPaths[0])
	default:
		names := make([]string, len(taskPaths))
		for i, p := range taskPaths {
			names[i] = filepath.Base(filepath.Dir(p))
		}
		return nil, fmt.Errorf("%w: %q matches %s", ErrAmbiguous, nameOrPartial, strings.Join(names, ", "))
	}
}

// List loads all tasks, applies the filter, and returns them sorted newest-first.
func (s *Store) List(f Filter) ([]*Task, error) {
	tasks, err := s.loadAll()
	if err != nil {
		return nil, err
	}
	result := Apply(tasks, f)
	sortByDateDesc(result)
	return result, nil
}

// UpdateFields loads the task identified by (planPartial, nameOrPartial),
// applies the supplied field updates, and writes the TASK.md back in-place
// (no directory move — status lives in frontmatter only).
//
// Supported keys: "status", "priority", "assignee".
//
// Special behaviour:
//   - "status" → "in_progress": hard error if IsBlocked returns true.
//   - "status" → "done": sets CompletedAt; calls CreateWalkthroughScaffold.
func (s *Store) UpdateFields(planPartial, nameOrPartial string, fields map[string]string) error {
	t, err := s.Get(planPartial, nameOrPartial)
	if err != nil {
		return err
	}

	for k, v := range fields {
		switch k {
		case "status":
			newStatus := Status(v)

			if !IsValidStatus(newStatus) {
				return fmt.Errorf("invalid status %q: must be one of open, in_progress, done", v)
			}

			if newStatus == StatusInProgress {
				// Load sibling tasks to check dependencies.
				planTasks, _ := s.loadPlanTasks(filepath.Dir(t.DirPath))
				if IsBlocked(t, planTasks) {
					return fmt.Errorf("%w: complete dependencies first", ErrBlocked)
				}
			}

			if newStatus == StatusDone && t.Status != StatusDone {
				wPath := filepath.Join(t.DirPath, walkthroughFileName)
				if !walkthroughHasContent(wPath) {
					return fmt.Errorf("WALKTHROUGH.md has no content: write WALKTHROUGH.md content first, then re-run")
				}
				now := time.Now()
				t.CompletedAt = &now
			}

			t.Status = newStatus

		case "priority":
			newPriority := Priority(v)
			if !IsValidPriority(newPriority) {
				return fmt.Errorf("invalid priority %q: must be one of low, medium, high", v)
			}
			t.Priority = newPriority

		case "assignee":
			t.Assignee = v

		default:
			return fmt.Errorf("unknown updatable field %q", k)
		}
	}

	// Write back in-place — no directory move.
	taskPath := filepath.Join(t.DirPath, taskFileName)
	data, err := Marshal(*t)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	if err := os.WriteFile(taskPath, data, 0o644); err != nil {
		return fmt.Errorf("write TASK.md: %w", err)
	}

	// Create walkthrough scaffold when task is marked done.
	if t.Status == StatusDone {
		if err := s.CreateWalkthroughScaffold(t); err != nil {
			// Non-fatal: warn but don't fail the update.
			fmt.Fprintf(os.Stderr, "warning: could not create walkthrough scaffold: %v\n", err)
		}
	}

	if s.cfg.Git.AutoPush {
		_ = gitutil.Add(s.projectRoot, taskPath)
	}

	// Best-effort index rebuild.
	_, _ = s.RebuildTaskIndex()
	if s.cfg.Git.AutoPush {
		_ = gitutil.Add(s.projectRoot, TaskIndexFilePath(s.projectRoot))
	}

	return nil
}

// Delete removes the task directory (including TASK.md and WALKTHROUGH.md)
// identified by (planPartial, nameOrPartial), then rebuilds the index.
func (s *Store) Delete(planPartial, nameOrPartial string) (*Task, error) {
	t, err := s.Get(planPartial, nameOrPartial)
	if err != nil {
		return nil, err
	}

	if s.cfg.Git.AutoPush {
		_ = gitutil.Remove(s.projectRoot, t.DirPath)
	}

	if err := os.RemoveAll(t.DirPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("remove task dir %s: %w", t.DirPath, err)
	}

	_, _ = s.RebuildTaskIndex()
	if s.cfg.Git.AutoPush {
		_ = gitutil.Add(s.projectRoot, TaskIndexFilePath(s.projectRoot))
	}

	return t, nil
}

// IsBlocked reports whether t has any unfinished dependencies within
// planTasks (same plan group).  A task is blocked when at least one seq
// number listed in t.DependsOn belongs to a task whose status is not done.
func IsBlocked(t *Task, planTasks []*Task) bool {
	if len(t.DependsOn) == 0 {
		return false
	}
	seqStatus := make(map[int]Status, len(planTasks))
	for _, pt := range planTasks {
		seqStatus[pt.Seq] = pt.Status
	}
	for _, dep := range t.DependsOn {
		if status, ok := seqStatus[dep]; !ok || status != StatusDone {
			return true
		}
	}
	return false
}

// defaultWalkthroughBody is the fallback section content used when
// .logosyncx/templates/walkthrough.md does not exist.
const defaultWalkthroughBody = `## Key Specification

<!-- What spec, task description, or requirements drove this implementation?
     Link to TASK.md sections, design docs, or paste the key constraints. -->

## What Was Done

<!-- Describe what was actually implemented or resolved. -->

## How It Was Done

<!-- Key steps, approach taken, alternatives considered. -->

## Gotchas & Lessons Learned

<!-- Anything that tripped you up, surprising behaviour, edge cases. -->

## Reusable Patterns

<!-- Code snippets, patterns, or conventions worth reusing. -->
`

// readWalkthroughTemplate reads .logosyncx/templates/walkthrough.md from the
// project root. Falls back to defaultWalkthroughBody if the file is missing.
func (s *Store) readWalkthroughTemplate() string {
	p := filepath.Join(s.projectRoot, ".logosyncx", "templates", "walkthrough.md")
	data, err := os.ReadFile(p)
	if err != nil {
		return defaultWalkthroughBody
	}
	return string(data)
}

// CreateWalkthroughScaffold writes a WALKTHROUGH.md scaffold into t.DirPath.
// If the file already exists, it is left untouched (idempotent).
// The section body is read from .logosyncx/templates/walkthrough.md when
// available, falling back to built-in defaults.
func (s *Store) CreateWalkthroughScaffold(t *Task) error {
	path := filepath.Join(t.DirPath, walkthroughFileName)

	// Idempotent: do nothing if the file already exists.
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	header := fmt.Sprintf("# Walkthrough: %s\n\n<!-- Auto-generated when this task was marked done. -->\n<!-- Fill in each section before running logos distill. -->\n\n", t.Title)
	content := header + s.readWalkthroughTemplate()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write WALKTHROUGH.md: %w", err)
	}

	if s.cfg.Git.AutoPush {
		_ = gitutil.Add(s.projectRoot, path)
	}

	return nil
}

// RebuildTaskIndex discards the existing task index and reconstructs it by
// scanning all TASK.md files. An empty index file is always created so that
// subsequent ReadAllTaskIndex calls succeed without triggering another rebuild.
func (s *Store) RebuildTaskIndex() (int, error) {
	path := TaskIndexFilePath(s.projectRoot)

	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		return 0, fmt.Errorf("create task index: %w", err)
	}

	tasks, loadErr := s.loadAll()

	// Group by plan to compute blocked status per plan group.
	planGroups := make(map[string][]*Task)
	for _, t := range tasks {
		planGroups[t.Plan] = append(planGroups[t.Plan], t)
	}

	for _, t := range tasks {
		entry := FromTask(t)
		entry.Blocked = IsBlocked(t, planGroups[t.Plan])
		if err := AppendTaskIndex(s.projectRoot, entry); err != nil {
			return 0, fmt.Errorf("append task index entry for %s: %w", t.DirPath, err)
		}
	}

	return len(tasks), loadErr
}

// ---------------------------------------------------------------------------
// Backward-compat shim used by cmd layer (single-arg Get)
// ---------------------------------------------------------------------------

// GetByName is a convenience wrapper that searches all plans.
// Callers that don't have a plan partial can use this instead of Get("", name).
func (s *Store) GetByName(nameOrPartial string) (*Task, error) {
	return s.Get("", nameOrPartial)
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// loadAll walks .logosyncx/tasks/<plan>/<task>/TASK.md and returns every
// successfully parsed task.  Parse errors are accumulated (non-fatal).
func (s *Store) loadAll() ([]*Task, error) {
	var tasks []*Task
	var errs []string

	planEntries, err := os.ReadDir(s.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tasks dir: %w", err)
	}

	for _, planEntry := range planEntries {
		if !planEntry.IsDir() {
			continue
		}
		planGroupDir := filepath.Join(s.dir, planEntry.Name())
		planTasks, parseErrs := s.loadPlanTasks(planGroupDir)
		errs = append(errs, parseErrs...)

		// Compute Blocked for each task in this plan group and set it on
		// the Task struct so that matchesFilter (in-memory path) can use it.
		for _, t := range planTasks {
			t.Blocked = IsBlocked(t, planTasks)
		}
		tasks = append(tasks, planTasks...)
	}

	if len(errs) > 0 {
		return tasks, fmt.Errorf("some task files could not be parsed:\n  %s",
			strings.Join(errs, "\n  "))
	}
	return tasks, nil
}

// loadPlanTasks loads all TASK.md files inside a single plan group directory.
// Returns parsed tasks and a (possibly empty) slice of error strings.
func (s *Store) loadPlanTasks(planGroupDir string) ([]*Task, []string) {
	var tasks []*Task
	var errs []string

	taskEntries, err := os.ReadDir(planGroupDir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Sprintf("%s: %v", planGroupDir, err))
		}
		return nil, errs
	}

	for _, taskEntry := range taskEntries {
		if !taskEntry.IsDir() {
			continue
		}
		taskPath := filepath.Join(planGroupDir, taskEntry.Name(), taskFileName)
		t, err := s.loadFile(taskPath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", taskPath, err))
			continue
		}
		tasks = append(tasks, t)
	}

	// Sort by Seq so dependency resolution is deterministic.
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Seq < tasks[j].Seq
	})

	return tasks, errs
}

// loadFile reads and parses a single TASK.md file at path.
// DirPath is set to the directory containing the file.
func (s *Store) loadFile(path string) (*Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t, err := ParseWithOptions(taskFileName, data, ParseOptions{
		ExcerptSection: s.cfg.Tasks.ExcerptSection,
	})
	if err != nil {
		return nil, err
	}
	t.DirPath = filepath.Dir(path)
	return &t, nil
}

// findTaskPaths returns the TASK.md paths that match planPartial and
// nameOrPartial.  planPartial empty → search all plan groups.
func (s *Store) findTaskPaths(planPartial, nameOrPartial string) ([]string, error) {
	planEntries, err := os.ReadDir(s.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read tasks dir: %w", err)
	}

	lowerPlan := strings.ToLower(planPartial)
	lowerName := strings.ToLower(nameOrPartial)

	var matches []string
	for _, planEntry := range planEntries {
		if !planEntry.IsDir() {
			continue
		}
		if lowerPlan != "" && !strings.Contains(strings.ToLower(planEntry.Name()), lowerPlan) {
			continue
		}

		planGroupDir := filepath.Join(s.dir, planEntry.Name())
		taskEntries, err := os.ReadDir(planGroupDir)
		if err != nil {
			continue
		}

		for _, taskEntry := range taskEntries {
			if !taskEntry.IsDir() {
				continue
			}
			if lowerName != "" && !strings.Contains(strings.ToLower(taskEntry.Name()), lowerName) {
				continue
			}
			candidate := filepath.Join(planGroupDir, taskEntry.Name(), taskFileName)
			if _, err := os.Stat(candidate); err == nil {
				matches = append(matches, candidate)
			}
		}
	}

	return matches, nil
}

// generateID returns a new unique task ID of the form "t-<6 hex chars>".
func generateID() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return idPrefix + hex.EncodeToString(b), nil
}

// parseSeqPrefix extracts the leading decimal number from a directory name
// like "001-add-jwt-middleware".  Returns 0 if no prefix is found.
func parseSeqPrefix(name string) int {
	idx := strings.IndexByte(name, '-')
	if idx <= 0 {
		return 0
	}
	n, err := strconv.Atoi(name[:idx])
	if err != nil {
		return 0
	}
	return n
}

// walkthroughHasContent reports whether the WALKTHROUGH.md at path exists and
// contains at least one substantive line — a non-empty line that does not start
// with "<!--" (HTML comment). Scaffold-only files (all HTML comment blocks)
// return false.
func walkthroughHasContent(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "<!--") {
			return true
		}
	}
	return false
}

// sortByDateDesc sorts tasks newest-first in-place.
func sortByDateDesc(tasks []*Task) {
	for i := 1; i < len(tasks); i++ {
		for j := i; j > 0 && tasks[j].Date.After(tasks[j-1].Date); j-- {
			tasks[j], tasks[j-1] = tasks[j-1], tasks[j]
		}
	}
}
