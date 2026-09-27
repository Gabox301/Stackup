package main

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Gabox301/Stackup/internal/apply"
	"github.com/Gabox301/Stackup/internal/diff"
	"github.com/Gabox301/Stackup/internal/generate"
	"github.com/Gabox301/Stackup/internal/tui"
)

func newApplyCommand(shared *sharedOpts) *cobra.Command {
	var ides []string
	var dryRun, force bool
	var backupDir, restoreFile string
	keep := -1
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Write IDE configs with backup (preview with --dry-run)",
		Args: func(cmd *cobra.Command, args []string) error {
			// The only positional apply takes is the --restore file
			// in space form (apply --restore <file>); pflag never
			// consumes a space-separated value for a flag that also
			// allows a bare form, so the bare-or-= forms arrive via
			// the flag itself and anything else is rejected.
			if cmd.Flags().Changed("restore") {
				if len(args) > 1 {
					return fmt.Errorf("apply: --restore takes at most one backup file, got %d", len(args))
				}
				return nil
			}
			return cobra.NoArgs(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireFormat(shared.format); err != nil {
				return err
			}
			// Fail fast with zero writes: a negative keep can never
			// prune, so reject it before any evidence or plan work.
			if cmd.Flags().Changed("keep") && keep < 0 {
				return fmt.Errorf("apply: --keep must be >= 0, got %d", keep)
			}
			evidences, err := resolveEvidence(cmd, shared)
			if err != nil {
				return err
			}
			plan, err := generate.Build(evidences, ides)
			if err != nil {
				return err
			}
			// Restore-only mode: revive backups without running apply.
			// --keep is a post-apply prune and does not apply here.
			if cmd.Flags().Changed("restore") {
				file := restoreFile
				if len(args) > 0 {
					if file != restoreNewest {
						return fmt.Errorf("apply: --restore takes one backup file, pass it once")
					}
					file = args[0]
				} else if file == restoreNewest {
					file = ""
				}
				return runRestore(cmd, shared, stacksJSON(evidences), plan, file, backupDir, dryRun, shared.format)
			}
			preview, err := diff.Compute(shared.path, plan, diff.Options{Force: force})
			if err != nil {
				return err
			}
			if dryRun {
				// Preview path: the Diff screen replaces the print on
				// qualifying runs. Launch failure warns and falls back to
				// byte-identical text with exit 2/0 preserved.
				if shouldUseTUI(shared) {
					if _, err := newTUILauncher(cmd.InOrStdin(), cmd.ErrOrStderr()).Run(tui.Input{
						Screen:  tui.ScreenDiff,
						Preview: preview,
					}); err != nil {
						warnTUIFallback(cmd, err)
					} else {
						if preview.HasChanges() {
							return &exitError{code: exitPreviewHasChanges}
						}
						return nil
					}
				}
				return printPreview(cmd, stacksJSON(evidences), preview, shared.format)
			}
			stacks := stacksJSON(evidences)
			if overwrites := pendingOverwrites(preview); len(overwrites) > 0 && !shared.yes {
				// Gate exclusion (piped stdout, piped stdin,
				// non-interactive, or json) blocks with exit 3 and zero
				// writes, keeping the wording byte-identical. Only a
				// qualifying TTY text run reaches the confirm screen.
				if !shouldUseTUI(shared) {
					if shared.format == "json" {
						if err := writeJSON(cmd.OutOrStdout(), blockedPayload{
							Stacks:  stacks,
							Pending: overwrites,
							Blocked: "non-interactive",
						}); err != nil {
							return err
						}
					} else {
						if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "apply blocked: %d file(s) would be overwritten; re-run with --yes to apply without prompting (no writes performed)\n", len(overwrites)); err != nil {
							return err
						}
					}
					return &exitError{code: exitBlockedNonInteractive}
				}
				act, err := newTUILauncher(cmd.InOrStdin(), cmd.ErrOrStderr()).Run(tui.Input{
					Screen:  tui.ScreenConfirm,
					Plan:    plan,
					Pending: overwrites,
				})
				if err != nil {
					// Confirm launch failure stays safe: warn, abort
					// with no writes, exit 0. The engine exit is never
					// coerced to 1.
					warnTUIFallback(cmd, err)
					if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "apply aborted (no writes performed)"); err != nil {
						return err
					}
					return nil
				}
				if act != tui.ActionApply {
					if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "apply aborted (no writes performed)"); err != nil {
						return err
					}
					return nil
				}
			}
			res, err := apply.Apply(shared.path, plan, apply.Options{Force: force, BackupDir: backupDir})
			if err != nil {
				// Mid-plan failure: report the applied[]/failed/pending[]
				// manifest through the same conventions as the write
				// summary (stderr text, stdout JSON) and keep exit 1 via
				// the plain error. Backups stay the recovery story.
				var applyErr *apply.ApplyError
				if errors.As(err, &applyErr) {
					if perr := printApplyFailure(cmd, stacks, applyErr, shared.format); perr != nil {
						return perr
					}
				}
				return err
			}
			// Post-apply retention: prune every planned target down to
			// the newest keep backups. Targets without backups are a
			// no-op; a negative keep was already rejected up front.
			if cmd.Flags().Changed("keep") {
				for _, f := range plan.Files {
					if _, err := apply.PruneBackups(shared.path, f.Path, keep, backupDir); err != nil {
						return err
					}
				}
			}
			// Always-present: a qualifying run replaces the summary print
			// with the read-only Result screen. Launch failure warns and
			// falls back to byte-identical text with exit 0 preserved.
			if shouldUseTUI(shared) {
				if _, err := newTUILauncher(cmd.InOrStdin(), cmd.ErrOrStderr()).Run(tui.Input{
					Screen:  tui.ScreenResult,
					Written: res,
				}); err != nil {
					warnTUIFallback(cmd, err)
				} else {
					return nil
				}
			}
			return printWritten(cmd, stacks, res, shared.format)
		},
	}
	cmd.Flags().StringSliceVar(&ides, "ide", nil, "limit to IDEs (comma-separated subset of vscode,cursor,devin,kiro)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the unified diff and write nothing (exit 2 when changes are pending)")
	cmd.Flags().BoolVar(&force, "force", false, "generated values overwrite user scalars (default merges and keeps them)")
	cmd.Flags().StringVar(&backupDir, "backup-dir", "", "write timestamped backups into <dir> instead of next to each file")
	cmd.Flags().IntVar(&keep, "keep", -1, "prune backups after apply, keeping N newest per target (0 prunes all)")
	cmd.Flags().StringVar(&restoreFile, "restore", "", "revive the newest backup per target, or the named backup file (writes nothing else)")
	// Bare --restore revives the newest backup per target; Changed()
	// tells it apart from the flag being absent entirely. The sentinel
	// must be non-empty: pflag ignores an empty NoOptDefVal and would
	// swallow the following flag as the value instead.
	cmd.Flags().Lookup("restore").NoOptDefVal = restoreNewest
	return cmd
}

