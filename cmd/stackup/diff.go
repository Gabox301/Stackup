package main

import (
	"github.com/spf13/cobra"

	"github.com/Gabox301/Stackup/internal/diff"
	"github.com/Gabox301/Stackup/internal/generate"
	"github.com/Gabox301/Stackup/internal/tui"
)

func newDiffCommand(shared *sharedOpts) *cobra.Command {
	var ides []string
	var force bool
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Preview pending config changes without writing",
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
			// Compute-first, present-after: a qualifying TTY run replaces
			// the print with the Diff screen. Launch failure warns and
			// falls back to byte-identical text with exit 2/0 preserved.
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
		},
	}
	cmd.Flags().StringSliceVar(&ides, "ide", nil, "limit to IDEs (comma-separated subset of vscode,cursor,devin,kiro)")
	cmd.Flags().BoolVar(&force, "force", false, "preview the overwrite path (generated values win) instead of the default merge")
	return cmd
}
