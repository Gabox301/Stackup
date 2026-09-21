package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Gabox301/Stackup/internal/apply"
	"github.com/Gabox301/Stackup/internal/diff"
	"github.com/Gabox301/Stackup/internal/tui"
)

// twoFilePreview builds a deterministic diff preview with one new and
// one modified file.
func twoFilePreview() diff.Result {
	return diff.Result{Files: []diff.FileDiff{
		{Path: ".vscode/settings.json", IsNew: true, Changed: true, Unified: "--- a/.vscode/settings.json\n+++ b/.vscode/settings.json\n"},
		{Path: ".vscode/tasks.json", Changed: true, Unified: "--- a/.vscode/tasks.json\n+++ b/.vscode/tasks.json\n"},
	}}
}

func TestScreenConstructorsReportScreen(t *testing.T) {
	t.Parallel()

	plan := vscodePlan(t)
	cases := []struct {
		name  string
		model tui.Model
		want  tui.Screen
	}{
		{"confirm", tui.NewConfirmModel(plan, nil), tui.ScreenConfirm},
		{"evidence", tui.NewEvidenceModel(nodeEvidence()), tui.ScreenEvidence},
		{"plan", tui.NewPlanModel(plan, nil), tui.ScreenPlan},
		{"diff", tui.NewDiffModel(twoFilePreview()), tui.ScreenDiff},
		{"result", tui.NewResultModel(apply.Result{Written: []string{"a"}}), tui.ScreenResult},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.model.Screen() != tt.want {
				t.Errorf("Screen() = %v, want %v", tt.model.Screen(), tt.want)
			}
			if tt.model.Decided() {
				t.Error("fresh model Decided() = true, want undecided")
			}
			if tt.model.Decision() != tui.ActionAbort {
				t.Error("fresh model Decision() is not the safe Abort default")
			}
		})
	}
}

func TestEvidenceViewShowsRowsAndEmpty(t *testing.T) {
	t.Parallel()

	view := stripANSI(tui.NewEvidenceModel(nodeEvidence()).View())
	for _, want := range []string{"Stackup detect", "node: high", "package.json", "pm=npm", "version=>=20"} {
		if !strings.Contains(view, want) {
			t.Errorf("evidence View() missing %q:\n%s", want, view)
		}
	}

	empty := stripANSI(tui.NewEvidenceModel(nil).View())
	if !strings.Contains(empty, "no supported stacks detected (use --allow-unknown to proceed)") {
		t.Errorf("empty evidence View() missing the exit-4 notice:\n%s", empty)
	}
}

func TestPresenceNavigationClamps(t *testing.T) {
	t.Parallel()

	// Evidence selector moves over findings and clamps at both ends.
	ev := tui.NewEvidenceModel(nodeEvidence())
	if moved := updateModel(ev, keyType(tea.KeyDown)); moved.Cursor() != 0 {
		t.Errorf("evidence down past single row cursor = %d, want 0", moved.Cursor())
	}
	if moved := updateModel(ev, keyType(tea.KeyUp)); moved.Cursor() != 0 || moved.Decided() {
		t.Errorf("evidence up at top cursor=%d decided=%v, want 0/false", moved.Cursor(), moved.Decided())
	}

	// Plan selector mirrors the confirm file list.
	plan := vscodePlan(t)
	pm := tui.NewPlanModel(plan, nil)
	if moved := updateModel(pm, keyType(tea.KeyDown)); moved.Cursor() != 1 {
		t.Errorf("plan down cursor = %d, want 1", moved.Cursor())
	}
	if pinned := updateModel(pm, keyType(tea.KeyUp)); pinned.Cursor() != 0 || pinned.Decided() {
		t.Errorf("plan up at top cursor=%d decided=%v, want 0/false", pinned.Cursor(), pinned.Decided())
	}
	bottom := pm
	for range 9 {
		bottom = updateModel(bottom, keyType(tea.KeyDown))
	}
	if bottom.Cursor() != len(plan.Files)-1 {
		t.Errorf("plan down at bottom cursor=%d, want %d", bottom.Cursor(), len(plan.Files)-1)
	}
	if bottom.Decided() {
		t.Error("plan navigation decided the screen, want undecided")
	}
}

func TestPresenceQuitIsAbort(t *testing.T) {
	t.Parallel()

	plan := vscodePlan(t)
	screens := map[string]tui.Model{
		"evidence": tui.NewEvidenceModel(nodeEvidence()),
		"plan":     tui.NewPlanModel(plan, nil),
		"diff":     tui.NewDiffModel(twoFilePreview()),
		"result":   tui.NewResultModel(apply.Result{Written: []string{".vscode/settings.json"}}),
	}
	quitKeys := []tea.Msg{
		keyRunes("q"),
		keyType(tea.KeyEsc),
		keyType(tea.KeyCtrlC),
		keyType(tea.KeyEnter),
		keyRunes("y"), // presence must never imply consent
		keyRunes("n"),
	}
	for name, m := range screens {
		for _, key := range quitKeys {
			next := updateModel(m, key)
			if !next.Decided() {
				t.Errorf("%s key %v left the screen undecided, want quit", name, key)
			}
			if next.Decision() != tui.ActionAbort {
				t.Errorf("%s key %v decision = %v, want ActionAbort", name, key, next.Decision())
			}
			if next.Confirmed() {
				t.Errorf("%s key %v confirmed, want never-confirmed on presence", name, key)
			}
		}
	}
}

