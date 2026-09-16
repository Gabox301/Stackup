package main

import (
	"github.com/spf13/cobra"

	"stackup/internal/apply"
	"stackup/internal/diff"
	"stackup/internal/generate"
	"stackup/internal/tui"
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
			// Write path: presence never gates the write. A qualifying run
			// shows the Plan screen first, then writes on quit with no
			// second print. Launch failure warns, then writes and prints
			// the plain summary as fallback.
			if shouldUseTUI(shared) {
				preview, err := diff.Compute(shared.path, plan, diff.Options{Force: force})
				if err != nil {
					return err
				}
				if _, err := newTUILauncher(cmd.InOrStdin(), cmd.ErrOrStderr()).Run(tui.Input{
					Screen:  tui.ScreenPlan,
					Plan:    plan,
					Pending: pendingOverwrites(preview),
				}); err != nil {
					warnTUIFallback(cmd, err)
				} else {
					res, err := apply.Apply(shared.path, plan, apply.Options{Force: force})
					if err != nil {
						return err
					}
					_ = res
					return nil
				}
			}
			res, err := apply.Apply(shared.path, plan, apply.Options{Force: force})
			if err != nil {
				return err
			}
			return printWritten(cmd, stacksJSON(evidences), res, shared.format)
		},
	}
	cmd.Flags().StringSliceVar(&ides, "ide", nil, "limit to IDEs (comma-separated subset of vscode,cursor,devin,kiro)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the unified diff and write nothing (exit 2 when changes are pending)")
	cmd.Flags().BoolVar(&force, "force", false, "generated values overwrite user scalars (default merges and keeps them)")
	return cmd
}
