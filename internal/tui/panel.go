package tui

import (
	"strings"
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

// compose assembles one app-style screen: the Banner header, a bold
// title, an optional pending summary, a grey section label (only when
// non-empty), blank-separated rows, and a dim footer with key hints.
// The runtime theme frames the whole body; the Ascii theme renders the
// same lines bare, banner included.
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
	return strings.Join(Banner, "\n") + "\n\n" + body + "\n"
}

// selected tints a cursor row behind the marker so the active entry pops
// without moving any text token. In the Ascii theme the tint renders as
// nothing (no escapes, no extra bytes).
func (m Model) selected(row string) string {
	return m.currentTheme().Highlight.Render(row)
}
