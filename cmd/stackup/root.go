package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Gabox301/Stackup/internal/apply"
	"github.com/Gabox301/Stackup/internal/detect"
	"github.com/Gabox301/Stackup/internal/diff"
	"github.com/Gabox301/Stackup/internal/tui"
)

// Exit codes for the stackup CLI tree.
const (
	exitPreviewHasChanges     = 2
	exitBlockedNonInteractive = 3
	exitUnknownStack          = 4
)

// exitError carries a mapped process exit status (2, 3, or 4) through
// Cobra. Commands print any user-facing output before returning it, so
// main exits quietly with the code. Any other error means exit 1.
type exitError struct{ code int }

func (e *exitError) Error() string { return fmt.Sprintf("exit %d", e.code) }

// sharedOpts binds the persistent flags every subcommand inherits.
type sharedOpts struct {
	path           string
	format         string
	yes            bool
	nonInteractive bool
	allowUnknown   bool
}

// stdinIsTTY reports whether stdin is an interactive terminal. It is a
// variable so tests can stub non-TTY behavior without real pipes.
var stdinIsTTY = func() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// stdoutIsTTY reports whether stdout is an interactive terminal. It is a
// variable so tests can stub piped output (e.g. `diff | head`) without
// real pipes. Presence requires both streams to be TTYs so a key-driven
// screen never swallows a pipe in either direction.
var stdoutIsTTY = func() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// newTUILauncher builds the interactive TUI screen. It is a variable so
// tests can swap the Bubble Tea program for a stub and prove the TUI
// boundary keeps core behavior unchanged. Pending lists travel in the
// Run Input, not on the constructor.
var newTUILauncher = func(in io.Reader, out io.Writer) tui.Launcher {
	return tui.NewTeaLauncher(in, out)
}

// shouldUseTUI reports whether the command must present instead of
// printing. JSON never presents, even on a TTY, so machine output stays
// parseable and byte-identical.
func shouldUseTUI(s *sharedOpts) bool {
	return stdoutIsTTY() && stdinIsTTY() && !s.yes && !s.nonInteractive && s.format == "text"
}

// warnTUIFallback tells stderr the screen failed and plain output
// follows. Callers then run today's print path byte-identical with the
// engine exit preserved, never coerced to 1.
func warnTUIFallback(cmd *cobra.Command, err error) {
	_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "warning: interactive display unavailable, showing plain output: %v\n", err)
}