// restoreNewest is the NoOptDefVal sentinel for a bare --restore.
const restoreNewest = "newest"

// applyFailedPayload carries the single write that aborted a run.
type applyFailedPayload struct {
	File  string `json:"file"`
	Error string `json:"error"`
}

// applyFailurePayload carries the mid-plan failure manifest. Backups
// ride along so the operator knows which pre-write images survived.
type applyFailurePayload struct {
	Stacks  []stackJSON        `json:"stacks"`
	Applied []string           `json:"applied"`
	Backups map[string]string  `json:"backups"`
	Failed  applyFailedPayload `json:"failed"`
	Pending []string           `json:"pending"`
}

// printApplyFailure reports a mid-plan apply abort. Text goes to stderr
// like the other apply notices (blocked, aborted) with one applied line
// per file in printWritten wording; JSON goes to stdout as a parseable
// manifest. The caller still returns the original error (exit 1).
func printApplyFailure(cmd *cobra.Command, stacks []stackJSON, applyErr *apply.ApplyError, format string) error {
	applied := applyErr.Applied
	if applied == nil {
		applied = []string{}
	}
	backups := applyErr.Backups
	if backups == nil {
		backups = map[string]string{}
	}
	pending := applyErr.Pending
	if pending == nil {
		pending = []string{}
	}
	cause := ""
	if applyErr.Cause != nil {
		cause = applyErr.Cause.Error()
	}
	if format == "json" {
		return writeJSON(cmd.OutOrStdout(), applyFailurePayload{
			Stacks:  stacks,
			Applied: applied,
			Backups: backups,
			Failed:  applyFailedPayload{File: applyErr.Failed, Error: cause},
			Pending: pending,
		})
	}
	errOut := cmd.ErrOrStderr()
	if _, err := fmt.Fprintf(errOut, "apply failed: %s: %s\n", applyErr.Failed, cause); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(errOut, "applied %d file(s):\n", len(applied)); err != nil {
		return err
	}
	for _, p := range applied {
		if bak, ok := backups[p]; ok {
			if _, err := fmt.Fprintf(errOut, "  %s (backup: %s)\n", p, bak); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(errOut, "  %s (new)\n", p); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintf(errOut, "pending %d file(s):\n", len(pending)); err != nil {
		return err
	}
	for _, p := range pending {
		if _, err := fmt.Fprintf(errOut, "  %s\n", p); err != nil {
			return err
		}
	}
	return nil
}

// restoreEntry pairs a revived target with the backup it came from.
type restoreEntry struct {
	Target string `json:"target"`
	Backup string `json:"backup"`
}

// restorePayload carries a restore-only run for --format json.
type restorePayload struct {
	Stacks   []stackJSON    `json:"stacks"`
	Restored []restoreEntry `json:"restored"`
}

// runRestore revives backups without running apply. A bare --restore
// (empty file) revives the newest backup per planned target discovered
// on disk via ListBackups; a named file revives exactly that backup
// after locating the planned target that lists it. DryRun previews the
// pairs without writing. Finding nothing is "no backups found" with
// exit 0, never an error.
func runRestore(cmd *cobra.Command, shared *sharedOpts, stacks []stackJSON, plan generate.Plan, file, backupDir string, dryRun bool, format string) error {
	targets := make([]string, 0, len(plan.Files))
	for _, f := range plan.Files {
		targets = append(targets, f.Path)
	}
	var pairs []restoreEntry
	if file == "" {
		for _, target := range targets {
			listed, err := apply.ListBackups(shared.path, target, backupDir)
			if err != nil {
				return err
			}
			if len(listed) == 0 {
				continue
			}
			pairs = append(pairs, restoreEntry{Target: target, Backup: listed[len(listed)-1]})
		}
	} else {
		rel := filepath.ToSlash(file)
		target, err := findBackupTarget(shared.path, targets, rel, backupDir)
		if err != nil {
			return err
		}
		if target == "" {
			return fmt.Errorf("apply: restore: %s is not a backup of any planned target", file)
		}
		pairs = append(pairs, restoreEntry{Target: target, Backup: rel})
	}
	if !dryRun {
		for _, p := range pairs {
			if err := apply.RestoreFile(shared.path, p.Target, p.Backup, backupDir); err != nil {
				return err
			}
		}
	}
	return printRestored(cmd, stacks, pairs, dryRun, format)
}

// findBackupTarget returns the planned target whose ListBackups set
// contains backup (a root-relative slash path), or "" when no planned
// target lists it. Matching reuses ListBackups, so the exact-pattern
// guard stays authoritative in the apply package.
func findBackupTarget(root string, targets []string, backup, backupDir string) (string, error) {
	for _, target := range targets {
		listed, err := apply.ListBackups(root, target, backupDir)
		if err != nil {
			return "", err
		}
		for _, b := range listed {
			if b == backup {
				return target, nil
			}
		}
	}
	return "", nil
}

// printRestored reports a restore-only run. DryRun previews the pairs
// without writing; an empty pair set prints "no backups found".
func printRestored(cmd *cobra.Command, stacks []stackJSON, pairs []restoreEntry, dryRun bool, format string) error {
	if pairs == nil {
		pairs = []restoreEntry{}
	}
	if format == "json" {
		return writeJSON(cmd.OutOrStdout(), restorePayload{Stacks: stacks, Restored: pairs})
	}
	if len(pairs) == 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "no backups found"); err != nil {
			return err
		}
		return nil
	}
	verb := "restored"
	if dryRun {
		verb = "would restore"
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %d file(s):\n", verb, len(pairs)); err != nil {
		return err
	}
	for _, p := range pairs {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s (from: %s)\n", p.Target, p.Backup); err != nil {
			return err
		}
	}
	return nil
}
