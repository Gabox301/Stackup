package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Adaptive triad for Charm vibrante styling. Light/Dark pairs keep the
// same hue readable on light and dark terminals.
var (
	themeGreen = lipgloss.AdaptiveColor{Light: "#1A7F37", Dark: "#3FB950"}
	themeAmber = lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#D29922"}
	themeGrey  = lipgloss.AdaptiveColor{Light: "#59636E", Dark: "#9198A1"}
	// themeHighlightBG tints the selected row so the cursor reads without
	// moving the text: the marker still carries the meaning after strip.
	themeHighlightBG = lipgloss.AdaptiveColor{Light: "#E3E7EB", Dark: "#333940"}
)

// Theme owns every lipgloss style used by the TUI screens. All styles
// are bound to one renderer: an Ascii renderer keeps View() free of
// escapes (goldens, pipes), while a runtime renderer enables color.
// Spacing stays within 0/1/2; styling wraps whole tokens only so
// ANSI-stripped text stays verbatim.
type Theme struct {
	Title, RoundedBorder, BadgeHigh, BadgeMedium, BadgeLow, PillNew, PillOverwrite, Footer, FooterEmph, Section, Highlight lipgloss.Style
	renderer                                                                                                               *lipgloss.Renderer
	// bordered reports whether the viewport border is drawn. Only the
	// runtime (color-detected) theme draws chrome; the Ascii theme stays
	// borderless so constructor View() output never gains border bytes
	// (goldens stay byte-identical, pipes stay plain).
	bordered bool
}

// NewAsciiTheme builds the deterministic theme used by constructors,
// tests, and non-TTY output: every style renders without escapes and no
// border chrome is drawn.
func NewAsciiTheme() Theme {
	th := NewTheme(ascii)
	th.RoundedBorder = ascii.NewStyle()
	th.bordered = false
	return th
}

// NewTheme binds every named style to r. A nil renderer falls back to
// the Ascii theme so a zero-value Model still renders plain output.
// The border is drawn: NewTheme backs the runtime (color-detected)
// theme, where the viewport frame is wanted.
func NewTheme(r *lipgloss.Renderer) Theme {
	if r == nil {
		return NewAsciiTheme()
	}
	return Theme{
		Title:         r.NewStyle().Bold(true),
		RoundedBorder: r.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(themeGrey),
		BadgeHigh:     r.NewStyle().Bold(true).Foreground(themeGreen),
		BadgeMedium:   r.NewStyle().Bold(true).Foreground(themeAmber),
		BadgeLow:      r.NewStyle().Bold(true).Foreground(themeGrey),
		PillNew:       r.NewStyle().Bold(true).Foreground(themeGreen),
		PillOverwrite: r.NewStyle().Bold(true).Foreground(themeAmber),
		Footer:        r.NewStyle().Foreground(themeGrey),
		FooterEmph:    r.NewStyle().Bold(true),
		Section:       r.NewStyle().Bold(true).Foreground(themeGrey),
		Highlight:     r.NewStyle().Background(themeHighlightBG),
		renderer:      r,
		bordered:      true,
	}
}

// drawsBorder reports whether the viewport border is rendered. The
// runtime theme frames the diff viewport; the Ascii theme never does,
// which keeps constructor and golden output border-free.
func (t Theme) drawsBorder() bool {
	return t.bordered
}

// currentTheme returns the model's theme, falling back to Ascii when
// the model was built without one (zero value). Constructors stay on
// the Ascii default; only the runtime launcher injects a renderer.
func (m Model) currentTheme() Theme {
	if m.theme.renderer == nil {
		return NewAsciiTheme()
	}
	return m.theme
}