func newRootCommand() *cobra.Command {
	shared := &sharedOpts{}
	root := &cobra.Command{
		Use:   "stackup",
		Short: "Detect project stacks and generate IDE configs",
		Long: `Stackup detects project stacks and generates IDE configs with preview-before-write.

Exit codes: 0 ok/no-change, 2 preview-has-changes, 3 blocked-non-interactive, 4 unknown-stack, 1 error.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       versionString(),
		PersistentPreRunE: func(*cobra.Command, []string) error {
			return requireValidPath(shared.path)
		},
	}
	root.PersistentFlags().StringVar(&shared.path, "path", ".", "project directory to scan")
	root.PersistentFlags().StringVar(&shared.format, "format", "text", "output format (text|json)")
	root.PersistentFlags().BoolVar(&shared.yes, "yes", false, "assume yes for overwrite prompts (never prompt)")
	root.PersistentFlags().BoolVar(&shared.nonInteractive, "non-interactive", false, "never prompt; abort gated writes instead of asking")
	root.PersistentFlags().BoolVar(&shared.allowUnknown, "allow-unknown", false, "proceed with empty results instead of failing")
	root.AddCommand(
		newDetectCommand(shared),
		newGenerateCommand(shared),
		newDiffCommand(shared),
		newApplyCommand(shared),
	)
	return root
}

// requireFormat rejects anything but the two supported output formats.
func requireFormat(format string) error {
	if format != "text" && format != "json" {
		return fmt.Errorf("invalid --format %q (want text|json)", format)
	}
	return nil
}

// requireValidPath stats the scan root upfront so a typo'd --path fails
// with exit 1 (plain error naming the bad path) before detection can
// misreport it as exit 4 unknown-stack. Existing directories pass
// through unchanged, preserving --allow-unknown for empty-but-real dirs.
func requireValidPath(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("invalid --path %q: %w", path, err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("invalid --path %q: not a directory", path)
	}
	return nil
}

// resolveEvidence runs stack detection and prints the unknown-stack
// notice, mapping it to exit 4. Generation stops in that case.
func resolveEvidence(cmd *cobra.Command, shared *sharedOpts) ([]detect.Evidence, error) {
	if err := requireValidPath(shared.path); err != nil {
		return nil, err
	}
	evidences, err := detect.Detect(shared.path, shared.allowUnknown)
	if err != nil {
		if errors.Is(err, detect.ErrUnknownStack) {
			return nil, unknownStack(cmd, shared.format)
		}
		return nil, err
	}
	return evidences, nil
}

// unknownStack prints the unknown-stack notice and maps it to exit 4.
// JSON output stays parseable so CI can consume it.
func unknownStack(cmd *cobra.Command, format string) error {
	if format == "json" {
		if err := writeJSON(cmd.OutOrStdout(), detectPayload{Stacks: []stackJSON{}}); err != nil {
			return err
		}
		return &exitError{code: exitUnknownStack}
	}
	if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "no supported stacks detected (use --allow-unknown to proceed)"); err != nil {
		return err
	}
	return &exitError{code: exitUnknownStack}
}

// printPreview renders a diff preview and maps it to its exit status:
// exit 2 when files differ, exit 0 when the tree is clean. It never writes.
func printPreview(cmd *cobra.Command, stacks []stackJSON, preview diff.Result, format string) error {
	out := cmd.OutOrStdout()
	if format == "json" {
		files := make([]filePayload, 0, len(preview.Files))
		for _, f := range preview.Files {
			files = append(files, filePayload{
				Path:    f.Path,
				IsNew:   f.IsNew,
				Changed: f.Changed,
				Unified: f.Unified,
			})
		}
		if err := writeJSON(out, previewPayload{Stacks: stacks, Files: files, HasChanges: preview.HasChanges()}); err != nil {
			return err
		}
	} else {
		changed := false
		for _, f := range preview.Files {
			if f.Changed {
				if _, err := fmt.Fprint(out, f.Unified); err != nil {
					return err
				}
				changed = true
			}
		}
		if !changed {
			if _, err := fmt.Fprintln(out, "no changes"); err != nil {
				return err
			}
		}
	}
	if preview.HasChanges() {
		return &exitError{code: exitPreviewHasChanges}
	}
	return nil
}

// pendingOverwrites lists planned files that would change an existing
// file on disk. Fresh files need no confirmation; these do.
func pendingOverwrites(preview diff.Result) []string {
	var out []string
	for _, f := range preview.Files {
		if f.Changed && !f.IsNew {
			out = append(out, f.Path)
		}
	}
	return out
}

// printWritten reports an apply/generate write. An empty write set
// prints the same no-change message as a clean preview.
func printWritten(cmd *cobra.Command, stacks []stackJSON, res apply.Result, format string) error {
	written := res.Written
	if written == nil {
		written = []string{}
	}
	backups := res.Backups
	if backups == nil {
		backups = map[string]string{}
	}
	if format == "json" {
		return writeJSON(cmd.OutOrStdout(), writtenPayload{Stacks: stacks, Written: written, Backups: backups})
	}
	if len(written) == 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "no changes"); err != nil {
			return err
		}
		return nil
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "wrote %d file(s):\n", len(written)); err != nil {
		return err
	}
	for _, p := range written {
		if bak, ok := backups[p]; ok {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s (backup: %s)\n", p, bak); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %s (new)\n", p); err != nil {
				return err
			}
		}
	}
	return nil
}

// formatEvidence renders one ranked finding as a single text line.
func formatEvidence(ev detect.Evidence) string {
	var b strings.Builder
	b.WriteString(ev.Ecosystem)
	b.WriteString(": ")
	b.WriteString(ev.Confidence.String())
	b.WriteString(" (signals: ")
	b.WriteString(strings.Join(ev.Signals, ", "))
	b.WriteString(")")
	if ev.PackageManager != "" {
		b.WriteString(" pm=")
		b.WriteString(ev.PackageManager)
	}
	if ev.VersionHint != "" {
		b.WriteString(" version=")
		b.WriteString(ev.VersionHint)
	}
	return b.String()
}

// stackJSON is the machine-readable view of one detection finding.
type stackJSON struct {
	Ecosystem      string   `json:"ecosystem"`
	Confidence     string   `json:"confidence"`
	Signals        []string `json:"signals"`
	VersionHint    string   `json:"versionHint,omitempty"`
	PackageManager string   `json:"packageManager,omitempty"`
}

// detectPayload carries ranked evidence for detect --format json.
type detectPayload struct {
	Stacks []stackJSON `json:"stacks"`
}

// filePayload carries one planned file for preview --format json.
type filePayload struct {
	Path    string `json:"path"`
	IsNew   bool   `json:"isNew"`
	Changed bool   `json:"changed"`
	Unified string `json:"unified,omitempty"`
}

// previewPayload carries the ranked evidence plus the diff.
type previewPayload struct {
	Stacks     []stackJSON   `json:"stacks"`
	Files      []filePayload `json:"files"`
	HasChanges bool          `json:"hasChanges"`
}

// writtenPayload carries the ranked evidence plus the write result.
type writtenPayload struct {
	Stacks  []stackJSON       `json:"stacks"`
	Written []string          `json:"written"`
	Backups map[string]string `json:"backups"`
}

// blockedPayload carries the pending overwrites for --format json
// when apply refuses to prompt in non-interactive mode (exit 3).
type blockedPayload struct {
	Stacks  []stackJSON `json:"stacks"`
	Pending []string    `json:"pending"`
	Blocked string      `json:"blocked"`
}

// stacksJSON converts ranked evidence to its machine-readable view.
// The result is never nil so it marshals as [] instead of null.
func stacksJSON(evidences []detect.Evidence) []stackJSON {
	out := make([]stackJSON, 0, len(evidences))
	for _, ev := range evidences {
		signals := ev.Signals
		if signals == nil {
			signals = []string{}
		}
		out = append(out, stackJSON{
			Ecosystem:      ev.Ecosystem,
			Confidence:     ev.Confidence.String(),
			Signals:        signals,
			VersionHint:    ev.VersionHint,
			PackageManager: ev.PackageManager,
		})
	}
	return out
}

// writeJSON encodes v as indented JSON followed by a newline.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