func TestPresenceIgnoresSizeAndUnboundKeys(t *testing.T) {
	t.Parallel()

	plan := vscodePlan(t)
	screens := []tui.Model{
		tui.NewEvidenceModel(nodeEvidence()),
		tui.NewPlanModel(plan, nil),
		tui.NewResultModel(apply.Result{}),
	}
	for i, m := range screens {
		if sized := updateModel(m, tea.WindowSizeMsg{Width: 100, Height: 30}); sized.Decided() {
			t.Errorf("screen %d WindowSizeMsg decided, want undecided", i)
		}
		if other := updateModel(m, keyRunes("x")); other.Decided() {
			t.Errorf("screen %d unbound key decided, want undecided", i)
		}
	}
}

func TestDiffSelectorMovesAndClamps(t *testing.T) {
	t.Parallel()

	m := tui.NewDiffModel(twoFilePreview())
	if m.DiffIndex() != 0 {
		t.Fatalf("DiffIndex() = %d, want 0", m.DiffIndex())
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "Stackup diff") || !strings.Contains(view, "settings.json") {
		t.Errorf("diff View() missing title or selector:\n%s", view)
	}
	if view := stripANSI(m.View()); !strings.Contains(view, "exit 2") {
		t.Errorf("diff View() missing the exit-2 footer:\n%s", view)
	}

	moved := updateModel(m, keyType(tea.KeyDown))
	if moved.DiffIndex() != 1 {
		t.Errorf("down DiffIndex() = %d, want 1", moved.DiffIndex())
	}
	if view := stripANSI(moved.View()); !strings.Contains(view, "tasks.json") {
		t.Errorf("moved diff View() missing the selected unified text:\n%s", view)
	}
	pinned := updateModel(moved, keyType(tea.KeyDown))
	if pinned.DiffIndex() != 1 || pinned.Decided() {
		t.Errorf("down at bottom index=%d decided=%v, want 1/false", pinned.DiffIndex(), pinned.Decided())
	}
	back := updateModel(moved, keyType(tea.KeyUp))
	if back.DiffIndex() != 0 {
		t.Errorf("up DiffIndex() = %d, want 0", back.DiffIndex())
	}

	sized := updateModel(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	if sized.Decided() {
		t.Error("diff WindowSizeMsg decided the screen, want undecided")
	}

	clean := tui.NewDiffModel(diff.Result{})
	if view := stripANSI(clean.View()); !strings.Contains(view, "no changes") {
		t.Errorf("clean diff View() missing the no-change notice:\n%s", view)
	}
}

func TestResultViewShowsWrittenAndNoChanges(t *testing.T) {
	t.Parallel()

	res := apply.Result{
		Written: []string{".vscode/settings.json", ".vscode/tasks.json"},
		Backups: map[string]string{".vscode/settings.json": ".vscode/settings.json.bak.20260101-000000"},
	}
	view := stripANSI(tui.NewResultModel(res).View())
	for _, want := range []string{"Stackup apply result", "wrote 2 file(s):", ".vscode/settings.json (backup:", ".vscode/tasks.json (new)"} {
		if !strings.Contains(view, want) {
			t.Errorf("result View() missing %q:\n%s", want, view)
		}
	}

	if view := stripANSI(tui.NewResultModel(apply.Result{}).View()); !strings.Contains(view, "no changes") {
		t.Errorf("empty result View() missing the no-change notice:\n%s", view)
	}
}

func TestPlanViewKeepsConfirmWording(t *testing.T) {
	t.Parallel()

	plan := vscodePlan(t)
	view := stripANSI(tui.NewPlanModel(plan, []string{".vscode/settings.json"}).View())
	for _, want := range []string{"Stackup generate", "4 files:", "3 new, 1 overwrite", ".vscode/settings.json (overwrite)"} {
		if !strings.Contains(view, want) {
			t.Errorf("plan View() missing %q:\n%s", want, view)
		}
	}
}

func TestStubRecordsInputs(t *testing.T) {
	t.Parallel()

	plan := vscodePlan(t)
	stub := &tui.StubLauncher{Action: tui.ActionAbort}
	pending := []string{".vscode/settings.json"}
	if _, err := stub.Run(tui.Input{Screen: tui.ScreenConfirm, Plan: plan, Pending: pending}); err != nil {
		t.Fatal(err)
	}
	if _, err := stub.Run(tui.Input{Screen: tui.ScreenEvidence, Evidence: nodeEvidence()}); err != nil {
		t.Fatal(err)
	}
	if stub.Calls != 2 {
		t.Fatalf("stub calls = %d, want 2", stub.Calls)
	}
	if len(stub.Inputs) != 2 || stub.Inputs[0].Screen != tui.ScreenConfirm || stub.Inputs[1].Screen != tui.ScreenEvidence {
		t.Fatalf("stub inputs = %+v, want confirm then evidence", stub.Inputs)
	}
	if len(stub.Inputs[0].Pending) != 1 || len(stub.Plans) != 2 || len(stub.Plans[0].Files) != 4 {
		t.Fatalf("stub plans/pending not carried: %+v", stub.Inputs[0])
	}
}
