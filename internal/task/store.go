// Package task provides Store — the read/write layer for .logosyncx/tasks/.
package task

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"crypto/rand"
	"encoding/hex"

	"github.com/senna-lang/logosyncx/internal/gitutil"
	"github.com/senna-lang/logosyncx/pkg/config"
)

// idPrefix is prepended to every auto-generated task ID.
const idPrefix = "t-"

// ErrNotFound is returned by Get and ResolveSession when no match is found.
var ErrNotFound = errors.New("not found")

// ErrAmbiguous is returned by Get and ResolveSession when more than one file
// matches the supplied partial name.
var ErrAmbiguous = errors.New("ambiguous: multiple matches")

// Store is the read/write gateway for task files in .logosyncx/tasks/.
// Tasks are organised into status subdirectories:
//
//	.logosyncx/tasks/
//	├── open/
//	├── in_progress/
//	├── done/
//	└── cancelled/
type Store struct {
	projectRoot string
	dir         string // absolute path to .logosyncx/tasks/ (root)
	sessionDir  string // absolute path to .logosyncx/sessions/
	cfg         *config.Config
}

// NewStore creates a Store rooted at projectRoot using the provided config.
func NewStore(projectRoot string, cfg *config.Config) *Store {
	return &Store{
		projectRoot: projectRoot,
		dir:         filepath.Join(projectRoot, ".logosyncx", "tasks"),
		sessionDir:  filepath.Join(projectRoot, ".logosyncx", "sessions"),
		cfg:         cfg,
	}
}

// statusDir returns the absolute path to the subdirectory for the given status.
func (s *Store) statusDir(status Status) string {
	return filepath.Join(s.dir, string(status))
}

// taskPath returns the absolute path to a task file given its status and base
// filename.
func (s *Store) taskPath(status Status, filename string) string {
	return filepath.Join(s.statusDir(status), filename)
}

// List reads every .md file from all status subdirectories, parses them,
// applies f, and returns the matching tasks sorted newest-first.
func (s *Store) List(f Filter) ([]*Task, error) {
	tasks, err := s.loadAll()
	if err != nil {
		return nil, err
	}
	result := Apply(tasks, f)
	sortByDateDesc(result)
	return result, nil
}

// Get returns the single task whose filename contains nameOrPartial as a
// case-insensitive substring, searching across all status subdirectories.
// Returns ErrNotFound if nothing matches, ErrAmbiguous if more than one file
// matches.
func (s *Store) Get(nameOrPartial string) (*Task, error) {
	lower := strings.ToLower(nameOrPartial)

	var matchPaths []string
	for _, status := range ValidStatuses {
		subDir := s.statusDir(status)
		entries, err := os.ReadDir(subDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read tasks/%s dir: %w", status, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			if strings.Contains(strings.ToLower(e.Name()), lower) {
				matchPaths = append(matchPaths, filepath.Join(subDir, e.Name()))
			}
		}
	}

	switch len(matchPaths) {
	case 0:
		return nil, fmt.Errorf("%w: %q in tasks/", ErrNotFound, nameOrPartial)
	case 1:
		return s.loadFile(matchPaths[0])
	default:
		names := make([]string, len(matchPaths))
		for i, p := range matchPaths {
			names[i] = filepath.Base(p)
		}
		return nil, fmt.Errorf("%w: %q matches %s", ErrAmbiguous, nameOrPartial, strings.Join(names, ", "))
	}
}

// Save auto-fills t.ID (if empty) and t.Date (if zero), marshals t to
// markdown, and writes it to tasks/<status>/<date>_<slug>.md.
// body is the markdown body (everything after the frontmatter closing ---).
// A git add is attempted after writing; failures are emitted as warnings
// rather than errors so the file operation is never rolled back.
// The returned string is the full path of the written file.
func (s *Store) Save(t *Task, body string) (string, error) {
	// Auto-fill missing fields.
	if t.ID == "" {
		id, err := generateID()
		if err != nil {
			return "", fmt.Errorf("generate task id: %w", err)
		}
		t.ID = id
	}
	if t.Date.IsZero() {
		t.Date = time.Now()
	}
	if t.Status == "" {
		t.Status = Status(s.cfg.Tasks.DefaultStatus)
	}
	if t.Priority == "" {
		t.Priority = Priority(s.cfg.Tasks.DefaultPriority)
	}

	// Ensure the status subdirectory exists.
	subDir := s.statusDir(t.Status)
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		return "", fmt.Errorf("create tasks/%s dir: %w", t.Status, err)
	}

	// Attach body so Marshal produces the full file.
	t.Body = body

	data, err := Marshal(*t)
	if err != nil {
		return "", fmt.Errorf("marshal task: %w", err)
	}

	filename := FileName(*t)
	path := s.taskPath(t.Status, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write task file: %w", err)
	}

	// Reload so Filename and Excerpt are populated from the written file.
	loaded, err := s.loadFile(path)
	if err == nil {
		*t = *loaded
	}

	// Best-effort git add (auto mode only).
	if s.cfg.Git.AutoPush {
		_ = gitutil.Add(s.projectRoot, path)
	}

	// Best-effort index append.
	_ = AppendTaskIndex(s.projectRoot, t.ToJSON())

	return path, nil
}

