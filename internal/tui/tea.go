package tui

import (
	"fmt"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// teaLauncher runs one TUI screen as a real Bubble Tea program.
type teaLauncher struct {
	in   io.Reader
	out  io.Writer
	opts []tea.ProgramOption
}

// NewTeaLauncher returns a Launcher that presents screens on an
// interactive terminal. in/out wire stdio (commands pass their own
// streams; tests pass buffers). opts tunes the program for exotic
// hosts; tests drive the Model directly or through teatest instead.
func NewTeaLauncher(in io.Reader, out io.Writer, opts ...tea.ProgramOption) Launcher {
	return &teaLauncher{in: in, out: out, opts: opts}
}

// Run builds the Model for inp.Screen and maps the outcome to an
// Action. An undecided shutdown (EOF, closed input) is Abort, never
// Apply; only the confirm screen can return Apply.
func (l *teaLauncher) Run(inp Input) (Action, error) {
	opts := append([]tea.ProgramOption{tea.WithInput(l.in), tea.WithOutput(l.out)}, l.opts...)
	final, err := tea.NewProgram(modelForInput(inp, l.out), opts...).Run()
	if err != nil {
		return ActionAbort, fmt.Errorf("tui: run %s screen: %w", inp.Screen, err)
	}
	m, ok := final.(Model)
	if !ok {
		return ActionAbort, fmt.Errorf("tui: unexpected final model %T", final)
	}
	return m.Decision(), nil
}

// modelForInput builds the initial Model for inp over out. Unknown
// screens fall back to the confirm dialog over inp.Plan, the safe
// default. The theme resolves from out+env with termenv default
// detection, so full-color TTYs get full style while pipes, NO_COLOR,
// dumb terminals, and legacy consoles degrade on their own.
// Constructors (used by tests and goldens) stay Ascii: only this
// runtime path injects a detected renderer.
func modelForInput(inp Input, out io.Writer) Model {
	var m Model
	switch inp.Screen {
	case ScreenEvidence:
		m = NewEvidenceModel(inp.Evidence)
	case ScreenPlan:
		m = NewPlanModel(inp.Plan, inp.Pending)
	case ScreenDiff:
		m = NewDiffModel(inp.Preview)
	case ScreenResult:
		m = NewResultModel(inp.Written)
	case ScreenConfirm:
		m = NewConfirmModel(inp.Plan, inp.Pending)
	default:
		m = NewConfirmModel(inp.Plan, inp.Pending)
	}
	m.theme = NewTheme(runtimeRenderer(out))
	return m
}

// runtimeRenderer builds the Lip Gloss renderer bound to out using
// termenv default detection. No color profile is ever forced: a nil
// writer falls back to the shared Ascii renderer so output stays plain
// instead of panicking.
func runtimeRenderer(out io.Writer) *lipgloss.Renderer {
	if out == nil {
		return ascii
	}
	return lipgloss.NewRenderer(out)
}
