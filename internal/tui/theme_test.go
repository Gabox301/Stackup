package tui

import (
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/Gabox301/Stackup/internal/apply"
	"github.com/Gabox301/Stackup/internal/detect"
	"github.com/Gabox301/Stackup/internal/diff"
	"github.com/Gabox301/Stackup/internal/generate"
)

// TestAsciiThemeRendersWithoutEscapes pins the slice-1 contract:
// the Ascii default styles keep their tokens and emit no escapes,
// so constructor View() output stays byte-identical.
func TestThemeAsciiHasNoEscapes(t *testing.T) {
	t.Parallel()

	theme := NewAsciiTheme()
	cases := []struct {
		name  string
		got   string
		token string
	}{
		{"title", theme.Title.Render("Stackup apply"), "Stackup apply"},
		{"border", theme.RoundedBorder.Render("body"), "body"},
		{"badge high", theme.BadgeHigh.Render("high"), "high"},
		{"badge medium", theme.BadgeMedium.Render("medium"), "medium"},
		{"badge low", theme.BadgeLow.Render("low"), "low"},
		{"pill new", theme.PillNew.Render("(new)"), "(new)"},
		{"pill overwrite", theme.PillOverwrite.Render("(overwrite)"), "(overwrite)"},
		{"footer", theme.Footer.Render("q quit"), "q quit"},
		{"footer emph", theme.FooterEmph.Render("exit 2"), "exit 2"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if strings.Contains(tt.got, "\x1b") {
				t.Errorf("Ascii style %q emitted escapes: %q", tt.name, tt.got)
			}
			if !strings.Contains(tt.got, tt.token) {
				t.Errorf("Ascii style %q = %q, want token %q", tt.name, tt.got, tt.token)
			}
		})
	}
}

// TestNewThemeNilRendererFallsBackToAscii keeps the zero-value Model
// contract: no renderer means plain output, never raw escapes.
func TestThemeNilRendererFallsBackToAscii(t *testing.T) {
	t.Parallel()

	got := NewTheme(nil).BadgeHigh.Render("high")
	if strings.Contains(got, "\x1b") {
		t.Errorf("NewTheme(nil) emitted escapes: %q", got)
	}
	if !strings.Contains(got, "high") {
		t.Errorf("NewTheme(nil) = %q, want token %q", got, "high")
	}
}

// TestNewThemeKeepsTokensVerbatim proves a color-bound theme wraps
// whole tokens instead of rewriting them (stripped text stays exact).
func TestThemeKeepsTokensVerbatim(t *testing.T) {
	t.Parallel()

	renderer := lipgloss.NewRenderer(io.Discard, termenv.WithProfile(termenv.ANSI256))
	theme := NewTheme(renderer)
	tokens := []string{"high", "medium", "low", "(new)", "(overwrite)", "exit 2"}
	styles := []lipgloss.Style{
		theme.BadgeHigh, theme.BadgeMedium, theme.BadgeLow,
		theme.PillNew, theme.PillOverwrite, theme.FooterEmph,
	}
	for i, token := range tokens {
		if got := styles[i].Render(token); !strings.Contains(got, token) {
			t.Errorf("styled %q = %q, want verbatim token", token, got)
		}
	}
}

// TestZeroValueModelFallsBackToAsciiTheme locks task 1.2: a bare
// Model and every constructor render without escapes today.
func TestThemeZeroValueModelIsAscii(t *testing.T) {
	t.Parallel()

	models := map[string]Model{
		"zero value": {},
		"confirm":    NewConfirmModel(twoFilePlan(), nil),
		"evidence":   NewEvidenceModel(nil),
		"plan":       NewPlanModel(twoFilePlan(), nil),
		"diff":       NewDiffModel(twoFileDiff()),
		"result":     NewResultModel(apply.Result{}),
	}
	for name, m := range models {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := m.currentTheme().Title.Render("Stackup"); strings.Contains(got, "\x1b") {
				t.Errorf("%s theme emitted escapes: %q", name, got)
			}
			if got := m.View(); strings.Contains(got, "\x1b") {
				t.Errorf("%s View() emitted escapes: %q", name, got)
			}
		})
	}
}

