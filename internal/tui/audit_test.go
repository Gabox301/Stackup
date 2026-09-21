package tui_test

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gabox301/Stackup/internal/apply"
	"github.com/Gabox301/Stackup/internal/diff"
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
