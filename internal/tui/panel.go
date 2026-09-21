package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// boxed wraps the given content lines in one rounded panel. Only the
// runtime (color-detected) theme draws the frame; the Ascii theme returns
// the lines untouched, so degraded output (pipes, legacy consoles) stays
// plain text. Content tokens are identical in both themes: the frame is
// chrome layered around the same lines, matching the diff-viewport
// precedent. Padding stays within the 0/1/2 spacing cap.
func (m Model) boxed(lines []string) []string {
	th := m.currentTheme()
	if !th.drawsBorder() {
		return lines
	}
	style := th.RoundedBorder.Padding(0, 1)
	return strings.Split(style.Render(strings.Join(lines, "\n")), "\n")
}

// clipLines enforces the narrow-terminal policy: once a WindowSizeMsg
// width is known, no output line may exceed it. Over-wide lines (banner
// art, long paths, signal lists, footers) truncate with an ellipsis;
// everything else passes through untouched. Width 0 (no size yet:
// constructor output and default goldens) clips nothing, so those
// renders stay byte-identical. Truncation is ANSI-width aware, so
// styled lines keep valid escapes.
func (m Model) clipLines(lines []string) []string {
	if m.width <= 0 {
		return lines
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		if ansi.StringWidth(line) > m.width {
			out[i] = ansi.Truncate(line, m.width, "…")
		} else {
			out[i] = line
		}
	}
	return out
}

// compose assembles one app-style screen: the Banner header, a bold
// title, an optional pending summary, a grey section label (only when
// non-empty), blank-separated rows, and a dim footer with key hints.
// The runtime theme frames the whole body; the Ascii theme renders the
// same lines bare, banner included. Final lines pass through the
// narrow truncate policy.
func (m Model) compose(title, pending, label string, rows []string, footer string) string {
	th := m.currentTheme()
	var lines []string
	lines = append(lines, th.Title.Render(title))
	if pending != "" {
		lines = append(lines, pending)
	}
	if label != "" {
		lines = append(lines, th.Section.Render(label))
	}
	lines = append(lines, "")
	lines = append(lines, rows...)
	lines = append(lines, "")
	lines = append(lines, th.Footer.Render(footer))
	body := strings.Join(m.boxed(lines), "\n")
	all := append([]string{}, Banner...)
	all = append(all, "")
	all = append(all, strings.Split(body, "\n")...)
	all = append(all, "")
	return strings.Join(m.clipLines(all), "\n")
}

// selected tints a cursor row behind the marker so the active entry pops
// without moving any text token. In the Ascii theme the tint renders as
// nothing (no escapes, no extra bytes).
func (m Model) selected(row string) string {
	return m.currentTheme().Highlight.Render(row)
}
