package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"stackup/internal/diff"
)

// Diff viewport defaults. They keep the first render deterministic;
// WindowSizeMsg resizes the viewport afterwards.
const (
	diffViewWidth  = 78
	diffViewHeight = 12
)

// newDiffViewport builds the viewport over content with deterministic
// defaults.
func newDiffViewport(content string) viewport.Model {
	vp := viewport.New(diffViewWidth, diffViewHeight)
	vp.SetContent(content)
	return vp
}

// selectedFile reports the diff entry under the selector. It returns
// the zero FileDiff when the preview holds no files.
func selectedFile(m Model) diff.FileDiff {
	if len(m.preview.Files) == 0 {
		return diff.FileDiff{}
	}
	i := m.diffIndex
	if i < 0 {
		i = 0
	}
	if i >= len(m.preview.Files) {
		i = len(m.preview.Files) - 1
	}
	return m.preview.Files[i]
}

// diffContent renders the viewport text for the selected file: its
// unified diff, or the no-change notice when nothing is pending.
func diffContent(m Model) string {
	if !m.preview.HasChanges() {
		return "no changes"
	}
	if sel := selectedFile(m); sel.Unified != "" {
		return sel.Unified
	}
	return "no changes"
}

// diffWidth fits the viewport inside total columns. The rounded border
// costs 2 columns when drawn, so the runtime theme shrinks the content
// and the framed total still fits the terminal; the Ascii theme draws no
// border and keeps the full width, which keeps constructor output (and
// goldens) byte-identical.
func (m Model) diffWidth(total int) int {
	if m.currentTheme().drawsBorder() {
		return max(total-2, 1)
	}
	return total
}

// diffFooter states the preview exit implication: pending changes
// keep exit 2, a clean tree keeps exit 0. The exit-2 token is wrapped
// whole in the emphasis style; the clean-tree notice stays unstyled.
func (m Model) diffFooter(hasChanges bool) string {
	if !hasChanges {
		return "no changes · q quit"
	}
	th := m.currentTheme()
	return th.Footer.Render("changes pending — ") +
		th.FooterEmph.Render("exit 2") +
		th.Footer.Render(" · q quit · up/down select file")
}

// diffTag words one selector row. Every tag is wrapped whole: (new)
// takes the new pill, (changed) the overwrite pill, (unchanged) the dim
// footer. Paths stay outside styles.
func (m Model) diffTag(f diff.FileDiff) string {
	th := m.currentTheme()
	switch {
	case !f.Changed:
		return th.Footer.Render("(unchanged)")
	case f.IsNew:
		return th.PillNew.Render("(new)")
	default:
		return th.PillOverwrite.Render("(changed)")
	}
}

// updateDiff moves the file selector on up/k and down/j, scrolls the
// viewport on any other viewport key, and quits read-only on every
// quit key (including y/n/enter) as Abort.
func updateDiff(m Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "y", "Y", "n", "N", "q", "Q", "enter", "esc", "ctrl+c":
			m.decided = true
			return m, tea.Quit
		case "up", "k":
			if m.diffIndex > 0 {
				m.diffIndex--
				m.viewport.SetContent(diffContent(m))
				m.viewport.GotoTop()
			}
			return m, nil
		case "down", "j":
			if m.diffIndex < len(m.preview.Files)-1 {
				m.diffIndex++
				m.viewport.SetContent(diffContent(m))
				m.viewport.GotoTop()
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// renderDiff joins the selector, the viewport, and the exit-2 footer.
// The unified content itself stays raw (copy-paste safe): no +/-
// tinting in v1. The runtime theme frames the selector panel and the
// viewport in the rounded border; the Ascii theme renders them plain, so
// constructor output stays byte-identical.
func renderDiff(m Model) string {
	th := m.currentTheme()
	var rows []string
	var label string
	if len(m.preview.Files) > 0 {
		label = "Files"
		rows = make([]string, 0, len(m.preview.Files))
		for i, f := range m.preview.Files {
			marker := "  "
			if i == m.diffIndex {
				marker = "> "
			}
			row := marker + f.Path + " " + m.diffTag(f)
			if i == m.diffIndex {
				row = m.selected(row)
			}
			rows = append(rows, row)
		}
	}
	body := strings.TrimRight(m.viewport.View(), "\n")
	if th.drawsBorder() {
		body = th.RoundedBorder.Render(body)
	}
	var lines []string
	lines = append(lines, th.Title.Render("Stackup diff"))
	if label != "" {
		lines = append(lines, th.Section.Render(label))
	}
	lines = append(lines, "")
	if len(rows) > 0 {
		lines = append(lines, rows...)
		lines = append(lines, "")
	}
	lines = append(lines, body)
	lines = append(lines, "")
	lines = append(lines, m.diffFooter(m.preview.HasChanges()))
	output := strings.Join(m.boxed(lines), "\n")
	return strings.Join(Banner, "\n") + "\n\n" + output + "\n"
}
