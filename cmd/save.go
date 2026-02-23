package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/senna-lang/logosyncx/internal/gitutil"
	"github.com/senna-lang/logosyncx/internal/project"
	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/index"
	"github.com/senna-lang/logosyncx/pkg/session"
	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save a session file to .logosyncx/sessions/",
	Long: `Save a session to .logosyncx/sessions/ using flag-based input.

  logos save --topic "..." [--tag <tag>] [--agent <agent>] \
             [--related <session>] [--body "..."] [--body-stdin]

git add is run automatically; git commit and push remain the user's responsibility.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		topic, _ := cmd.Flags().GetString("topic")
		tags, _ := cmd.Flags().GetStringArray("tag")
		agent, _ := cmd.Flags().GetString("agent")
		related, _ := cmd.Flags().GetStringArray("related")
		body, _ := cmd.Flags().GetString("body")
		bodyStdin, _ := cmd.Flags().GetBool("body-stdin")
		return runSave(topic, tags, agent, related, body, bodyStdin)
	},
}

func init() {
	saveCmd.Flags().StringP("topic", "t", "", "Session topic (required)")
	saveCmd.Flags().StringArray("tag", []string{}, "Tag to attach (repeatable: --tag go --tag cli)")
	saveCmd.Flags().StringP("agent", "a", "", "Agent name (e.g. claude-code)")
	saveCmd.Flags().StringArray("related", []string{}, "Related session filename (repeatable)")
	saveCmd.Flags().StringP("body", "b", "", "Session body text (inline)")
	saveCmd.Flags().Bool("body-stdin", false, "Read session body prose from stdin (no frontmatter needed)")
	rootCmd.AddCommand(saveCmd)
}

func runSave(topic string, tags []string, agent string, related []string, body string, bodyStdin bool) error {
	if strings.TrimSpace(topic) == "" {
		return errors.New("provide --topic <topic>")
	}
	if body != "" && bodyStdin {
		return errors.New("--body and --body-stdin are mutually exclusive")
	}

	var bodyText string
	if bodyStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		bodyText = string(data)
	} else {
		bodyText = body
	}

	s := session.Session{
		Topic:   topic,
		Tags:    tags,
		Agent:   agent,
		Related: related,
		Body:    bodyText,
	}

	var err error

	// Auto-fill missing frontmatter fields.
	s.ID, err = generateID()
	if err != nil {
		return fmt.Errorf("generate id: %w", err)
	}
	s.Date = time.Now()

	// Find the project root.
	root, err := project.FindRoot()
	if err != nil {
		return err
	}

	// Load config for privacy filter patterns.
	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Check privacy filter patterns and warn on matches.
	warnPrivacy(s.Body, cfg.Privacy.FilterPatterns)

	// Write the session file.
	savedPath, err := session.Write(root, s)
	if err != nil {
		return fmt.Errorf("write session: %w", err)
	}

	fmt.Printf("✓ Saved session to %s\n", savedPath)

	// Update the session index (append-only, best-effort).
	savedSession, loadErr := session.LoadFile(savedPath)
	if loadErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load saved session for indexing (%v)\n", loadErr)
	} else {
		if indexErr := index.Append(root, index.FromSession(savedSession)); indexErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not update index (%v) — run `logos sync` to rebuild\n", indexErr)
		}
	}

	// Stage both the session file and the index with git add (best-effort).
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
