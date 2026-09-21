package tui_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/Gabox301/Stackup/internal/apply"
	"github.com/Gabox301/Stackup/internal/detect"
	"github.com/Gabox301/Stackup/internal/diff"
	"github.com/Gabox301/Stackup/internal/generate"
	"github.com/Gabox301/Stackup/internal/tui"
)

// tallPreview builds a diff preview whose first file needs scrolling:
// 42 unified lines over the default 12-line viewport.
func tallPreview() diff.Result {
	var b strings.Builder
	b.WriteString("--- a/big.txt\n+++ b/big.txt\n")
	for i := 1; i <= 40; i++ {
		fmt.Fprintf(&b, "+line %02d\n", i)
	}
	return diff.Result{Files: []diff.FileDiff{
		{Path: "big.txt", Changed: true, Unified: b.String()},
		{Path: "other.txt", Changed: true, Unified: "--- a/other.txt\n+++ b/other.txt\n"},
	}}
}

// TestAuditDiffLineScrollReachableAndAdvertised proves T1: J scrolls the
// viewport one line down (K back up) without moving the file selector,
// while up/down still move the selector instead of scrolling.
func TestAuditDiffLineScrollReachableAndAdvertised(t *testing.T) {
	t.Parallel()

	m := tui.NewDiffModel(tallPreview())
	if view := stripANSI(m.View()); !strings.Contains(view, "J/K scroll") {
		t.Errorf("diff footer missing the line-scroll hint:\n%s", view)
	}

	before := stripANSI(m.View())
	if !strings.Contains(before, "--- a/big.txt") {
		t.Fatalf("unscrolled diff View() missing the first unified line:\n%s", before)
	}

	scrolled := updateModel(m, keyRunes("J"))
	if scrolled.DiffIndex() != 0 {
		t.Errorf("J moved the selector to %d, want 0 (scroll must not select)", scrolled.DiffIndex())
	}
	after := stripANSI(scrolled.View())
	if strings.Contains(after, "--- a/big.txt") {
		t.Errorf("J left the first unified line visible, want one line scrolled:\n%s", after)
	}
	if !strings.Contains(after, "+line 11") {
		t.Errorf("J View() missing the newly visible +line 11:\n%s", after)
	}

	back := updateModel(scrolled, keyRunes("K"))
	if !strings.Contains(stripANSI(back.View()), "--- a/big.txt") {
		t.Errorf("K did not scroll back to the top:\n%s", stripANSI(back.View()))
	}

	// up/down still own the selector: moving files resets the viewport.
	moved := updateModel(m, keyType(tea.KeyDown))
	if moved.DiffIndex() != 1 {
		t.Errorf("down DiffIndex() = %d, want 1 (selector owns up/down)", moved.DiffIndex())
	}
}

// TestAuditConfirmAndPlanFootersListAbortKeys proves T2: both gate
// footers name every handled abort key.
func TestAuditConfirmAndPlanFootersListAbortKeys(t *testing.T) {
	t.Parallel()

	plan := vscodePlan(t)
	screens := map[string]tui.Model{
		"confirm": tui.NewConfirmModel(plan, []string{".vscode/settings.json"}),
		"plan":    tui.NewPlanModel(plan, []string{".vscode/settings.json"}),
	}
	for name, m := range screens {
		view := stripANSI(m.View())
		for _, want := range []string{"q/Q/esc/ctrl+c"} {
			if !strings.Contains(view, want) {
				t.Errorf("%s footer missing %q:\n%s", name, want, view)
			}
		}
	}
}

// TestAuditResultCursorFrozenWithoutMarker proves T4: navigation never
// mutates the marker-less result screen (same render, undecided),
// while quit keys still abort.
func TestAuditResultCursorFrozenWithoutMarker(t *testing.T) {
	t.Parallel()

	m := tui.NewResultModel(apply.Result{Written: []string{"a"}})
	want := stripANSI(m.View())
	for _, key := range []tea.Msg{keyType(tea.KeyUp), keyType(tea.KeyDown), keyRunes("k"), keyRunes("j")} {
		next := updateModel(m, key)
		if next.Cursor() != 0 || next.Decided() {
			t.Errorf("key %v cursor=%d decided=%v, want 0/false on marker-less result", key, next.Cursor(), next.Decided())
		}
		if got := stripANSI(next.View()); got != want {
			t.Errorf("key %v changed the marker-less render:\n%s", key, got)
		}
	}
	if next := updateModel(m, keyRunes("q")); !next.Decided() || next.Decision() != tui.ActionAbort {
		t.Errorf("q decided=%v decision=%v, want decided Abort", next.Decided(), next.Decision())
	}
}

