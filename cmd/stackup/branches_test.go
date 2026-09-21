package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/Gabox301/Stackup/internal/tui"
)

// TestDetectPresentsEvidenceScreen proves compute-then-present for detect:
// a qualifying run shows Evidence with no stdout print and exit 0.
func TestDetectPresentsEvidenceScreen(t *testing.T) {
	stubGates(t, true, true)
	stub := &tui.StubLauncher{Action: tui.ActionAbort}
	stubLauncher(t, stub)

	stdout, _, code := runCLI(t, []string{"detect", "--path", nodeProject(t)}, "")
	if code != 0 {
		t.Fatalf("detect on TTY exit = %d, want 0", code)
	}
	if stub.Calls != 1 {
		t.Fatalf("stub calls = %d, want 1 evidence screen", stub.Calls)
	}
	if stub.Inputs[0].Screen != tui.ScreenEvidence {
		t.Errorf("stub screen = %v, want evidence", stub.Inputs[0].Screen)
	}
	if len(stub.Inputs[0].Evidence) == 0 {
		t.Error("evidence screen carries no findings, want the node row")
	}
	if stdout != "" {
		t.Errorf("detect on TTY printed %q to stdout, want empty (TUI replaces the print)", stdout)
	}
}

// TestDiffPresentsDiffScreen proves compute-then-present for diff with
// exit 2 preserved and no stdout print on the TUI leg.
func TestDiffPresentsDiffScreen(t *testing.T) {
	stubGates(t, true, true)
	stub := &tui.StubLauncher{Action: tui.ActionAbort}
	stubLauncher(t, stub)

	stdout, _, code := runCLI(t, []string{"diff", "--path", nodeProject(t)}, "")
	if code != 2 {
		t.Fatalf("diff on TTY exit = %d, want 2 (preview-has-changes)", code)
	}
	if stub.Calls != 1 || stub.Inputs[0].Screen != tui.ScreenDiff {
		t.Fatalf("stub screens = %+v, want exactly 1 diff screen", stub.Inputs)
	}
	if !stub.Inputs[0].Preview.HasChanges() {
		t.Error("diff screen carries a clean preview, want pending changes")
	}
	if stdout != "" {
		t.Errorf("diff on TTY printed %q to stdout, want empty (TUI replaces the print)", stdout)
	}
}

// TestGenerateDryRunPresentsPlanScreen proves the dry-run leg shows Plan
// and maps to exit 2 without printing or writing.
func TestGenerateDryRunPresentsPlanScreen(t *testing.T) {
	stubGates(t, true, true)
	stub := &tui.StubLauncher{Action: tui.ActionAbort}
	stubLauncher(t, stub)
	root := nodeProject(t)

	stdout, _, code := runCLI(t, []string{"generate", "--path", root, "--dry-run"}, "")
	if code != 2 {
		t.Fatalf("generate --dry-run on TTY exit = %d, want 2", code)
	}
	if stub.Calls != 1 || stub.Inputs[0].Screen != tui.ScreenPlan {
		t.Fatalf("stub screens = %+v, want exactly 1 plan screen", stub.Inputs)
	}
	if len(stub.Inputs[0].Plan.Files) != 16 {
		t.Errorf("plan screen files = %d, want the 16-file plan", len(stub.Inputs[0].Plan.Files))
	}
	if stdout != "" {
		t.Errorf("generate --dry-run on TTY printed %q, want empty (TUI replaces the print)", stdout)
	}
	if got := treeFiles(t, root); len(got) != 1 {
		t.Errorf("generate --dry-run wrote %v, want zero writes", got)
	}
}

// TestGenerateWritePresentsPlanThenWrites proves presence never gates the
// write: the Plan screen shows first, then files land with no summary print.
func TestGenerateWritePresentsPlanThenWrites(t *testing.T) {
	stubGates(t, true, true)
	stub := &tui.StubLauncher{Action: tui.ActionAbort}
	stubLauncher(t, stub)
	root := nodeProject(t)

	stdout, _, code := runCLI(t, []string{"generate", "--path", root}, "")
	if code != 0 {
		t.Fatalf("generate on TTY exit = %d, want 0", code)
	}
	if stub.Calls != 1 || stub.Inputs[0].Screen != tui.ScreenPlan {
		t.Fatalf("stub screens = %+v, want exactly 1 plan screen", stub.Inputs)
	}
	if stdout != "" {
		t.Errorf("generate on TTY printed %q, want empty (plan was the output)", stdout)
	}
	if got := treeFiles(t, root); len(got) <= 1 {
		t.Errorf("generate on TTY wrote %v, want the fresh plan files", got)
	}
}

// TestApplyFreshPresentsResultScreen proves always-present: even with no
// overwrites (no confirm), the write is followed by the Result screen.
func TestApplyFreshPresentsResultScreen(t *testing.T) {
	stubGates(t, true, true)
	stub := &tui.StubLauncher{Action: tui.ActionAbort}
	stubLauncher(t, stub)
	root := nodeProject(t)

	stdout, _, code := runCLI(t, []string{"apply", "--path", root}, "")
	if code != 0 {
		t.Fatalf("fresh apply on TTY exit = %d, want 0", code)
	}
	if stub.Calls != 1 || stub.Inputs[0].Screen != tui.ScreenResult {
		t.Fatalf("stub screens = %+v, want exactly 1 result screen", stub.Inputs)
	}
	if len(stub.Inputs[0].Written.Written) == 0 {
		t.Error("result screen carries no written summary, want the fresh write")
	}
	if stdout != "" {
		t.Errorf("fresh apply on TTY printed %q, want empty (result was the output)", stdout)
	}
	if got := treeFiles(t, root); len(got) <= 1 {
		t.Errorf("fresh apply on TTY wrote %v, want the fresh plan files", got)
	}
}

// TestDiffLaunchFailureFallsBack proves one fallback leg: a launcher error
// warns on stderr while stdout stays byte-identical with exit 2 preserved.
// The full gate-bytes matrix belongs to slice 3.
func TestDiffLaunchFailureFallsBack(t *testing.T) {
	stubGates(t, true, true)
	stubLauncher(t, &tui.StubLauncher{Err: errors.New("boom")})
	root := nodeProject(t)

	stdout, stderr, code := runCLI(t, []string{"diff", "--path", root}, "")
	if code != 2 {
		t.Fatalf("diff fallback exit = %d, want 2 (engine exit preserved, never 1)", code)
	}
	if !strings.Contains(stdout, "--- a/") {
		t.Errorf("diff fallback stdout missing the unified header:\n%s", stdout)
	}
	if !strings.Contains(stderr, "warning: interactive display unavailable") {
		t.Errorf("diff fallback stderr missing the warning:\n%s", stderr)
	}
	if got := treeFiles(t, root); len(got) != 1 {
		t.Errorf("diff fallback wrote %v, want zero writes", got)
	}
}
