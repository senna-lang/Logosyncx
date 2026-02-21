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
	"github.com/senna-lang/logosyncx/pkg/session"
	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save a session file to .logosyncx/sessions/",
	Long: `Accept a Markdown session file via --file or --stdin, auto-fill missing
frontmatter fields (id, date), and save it to .logosyncx/sessions/.
git add is run automatically; git commit and push remain the user's responsibility.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, _ := cmd.Flags().GetString("file")
		useStdin, _ := cmd.Flags().GetBool("stdin")
		return runSave(filePath, useStdin)
	},
}

func init() {
	saveCmd.Flags().StringP("file", "f", "", "Path to the session Markdown file to save")
	saveCmd.Flags().Bool("stdin", false, "Read session Markdown from stdin")
	rootCmd.AddCommand(saveCmd)
}

func runSave(filePath string, useStdin bool) error {
	// Validate flags: exactly one source must be provided.
	if filePath == "" && !useStdin {
		return errors.New("provide --file <path> or --stdin")
	}
	if filePath != "" && useStdin {
		return errors.New("--file and --stdin are mutually exclusive")
	}

	// Read raw markdown from the chosen source.
	var data []byte
	var err error
	if useStdin {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
	} else {
		data, err = os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read file %s: %w", filePath, err)
		}
	}

	// Parse the session from the raw markdown.
	s, err := session.Parse("input", data)
	if err != nil {
		return fmt.Errorf("parse session: %w", err)
	}

	// Auto-fill missing frontmatter fields.
	if s.ID == "" {
		s.ID, err = generateID()
		if err != nil {
			return fmt.Errorf("generate id: %w", err)
		}
	}
	if s.Date.IsZero() {
		s.Date = time.Now()
	}

	// Warn (but do not block) if topic is missing.
	if strings.TrimSpace(s.Topic) == "" {
		fmt.Fprintln(os.Stderr, "warning: frontmatter 'topic' is empty — filename will use 'untitled'")
		s.Topic = "untitled"
	}

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

	// Stage the file with git add (best-effort).
	if err := gitutil.Add(root, savedPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: git add failed (%v) — stage the file manually\n", err)
	} else {
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
