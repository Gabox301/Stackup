// Command stackup detects project stacks and generates IDE configs.
//
// The Cobra tree (detect|generate|diff|apply) maps engine results to
// process exit codes: 0 ok/no-change, 2 preview-has-changes,
// 3 blocked-non-interactive, 4 unknown-stack, 1 error.
package main

import (
	"errors"
	"fmt"
	"os"
)

// Build metadata, overridden by GoReleaser via ldflags:
// -X main.version=... -X main.commit=... -X main.date=...
// Defaults keep local builds working without flags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// versionString reports the human-readable build identity used by
// --version and (later) release tooling.
func versionString() string {
	if version == "dev" {
		return "dev (commit none, built unknown)"
	}
	return fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
}

func main() {
	if err := newRootCommand().Execute(); err != nil {
		var ee *exitError
		if errors.As(err, &ee) {
			os.Exit(ee.code)
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
