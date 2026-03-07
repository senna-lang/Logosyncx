// Package index manages the JSONL plan index stored at
// .logosyncx/index.jsonl.  Each line is a JSON-encoded Entry representing
// one saved plan.  The index lets logos ls and logos search operate without
// reading individual plan Markdown files on every invocation.
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

	"github.com/senna-lang/logosyncx/pkg/plan"
)

const indexFileName = "index.jsonl"

// Entry is a single row in the index file.
// Fields mirror the plan frontmatter plus the excerpt and derived fields.
type Entry struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	Date      time.Time `json:"date"`
	Topic     string    `json:"topic"`
	Tags      []string  `json:"tags"`
	Agent     string    `json:"agent"`
	Related   []string  `json:"related"`
	DependsOn []string  `json:"depends_on"`
	TasksDir  string    `json:"tasks_dir"`
	Distilled bool      `json:"distilled"`
	Blocked   bool      `json:"blocked"` // true if any DependsOn plan is not yet distilled
	Excerpt   string    `json:"excerpt"`
}

// FilePath returns the absolute path to the index file under projectRoot.
func FilePath(projectRoot string) string {
	return filepath.Join(projectRoot, ".logosyncx", indexFileName)
}

// FromPlan converts a plan.Plan to an Entry. The Blocked field is computed:
// true when any filename listed in DependsOn is not yet distilled, based on
// the provided allPlans slice.
func FromPlan(p plan.Plan, allPlans []plan.Plan) Entry {
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	related := p.Related
	if related == nil {
		related = []string{}
	}
	dependsOn := p.DependsOn
	if dependsOn == nil {
		dependsOn = []string{}
	}
	date := time.Now()
	if p.Date != nil {
		date = *p.Date
	}

	blocked := false
	if len(p.DependsOn) > 0 {
		distilled := make(map[string]bool, len(allPlans))
		for _, ap := range allPlans {
			distilled[ap.Filename] = ap.Distilled
		}
		for _, dep := range p.DependsOn {
			if !distilled[dep] {
				blocked = true
				break
			}
		}
	}

	return Entry{
		ID:        p.ID,
		Filename:  p.Filename,
		Date:      date,
		Topic:     p.Topic,
		Tags:      tags,
		Agent:     p.Agent,
		Related:   related,
		DependsOn: dependsOn,
		TasksDir:  p.TasksDir,
		Distilled: p.Distilled,
		Blocked:   blocked,
		Excerpt:   p.Excerpt,
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
// under projectRoot. The file and any missing parent directories are created
// automatically.
func Append(projectRoot string, e Entry) error {
	path := FilePath(projectRoot)

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
// .md file under the plans directory. An empty index file is always created,
// even when there are no plans, so that subsequent ReadAll calls succeed
// without triggering another rebuild.
//
// excerptSection is the heading name used to extract each plan's excerpt
// (e.g. cfg.Plans.ExcerptSection). An empty string falls back to "Background".
//
// The first return value is the number of plans successfully indexed.
func Rebuild(projectRoot string, excerptSection string) (int, error) {
	path := FilePath(projectRoot)

	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		return 0, fmt.Errorf("create index: %w", err)
	}

	plans, loadErr := plan.LoadAllWithOptions(projectRoot, plan.ParseOptions{
		ExcerptSection: excerptSection,
	})

	for _, p := range plans {
		if err := Append(projectRoot, FromPlan(p, plans)); err != nil {
			return 0, fmt.Errorf("append entry for %s: %w", p.Filename, err)
		}
	}

	return len(plans), loadErr
}