// TestAuditDryRunPlanFooterPromisesNoWrites proves T5: the plan footer
// (shared by the dry-run leg, which writes nothing) uses confirm
// wording while the gate decisions stay intact.
func TestAuditDryRunPlanFooterPromisesNoWrites(t *testing.T) {
	t.Parallel()

	plan := vscodePlan(t)
	pending := []string{".vscode/settings.json"}
	view := stripANSI(tui.NewPlanModel(plan, pending).View())
	for _, bad := range []string{"y write", "enter write"} {
		if strings.Contains(view, bad) {
			t.Errorf("dry-run plan footer promises %q, want confirm wording:\n%s", bad, view)
		}
	}
	if !strings.Contains(view, "enter confirm") {
		t.Errorf("plan footer missing the confirm verb:\n%s", view)
	}
	for _, key := range []tea.Msg{keyRunes("y"), keyType(tea.KeyEnter)} {
		next := updateModel(tui.NewPlanModel(plan, pending), key)
		if !next.Decided() || !next.Confirmed() || next.Decision() != tui.ActionApply {
			t.Errorf("key %v left decided=%v confirmed=%v decision=%v, want confirmed Apply",
				key, next.Decided(), next.Confirmed(), next.Decision())
		}
	}
}

// narrowPath is a 72-column plan path: every row carrying it exceeds a
// 60-column terminal, so the narrow goldens exercise the truncate rule.
func narrowPath() string {
	return ".vscode/" + strings.Repeat("a", 58) + ".json"
}

// narrowPlan builds a two-file plan around the over-wide path.
func narrowPlan() generate.Plan {
	return generate.Plan{Files: []generate.FileOp{
		{Path: narrowPath()},
		{Path: ".vscode/tasks.json"},
	}}
}

// TestAuditNarrowWidthTruncatesWithEllipsis proves T3: at 60 columns no
// stripped line exceeds the width and the over-wide row carries the
// ellipsis tail.
func TestAuditNarrowWidthTruncatesWithEllipsis(t *testing.T) {
	t.Parallel()

	m := updateModel(tui.NewConfirmModel(narrowPlan(), nil), tea.WindowSizeMsg{Width: 60, Height: 30})
	view := stripANSI(m.View())
	if !strings.Contains(view, "…") {
		t.Errorf("narrow View() missing the truncation ellipsis:\n%s", view)
	}
	for _, line := range strings.Split(strings.TrimRight(view, "\n"), "\n") {
		if w := ansi.StringWidth(line); w > 60 {
			t.Errorf("narrow line width %d > 60: %q", w, line)
		}
	}
}

// Narrow goldens (T3/T6) pin every screen at 60x30 with over-wide
// content where the screen carries user paths. Refresh with
// go test ./internal/tui -update, inspect the diff, rerun clean.
func TestConfirmNarrowGolden(t *testing.T) {
	t.Parallel()

	m := updateModel(tui.NewConfirmModel(narrowPlan(), nil), tea.WindowSizeMsg{Width: 60, Height: 30})
	teatest.RequireEqualOutput(t, []byte(stripANSI(m.View())))
}

func TestEvidenceNarrowGolden(t *testing.T) {
	t.Parallel()

	ev := []detect.Evidence{{
		Ecosystem:      "node",
		Confidence:     detect.ConfidenceHigh,
		Signals:        []string{"package.json", strings.Repeat("s", 60)},
		PackageManager: "npm",
	}}
	m := updateModel(tui.NewEvidenceModel(ev), tea.WindowSizeMsg{Width: 60, Height: 30})
	teatest.RequireEqualOutput(t, []byte(stripANSI(m.View())))
}

func TestPlanNarrowGolden(t *testing.T) {
	t.Parallel()

	m := updateModel(tui.NewPlanModel(narrowPlan(), []string{narrowPath()}), tea.WindowSizeMsg{Width: 60, Height: 30})
	teatest.RequireEqualOutput(t, []byte(stripANSI(m.View())))
}

