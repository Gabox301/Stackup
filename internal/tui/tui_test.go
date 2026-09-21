package tui_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/Gabox301/Stackup/internal/apply"
	"github.com/Gabox301/Stackup/internal/detect"
	"github.com/Gabox301/Stackup/internal/diff"
	"github.com/Gabox301/Stackup/internal/generate"
	"github.com/Gabox301/Stackup/internal/tui"
)

func nodeEvidence() []detect.Evidence {
	return []detect.Evidence{
		{
			Ecosystem:      "node",
			Confidence:     detect.ConfidenceHigh,
			Signals:        []string{"package.json", "package-lock.json"},
			VersionHint:    ">=20",
			PackageManager: "npm",
		},
	}
}

// vscodePlan builds the compact 4-file plan used across TUI tests.
func vscodePlan(t *testing.T) generate.Plan {
	t.Helper()
	plan, err := generate.Build(nodeEvidence(), []string{"vscode"})
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	return plan
}

// updateModel drives one message through the confirm Model.
func updateModel(m tui.Model, msg tea.Msg) tui.Model {
	next, _ := m.Update(msg)
	model, ok := next.(tui.Model)
	if !ok {
		panic("tui.Model.Update returned non-Model")
	}
	return model
}

func keyRunes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func keyType(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

// stripANSI removes ANSI escapes so golden files pin plain text only:
// palette tweaks never churn goldens (see palette-change-keeps-goldens).
// It lives here so every package tui_test golden shares one helper.
func stripANSI(s string) string {
	return ansi.Strip(s)
}

func TestConfirmKeysDecide(t *testing.T) {
	t.Parallel()

	plan := vscodePlan(t)
	pending := []string{".vscode/settings.json"}

	applyKeys := []tea.Msg{
		keyRunes("y"),
		keyRunes("Y"),
		keyType(tea.KeyEnter),
	}
	for _, msg := range applyKeys {
		m := updateModel(tui.NewConfirmModel(plan, pending), msg)
		if !m.Decided() || !m.Confirmed() {
			t.Errorf("key %v decided=%v confirmed=%v, want both true", msg, m.Decided(), m.Confirmed())
		}
		if m.Decision() != tui.ActionApply {
			t.Errorf("key %v decision = %v, want ActionApply", msg, m.Decision())
		}
	}

	abortKeys := []tea.Msg{
		keyRunes("n"),
		keyRunes("N"),
		keyRunes("q"),
		keyRunes("Q"),
		keyType(tea.KeyEsc),
		keyType(tea.KeyCtrlC),
	}
	for _, msg := range abortKeys {
		m := updateModel(tui.NewConfirmModel(plan, pending), msg)
		if !m.Decided() || m.Confirmed() {
			t.Errorf("key %v decided=%v confirmed=%v, want decided-only", msg, m.Decided(), m.Confirmed())
		}
		if m.Decision() != tui.ActionAbort {
			t.Errorf("key %v decision = %v, want ActionAbort", msg, m.Decision())
		}
	}
}

func TestConfirmIgnoresNavigationAndSize(t *testing.T) {
	t.Parallel()

	plan := vscodePlan(t)
	m := tui.NewConfirmModel(plan, nil)

	moved := updateModel(m, keyType(tea.KeyDown))
	if moved.Cursor() != 1 {
		t.Errorf("down cursor = %d, want 1", moved.Cursor())
	}
	moved = updateModel(moved, keyRunes("j"))
	if moved.Cursor() != 2 {
		t.Errorf("j cursor = %d, want 2", moved.Cursor())
	}
	moved = updateModel(moved, keyType(tea.KeyUp))
	if moved.Cursor() != 1 {
		t.Errorf("up cursor = %d, want 1", moved.Cursor())
	}
	moved = updateModel(moved, keyRunes("k"))
	if moved.Cursor() != 0 {
		t.Errorf("k cursor = %d, want 0", moved.Cursor())
	}

	// Clamps hold at both ends and navigation never decides.
	pinned := updateModel(m, keyType(tea.KeyUp))
	if pinned.Cursor() != 0 || pinned.Decided() {
		t.Errorf("up at top cursor=%d decided=%v, want 0/false", pinned.Cursor(), pinned.Decided())
	}
	bottom := m
	for range 9 {
		bottom = updateModel(bottom, keyType(tea.KeyDown))
	}
	if bottom.Cursor() != len(plan.Files)-1 {
		t.Errorf("down at bottom cursor=%d, want %d", bottom.Cursor(), len(plan.Files)-1)
	}
	if bottom.Decided() {
		t.Error("navigation decided the dialog, want undecided")
	}

	sized := updateModel(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	if sized.Decided() {
		t.Error("WindowSizeMsg decided the dialog, want undecided")
	}

	other := updateModel(m, keyRunes("x"))
	if other.Decided() {
		t.Error("unbound key decided the dialog, want undecided")
	}
	if other.Decision() != tui.ActionAbort {
		t.Error("undecided dialog decision is not the safe Abort default")
	}
}

func TestConfirmViewGolden(t *testing.T) {
	t.Parallel()

	plan := vscodePlan(t)
	m := tui.NewConfirmModel(plan, []string{".vscode/settings.json"})
	// Golden file: testdata/TestConfirmViewGolden.golden, refreshed
	// with go test -update (the flag comes from x/exp/golden via
	// teatest). The view is stripped before compare so styled bytes
	// never churn the golden; the file stays byte-identical.
	teatest.RequireEqualOutput(t, []byte(stripANSI(m.View())))
}

// Per-screen goldens pin View() at a fixed 80x30 ASCII render. Each test
// applies WindowSizeMsg{80,30} first so the diff viewport size is fixed;
// the other screens ignore size by design, but the message keeps the
// harness uniform. Refresh with go test ./internal/tui -update, inspect
// the diff, then rerun clean without -update.
func TestEvidenceViewGolden(t *testing.T) {
	t.Parallel()

	m := updateModel(tui.NewEvidenceModel(nodeEvidence()), tea.WindowSizeMsg{Width: 80, Height: 30})
	teatest.RequireEqualOutput(t, []byte(stripANSI(m.View())))
}

func TestPlanViewGolden(t *testing.T) {
	t.Parallel()

	plan := vscodePlan(t)
	m := updateModel(tui.NewPlanModel(plan, []string{".vscode/settings.json"}), tea.WindowSizeMsg{Width: 80, Height: 30})
	teatest.RequireEqualOutput(t, []byte(stripANSI(m.View())))
}

func TestDiffViewGolden(t *testing.T) {
	t.Parallel()

	m := updateModel(tui.NewDiffModel(twoFilePreview()), tea.WindowSizeMsg{Width: 80, Height: 30})
	teatest.RequireEqualOutput(t, []byte(stripANSI(m.View())))
}

func TestResultViewGolden(t *testing.T) {
	t.Parallel()

	res := apply.Result{
		Written: []string{".vscode/settings.json", ".vscode/tasks.json"},
		Backups: map[string]string{".vscode/settings.json": ".vscode/settings.json.bak.20260101-000000"},
	}
	m := updateModel(tui.NewResultModel(res), tea.WindowSizeMsg{Width: 80, Height: 30})
	teatest.RequireEqualOutput(t, []byte(stripANSI(m.View())))
}

// TestPlanKeysDecide covers the generate write gate: y/enter confirms
// the write, n/q/esc aborts, navigation moves and clamps without ever
// deciding, and size/unbound keys leave the gate undecided.
func TestPlanKeysDecide(t *testing.T) {
	t.Parallel()

	plan := vscodePlan(t)
	pending := []string{".vscode/settings.json"}

	writeKeys := []tea.Msg{
		keyRunes("y"),
		keyRunes("Y"),
		keyType(tea.KeyEnter),
	}
	for _, msg := range writeKeys {
		m := updateModel(tui.NewPlanModel(plan, pending), msg)
		if !m.Decided() || !m.Confirmed() {
			t.Errorf("key %v decided=%v confirmed=%v, want both true", msg, m.Decided(), m.Confirmed())
		}
		if m.Decision() != tui.ActionApply {
			t.Errorf("key %v decision = %v, want ActionApply", msg, m.Decision())
		}
	}

	abortKeys := []tea.Msg{
		keyRunes("n"),
		keyRunes("N"),
		keyRunes("q"),
		keyRunes("Q"),
		keyType(tea.KeyEsc),
		keyType(tea.KeyCtrlC),
	}
	for _, msg := range abortKeys {
		m := updateModel(tui.NewPlanModel(plan, pending), msg)
		if !m.Decided() || m.Confirmed() {
			t.Errorf("key %v decided=%v confirmed=%v, want decided-only", msg, m.Decided(), m.Confirmed())
		}
		if m.Decision() != tui.ActionAbort {
			t.Errorf("key %v decision = %v, want ActionAbort", msg, m.Decision())
		}
	}

	// Navigation moves and clamps without deciding.
	m := tui.NewPlanModel(plan, pending)
	if moved := updateModel(m, keyType(tea.KeyDown)); moved.Cursor() != 1 || moved.Decided() {
		t.Errorf("down cursor=%d decided=%v, want 1/false", moved.Cursor(), moved.Decided())
	}
	if pinned := updateModel(m, keyType(tea.KeyUp)); pinned.Cursor() != 0 || pinned.Decided() {
		t.Errorf("up at top cursor=%d decided=%v, want 0/false", pinned.Cursor(), pinned.Decided())
	}
	bottom := m
	for range len(plan.Files) + 2 {
		bottom = updateModel(bottom, keyType(tea.KeyDown))
	}
	if bottom.Cursor() != len(plan.Files)-1 || bottom.Decided() {
		t.Errorf("down at bottom cursor=%d decided=%v, want %d/false", bottom.Cursor(), bottom.Decided(), len(plan.Files)-1)
	}
	if sized := updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 30}); sized.Decided() {
		t.Error("WindowSizeMsg decided the gate, want undecided")
	}
	if other := updateModel(m, keyRunes("x")); other.Decided() {
		t.Error("unbound key decided the gate, want undecided")
	}
	if fresh := tui.NewPlanModel(plan, pending); fresh.Decided() || fresh.Decision() != tui.ActionAbort {
		t.Error("fresh gate is not the safe undecided-Abort default")
	}
}

