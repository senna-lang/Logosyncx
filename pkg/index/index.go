// Package index manages the JSONL session index stored at
// .logosyncx/index.jsonl.  Each line is a JSON-encoded Entry representing
// one saved session.  The index lets logos ls and logos search operate
// without reading individual session Markdown files on every invocation.
package index

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/senna-lang/logosyncx/pkg/session"
)

const indexFileName = "index.jsonl"

// Entry is a single row in the index file.
// Fields mirror the session frontmatter plus the excerpt derived from
// the ## Summary section.
type Entry struct {
	ID       string    `json:"id"`
	Filename string    `json:"filename"`
	Date     time.Time `json:"date"`
	Topic    string    `json:"topic"`
	Tags     []string  `json:"tags"`
	Agent    string    `json:"agent"`
	Related  []string  `json:"related"`
	Excerpt  string    `json:"excerpt"`
}

// FilePath returns the absolute path to the index file under projectRoot.
func FilePath(projectRoot string) string {
	return filepath.Join(projectRoot, ".logosyncx", indexFileName)
}

// FromSession converts a session.Session to an Entry suitable for writing to
// the index.  Nil slice fields are normalised to empty slices so that JSON
// serialisation always produces [] rather than null.
func FromSession(s session.Session) Entry {
	tags := s.Tags
	if tags == nil {
		tags = []string{}
	}
	related := s.Related
	if related == nil {
		related = []string{}
	}
	date := time.Now()
	if s.Date != nil {
		date = *s.Date
	}
	return Entry{
		ID:       s.ID,
		Filename: s.Filename,
		Date:     date,
		Topic:    s.Topic,
		Tags:     tags,
		Agent:    s.Agent,
		Related:  related,
		Excerpt:  s.Excerpt,
	}
}

// ReadAll reads every entry from the index file under projectRoot.
// If the file does not exist os.ErrNotExist is returned (unwrapped so callers
// can use errors.Is).  Lines that are blank are silently skipped; a malformed
// line causes ReadAll to return whatever it has collected so far plus an error.
func ReadAll(projectRoot string) ([]Entry, error) {
	path := FilePath(projectRoot)
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("open index: %w", err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return entries, fmt.Errorf("parse index line %d: %w", lineNum, err)
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("read index: %w", err)
	}
	return entries, nil
}

// Append serialises e as a single JSON line and appends it to the index file
// under projectRoot.  The file and any missing parent directories are created
// automatically.
func Append(projectRoot string, e Entry) error {
	path := FilePath(projectRoot)

	// Ensure the parent directory exists (it should, but be safe).
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create index directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open index for append: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal index entry: %w", err)
	}
	if _, err := fmt.Fprintf(f, "%s\n", data); err != nil {
		return fmt.Errorf("write index entry: %w", err)
	}
	return nil
}

// Rebuild discards the existing index and reconstructs it by scanning every
// .md file under the sessions directory.  An empty index file is always
// created, even when there are no sessions, so that subsequent ReadAll calls
// succeed without triggering another rebuild.
//
// excerptSection is the heading name used to extract each session's excerpt
// (e.g. cfg.Sessions.ExcerptSection).  An empty string falls back to "Summary".
//
// The first return value is the number of sessions successfully indexed.
// A non-nil error indicates either an I/O failure (fatal) or parse warnings
// from session files (non-fatal, sessions still indexed where possible).
func Rebuild(projectRoot string, excerptSection string) (int, error) {
	path := FilePath(projectRoot)

	// Always create / truncate the index file so it exists after this call.
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		return 0, fmt.Errorf("create index: %w", err)
	}

	// Load all sessions from disk; LoadAllWithOptions returns partial results
	// on parse errors so we index as many as possible.
	sessions, loadErr := session.LoadAllWithOptions(projectRoot, session.ParseOptions{
		ExcerptSection: excerptSection,
	})

	for _, s := range sessions {
		if err := Append(projectRoot, FromSession(s)); err != nil {
			return 0, fmt.Errorf("append entry for %s: %w", s.Filename, err)
		}
	}

	// loadErr is non-nil only when some files could not be parsed; surface it
	// to the caller for display as a warning rather than a hard failure.
	return len(sessions), loadErr
}
