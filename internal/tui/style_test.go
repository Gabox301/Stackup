package tui

import (
	"io"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"stackup/internal/apply"
	"stackup/internal/detect"
	"stackup/internal/diff"
)

// ansi256Theme binds the full style set to a fixed color renderer so
// wrapping tests prove styles apply without migrating any goldens.
// SetColorProfile is lipgloss's testing seam: termenv output options
// alone (WithProfile) still degrade non-TTY writers to Ascii, which is
// exactly the runtime behavior production relies on.
func ansi256Theme() Theme {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.ANSI256)
	return NewTheme(r)
}

// withStyle returns m carrying the fixed color theme, the way the
// runtime launcher injects a detected renderer.
func withStyle(m Model) Model {
	m.theme = ansi256Theme()
	return m
}

// styledEvidence builds a single finding at the given grade.
func styledEvidence(conf detect.Confidence) []detect.Evidence {
	return []detect.Evidence{{
		Ecosystem:      "node",
		Confidence:     conf,
		Signals:        []string{"package.json"},
		PackageManager: "npm",
	}}
}

// TestStyleWrapsWholeTokens proves the screen glow-up styles whole
// tokens only: every token stays verbatim while escapes prove the style
// rendered. Constructors keep the Ascii default; only the injected
// runtime theme styles.
func TestStyleWrapsWholeTokens(t *testing.T) {
	t.Parallel()

	styledResult := apply.Result{
		Written: []string{".vscode/settings.json", ".vscode/tasks.json"},
		Backups: map[string]string{".vscode/settings.json": ".vscode/settings.json.bak.1"},
	}
	cases := []struct {
		name   string
		model  Model
		tokens []string
	}{
		{"evidence high badge", NewEvidenceModel(styledEvidence(detect.ConfidenceHigh)), []string{"Stackup detect", "node", "high"}},
		{"evidence medium badge", NewEvidenceModel(styledEvidence(detect.ConfidenceMedium)), []string{"medium"}},
		{"evidence low badge", NewEvidenceModel(styledEvidence(detect.ConfidenceLow)), []string{"low"}},
		{"confirm pills", NewConfirmModel(twoFilePlan(), []string{".vscode/settings.json"}), []string{"Stackup apply", "2 files:", "(overwrite)", "(new)"}},
		{"plan pills", NewPlanModel(twoFilePlan(), []string{".vscode/settings.json"}), []string{"Stackup generate", "(overwrite)", "(new)"}},
		{"result pills", NewResultModel(styledResult), []string{"Stackup apply result", "(backup:", "(new)"}},
		{"diff tags and footer", NewDiffModel(twoFileDiff()), []string{"Stackup diff", "(new)", "(changed)", "exit 2"}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			view := withStyle(tt.model).View()
			for _, token := range tt.tokens {
				if !strings.Contains(view, token) {
					t.Errorf("styled %s View() missing token %q:\n%s", tt.name, token, view)
				}
			}
			if !strings.Contains(view, "\x1b") {
				t.Errorf("styled %s View() has no escapes, want styles applied", tt.name)
			}
		})
	}
}

// TestStyleKeepsMarkersAndNoticesPlain proves what styling must not
// touch: markers and paths stay adjacent and plain, and the no-change
// notices stay unstyled.
func TestStyleKeepsMarkersAndNoticesPlain(t *testing.T) {
	t.Parallel()

	// Marker and path stay outside styles: they read contiguous even in
	// the styled view.
	view := withStyle(NewConfirmModel(twoFilePlan(), nil)).View()
	if !strings.Contains(view, "> .vscode/settings.json") {
		t.Errorf("styled confirm View() split marker and path:\n%s", view)
	}

	// The no-change notices stay unstyled: plain under both themes.
	plainNotices := []Model{
		NewEvidenceModel(nil),
		NewResultModel(apply.Result{}),
		NewDiffModel(diff.Result{}),
	}
	for _, m := range plainNotices {
		if got := m.View(); !strings.Contains(got, "no") {
			t.Errorf("Ascii notice View() missing its notice:\n%s", got)
		}
		styled := withStyle(m).View()
		if !strings.Contains(styled, "no changes") && !strings.Contains(styled, "no supported stacks") {
			t.Errorf("styled notice View() missing its notice:\n%s", styled)
		}
	}
}