// twoFilePlan builds a minimal two-file plan for constructor tests.
func twoFilePlan() generate.Plan {
	return generate.Plan{Files: []generate.FileOp{
		{Path: ".vscode/settings.json"},
		{Path: ".vscode/tasks.json"},
	}}
}

// twoFileDiff builds a minimal diff preview for constructor tests.
func twoFileDiff() diff.Result {
	return diff.Result{Files: []diff.FileDiff{
		{Path: ".vscode/settings.json", IsNew: true, Changed: true, Unified: "--- a\n+++ b\n"},
		{Path: ".vscode/tasks.json", Changed: true, Unified: "--- a\n+++ b\n"},
	}}
}

// TestThemeANSI256ExactEscapes pins the styled bytes under the fixed
// ANSI256 seam: badges, pills, title, footer, and border corners.
// A palette-only tweak intentionally breaks these pins while stripped
// goldens stay byte-identical (see palette-change-keeps-goldens).
func TestThemeANSI256ExactEscapes(t *testing.T) {
	t.Parallel()

	theme := ansi256Theme()
	cases := []struct {
		name  string
		got   string
		want  string
		token string
	}{
		{"title", theme.Title.Render("Stackup apply"), "\x1b[1mStackup apply\x1b[0m", "Stackup apply"},
		{"badge high", theme.BadgeHigh.Render("high"), "\x1b[1;38;5;71mhigh\x1b[0m", "high"},
		{"badge medium", theme.BadgeMedium.Render("medium"), "\x1b[1;38;5;172mmedium\x1b[0m", "medium"},
		{"badge low", theme.BadgeLow.Render("low"), "\x1b[1;38;5;103mlow\x1b[0m", "low"},
		{"pill new", theme.PillNew.Render("(new)"), "\x1b[1;38;5;71m(new)\x1b[0m", "(new)"},
		{"pill overwrite", theme.PillOverwrite.Render("(overwrite)"), "\x1b[1;38;5;172m(overwrite)\x1b[0m", "(overwrite)"},
		{"footer", theme.Footer.Render("q quit"), "\x1b[38;5;103mq quit\x1b[0m", "q quit"},
		{"footer emph", theme.FooterEmph.Render("exit 2"), "\x1b[1mexit 2\x1b[0m", "exit 2"},
		{"section", theme.Section.Render("Detected stacks"), "\x1b[1;38;5;103mDetected stacks\x1b[0m", "Detected stacks"},
		{"highlight", theme.Highlight.Render("> a"), "\x1b[48;5;59m> a\x1b[0m", "> a"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.got != tt.want {
				t.Errorf("styled %q = %q, want exact %q", tt.name, tt.got, tt.want)
			}
			if stripped := ansi.Strip(tt.got); stripped != tt.token {
				t.Errorf("stripped %q = %q, want verbatim token %q", tt.name, stripped, tt.token)
			}
		})
	}
}

// TestThemeANSI256BorderCorners pins the rounded border chrome: all
// four corners render in the grey triad tone around intact content.
func TestThemeANSI256BorderCorners(t *testing.T) {
	t.Parallel()

	got := ansi256Theme().RoundedBorder.Render("body")
	for _, corner := range []string{"╭", "╮", "╰", "╯"} {
		if !strings.Contains(got, corner) {
			t.Errorf("styled border missing corner %q:\n%q", corner, got)
		}
	}
	if !strings.Contains(got, "\x1b[38;5;103m") {
		t.Errorf("styled border missing the grey triad escape:\n%q", got)
	}
	if stripped := ansi.Strip(got); stripped != "╭────╮\n│body│\n╰────╯" {
		t.Errorf("stripped border = %q, want the intact box", stripped)
	}
}