func TestDiffNarrowGolden(t *testing.T) {
	t.Parallel()

	preview := diff.Result{Files: []diff.FileDiff{
		{Path: narrowPath(), IsNew: true, Changed: true, Unified: "--- a/big\n+++ b/big\n+line\n"},
		{Path: ".vscode/tasks.json", Changed: true, Unified: "--- a/tasks\n+++ b/tasks\n"},
	}}
	m := updateModel(tui.NewDiffModel(preview), tea.WindowSizeMsg{Width: 60, Height: 30})
	teatest.RequireEqualOutput(t, []byte(stripANSI(m.View())))
}

func TestResultNarrowGolden(t *testing.T) {
	t.Parallel()

	res := apply.Result{
		Written: []string{narrowPath()},
		Backups: map[string]string{narrowPath(): narrowPath() + ".bak.20260101-000000"},
	}
	m := updateModel(tui.NewResultModel(res), tea.WindowSizeMsg{Width: 60, Height: 30})
	teatest.RequireEqualOutput(t, []byte(stripANSI(m.View())))
}

// TestAuditConfirmInteractiveEnterQuits drives the real confirm program
// with enter and proves it quits (WaitFinished): the Apply decision
// itself stays pinned Update-direct below, so this leg asserts no
// FinalModel type (esc-race lesson).
func TestAuditConfirmInteractiveEnterQuits(t *testing.T) {
	plan := vscodePlan(t)
	tm := teatest.NewTestModel(
		t,
		tui.NewConfirmModel(plan, []string{".vscode/settings.json"}),
		teatest.WithInitialTermSize(80, 30),
	)
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte(".vscode/settings.json"))
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	decided := updateModel(tui.NewConfirmModel(plan, []string{".vscode/settings.json"}), keyType(tea.KeyEnter))
	if !decided.Decided() || !decided.Confirmed() || decided.Decision() != tui.ActionApply {
		t.Errorf("direct enter decided=%v confirmed=%v decision=%v, want confirmed Apply",
			decided.Decided(), decided.Confirmed(), decided.Decision())
	}
}

// TestAuditDiffInteractiveSelectorSwapsContent drives the real diff
// program: down selects the second file and its unified text renders.
// Output-based only: no FinalModel assertion (esc-race lesson).
func TestAuditDiffInteractiveSelectorSwapsContent(t *testing.T) {
	tm := teatest.NewTestModel(
		t,
		tui.NewDiffModel(twoFilePreview()),
		teatest.WithInitialTermSize(80, 30),
	)
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("settings.json"))
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("--- a/.vscode/tasks.json"))
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}

// TestAuditNonTTYStrippedViewMatchesPlainPrint proves the non-TTY
// parity leg: the stripped (unstyled, pipe-equivalent) view carries
// every plain-print token verbatim — evidence row, diff unified,
// result summary, and both no-change notices.
func TestAuditNonTTYStrippedViewMatchesPlainPrint(t *testing.T) {
	t.Parallel()

	ev := stripANSI(tui.NewEvidenceModel(nodeEvidence()).View())
	if !strings.Contains(ev, "node: high (signals: package.json, package-lock.json) pm=npm version=>=20") {
		t.Errorf("stripped evidence View() differs from the plain detect row:\n%s", ev)
	}
	if !strings.Contains(stripANSI(tui.NewEvidenceModel(nil).View()), "no supported stacks detected (use --allow-unknown to proceed)") {
		t.Error("stripped empty evidence View() lost the exit-4 notice")
	}

	preview := twoFilePreview()
	dv := stripANSI(tui.NewDiffModel(preview).View())
	for _, want := range []string{"--- a/.vscode/settings.json", "+++ b/.vscode/settings.json"} {
		if !strings.Contains(dv, want) {
			t.Errorf("stripped diff View() lost the plain unified line %q:\n%s", want, dv)
		}
	}
	if !strings.Contains(stripANSI(tui.NewDiffModel(diff.Result{}).View()), "no changes") {
		t.Error("stripped clean diff View() lost the no-change notice")
	}

	rv := stripANSI(tui.NewResultModel(apply.Result{Written: []string{"a"}}).View())
	for _, want := range []string{"wrote 1 file(s):", "  a (new)"} {
		if !strings.Contains(rv, want) {
			t.Errorf("stripped result View() lost the plain summary line %q:\n%s", want, rv)
		}
	}
	if !strings.Contains(stripANSI(tui.NewResultModel(apply.Result{}).View()), "no changes") {
		t.Error("stripped empty result View() lost the no-change notice")
	}
}
