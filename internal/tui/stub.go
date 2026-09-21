package tui

import "github.com/Gabox301/Stackup/internal/generate"

// StubLauncher answers every Run with a canned Action. Tests swap the
// real Bubble Tea launcher for it to prove the TUI boundary keeps the
// core green: the engines never observe which Launcher answered.
type StubLauncher struct {
	Action Action
	Err    error

	Calls int
	// Inputs records every Run payload in order.
	Inputs []Input
	// Plans records the plan carried by each Run, in order. It mirrors
	// the confirm path so command tests keep asserting the carried
	// plan; prefer Inputs for screen-aware assertions.
	Plans []generate.Plan
}

// Run records inp and returns the canned answer.
func (s *StubLauncher) Run(inp Input) (Action, error) {
	s.Calls++
	s.Inputs = append(s.Inputs, inp)
	s.Plans = append(s.Plans, inp.Plan)
	return s.Action, s.Err
}
