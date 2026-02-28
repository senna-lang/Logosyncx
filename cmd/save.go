package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/senna-lang/logosyncx/internal/gitutil"
	"github.com/senna-lang/logosyncx/internal/project"
	"github.com/senna-lang/logosyncx/internal/task"
	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/index"
	"github.com/senna-lang/logosyncx/pkg/session"
	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:   "save",
	Args:  cobra.NoArgs,
	Short: "Save a session file to .logosyncx/sessions/",
	Long: `Save a session to .logosyncx/sessions/ using flag-based input.

  logos save --topic "..." [--tag <tag>] [--agent <agent>] \
             [--related <session>] [--task <task>] \
             [--section "Name=content"] [--section "Name2=content2"] ...

Body content is provided exclusively via --section flags. Each --section value
must be formatted as "Name=content" where Name matches one of the section names
defined in .logosyncx/config.json (sessions.sections). Unknown section names are
rejected. --section may be repeated once per section; providing the same section
name more than once is an error.

Use --task to link this session to one or more existing tasks (partial name
match). The resolved task filenames are stored in the session's tasks: field.

When git.auto_push is false (the default), no git operations are performed —
commit and push remain entirely the user's responsibility.

When git.auto_push is true in .logosyncx/config.json, logos save automatically
runs git add, git commit, and git push after writing the session file so that
AI agents can share context with the team without manual git interaction.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		topic, _ := cmd.Flags().GetString("topic")
		tags, _ := cmd.Flags().GetStringArray("tag")
		agent, _ := cmd.Flags().GetString("agent")
		related, _ := cmd.Flags().GetStringArray("related")
		taskPartials, _ := cmd.Flags().GetStringArray("task")
		sections, _ := cmd.Flags().GetStringArray("section")
		return runSave(topic, tags, agent, related, taskPartials, sections)
	},
}

func init() {
	saveCmd.Flags().StringP("topic", "t", "", "Session topic (required)")
	saveCmd.Flags().StringArray("tag", []string{}, "Tag to attach (repeatable: --tag go --tag cli)")
	saveCmd.Flags().StringP("agent", "a", "", "Agent name (e.g. claude-code)")
	saveCmd.Flags().StringArray("related", []string{}, "Related session filename (repeatable)")
	saveCmd.Flags().StringArray("task", []string{}, "Task to link (partial name, repeatable: --task impl --task docs)")
	saveCmd.Flags().StringArray("section", []string{}, "Section content as 'Name=content' (repeatable; name must be defined in config)")
	rootCmd.AddCommand(saveCmd)
}

func runSave(topic string, tags []string, agent string, related []string, taskPartials []string, sections []string) error {
	if strings.TrimSpace(topic) == "" {
		return errors.New("provide --topic <topic>")
	}

	s := session.Session{
		Topic:   topic,
		Tags:    tags,
		Agent:   agent,
		Related: related,
	}

	// Resolve --task partials to full task filenames before loading project
	// root (we need root first, so this is deferred below).

	var err error

	// Auto-fill missing frontmatter fields.
	s.ID, err = generateID()
	if err != nil {
		return fmt.Errorf("generate id: %w", err)
	}
	now := time.Now()
	s.Date = &now

	// Find the project root.
	root, err := project.FindRoot()
	if err != nil {
		return err
	}

	// Load config for section definitions, privacy filters, and git settings.
	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Resolve --task partial names to canonical task filenames.
	if len(taskPartials) > 0 {
		store := task.NewStore(root, &cfg)
		for _, partial := range taskPartials {
			t, err := store.Get(partial)
			if err != nil {
				return fmt.Errorf("resolve task %q: %w", partial, err)
			}
			s.Tasks = append(s.Tasks, t.Filename)
		}
	}

	// Build the body from --section flags.
	// Each flag value must be "Name=content" where Name is defined in config.
	// An empty sections list produces an empty body.
	bodyText, err := buildBodyFromSections(sections, cfg.Sessions.Sections)
	if err != nil {
		return err
	}
	s.Body = bodyText

	// Check privacy filter patterns and warn on matches.
	warnPrivacy(s.Body, cfg.Privacy.FilterPatterns)

	// Warn if required sections are missing from the body.
	warnMissingSections(s.Body, cfg.Sessions.Sections)

	// Write the session file.
	savedPath, err := session.Write(root, s)
	if err != nil {
		return fmt.Errorf("write session: %w", err)
	}

	fmt.Printf("✓ Saved session to %s\n", savedPath)

	// Update the session index (append-only, best-effort).
	// Use ParseWithOptions so the excerpt respects the project's excerpt_section.
	savedSession, loadErr := session.ParseWithOptions(
		savedPath,
		func() []byte {
			data, _ := os.ReadFile(savedPath)
			return data
		}(),
		session.ParseOptions{ExcerptSection: cfg.Sessions.ExcerptSection},
	)
	if loadErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load saved session for indexing (%v)\n", loadErr)
	} else {
		savedSession.Filename = filepath.Base(savedPath)
		if indexErr := index.Append(root, index.FromSession(savedSession)); indexErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not update index (%v) — run `logos sync` to rebuild\n", indexErr)
		}
	}

	// In auto mode: stage, commit, and push so agents can share context
	// without any manual git interaction.
	// In manual mode: leave all git operations to the user.
	if cfg.Git.AutoPush {
		filesToStage := []string{savedPath, index.FilePath(root)}
		allStaged := true
		for _, f := range filesToStage {
			if err := gitutil.Add(root, f); err != nil {
				fmt.Fprintf(os.Stderr, "warning: git add failed for %s (%v) — stage the file manually\n", f, err)
				allStaged = false
			}
		}
		if allStaged {
			fmt.Println("✓ Staged with git add")
		}

		commitMsg := fmt.Sprintf("logos: save session %q", topic)
		if err := gitutil.Commit(root, commitMsg); err != nil {
			fmt.Fprintf(os.Stderr, "warning: git commit failed (%v) — commit and push manually\n", err)
			fmt.Println()
			fmt.Println("Next: commit and push to share context with your team.")
			return nil
		}
		fmt.Println("✓ Committed with git commit")

		if err := gitutil.Push(root); err != nil {
			fmt.Fprintf(os.Stderr, "warning: git push failed (%v) — push manually\n", err)
			fmt.Println()
			fmt.Println("Next: push to share context with your team.")
			return nil
		}
		fmt.Println("✓ Pushed with git push")
		fmt.Println()
		fmt.Println("Context shared with your team.")
		return nil
	}

	fmt.Println()
	fmt.Println("Next: commit and push to share context with your team.")
	return nil
}

// generateID returns a random 6-character lowercase hex string.
func generateID() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// warnMissingSections checks that every required section defined in the project
// config is present in the body. A missing required section triggers a warning
// (not an error) so the save is never blocked by structural issues.
// Sections are matched by name only — any heading level (h1–h6) is accepted.
func warnMissingSections(body string, sections []config.SectionConfig) {
	for _, sec := range sections {
		if !sec.Required {
			continue
		}
		if !hasHeading(body, sec.Name) {
			fmt.Fprintf(os.Stderr, "warning: required section %q is missing from the session body\n", sec.Name)
		}
	}
}

// hasHeading returns true if the body contains a markdown ATX heading whose
// text matches name (case-insensitive) at any heading level (h1–h6).
func hasHeading(body, name string) bool {
	for _, line := range strings.Split(body, "\n") {
		i := 0
		for i < len(line) && line[i] == '#' {
			i++
		}
		if i == 0 || i > 6 || i >= len(line) || line[i] != ' ' {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(line[i+1:]), name) {
			return true
		}
	}
	return false
}

// warnPrivacy checks the session body against each compiled regex pattern
// in filterPatterns and prints a warning for each match found.
func warnPrivacy(body string, filterPatterns []string) {
	for _, pattern := range filterPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: invalid privacy filter pattern %q: %v\n", pattern, err)
			continue
		}
		if re.MatchString(body) {
			fmt.Fprintf(os.Stderr, "warning: session content matches privacy filter pattern %q — review before committing\n", pattern)
		}
	}
}
