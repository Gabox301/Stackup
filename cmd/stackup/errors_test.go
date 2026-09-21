package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// runExit executes the Cobra tree and maps the result to an exit code the
// same way main does: exitError codes pass through, any other error is
// exit 1, success is 0.
func runExit(t *testing.T, args []string) (string, string, int, error) {
	t.Helper()
	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		var ee *exitError
		if errors.As(err, &ee) {
			return stdout.String(), stderr.String(), ee.code, err
		}
		return stdout.String(), stderr.String(), 1, err
	}
	return stdout.String(), stderr.String(), 0, nil
}

// TestInvalidCLIArgumentErrors tables the two deterministic CLI argument
// failures: an unknown --format and an unknown --ide. Both are plain
// errors (exit 1, empty stdout) on every command that accepts them, and
// neither writes anything.
func TestInvalidCLIArgumentErrors(t *testing.T) {
	t.Parallel()

	formatCases := []struct {
		name string
		args func(root string) []string
	}{
		{"detect", func(root string) []string { return []string{"detect", "--path", root, "--format", "yaml"} }},
		{"generate", func(root string) []string { return []string{"generate", "--path", root, "--format", "yaml"} }},
		{"diff", func(root string) []string { return []string{"diff", "--path", root, "--format", "yaml"} }},
		{"apply", func(root string) []string { return []string{"apply", "--path", root, "--format", "yaml"} }},
	}
	for _, tc := range formatCases {
		t.Run("format/"+tc.name, func(t *testing.T) {
			t.Parallel()
			root := nodeProject(t)
			stdout, _, code, err := runExit(t, tc.args(root))
			if code != 1 {
				t.Errorf("%s --format yaml exit = %d, want 1 (plain error)", tc.name, code)
			}
			if err == nil || !strings.Contains(err.Error(), `invalid --format "yaml"`) {
				t.Errorf("%s --format yaml error = %v, want it to name the bad format", tc.name, err)
			}
			if stdout != "" {
				t.Errorf("%s --format yaml stdout = %q, want empty (plain error, no payload)", tc.name, stdout)
			}
			if got := treeFiles(t, root); len(got) != 1 || got[0] != "package.json" {
				t.Errorf("%s --format yaml wrote %v, want zero writes", tc.name, got)
			}
		})
	}

	ideCases := []struct {
		name string
		args func(root string) []string
	}{
		{"generate", func(root string) []string { return []string{"generate", "--path", root, "--dry-run", "--ide", "nope"} }},
		{"diff", func(root string) []string { return []string{"diff", "--path", root, "--ide", "nope"} }},
		{"apply", func(root string) []string { return []string{"apply", "--path", root, "--dry-run", "--ide", "nope"} }},
	}
	for _, tc := range ideCases {
		t.Run("ide/"+tc.name, func(t *testing.T) {
			t.Parallel()
			root := nodeProject(t)
			stdout, _, code, err := runExit(t, tc.args(root))
			if code != 1 {
				t.Errorf("%s --ide nope exit = %d, want 1 (plain error)", tc.name, code)
			}
			if err == nil || !strings.Contains(err.Error(), `unknown IDE "nope"`) {
				t.Errorf("%s --ide nope error = %v, want it to name the bad IDE", tc.name, err)
			}
			if stdout != "" {
				t.Errorf("%s --ide nope stdout = %q, want empty (plain error, no payload)", tc.name, stdout)
			}
			if got := treeFiles(t, root); len(got) != 1 || got[0] != "package.json" {
				t.Errorf("%s --ide nope wrote %v, want zero writes", tc.name, got)
			}
		})
	}
}