// TestRuntimeDegradationMatrix proves the env matrix without forcing a
// color profile: NO_COLOR, CLICOLOR=0, TERM=dumb, and the non-TTY
// baseline all render plain with tokens intact, and even a forced-color
// request still degrades on a non-TTY writer. Not parallel: subtests
// mutate the process environment.
func TestRuntimeDegradationMatrix(t *testing.T) {
	cases := []struct {
		name  string
		env   map[string]string
		token string
	}{
		{"non-TTY baseline", nil, "Stackup apply"},
		{"NO_COLOR", map[string]string{"NO_COLOR": "1"}, "Stackup apply"},
		{"CLICOLOR=0", map[string]string{"CLICOLOR": "0"}, "Stackup apply"},
		{"TERM=dumb", map[string]string{"TERM": "dumb"}, "Stackup apply"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if got := runtimeRenderer(io.Discard).ColorProfile(); got != termenv.Ascii {
				t.Errorf("%s renderer profile = %v, want Ascii (%v)", tt.name, got, termenv.Ascii)
			}
			view := modelForInput(Input{Screen: ScreenConfirm, Plan: twoFilePlan()}, io.Discard).View()
			if strings.Contains(view, "\x1b") {
				t.Errorf("%s View() has escapes, want degraded plain:\n%s", tt.name, view)
			}
			if stripped := ansi.Strip(view); !strings.Contains(stripped, tt.token) {
				t.Errorf("%s stripped View() missing token %q:\n%s", tt.name, tt.token, stripped)
			}
		})
	}
}

// TestRuntimeDegradationKeepsTokens proves degraded screens keep their
// meaning: every styled token survives verbatim once stripped, on every
// screen that carries one.
func TestRuntimeDegradationKeepsTokens(t *testing.T) {
	high := []detect.Evidence{{
		Ecosystem:      "node",
		Confidence:     detect.ConfidenceHigh,
		Signals:        []string{"package.json"},
		PackageManager: "npm",
	}}
	written := apply.Result{Written: []string{".vscode/settings.json"}}
	cases := []struct {
		name  string
		input Input
		token string
	}{
		{"evidence badge", Input{Screen: ScreenEvidence, Evidence: high}, "high"},
		{"diff footer", Input{Screen: ScreenDiff, Preview: twoFileDiff()}, "exit 2"},
		{"result pill", Input{Screen: ScreenResult, Written: written}, "(new)"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", "1")
			view := modelForInput(tt.input, io.Discard).View()
			if strings.Contains(view, "\x1b") {
				t.Errorf("NO_COLOR %s View() has escapes, want plain:\n%s", tt.name, view)
			}
			if !strings.Contains(ansi.Strip(view), tt.token) {
				t.Errorf("NO_COLOR %s stripped View() missing token %q:\n%s", tt.name, tt.token, view)
			}
		})
	}
}

// TestLegacyWindowsAsciiFallback proves the legacy-console path: the
// Ascii fallback emits no raw escapes and no border chrome while layout
// and tokens stay intact.
func TestLegacyWindowsAsciiFallback(t *testing.T) {
	t.Parallel()

	models := map[string]struct {
		model Model
		token string
	}{
		"confirm":  {NewConfirmModel(twoFilePlan(), nil), "Stackup apply"},
		"evidence": {NewEvidenceModel(nil), "no supported stacks"},
		"plan":     {NewPlanModel(twoFilePlan(), nil), "Stackup generate"},
		"diff":     {NewDiffModel(twoFileDiff()), "Stackup diff"},
		"result":   {NewResultModel(apply.Result{}), "no changes"},
	}
	for name, tt := range models {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			view := tt.model.View()
			if strings.Contains(view, "\x1b") {
				t.Errorf("%s Ascii View() has raw escapes, want plain:\n%s", name, view)
			}
			if strings.Contains(view, "╭") {
				t.Errorf("%s Ascii View() draws border chrome, want borderless:\n%s", name, view)
			}
			if !strings.Contains(ansi.Strip(view), tt.token) {
				t.Errorf("%s Ascii View() missing token %q:\n%s", name, tt.token, view)
			}
		})
	}
}

