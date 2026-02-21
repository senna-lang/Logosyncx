package cmd

import (
	"fmt"
	"os"

	"github.com/senna-lang/logosyncx/internal/gitutil"
	"github.com/senna-lang/logosyncx/internal/project"
	"github.com/senna-lang/logosyncx/pkg/index"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Rebuild the session index from sessions/",
	Long: `Delete and rebuild index.jsonl by scanning every session file under
.logosyncx/sessions/. Run this after manually editing or deleting session
files to bring the index back in sync with the filesystem.`,
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

	fmt.Println("Rebuilding index from sessions/...")

	n, err := index.Rebuild(root)
	if err != nil {
		// Non-fatal parse warnings: print but continue.
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	fmt.Printf("Done. %d sessions indexed.\n", n)

	// Stage the rebuilt index file with git add (best-effort).
	indexPath := index.FilePath(root)
	if gitErr := gitutil.Add(root, indexPath); gitErr != nil {
		fmt.Fprintf(os.Stderr, "warning: git add failed (%v) — stage the file manually\n", gitErr)
	}

	return nil
}
