package tui

import (
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"stackup/internal/apply"
	"stackup/internal/detect"
	"stackup/internal/diff"
	"stackup/internal/generate"
)

// ascii renders screens without ANSI escapes, so piped output and
// golden files stay byte-identical on every terminal.
var ascii = lipgloss.NewRenderer(io.Discard, termenv.WithProfile(termenv.Ascii))

// Model is the Bubble Tea screen over one Input payload.
//
// ScreenConfirm keeps the original dialog: y applies, n/q/esc aborts,
// up/k and down/j move the highlight, enter applies. An undecided
// shutdown (EOF, closed input) reads as Abort, so the dialog can never
// confirm by accident.
//
// Presence screens (evidence, plan, diff, result) are read-only: any
// quit key leaves Decided set with apply false, so Decision stays
// Abort and no write is ever gated by them.
type Model struct {
	screen      Screen
	files       []string
	isOverwrite map[string]bool
	cursor      int
	decided     bool
	apply       bool
	evidence    []detect.Evidence
	plan        generate.Plan
	preview     diff.Result
	written     apply.Result
	diffIndex   int
	viewport    viewport.Model
	// theme owns the screen styles. The zero value renders Ascii
	// (see currentTheme); only the runtime launcher injects a
	// renderer, so constructors keep their signatures and output.
	theme Theme
}

// NewConfirmModel builds the confirm screen for p. pending lists the
// plan paths that would overwrite existing files; entries outside the
// plan are ignored. The model starts undecided on the first file.
func NewConfirmModel(p generate.Plan, pending []string) Model {
	m := Model{screen: ScreenConfirm, isOverwrite: map[string]bool{}, plan: p}
	inPlan := map[string]bool{}
	for _, f := range p.Files {
		m.files = append(m.files, f.Path)
		inPlan[f.Path] = true
	}
	for _, path := range pending {
		if inPlan[path] {
			m.isOverwrite[path] = true
		}
	}
	return m
}

// NewEvidenceModel builds the read-only detect screen over ev. The
// model starts undecided on the first finding.
func NewEvidenceModel(ev []detect.Evidence) Model {
	return Model{
		screen:   ScreenEvidence,
		evidence: append([]detect.Evidence(nil), ev...),
	}
}

// NewPlanModel builds the read-only generate preview for p. pending
// marks overwrites with the same wording as the confirm screen.
func NewPlanModel(p generate.Plan, pending []string) Model {
	m := NewConfirmModel(p, pending)
	m.screen = ScreenPlan
	m.decided = false
	m.apply = false
	m.cursor = 0
	return m
}

// NewDiffModel builds the diff screen over r. The selector starts on
// the first file and the viewport shows its unified text.
func NewDiffModel(r diff.Result) Model {
	m := Model{screen: ScreenDiff, preview: r}
	m.viewport = newDiffViewport(diffContent(m))
	return m
}

// NewResultModel builds the read-only apply summary over res.
func NewResultModel(res apply.Result) Model {
	return Model{screen: ScreenResult, written: res}
}

// Screen reports which presence screen the model shows.
func (m Model) Screen() Screen {
	return m.screen
}

// Files reports the plan paths in order.
func (m Model) Files() []string {
	return append([]string(nil), m.files...)
}

// Overwrites reports how many planned files would overwrite existing ones.
func (m Model) Overwrites() int {
	return len(m.isOverwrite)
}

// Cursor reports the highlighted file index.
func (m Model) Cursor() int {
	return m.cursor
}

// DiffIndex reports the selected diff file index.
func (m Model) DiffIndex() int {
	return m.diffIndex
}

// Evidence reports the detect findings in order.
func (m Model) Evidence() []detect.Evidence {
	return append([]detect.Evidence(nil), m.evidence...)
}

// Preview reports the diff preview.
func (m Model) Preview() diff.Result {
	return m.preview
}

// Written reports the apply write summary.
func (m Model) Written() apply.Result {
	return m.written
}

// Decided reports whether the user picked an answer.
func (m Model) Decided() bool {
	return m.decided
}

