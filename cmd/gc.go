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
	"github.com/senna-lang/logosyncx/pkg/plan"
	"github.com/spf13/cobra"
)

// --- logos gc ----------------------------------------------------------------

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Archive stale plan files to plans/archive/",
	Long: `Scan all plans and move stale ones to .logosyncx/plans/archive/.

A plan is a GC candidate when one of the following is true:

  Strong candidate (--linked-days, default 30):
    plan.distilled == true, all tasks done, and at least linked-days have
    passed since the latest task completion (or the plan creation date when
    completed_at is not recorded).

  Weak candidate (--orphan-days, default 90):
    The plan has no tasks and is older than orphan-days.

Plans with at least one linked task still open or in_progress are
protected and will never be selected.

Use --dry-run to preview candidates without moving any files.
Run "logos gc purge" to permanently delete all archived plans.`,
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
	Short: "Permanently delete all plans in plans/archive/",
	Long: `Delete every plan file stored in .logosyncx/plans/archive/.

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
	gcCmd.Flags().Int("linked-days", 0, "Days since task completion before a distilled plan is archived (default from config: 30)")
	gcCmd.Flags().Int("orphan-days", 0, "Days since creation before a plan with no tasks is archived (default from config: 90)")

	gcPurgeCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	gcCmd.AddCommand(gcPurgeCmd)
	rootCmd.AddCommand(gcCmd)
}

// --- GC candidate types ------------------------------------------------------

// gcTier describes how strongly a plan qualifies for archival.
type gcTier int

const (
	gcTierStrong gcTier = 1 // distilled + all tasks done
	gcTierWeak   gcTier = 2 // no linked tasks
)

// gcCandidate holds a plan and the reason it was selected.
type gcCandidate struct {
	p       plan.Plan
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
		orphanDays = cfg.GC.OrphanPlanDays
	}

	candidates, err := findGCCandidates(root, &cfg, linkedDays, orphanDays)
	if err != nil {
		return err
	}

	if len(candidates) == 0 {
		fmt.Println("No plans eligible for archival.")
		return nil
	}

	if dryRun {
		printGCCandidates(candidates, linkedDays, orphanDays)
		fmt.Printf("\n%d plan(s) would be archived. Run without --dry-run to proceed.\n", len(candidates))
		return nil
	}

	// Archive each candidate.
	archived := 0
	for _, c := range candidates {
		dst, err := plan.Archive(root, c.p.Filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not archive %s: %v\n", c.p.Filename, err)
			continue
		}

		// git: remove old path, stage new path (best-effort).
		if cfg.Git.AutoPush {
			oldPath := filepath.Join(plan.PlansDir(root), c.p.Filename)
			_ = gitutil.Remove(root, oldPath)
			_ = gitutil.Add(root, dst)
		}

		fmt.Printf("  → archived %s\n", c.p.Filename)
		archived++
	}

	if archived == 0 {
		return fmt.Errorf("all archive operations failed — check warnings above")
	}

	// Rebuild plan index so archived plans no longer appear in logos ls.
	n, err := index.Rebuild(root, cfg.Plans.ExcerptSection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: plan index rebuild: %v\n", err)
	}
	if cfg.Git.AutoPush {
		_ = gitutil.Add(root, index.FilePath(root))
	}

	fmt.Printf("✓ Archived %d plan(s). Plan index rebuilt (%d active plans).\n", archived, n)
	fmt.Println("  Run `logos gc purge --force` to permanently delete archived plans.")
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

	archivedFiles, err := loadArchivedPlanFilenames(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
	if len(archivedFiles) == 0 {
		fmt.Println("No archived plans to purge.")
		return nil
	}

	fmt.Printf("This will permanently delete %d archived plan(s):\n", len(archivedFiles))
	for _, f := range archivedFiles {
		fmt.Printf("  - %s\n", f)
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

	archiveDir := plan.ArchiveDir(root)
	count := 0
	for _, f := range archivedFiles {
		path := filepath.Join(archiveDir, f)
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "warning: could not delete %s: %v\n", f, err)
			continue
		}
		if cfg.Git.AutoPush {
			_ = gitutil.Remove(root, path)
		}
		count++
	}

	fmt.Printf("✓ Permanently deleted %d archived plan(s).\n", count)
	return nil
}

// loadArchivedPlanFilenames returns the filenames of all .md files in plans/archive/.
func loadArchivedPlanFilenames(root string) ([]string, error) {
	archiveDir := plan.ArchiveDir(root)
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read archive dir: %w", err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			files = append(files, e.Name())
		}
	}
	return files, nil
}

// findGCCandidates loads all active plans and evaluates each one against
// the GC criteria, returning the list of plans eligible for archival.
func findGCCandidates(root string, cfg *config.Config, linkedDays, orphanDays int) ([]gcCandidate, error) {
	plans, err := plan.LoadAll(root)
	if err != nil {
		// Non-fatal: LoadAll returns partial results on parse errors.
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}

	store := task.NewStore(root, cfg)
	now := time.Now()
	var candidates []gcCandidate

	for _, p := range plans {
		planSlug := strings.TrimSuffix(p.Filename, ".md")

		tasks, _ := store.List(task.Filter{Plan: planSlug})

		if len(tasks) == 0 {
			// Weak candidate: no linked tasks — age-based.
			if p.Date == nil {
				continue
			}
			days := int(now.Sub(*p.Date).Hours() / 24)
			if days >= orphanDays {
				candidates = append(candidates, gcCandidate{
					p:       p,
					reason:  fmt.Sprintf("no linked tasks, %d days old", days),
					ageDays: days,
					tier:    gcTierWeak,
				})
			}
			continue
		}

		// Has linked tasks — check for active (protected) tasks.
		hasActive := false
		allDone := true
		var latestCompletion *time.Time

		for _, t := range tasks {
			switch t.Status {
			case task.StatusOpen, task.StatusInProgress:
				hasActive = true
				allDone = false
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

		if !allDone {
			continue
		}

		// Only distilled plans qualify as strong candidates.
		if !p.Distilled {
			continue
		}

		// Strong candidate: distilled + all tasks done.
		var refTime time.Time
		var reasonSuffix string
		if latestCompletion != nil {
			refTime = *latestCompletion
			reasonSuffix = fmt.Sprintf("%d days since last task completed", int(now.Sub(refTime).Hours()/24))
		} else if p.Date != nil {
			refTime = *p.Date
			reasonSuffix = fmt.Sprintf("%d days old (no completed_at recorded)", int(now.Sub(refTime).Hours()/24))
		} else {
			continue
		}

		days := int(now.Sub(refTime).Hours() / 24)
		if days >= linkedDays {
			candidates = append(candidates, gcCandidate{
				p:       p,
				reason:  fmt.Sprintf("distilled, all tasks done, %s", reasonSuffix),
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

	fmt.Printf("GC candidates (%d plan(s)):\n", len(candidates))
	fmt.Printf("  Thresholds: linked-days=%d, orphan-days=%d\n\n", linkedDays, orphanDays)

	for _, c := range candidates {
		tier := "strong"
		if c.tier == gcTierWeak {
			tier = "weak"
		}
		fmt.Printf("  [%s] %s\n", tier, c.p.Filename)
		fmt.Printf("        Reason : %s\n", c.reason)
		fmt.Println()
	}

	if strong > 0 {
		fmt.Printf("  %d strong candidate(s): distilled + all tasks done\n", strong)
	}
	if weak > 0 {
		fmt.Printf("  %d weak candidate(s):   no linked tasks, aged out\n", weak)
	}
}