// TestStyleDiffTagsCoverAllStates proves every selector tag wraps its
// whole token: new, changed, and unchanged alike.
func TestStyleDiffTagsCoverAllStates(t *testing.T) {
	t.Parallel()

	m := withStyle(Model{})
	cases := []struct {
		name  string
		file  diff.FileDiff
		token string
	}{
		{"new", diff.FileDiff{Path: "a", IsNew: true, Changed: true}, "(new)"},
		{"changed", diff.FileDiff{Path: "b", Changed: true}, "(changed)"},
		{"unchanged", diff.FileDiff{Path: "c"}, "(unchanged)"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := m.diffTag(tt.file)
			if !strings.Contains(got, tt.token) {
				t.Errorf("diffTag() = %q, want whole token %q", got, tt.token)
			}
			if !strings.Contains(got, "\x1b") {
				t.Errorf("styled diffTag() = %q, want escapes proving the style", got)
			}
		})
	}
}

// TestDiffBorderFramesViewport proves the width-2 math and the Ascii
// golden guard: at 80 columns the runtime theme shrinks content to 78
// and frames it, while the Ascii default keeps 80 borderless columns so
// constructor output (and goldens) stay byte-identical.
func TestDiffBorderFramesViewport(t *testing.T) {
	t.Parallel()

	sized := func(m Model) Model {
		next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
		model, ok := next.(Model)
		if !ok {
			t.Fatal("Model.Update returned non-Model")
		}
		return model
	}

	asciiModel := sized(NewDiffModel(twoFileDiff()))
	if asciiModel.viewport.Width != 80 {
		t.Errorf("Ascii viewport Width = %d, want 80 (no border chrome)", asciiModel.viewport.Width)
	}
	if view := asciiModel.View(); strings.Contains(view, "\x1b") || strings.Contains(view, "╭") {
		t.Errorf("Ascii diff View() gained styling or border, want plain:\n%s", view)
	}

	styledModel := sized(withStyle(NewDiffModel(twoFileDiff())))
	if styledModel.viewport.Width != 78 {
		t.Errorf("styled viewport Width = %d, want 78 (80 cols minus border)", styledModel.viewport.Width)
	}
	if view := styledModel.View(); !strings.Contains(view, "╭") {
		t.Errorf("styled diff View() missing the rounded border:\n%s", view)
	}
}

// TestModelForInputInjectsRuntimeTheme proves the tea.go wiring: every
// screen resolves a theme from out+env, unknown screens fall back to
// the confirm dialog, and a non-TTY writer degrades to no escapes
// without any forced profile.
func TestModelForInputInjectsRuntimeTheme(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input Input
		want  Screen
	}{
		{"confirm", Input{Screen: ScreenConfirm, Plan: twoFilePlan()}, ScreenConfirm},
		{"evidence", Input{Screen: ScreenEvidence}, ScreenEvidence},
		{"plan", Input{Screen: ScreenPlan, Plan: twoFilePlan()}, ScreenPlan},
		{"diff", Input{Screen: ScreenDiff, Preview: twoFileDiff()}, ScreenDiff},
		{"result", Input{Screen: ScreenResult}, ScreenResult},
		{"unknown falls back to confirm", Input{Screen: Screen(99), Plan: twoFilePlan()}, ScreenConfirm},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := modelForInput(tt.input, io.Discard)
			if m.Screen() != tt.want {
				t.Errorf("modelForInput screen = %v, want %v", m.Screen(), tt.want)
			}
			if m.theme.renderer == nil {
				t.Error("modelForInput left a zero theme, want the runtime renderer")
			}
			if view := m.View(); strings.Contains(view, "\x1b") {
				t.Errorf("buffer-output View() has escapes, want degraded plain (no forced profile):\n%s", view)
			}
		})
	}
}

// TestRuntimeRendererNilOutFallsBackToAscii keeps the nil-writer path
// total: plain output instead of a panic.
func TestRuntimeRendererNilOutFallsBackToAscii(t *testing.T) {
	t.Parallel()

	m := modelForInput(Input{Screen: ScreenConfirm, Plan: twoFilePlan()}, nil)
	if view := m.View(); strings.Contains(view, "\x1b") {
		t.Errorf("nil-out View() has escapes, want plain Ascii:\n%s", view)
	}
}