// UpdateFields loads the task matching nameOrPartial, applies the given field
// updates, re-serialises, and writes the file back. If the status field
// changes, the task file is moved to the new status subdirectory.
// Supported keys: "status", "priority", "assignee", "session".
// git operations are attempted after writing (best-effort).
func (s *Store) UpdateFields(nameOrPartial string, fields map[string]string) error {
	t, err := s.Get(nameOrPartial)
	if err != nil {
		return err
	}

	oldStatus := t.Status
	oldPath := s.taskPath(oldStatus, t.Filename)

	for k, v := range fields {
		switch k {
		case "status":
			t.Status = Status(v)
		case "priority":
			t.Priority = Priority(v)
		case "assignee":
			t.Assignee = v
		case "session":
			t.Session = v
		default:
			return fmt.Errorf("unknown updatable field %q", k)
		}
	}

	newPath := s.taskPath(t.Status, t.Filename)

	// Ensure the destination directory exists.
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return fmt.Errorf("create status dir: %w", err)
	}

	// Write updated content to the new path.
	data, err := Marshal(*t)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	if err := os.WriteFile(newPath, data, 0o644); err != nil {
		return fmt.Errorf("write task file: %w", err)
	}

	// If the status changed, remove the old file and update git accordingly.
	if oldPath != newPath {
		if err := os.Remove(oldPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove old task file: %w", err)
		}
		if s.cfg.Git.AutoPush {
			_ = gitutil.Remove(s.projectRoot, oldPath)
		}
	}

	if s.cfg.Git.AutoPush {
		_ = gitutil.Add(s.projectRoot, newPath)
	}

	// Rebuild index to reflect updated field values (best-effort).
	_, _ = s.RebuildTaskIndex()

	return nil
}

// Delete removes the task file matching nameOrPartial from its status
// subdirectory. git rm is attempted after deletion (best-effort).
func (s *Store) Delete(nameOrPartial string) error {
	t, err := s.Get(nameOrPartial)
	if err != nil {
		return err
	}

	path := s.taskPath(t.Status, t.Filename)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove task file: %w", err)
	}

	if s.cfg.Git.AutoPush {
		_ = gitutil.Remove(s.projectRoot, path)
	}

	// Rebuild index to remove the deleted entry (best-effort).
	_, _ = s.RebuildTaskIndex()

	return nil
}

// Purge deletes all task files in the given status subdirectory.
// git rm is attempted for each deleted file (best-effort).
// The first return value is the number of files successfully deleted.
func (s *Store) Purge(status Status) (int, error) {
	subDir := s.statusDir(status)
	entries, err := os.ReadDir(subDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("read tasks/%s dir: %w", status, err)
	}

	count := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(subDir, e.Name())
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return count, fmt.Errorf("remove %s: %w", e.Name(), err)
		}
		if s.cfg.Git.AutoPush {
			_ = gitutil.Remove(s.projectRoot, path)
		}
		count++
	}

	if count > 0 {
		_, _ = s.RebuildTaskIndex()
	}

	return count, nil
}

// ResolveSession finds the session filename in sessions/ that contains
// partial as a case-insensitive substring.  Returns ErrNotFound if nothing
// matches, ErrAmbiguous if more than one file matches.
func (s *Store) ResolveSession(partial string) (string, error) {
	entries, err := os.ReadDir(s.sessionDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %q in sessions/", ErrNotFound, partial)
		}
		return "", fmt.Errorf("read sessions dir: %w", err)
	}

	var matches []string
	lower := strings.ToLower(partial)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if strings.Contains(strings.ToLower(e.Name()), lower) {
			matches = append(matches, e.Name())
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%w: %q in sessions/", ErrNotFound, partial)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("%w: %q matches %s", ErrAmbiguous, partial, strings.Join(matches, ", "))
	}
}

// --- private helpers ---------------------------------------------------------

// loadAll reads every .md file from all status subdirectories and returns all
// successfully parsed tasks. Parse errors are accumulated but do not abort
// the overall read.
func (s *Store) loadAll() ([]*Task, error) {
	var tasks []*Task
	var errs []string

	for _, status := range ValidStatuses {
		subDir := s.statusDir(status)
		entries, err := os.ReadDir(subDir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read tasks/%s dir: %w", status, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			t, err := s.loadFile(filepath.Join(subDir, e.Name()))
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s/%s: %v", status, e.Name(), err))
				continue
			}
			tasks = append(tasks, t)
		}
	}

	if len(errs) > 0 {
		return tasks, fmt.Errorf("some task files could not be parsed:\n  %s",
			strings.Join(errs, "\n  "))
	}
	return tasks, nil
}

// loadFile reads and parses a single task file at path.
// Task.Filename is set to the base filename only (not including the status
// subdirectory), as the status field on the Task itself encodes the subdir.
func (s *Store) loadFile(path string) (*Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t, err := Parse(filepath.Base(path), data)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// generateID returns a new unique task ID of the form "t-<6 hex chars>".
func generateID() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return idPrefix + hex.EncodeToString(b), nil
}

// sortByDateDesc sorts tasks newest-first in-place.
func sortByDateDesc(tasks []*Task) {
	for i := 1; i < len(tasks); i++ {
		for j := i; j > 0 && tasks[j].Date.After(tasks[j-1].Date); j-- {
			tasks[j], tasks[j-1] = tasks[j-1], tasks[j]
		}
	}
}
