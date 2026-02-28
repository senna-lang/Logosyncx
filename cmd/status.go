package cmd

import (
	"fmt"
	"strings"

	"github.com/senna-lang/logosyncx/internal/gitutil"
	"github.com/senna-lang/logosyncx/internal/project"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show uncommitted sessions and tasks in .logosyncx/",
	Long: `Display the git status of every file under .logosyncx/, grouped by
staging state:

  Staged      — added to the index, ready to commit
  Unstaged    — modified in the worktree but not yet staged
  Untracked   — new files not yet added to git

This command is informational and never modifies any file or git state.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus() error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}

	entries, err := gitutil.StatusUnderDir(root, ".logosyncx/")
	if err != nil {
		return fmt.Errorf("query git status: %w", err)
	}

	// Partition entries into three buckets.
	var staged, unstaged, untracked []gitutil.FileStatus

	for _, e := range entries {
		switch {
		case e.Staging == gitutil.StatusUntracked && e.Worktree == gitutil.StatusUntracked:
			untracked = append(untracked, e)
		case e.Staging != gitutil.StatusUnmodified && e.Staging != gitutil.StatusUntracked:
			// Something in the index (added, modified, deleted, renamed…).
			staged = append(staged, e)
		case e.Worktree != gitutil.StatusUnmodified && e.Worktree != gitutil.StatusUntracked:
			// Worktree change that hasn't been staged yet.
			unstaged = append(unstaged, e)
		}
	}

	if len(staged) == 0 && len(unstaged) == 0 && len(untracked) == 0 {
		fmt.Println("✓ Nothing uncommitted in .logosyncx/ — all saved and committed.")
		return nil
	}

	if len(staged) > 0 {
		fmt.Printf("Staged (ready to commit):\n")
		for _, e := range staged {
			label := statusLabel(e.Staging)
			fmt.Printf("  %-12s %s\n", "("+label+")", trimPrefix(e.Path))
		}
		fmt.Println()
	}

	if len(unstaged) > 0 {
		fmt.Printf("Unstaged changes:\n")
		for _, e := range unstaged {
			label := statusLabel(e.Worktree)
			fmt.Printf("  %-12s %s\n", "("+label+")", trimPrefix(e.Path))
		}
		fmt.Println()
	}

	if len(untracked) > 0 {
		fmt.Printf("Untracked (not staged):\n")
		for _, e := range untracked {
			fmt.Printf("  %-12s %s\n", "(new)", trimPrefix(e.Path))
		}
		fmt.Println()
	}

	fmt.Println("Run `git add .logosyncx/ && git commit` to commit the above.")
	return nil
}

// statusLabel returns a short human-readable label for a git status code.
func statusLabel(sc gitutil.StatusCode) string {
	switch sc {
	case gitutil.StatusAdded:
		return "added"
	case gitutil.StatusModified:
		return "modified"
	case gitutil.StatusDeleted:
		return "deleted"
	case gitutil.StatusRenamed:
		return "renamed"
	default:
		return strings.ToLower(string(rune(sc)))
	}
}

// trimPrefix strips the leading ".logosyncx/" from a path for cleaner display.
func trimPrefix(path string) string {
	return strings.TrimPrefix(path, ".logosyncx/")
}
