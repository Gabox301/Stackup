package tui_test

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"stackup/internal/apply"
	"stackup/internal/tui"
)

// TestConfirmInteractiveApplies drives the real Bubble Tea program
// once: it waits for the file list to render, answers y, and expects
// the Apply decision out of the finished model.
func TestConfirmInteractiveApplies(t *testing.T) {
	plan := vscodePlan(t)
	tm := teatest.NewTestModel(
		t,
		tui.NewConfirmModel(plan, []string{".vscode/settings.json"}),
		teatest.WithInitialTermSize(80, 30),
	)
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte(".vscode/settings.json"))
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	final, ok := tm.FinalModel(t, teatest.WithFinalTimeout(5*time.Second)).(tui.Model)
	if !ok {
		t.Fatalf("FinalModel() is not a tui.Model, cannot prove the Apply decision")
	}
	if !final.Decided() || !final.Confirmed() {
		t.Fatal("interactive y left the dialog undecided, want confirmed apply")
	}
	if final.Decision() != tui.ActionApply {
		t.Errorf("interactive decision = %v, want ActionApply", final.Decision())
	}
}

// TestPresenceInteractiveQuitAborts drives one real Bubble Tea program per
// presence screen: it waits for the screen to render at fixed 80x30, sends
// q, and expects quit-as-Abort (never confirmed). Confirm y/n stays in
// TestConfirmInteractiveApplies. Not parallel: each leg runs a live program.
// Every quit key returns tea.Quit (slice-1 lesson), so WaitFinished never
// times out on a dropped quit.
func TestPresenceInteractiveQuitAborts(t *testing.T) {
	plan := vscodePlan(t)
	cases := []struct {
		name    string
		model   tui.Model
		waitFor string
	}{
		{"evidence", tui.NewEvidenceModel(nodeEvidence()), "node"},
		{"plan", tui.NewPlanModel(plan, nil), ".vscode/settings.json"},
		{"diff", tui.NewDiffModel(twoFilePreview()), "Stackup diff"},
		{"result", tui.NewResultModel(apply.Result{Written: []string{".vscode/settings.json"}}), "Stackup apply result"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tm := teatest.NewTestModel(t, tc.model, teatest.WithInitialTermSize(80, 30))
			teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
				return bytes.Contains(b, []byte(tc.waitFor))
			}, teatest.WithDuration(5*time.Second))

			tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
			tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

			final, ok := tm.FinalModel(t, teatest.WithFinalTimeout(5*time.Second)).(tui.Model)
			if !ok {
				t.Fatalf("%s FinalModel() is not a tui.Model", tc.name)
			}
			if !final.Decided() {
				t.Errorf("%s interactive q left the screen undecided, want quit", tc.name)
			}
			if final.Decision() != tui.ActionAbort {
				t.Errorf("%s interactive decision = %v, want ActionAbort", tc.name, final.Decision())
			}
			if final.Confirmed() {
				t.Errorf("%s interactive q confirmed, want never-confirmed on presence", tc.name)
			}
		})
	}
}
