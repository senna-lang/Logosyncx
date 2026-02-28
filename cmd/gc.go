package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

// --- logos gc ----------------------------------------------------------------

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Archive stale session files to sessions/archive/",
	Long: `Scan all sessions and move stale ones to .logosyncx/sessions/archive/.

A session is a GC candidate when one of the following is true:

  Strong candidate (--linked-days, default 30):
    All tasks linked to the session are done or cancelled, AND at least
    linked-days have passed since the latest task completion (or since the
    session was created when completed_at is not recorded).

  Weak candidate (--orphan-days, default 90):
    The session has no linked tasks and is older than orphan-days.

Sessions with at least one linked task still open or in_progress are
protected and will never be selected.

Use --dry-run to preview candidates without moving any files.
Run "logos gc purge" to permanently delete all archived sessions.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		linkedDays, _ := cmd.Flags().GetInt("linked-days")
		orphanDays, _ := cmd.Flags().GetInt("orphan-days")
		linkedChanged := cmd.Flags().Changed("linked-days")
		orphanChanged := cmd.Flags().Changed("orphan-days")
		return runGC(dryRun, linkedDays, orphanDays, linkedChanged, orphanChanged)
	},
}

var gcPurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Permanently delete all sessions in sessions/archive/",
	Long: `Delete every session file stored in .logosyncx/sessions/archive/.

This is irreversible. Use --dry-run on "logos gc" first to inspect what
was archived before running this command.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		return runGCPurge(force)
	},
}

func init() {
	gcCmd.Flags().Bool("dry-run", false, "Preview candidates without moving any files")
	gcCmd.Flags().Int("linked-days", 0, "Days since task completion before a linked session is archived (default from config: 30)")
	gcCmd.Flags().Int("orphan-days", 0, "Days since creation before a session with no linked tasks is archived (default from config: 90)")

	gcPurgeCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	gcCmd.AddCommand(gcPurgeCmd)
	rootCmd.AddCommand(gcCmd)
}

// --- GC candidate types ------------------------------------------------------

// gcTier describes how strongly a session qualifies for archival.
type gcTier int

const (
	gcTierStrong gcTier = 1 // all linked tasks done/cancelled
	gcTierWeak   gcTier = 2 // no linked tasks
)

// gcCandidate holds a session and the reason it was selected.
type gcCandidate struct {
	sess    session.Session
	reason  string
	ageDays int
	tier    gcTier
}

// --- core logic --------------------------------------------------------------

func runGC(dryRun bool, linkedDays, orphanDays int, linkedChanged, orphanChanged bool) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Fall back to config values when flags were not explicitly set.
	if !linkedChanged {
		linkedDays = cfg.GC.LinkedTaskDoneDays
	}
	if !orphanChanged {
		orphanDays = cfg.GC.OrphanSessionDays
	}

	candidates, err := findGCCandidates(root, &cfg, linkedDays, orphanDays)
	if err != nil {
		return err
	}

	if len(candidates) == 0 {
		fmt.Println("No sessions eligible for archival.")
		return nil
	}

	if dryRun {
		printGCCandidates(candidates, linkedDays, orphanDays)
		fmt.Printf("\n%d session(s) would be archived. Run without --dry-run to proceed.\n", len(candidates))
		return nil
	}

	// Archive each candidate.
	archived := 0
	for _, c := range candidates {
		dst, err := session.Archive(root, c.sess.Filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not archive %s: %v\n", c.sess.Filename, err)
			continue
		}

		// git: remove old path, stage new path (best-effort).
		if cfg.Git.AutoPush {
			oldPath := filepath.Join(session.SessionsDir(root), c.sess.Filename)
			_ = gitutil.Remove(root, oldPath)
			_ = gitutil.Add(root, dst)
		}

		fmt.Printf("  → archived %s\n", c.sess.Filename)
		archived++
	}

	if archived == 0 {
		return fmt.Errorf("all archive operations failed — check warnings above")
	}

	// Rebuild session index so archived sessions no longer appear in logos ls.
	n, err := index.Rebuild(root, cfg.Sessions.ExcerptSection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: session index rebuild: %v\n", err)
	}
	if cfg.Git.AutoPush {
		_ = gitutil.Add(root, index.FilePath(root))
	}

	fmt.Printf("✓ Archived %d session(s). Session index rebuilt (%d active sessions).\n", archived, n)
	fmt.Println("  Run `logos gc purge --force` to permanently delete archived sessions.")
	return nil
}

