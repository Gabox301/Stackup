//go:build tools

// Package tools pins CLI/TUI dependencies that later slices import
// directly (Bubble Tea stack, huh forms, hujson merge engine, teatest).
// The tools build tag keeps this file out of normal builds while
// `go mod tidy` still records the pinned versions in go.mod/go.sum.
package tools

import (
	_ "charm.land/huh/v2"
	_ "github.com/charmbracelet/bubbles/spinner"
	_ "github.com/charmbracelet/bubbletea"
	_ "github.com/charmbracelet/lipgloss"
	_ "github.com/charmbracelet/x/exp/teatest"
	_ "github.com/tailscale/hujson"
)