// Confirmed reports whether the user chose to apply. It is meaningful
// only after Decided, and only on the confirm screen.
func (m Model) Confirmed() bool {
	return m.screen == ScreenConfirm && m.decided && m.apply
}

// Decision maps the dialog outcome to the Launcher answer. Undecided
// (EOF, killed program) is Abort: the safe default. Only the confirm
// screen can return Apply.
func (m Model) Decision() Action {
	if m.Confirmed() {
		return ActionApply
	}
	return ActionAbort
}

// Init implements tea.Model. Screens are static until keys arrive.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. WindowSizeMsg resizes the diff viewport
// and is ignored on every other screen: those layouts are plain lines
// that need no reflow.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		if m.screen == ScreenDiff {
			m.viewport.Width = m.diffWidth(size.Width)
			m.viewport.Height = max(size.Height-10-m.bannerHeight(), 1)
			m.viewport.SetContent(diffContent(m))
			return m, nil
		}
		return m, nil
	}
	if m.screen == ScreenDiff {
		return updateDiff(m, msg)
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.screen == ScreenConfirm {
		return updateConfirm(m, key)
	}
	return updatePresence(m, key)
}

// updateConfirm handles the confirm dialog keys verbatim: y applies,
// n/q/esc aborts, up/k and down/j move the highlight, enter applies.
// Deciding keys quit the program so Run returns the answer.
func updateConfirm(m Model, key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "y", "Y", "enter":
		m.decided, m.apply = true, true
		return m, tea.Quit
	case "n", "N", "q", "esc", "ctrl+c":
		m.decided = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down", "j":
		if m.cursor < len(m.files)-1 {
			m.cursor++
		}
		return m, nil
	}
	return m, nil
}

// updatePresence handles read-only screens: navigation moves the
// highlight, every quit key (including y/n/enter) quits as Abort so
// presence can never imply consent.
func updatePresence(m Model, key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "y", "Y", "n", "N", "q", "Q", "enter", "esc", "ctrl+c":
		m.decided = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down", "j":
		top := len(m.files) - 1
		if m.screen == ScreenEvidence {
			top = len(m.evidence) - 1
		}
		if m.cursor < top {
			m.cursor++
		}
		return m, nil
	}
	return m, nil
}

// View implements tea.Model. It joins plain lines (no error-returning
// writers), keeping the render deterministic for goldens.
func (m Model) View() string {
	switch m.screen {
	case ScreenEvidence:
		return m.viewEvidence()
	case ScreenPlan:
		return m.viewPlan()
	case ScreenDiff:
		return renderDiff(m)
	case ScreenResult:
		return m.viewResult()
	default:
		return m.viewConfirm()
	}
}

// viewConfirm renders the apply confirmation dialog verbatim: pending
// summary, the file list (selected row tinted), and the key hints footer.
func (m Model) viewConfirm() string {
	rows := make([]string, 0, len(m.files))
	for i, path := range m.files {
		row := m.fileLine(i == m.cursor, path, m.isOverwrite[path])
		if i == m.cursor {
			row = m.selected(row)
		}
		rows = append(rows, row)
	}
	return m.compose("Stackup apply", m.pendingLine(len(m.files), len(m.isOverwrite)), "Files to write", rows, "y apply · n abort · up/down move · enter apply")
}

// viewEvidence renders one row per finding with the same wording as
// the plain-text detect output. Empty mirrors the exit-4 notice.
func (m Model) viewEvidence() string {
	var rows []string
	if len(m.evidence) == 0 {
		rows = []string{"no supported stacks detected (use --allow-unknown to proceed)"}
	} else {
		rows = make([]string, 0, len(m.evidence))
		for i, ev := range m.evidence {
			row := m.evidenceLine(i == m.cursor, ev)
			if i == m.cursor {
				row = m.selected(row)
			}
			rows = append(rows, row)
		}
	}
	return m.compose("Stackup detect", "", "Detected stacks", rows, "q quit · up/down move")
}

