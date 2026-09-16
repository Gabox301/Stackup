package main

import "testing"

// TestShouldUseTUI covers the presence gate: both streams must be TTYs,
// --yes/--non-interactive stay text, and json never presents.
func TestShouldUseTUI(t *testing.T) {
	cases := []struct {
		name           string
		stdin          bool
		stdout         bool
		yes            bool
		nonInteractive bool
		format         string
		want           bool
	}{
		{"tty text presents", true, true, false, false, "text", true},
		{"piped stdout stays text", true, false, false, false, "text", false},
		{"piped stdin stays text", false, true, false, false, "text", false},
		{"both piped stays text", false, false, false, false, "text", false},
		{"yes flag stays text", true, true, true, false, "text", false},
		{"non-interactive stays text", true, true, false, true, "text", false},
		{"json never presents", true, true, false, false, "json", false},
		{"json piped stays text", false, false, false, false, "json", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			stubGates(t, tt.stdin, tt.stdout)
			shared := &sharedOpts{yes: tt.yes, nonInteractive: tt.nonInteractive, format: tt.format}
			if got := shouldUseTUI(shared); got != tt.want {
				t.Errorf("shouldUseTUI(stdin=%v stdout=%v yes=%v nonInteractive=%v format=%q) = %v, want %v",
					tt.stdin, tt.stdout, tt.yes, tt.nonInteractive, tt.format, got, tt.want)
			}
		})
	}
}
