package cmd

import (
	"fmt"
	"os"

	"github.com/senna-lang/logosyncx/internal/gitutil"
	"github.com/senna-lang/logosyncx/internal/project"
	"github.com/senna-lang/logosyncx/internal/task"
	"github.com/senna-lang/logosyncx/pkg/config"
	"github.com/senna-lang/logosyncx/pkg/index"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Rebuild session and task indexes from the filesystem",
	Long: `Delete and rebuild index.jsonl and task-index.jsonl by scanning every
file under .logosyncx/sessions/ and .logosyncx/tasks/ respectively.
Run this after manually editing, adding, or deleting session or task files
to bring both indexes back in sync with the filesystem.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSync()
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync() error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}

	// --- sessions ------------------------------------------------------------
	fmt.Println("Rebuilding session index from sessions/...")
	n, err := index.Rebuild(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
	fmt.Printf("Done. %d sessions indexed.\n", n)

	sessionIndexPath := index.FilePath(root)
	if gitErr := gitutil.Add(root, sessionIndexPath); gitErr != nil {
		fmt.Fprintf(os.Stderr, "warning: git add failed for session index (%v) — stage the file manually\n", gitErr)
	}

	// --- tasks ---------------------------------------------------------------
	fmt.Println("\nRebuilding task index from tasks/...")
	cfg, err := config.Load(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load config (%v) — using defaults\n", err)
		cfg = config.Config{}
	}
	store := task.NewStore(root, &cfg)
	m, err := store.RebuildTaskIndex()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
	fmt.Printf("Done. %d tasks indexed.\n", m)

	taskIndexPath := task.TaskIndexFilePath(root)
	if gitErr := gitutil.Add(root, taskIndexPath); gitErr != nil {
		fmt.Fprintf(os.Stderr, "warning: git add failed for task index (%v) — stage the file manually\n", gitErr)
	}

	return nil
}
