// Package tui presents stackup results on interactive terminals.
//
// Commands talk to the TUI only through the Launcher interface, so
// tests (and future front ends) can swap the Bubble Tea program for a
// stub without touching the detect/generate/diff/apply core.
// Non-interactive runs (exit 3, zero writes) never reach this package.
package tui

import (
	"github.com/Gabox301/Stackup/internal/apply"
	"github.com/Gabox301/Stackup/internal/detect"
	"github.com/Gabox301/Stackup/internal/diff"
	"github.com/Gabox301/Stackup/internal/generate"
)

// Action is the user's answer to the apply confirmation. Only the
// confirm screen ever returns ActionApply; every presence screen
// returns ActionAbort and never gates a write.
type Action int

const (
	// ActionAbort leaves the tree untouched (exit 0, no writes).
	ActionAbort Action = iota
	// ActionApply proceeds to the backup-gated apply write.
	ActionApply
)

// Screen selects which presence screen the Launcher shows.
type Screen int

const (
	// ScreenConfirm is the apply overwrite confirmation dialog.
	ScreenConfirm Screen = iota
	// ScreenEvidence presents detect findings read-only.
	ScreenEvidence
	// ScreenPlan presents a generate plan preview read-only.
	ScreenPlan
	// ScreenDiff presents a unified diff with selector + viewport.
	ScreenDiff
	// ScreenResult presents an apply write summary read-only.
	ScreenResult
)

// String names the screen for errors and debugging.
func (s Screen) String() string {
	switch s {
	case ScreenConfirm:
		return "confirm"
	case ScreenEvidence:
		return "evidence"
	case ScreenPlan:
		return "plan"
	case ScreenDiff:
		return "diff"
	case ScreenResult:
		return "result"
	default:
		return "unknown"
	}
}

// Input carries one screen plus its payload. Only the fields for
// Screen are read; the rest stay zero. Pending lists the plan paths
// that would overwrite existing files.
type Input struct {
	Screen   Screen
	Evidence []detect.Evidence
	Plan     generate.Plan
	Pending  []string
	Preview  diff.Result
	Written  apply.Result
}

// Launcher runs one TUI screen over inp.
type Launcher interface {
	Run(inp Input) (Action, error)
}