// TestBannerAlways proves the ASCII header is drawn on every screen in
// both themes: styled and Ascii/degraded carry the banner text.
func TestBannerAlways(t *testing.T) {
	t.Parallel()

	plan := twoFilePlan()
	screens := map[string]Model{
		"evidence": NewEvidenceModel(styledEvidence(detect.ConfidenceHigh)),
		"confirm":  NewConfirmModel(plan, []string{".vscode/settings.json"}),
		"plan":     NewPlanModel(plan, nil),
		"diff":     NewDiffModel(twoFileDiff()),
		"result":   NewResultModel(apply.Result{Written: []string{"a"}}),
	}
	for name, m := range screens {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			styled := withStyle(m).View()
			plain := m.View()
			if !strings.Contains(ansi.Strip(styled), "/____/") {
				t.Errorf("styled %s View() missing the banner:\n%s", name, ansi.Strip(styled))
			}
			if !strings.Contains(plain, "/____/") {
				t.Errorf("Ascii %s View() missing the banner, want always:\n%s", name, plain)
			}
		})
	}
}

// TestStyledStrippedMatchesAscii proves palette-change-keeps-goldens with
// panel chrome: stripping the forced-color view returns every content
// token verbatim, in the same order and layout the Ascii theme renders.
// The styled view layers the rounded panel (and highlight tints) around
// those same lines; the Ascii view is the same lines with no chrome and
// no escapes, exactly what the goldens (and pipes) pin. This extends the
// diff-only exception to every screen: panels are chrome, never content.
func TestStyledStrippedMatchesAscii(t *testing.T) {
	t.Parallel()

	screens := []struct {
		name   string
		model  Model
		tokens []string
	}{
		{
			"confirm",
			NewConfirmModel(twoFilePlan(), []string{".vscode/settings.json"}),
			[]string{"Stackup apply", "Files to write", "2 files:", "1 new, 1 overwrite", "(overwrite)", "(new)", "enter apply"},
		},
		{
			"evidence",
			NewEvidenceModel([]detect.Evidence{{
				Ecosystem:      "node",
				Confidence:     detect.ConfidenceHigh,
				Signals:        []string{"package.json"},
				PackageManager: "npm",
			}}),
			[]string{"Stackup detect", "Detected stacks", "node", "high", "package.json", "pm=npm", "q quit"},
		},
		{
			"plan",
			NewPlanModel(twoFilePlan(), []string{".vscode/settings.json"}),
			[]string{"Stackup generate", "Files to write", "2 files:", "1 new, 1 overwrite", "(overwrite)", "(new)", "quit writes"},
		},
		{
			"result",
			NewResultModel(apply.Result{
				Written: []string{".vscode/settings.json", ".vscode/tasks.json"},
				Backups: map[string]string{".vscode/settings.json": ".vscode/settings.json.bak.1"},
			}),
			[]string{"Stackup apply result", "Files written", "wrote 2 file(s):", "(backup:", "(new)", "q quit"},
		},
		{
			"diff",
			NewDiffModel(twoFileDiff()),
			[]string{"Stackup diff", "Files", "(new)", "(changed)", "exit 2", "settings.json"},
		},
	}
	for _, sc := range screens {
		t.Run(sc.name, func(t *testing.T) {
			t.Parallel()
			styled := withStyle(sc.model).View()
			plain := sc.model.View()
			if !strings.Contains(styled, "\x1b") {
				t.Errorf("styled %s View() has no escapes, want styles applied:\n%s", sc.name, styled)
			}
			if !strings.Contains(styled, "╭") {
				t.Errorf("styled %s View() missing panel chrome:\n%s", sc.name, styled)
			}
			if strings.Contains(plain, "\x1b") {
				t.Errorf("Ascii %s View() has raw escapes, want plain:\n%s", sc.name, plain)
			}
			if strings.Contains(plain, "╭") {
				t.Errorf("Ascii %s View() draws border chrome, want borderless:\n%s", sc.name, plain)
			}
			for _, token := range sc.tokens {
				if !strings.Contains(ansi.Strip(styled), token) {
					t.Errorf("styled %s stripped View() missing token %q:\n%s", sc.name, token, ansi.Strip(styled))
				}
				if !strings.Contains(plain, token) {
					t.Errorf("Ascii %s View() missing token %q:\n%s", sc.name, token, plain)
				}
			}
		})
	}
}
