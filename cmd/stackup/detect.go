package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Gabox301/Stackup/internal/detect"
	"github.com/Gabox301/Stackup/internal/tui"
)

func newDetectCommand(shared *sharedOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "detect",
		Short: "Detect stacks in a project directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requireFormat(shared.format); err != nil {
				return err
			}
			evidences, err := detect.Detect(shared.path, shared.allowUnknown)
			if err != nil {
				if errors.Is(err, detect.ErrUnknownStack) {
					return unknownStack(cmd, shared.format)
				}
				return err
			}
			if shared.format == "json" {
				return writeJSON(cmd.OutOrStdout(), detectPayload{Stacks: stacksJSON(evidences)})
			}
			// Compute-first, present-after: a qualifying TTY run replaces
			// the print with the read-only Evidence screen. Launch failure
			// warns on stderr and falls through to the byte-identical
			// fallback with the engine exit preserved.
			if shouldUseTUI(shared) {
				if _, err := newTUILauncher(cmd.InOrStdin(), cmd.ErrOrStderr()).Run(tui.Input{
					Screen:   tui.ScreenEvidence,
					Evidence: evidences,
				}); err != nil {
					warnTUIFallback(cmd, err)
				} else {
					return nil
				}
			}
			if len(evidences) == 0 {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), "no supported stacks detected (unknown allowed)"); err != nil {
					return err
				}
				return nil
			}
			for _, ev := range evidences {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), formatEvidence(ev)); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