func runGCPurge(force bool) error {
	root, err := project.FindRoot()
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	archived, err := session.LoadArchived(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
	if len(archived) == 0 {
		fmt.Println("No archived sessions to purge.")
		return nil
	}

	fmt.Printf("This will permanently delete %d archived session(s):\n", len(archived))
	for _, s := range archived {
		fmt.Printf("  - %s\n", s.Filename)
	}

	if !force {
		fmt.Print("\nConfirm permanent deletion? [y/N]: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	archiveDir := session.ArchiveDir(root)
	count := 0
	for _, s := range archived {
		path := filepath.Join(archiveDir, s.Filename)
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "warning: could not delete %s: %v\n", s.Filename, err)
			continue
		}
		if cfg.Git.AutoPush {
			_ = gitutil.Remove(root, path)
		}
		count++
	}

	fmt.Printf("✓ Permanently deleted %d archived session(s).\n", count)
	return nil
}

// findGCCandidates loads all active sessions and evaluates each one against
// the GC criteria, returning the list of sessions eligible for archival.
func findGCCandidates(root string, cfg *config.Config, linkedDays, orphanDays int) ([]gcCandidate, error) {
	sessions, err := session.LoadAllWithOptions(root, session.ParseOptions{
		ExcerptSection: cfg.Sessions.ExcerptSection,
	})
	if err != nil {
		// Non-fatal: LoadAllWithOptions returns partial results on parse errors.
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	store := task.NewStore(root, cfg)
	now := time.Now()
	var candidates []gcCandidate

	for _, s := range sessions {
		if len(s.Tasks) == 0 {
			// Weak candidate: no linked tasks — age-based.
			if s.Date == nil {
				continue
			}
			days := int(now.Sub(*s.Date).Hours() / 24)
			if days >= orphanDays {
				candidates = append(candidates, gcCandidate{
					sess:    s,
					reason:  fmt.Sprintf("no linked tasks, %d days old", days),
					ageDays: days,
					tier:    gcTierWeak,
				})
			}
			continue
		}

		// Has linked tasks — determine if all are terminal (done/cancelled).
		allTerminal := true
		hasActive := false
		var latestCompletion *time.Time

		for _, taskFilename := range s.Tasks {
			t, err := store.Get(taskFilename)
			if err != nil {
				// Task file not found or ambiguous: treat as terminal so it
				// does not block archival of the session.
				continue
			}
			switch t.Status {
			case task.StatusOpen, task.StatusInProgress:
				hasActive = true
				allTerminal = false
			}
			if t.CompletedAt != nil {
				if latestCompletion == nil || t.CompletedAt.After(*latestCompletion) {
					latest := *t.CompletedAt
					latestCompletion = &latest
				}
			}
		}

		if hasActive {
			// Protected: at least one task still active.
			continue
		}

		if !allTerminal {
			continue
		}

		// Strong candidate: all tasks terminal — use completion time or session date.
		var refTime time.Time
		var reasonSuffix string
		if latestCompletion != nil {
			refTime = *latestCompletion
			reasonSuffix = fmt.Sprintf("%d days since last task completed", int(now.Sub(refTime).Hours()/24))
		} else if s.Date != nil {
			refTime = *s.Date
			reasonSuffix = fmt.Sprintf("%d days old (no completed_at recorded)", int(now.Sub(refTime).Hours()/24))
		} else {
			continue
		}

		days := int(now.Sub(refTime).Hours() / 24)
		if days >= linkedDays {
			candidates = append(candidates, gcCandidate{
				sess:    s,
				reason:  fmt.Sprintf("all linked tasks done/cancelled, %s", reasonSuffix),
				ageDays: days,
				tier:    gcTierStrong,
			})
		}
	}

	return candidates, nil
}

// printGCCandidates renders a human-readable preview of the GC candidates.
func printGCCandidates(candidates []gcCandidate, linkedDays, orphanDays int) {
	strong := 0
	weak := 0
	for _, c := range candidates {
		if c.tier == gcTierStrong {
			strong++
		} else {
			weak++
		}
	}

	fmt.Printf("GC candidates (%d session(s)):\n", len(candidates))
	fmt.Printf("  Thresholds: linked-days=%d, orphan-days=%d\n\n", linkedDays, orphanDays)

	for _, c := range candidates {
		tier := "strong"
		if c.tier == gcTierWeak {
			tier = "weak"
		}
		fmt.Printf("  [%s] %s\n", tier, c.sess.Filename)
		fmt.Printf("        Reason : %s\n", c.reason)
		if len(c.sess.Tasks) > 0 {
			fmt.Printf("        Tasks  : %s\n", strings.Join(c.sess.Tasks, ", "))
		}
		fmt.Println()
	}

	if strong > 0 {
		fmt.Printf("  %d strong candidate(s): linked tasks all done/cancelled\n", strong)
	}
	if weak > 0 {
		fmt.Printf("  %d weak candidate(s):   no linked tasks, aged out\n", weak)
	}
}
