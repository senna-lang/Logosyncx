// index.go manages the JSONL task index stored at .logosyncx/task-index.jsonl.
// Each line is a JSON-encoded TaskJSON representing one saved task.
// The index lets logos task ls operate without reading individual task
// Markdown files on every invocation.
package task

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const taskIndexFileName = "task-index.jsonl"

// TaskIndexFilePath returns the absolute path to the task index file under
// projectRoot.
func TaskIndexFilePath(projectRoot string) string {
	return filepath.Join(projectRoot, ".logosyncx", taskIndexFileName)
}

// ReadAllTaskIndex reads every entry from the task index file under
// projectRoot.  If the file does not exist os.ErrNotExist is returned
// (unwrapped) so callers can use errors.Is.  Blank lines are silently
// skipped; a malformed line causes ReadAllTaskIndex to return whatever it has
// collected so far plus an error.
func ReadAllTaskIndex(projectRoot string) ([]TaskJSON, error) {
	path := TaskIndexFilePath(projectRoot)
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("open task index: %w", err)
	}
	defer f.Close()

	var entries []TaskJSON
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e TaskJSON
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return entries, fmt.Errorf("parse task index line %d: %w", lineNum, err)
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("read task index: %w", err)
	}
	return entries, nil
}

// AppendTaskIndex serialises e as a single JSON line and appends it to the
// task index file under projectRoot.  The file and any missing parent
// directories are created automatically.
func AppendTaskIndex(projectRoot string, e TaskJSON) error {
	path := TaskIndexFilePath(projectRoot)

	// Ensure the parent directory exists (it should, but be safe).
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create task index directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open task index for append: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal task index entry: %w", err)
	}
	if _, err := fmt.Fprintf(f, "%s\n", data); err != nil {
		return fmt.Errorf("write task index entry: %w", err)
	}
	return nil
}

// RebuildTaskIndex discards the existing task index and reconstructs it by
// scanning every .md file under .logosyncx/tasks/.  An empty index file is
// always created, even when there are no tasks, so that subsequent
// ReadAllTaskIndex calls succeed without triggering another rebuild.
//
// The first return value is the number of tasks successfully indexed.
// A non-nil error indicates either an I/O failure (fatal) or parse warnings
// from task files (non-fatal; tasks are still indexed where possible).
func (s *Store) RebuildTaskIndex() (int, error) {
	path := TaskIndexFilePath(s.projectRoot)

	// Always create / truncate the index file so it exists after this call.
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		return 0, fmt.Errorf("create task index: %w", err)
	}

	// Load all tasks from disk; loadAll returns partial results on parse
	// errors so we index as many as possible.
	tasks, loadErr := s.loadAll()

	for _, t := range tasks {
		if err := AppendTaskIndex(s.projectRoot, t.ToJSON()); err != nil {
			return 0, fmt.Errorf("append task index entry for %s: %w", t.Filename, err)
		}
	}

	// loadErr is non-nil only when some files could not be parsed; surface it
	// to the caller for display as a warning rather than a hard failure.
	return len(tasks), loadErr
}