// TestPresenceScreensUpdatePerScreen covers per-screen Update keys and
// clamps for the read-only presence screens (evidence, result):
// navigation moves and clamps, quit keys (including y/n/enter) abort
// without confirming, and size/unbound keys never decide. The plan
// gate owns its own key contract in TestPlanKeysDecide.
func TestPresenceScreensUpdatePerScreen(t *testing.T) {
	t.Parallel()

	screens := []struct {
		name  string
		model tui.Model
		top   int
	}{
		{"evidence", tui.NewEvidenceModel(nodeEvidence()), 0},
		{"result", tui.NewResultModel(apply.Result{Written: []string{"a"}}), 0},
	}
	for _, sc := range screens {
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()
			// Navigation clamps at the top.
			if pinned := updateModel(sc.model, keyType(tea.KeyUp)); pinned.Cursor() != 0 || pinned.Decided() {
				t.Errorf("up at top cursor=%d decided=%v, want 0/false", pinned.Cursor(), pinned.Decided())
			}
			// Down moves only when rows exist below, never decides.
			moved := updateModel(sc.model, keyType(tea.KeyDown))
			wantCursor := 0
			if sc.top > 0 {
				wantCursor = 1
			}
			if moved.Cursor() != wantCursor {
				t.Errorf("down cursor = %d, want %d", moved.Cursor(), wantCursor)
			}
			if moved.Decided() {
				t.Error("navigation decided the screen, want undecided")
			}
			// Bottom clamp holds.
			bottom := sc.model
			for range sc.top + 2 {
				bottom = updateModel(bottom, keyType(tea.KeyDown))
			}
			if bottom.Cursor() != sc.top {
				t.Errorf("down at bottom cursor=%d, want %d", bottom.Cursor(), sc.top)
			}
			if bottom.Decided() {
				t.Error("bottom navigation decided the screen, want undecided")
			}
			// Quit keys abort without confirming.
			for _, key := range []tea.Msg{keyRunes("q"), keyType(tea.KeyEsc), keyType(tea.KeyCtrlC), keyType(tea.KeyEnter), keyRunes("y"), keyRunes("n")} {
				next := updateModel(sc.model, key)
				if !next.Decided() {
					t.Errorf("key %v left %s undecided, want quit", key, sc.name)
				}
				if next.Decision() != tui.ActionAbort {
					t.Errorf("key %v %s decision = %v, want ActionAbort", key, sc.name, next.Decision())
				}
				if next.Confirmed() {
					t.Errorf("key %v confirmed %s, want never-confirmed on presence", key, sc.name)
				}
			}
			// Size and unbound keys never decide.
			if sized := updateModel(sc.model, tea.WindowSizeMsg{Width: 80, Height: 30}); sized.Decided() {
				t.Errorf("WindowSizeMsg decided %s, want undecided (viewport only on diff)", sc.name)
			}
			if other := updateModel(sc.model, keyRunes("x")); other.Decided() {
				t.Errorf("unbound key decided %s, want undecided", sc.name)
			}
		})
	}
}

