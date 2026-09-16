package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"stackup/internal/apply"
	"stackup/internal/diff"
	"stackup/internal/generate"
	"stackup/internal/tui"
)

func newApplyCommand(shared *sharedOpts) *cobra.Command {
	var ides []string
	var dryRun, force bool
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Write IDE configs with backup (preview with --dry-run)",
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
			res, err := apply.Apply(shared.path, plan, apply.Options{Force: force})
			if err != nil {
				return err
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
	return cmd
}