// viewPlan renders the generate preview with confirm wording. Quitting
// returns to the caller, which performs the write.
func (m Model) viewPlan() string {
	rows := make([]string, 0, len(m.files))
	for i, path := range m.files {
		row := m.fileLine(i == m.cursor, path, m.isOverwrite[path])
		if i == m.cursor {
			row = m.selected(row)
		}
		rows = append(rows, row)
	}
	return m.compose("Stackup generate", m.pendingLine(len(m.files), len(m.isOverwrite)), "Files to write", rows, "q quit · up/down move · quit writes")
}

// viewResult renders the apply write summary with the same lines as
// the plain-text path, or the no-change notice when nothing was written.
func (m Model) viewResult() string {
	if len(m.written.Written) == 0 {
		return m.compose("Stackup apply result", "", "Apply result", []string{"no changes"}, "q quit")
	}
	return m.compose("Stackup apply result", "", "Files written", m.resultLines(m.written), "q quit")
}

// evidenceLine renders one finding with its highlight. The wording
// matches the plain-text detect row: only the whole confidence word
// takes its badge, so stripped text stays verbatim. Markers, paths,
// signals, and hints stay outside styles.
func (m Model) evidenceLine(highlighted bool, ev detect.Evidence) string {
	marker := "  "
	if highlighted {
		marker = "> "
	}
	var b strings.Builder
	b.WriteString(marker)
	b.WriteString(ev.Ecosystem)
	b.WriteString(": ")
	b.WriteString(m.confidenceBadge(ev.Confidence).Render(ev.Confidence.String()))
	b.WriteString(" (signals: ")
	b.WriteString(strings.Join(ev.Signals, ", "))
	b.WriteString(")")
	if ev.PackageManager != "" {
		b.WriteString(" pm=")
		b.WriteString(ev.PackageManager)
	}
	if ev.VersionHint != "" {
		b.WriteString(" version=")
		b.WriteString(ev.VersionHint)
	}
	return b.String()
}

// confidenceBadge maps a grade to its badge style. Unknown grades read
// as Low: the text token always carries the meaning, never color alone.
func (m Model) confidenceBadge(c detect.Confidence) lipgloss.Style {
	th := m.currentTheme()
	switch c {
	case detect.ConfidenceHigh:
		return th.BadgeHigh
	case detect.ConfidenceMedium:
		return th.BadgeMedium
	default:
		return th.BadgeLow
	}
}

// resultLines renders an apply summary with the same lines as the
// plain-text path. Whole tokens only: (new) and (backup: …) take pill
// styles while paths stay outside; the no-change notice stays unstyled.
func (m Model) resultLines(res apply.Result) []string {
	if len(res.Written) == 0 {
		return []string{"no changes"}
	}
	th := m.currentTheme()
	lines := []string{"wrote " + strconv.Itoa(len(res.Written)) + " file(s):"}
	for _, p := range res.Written {
		if bak, ok := res.Backups[p]; ok {
			lines = append(lines, "  "+p+" "+th.PillOverwrite.Render("(backup: "+bak+")"))
		} else {
			lines = append(lines, "  "+p+" "+th.PillNew.Render("(new)"))
		}
	}
	return lines
}

// pendingLine summarizes the plan counts in stable wording, dimmed as a
// subtitle. The whole line is styled so no count token is ever split;
// stripped text stays verbatim.
func (m Model) pendingLine(total, overwrites int) string {
	noun := " files"
	if total == 1 {
		noun = " file"
	}
	return m.currentTheme().Footer.Render(strconv.Itoa(total) + noun + ": " +
		strconv.Itoa(total-overwrites) + " new, " +
		strconv.Itoa(overwrites) + " overwrite")
}

// fileLine renders one planned file with its highlight and merge tag.
// The marker and path stay outside styles; the whole tag takes its pill.
func (m Model) fileLine(highlighted bool, path string, overwrite bool) string {
	marker := "  "
	if highlighted {
		marker = "> "
	}
	tag := m.currentTheme().PillNew.Render("(new)")
	if overwrite {
		tag = m.currentTheme().PillOverwrite.Render("(overwrite)")
	}
	return marker + path + " " + tag
}

// max returns the larger of a and b.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