// TestDiffScreenUpdateKeysAndViewport covers the diff selector and proves
// the viewport resizes only on the diff screen: size is ignored on every
// other screen, selector moves/clamps on diff, and every quit key aborts.
func TestDiffScreenUpdateKeysAndViewport(t *testing.T) {
	t.Parallel()

	m := tui.NewDiffModel(twoFilePreview())
	if m.DiffIndex() != 0 {
		t.Fatalf("DiffIndex() = %d, want 0", m.DiffIndex())
	}
	moved := updateModel(m, keyType(tea.KeyDown))
	if moved.DiffIndex() != 1 {
		t.Errorf("down DiffIndex() = %d, want 1", moved.DiffIndex())
	}
	if moved.Decided() {
		t.Error("diff navigation decided the screen, want undecided")
	}
	pinned := updateModel(moved, keyType(tea.KeyDown))
	if pinned.DiffIndex() != 1 || pinned.Decided() {
		t.Errorf("down at bottom index=%d decided=%v, want 1/false", pinned.DiffIndex(), pinned.Decided())
	}
	back := updateModel(moved, keyType(tea.KeyUp))
	if back.DiffIndex() != 0 {
		t.Errorf("up DiffIndex() = %d, want 0", back.DiffIndex())
	}
	// WindowSizeMsg never decides, even on diff; it only resizes the
	// viewport there.
	sized := updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 30})
	if sized.Decided() {
		t.Error("diff WindowSizeMsg decided the screen, want undecided")
	}
	if sized.DiffIndex() != 0 {
		t.Errorf("diff WindowSizeMsg moved the selector to %d, want 0", sized.DiffIndex())
	}
	// Size is ignored on non-diff screens: cursor and decision untouched.
	plan := vscodePlan(t)
	for name, model := range map[string]tui.Model{
		"evidence": tui.NewEvidenceModel(nodeEvidence()),
		"plan":     tui.NewPlanModel(plan, nil),
		"result":   tui.NewResultModel(apply.Result{}),
	} {
		after := updateModel(model, tea.WindowSizeMsg{Width: 80, Height: 30})
		if after.Decided() || after.Cursor() != 0 {
			t.Errorf("%s size changed state: cursor=%d decided=%v, want 0/false", name, after.Cursor(), after.Decided())
		}
	}
	// Every quit key (including y/n/enter) aborts on diff.
	for _, key := range []tea.Msg{keyRunes("q"), keyType(tea.KeyEsc), keyType(tea.KeyCtrlC), keyType(tea.KeyEnter), keyRunes("y"), keyRunes("n")} {
		next := updateModel(m, key)
		if !next.Decided() || next.Decision() != tui.ActionAbort || next.Confirmed() {
			t.Errorf("key %v diff decided=%v decision=%v confirmed=%v, want decided Abort only",
				key, next.Decided(), next.Decision(), next.Confirmed())
		}
	}
	if other := updateModel(m, keyRunes("x")); other.Decided() {
		t.Error("unbound key decided the diff screen, want undecided")
	}
}

