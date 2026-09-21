package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Gabox301/Stackup/internal/apply"
	"github.com/Gabox301/Stackup/internal/diff"
	"github.com/Gabox301/Stackup/internal/generate"
	"github.com/Gabox301/Stackup/internal/tui"
)

func newGenerateCommand(shared *sharedOpts) *cobra.Command {
	var ides []string
	var dryRun, force bool
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate IDE configs (preview with --dry-run)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requireFormat(shared.format); err != nil {
				return err
			}
			evidences, err := resolveEvidence(cmd, shared)
			if err != nil {
				return err
			}
			plan, err := generate.Build(evidences, ides)
			if err != nil {
				return err
			}
			if dryRun {
				preview, err := diff.Compute(shared.path, plan, diff.Options{Force: force})
				if err != nil {
					return err
				}
				// Compute-first, present-after: the Plan screen shows the
				// file list with (new)/(overwrite) wording, then quit maps
				// to exit 2/0 without printing. Launch failure warns and
				// falls back to byte-identical preview text.
				if shouldUseTUI(shared) {
					if _, err := newTUILauncher(cmd.InOrStdin(), cmd.ErrOrStderr()).Run(tui.Input{
						Screen:  tui.ScreenPlan,
						Plan:    plan,
						Pending: pendingOverwrites(preview),
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
			// Write path: compute-first, gate-after. Pending overwrites
			// without --yes block with exit 3 off-gate (zero writes); a
			// qualifying run shows the Plan gate (explicit y writes,
			// n/q/esc aborts with zero writes), then the Result screen
			// replaces the summary print. Launch failures warn: a gated
			// Plan failure aborts safely, a fresh Plan failure writes and
			// prints, a Result failure prints byte-identical text.
			preview, err := diff.Compute(shared.path, plan, diff.Options{Force: force})
			if err != nil {
				return err
			}
			stacks := stacksJSON(evidences)
			overwrites := pendingOverwrites(preview)
			if len(overwrites) > 0 && !shared.yes && !shouldUseTUI(shared) {
				// Gate exclusion (piped stdout, piped stdin,
				// non-interactive, or json) blocks with exit 3 and zero
				// writes, keeping the apply wording with our verb. Only
				// a qualifying TTY text run reaches the Plan gate.
				if shared.format == "json" {
					if err := writeJSON(cmd.OutOrStdout(), blockedPayload{
						Stacks:  stacks,
						Pending: overwrites,
						Blocked: "non-interactive",
					}); err != nil {
						return err
					}
				} else {
					if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "generate blocked: %d file(s) would be overwritten; re-run with --yes to apply without prompting (no writes performed)\n", len(overwrites)); err != nil {
						return err
					}
				}
				return &exitError{code: exitBlockedNonInteractive}
			}
			planOK := true
			if shouldUseTUI(shared) {
				act, err := newTUILauncher(cmd.InOrStdin(), cmd.ErrOrStderr()).Run(tui.Input{
					Screen:  tui.ScreenPlan,
					Plan:    plan,
					Pending: overwrites,
				})
				if err != nil {
					// Plan launch failure stays safe when gated: warn,
					// abort with no writes, exit 0. A fresh run has no
					// consent to lose, so it writes and prints below.
					warnTUIFallback(cmd, err)
					if len(overwrites) > 0 && !shared.yes {
						if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "generate aborted (no writes performed)"); err != nil {
							return err
						}
						return nil
					}
					planOK = false
				} else if act != tui.ActionApply {
					if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "generate aborted (no writes performed)"); err != nil {
						return err
					}
					return nil
				}
			}
			res, err := apply.Apply(shared.path, plan, apply.Options{Force: force})
			if err != nil {
				return err
			}
			// Always-present: a qualifying Plan run replaces the summary
			// print with the read-only Result screen. Launch failure warns
			// and falls back to byte-identical text with exit 0 preserved.
			if shouldUseTUI(shared) && planOK {
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
	return cmd
}