func TestStubSwapKeepsCoreGreen(t *testing.T) {
	t.Parallel()

	// The TUI-swap scenario: with the Bubble Tea program replaced by a
	// stub, the core engines behave exactly as without the TUI.
	plan, err := generate.Build(nodeEvidence(), nil)
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if len(plan.Files) != 16 {
		t.Fatalf("Build() files = %d, want the full 16-file plan", len(plan.Files))
	}

	stub := &tui.StubLauncher{Action: tui.ActionApply}
	act, err := stub.Run(tui.Input{Screen: tui.ScreenConfirm, Plan: plan})
	if err != nil {
		t.Fatalf("StubLauncher.Run() unexpected error: %v", err)
	}
	if act != tui.ActionApply {
		t.Fatalf("StubLauncher.Run() = %v, want ActionApply", act)
	}
	if stub.Calls != 1 || len(stub.Plans) != 1 || len(stub.Plans[0].Files) != 16 {
		t.Fatalf("stub recorded calls=%d plans=%d, want 1 call carrying the 16-file plan",
			stub.Calls, len(stub.Plans))
	}

	// Core runs unchanged under the stub: fresh apply writes, diff
	// previews nothing afterwards, re-apply is idempotent.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte("{\"name\": \"tui-swap\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := apply.Apply(root, plan, apply.Options{})
	if err != nil {
		t.Fatalf("Apply() under stub unexpected error: %v", err)
	}
	if len(res.Written) != 16 {
		t.Errorf("Apply() wrote %d files, want 16", len(res.Written))
	}
	preview, err := diff.Compute(root, plan, diff.Options{})
	if err != nil {
		t.Fatalf("Compute() unexpected error: %v", err)
	}
	if preview.HasChanges() {
		t.Error("diff after stub-confirmed apply still has changes, want clean")
	}

	aborter := &tui.StubLauncher{Action: tui.ActionAbort}
	if act, _ := aborter.Run(tui.Input{Screen: tui.ScreenConfirm, Plan: plan}); act != tui.ActionAbort {
		t.Errorf("abort stub decision = %v, want ActionAbort", act)
	}

	boomer := &tui.StubLauncher{Err: errors.New("boom")}
	if _, err := boomer.Run(tui.Input{Screen: tui.ScreenConfirm, Plan: plan}); err == nil || err.Error() != "boom" {
		t.Errorf("error stub err = %v, want the canned boom", err)
	}
}
