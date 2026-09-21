package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gabox301/Stackup/internal/tui"
)

// runCLI executes the Cobra tree with args and returns stdout, stderr,
// and the mapped exit code (0 on success).
func runCLI(t *testing.T, args []string, stdin string) (string, string, int) {
	t.Helper()
	cmd := newRootCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if stdin != "" {
		cmd.SetIn(strings.NewReader(stdin))
	}
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		if ee, ok := err.(*exitError); ok {
			return stdout.String(), stderr.String(), ee.code
		}
		t.Fatalf("Execute(%q) unexpected error: %v (stderr: %s)", args, err, stderr.String())
	}
	return stdout.String(), stderr.String(), 0
}

// seedFile writes content at root/rel, creating parent directories.
func seedFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const nodeManifest = "{\"name\": \"cli-test\", \"version\": \"0.0.1\"}\n"

// nodeProject seeds a minimal detectable Node project in a temp dir.
func nodeProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	seedFile(t, root, "package.json", nodeManifest)
	return root
}

// stubTTY replaces the terminal probe for the duration of a test.
func stubTTY(t *testing.T, tty bool) {
	t.Helper()
	old := stdinIsTTY
	stdinIsTTY = func() bool { return tty }
	t.Cleanup(func() { stdinIsTTY = old })
}

// stubStdoutTTY replaces the stdout terminal probe for the duration of a
// test, so piped-vs-TTY branches stay hermetic under go test.
func stubStdoutTTY(t *testing.T, tty bool) {
	t.Helper()
	old := stdoutIsTTY
	stdoutIsTTY = func() bool { return tty }
	t.Cleanup(func() { stdoutIsTTY = old })
}

// stubGates replaces both terminal probes at once.
func stubGates(t *testing.T, stdin, stdout bool) {
	t.Helper()
	stubTTY(t, stdin)
	stubStdoutTTY(t, stdout)
}

// treeFiles lists every file under root as slash-separated relative paths.
func treeFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestHelpListsCoreCommands(t *testing.T) {
	t.Parallel()

	stdout, _, code := runCLI(t, []string{"--help"}, "")
	if code != 0 {
		t.Fatalf("--help exit = %d, want 0", code)
	}
	for _, name := range []string{"detect", "generate", "diff", "apply"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("--help output missing %q:\n%s", name, stdout)
		}
	}
}

func TestDetectJSONIsParseable(t *testing.T) {
	t.Parallel()

	root := nodeProject(t)
	stdout, _, code := runCLI(t, []string{"detect", "--path", root, "--format", "json"}, "")
	if code != 0 {
		t.Fatalf("detect --format json exit = %d, want 0", code)
	}
	var payload struct {
		Stacks []struct {
			Ecosystem  string `json:"ecosystem"`
			Confidence string `json:"confidence"`
		} `json:"stacks"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("detect --format json is not parseable JSON: %v\n%s", err, stdout)
	}
	if len(payload.Stacks) == 0 {
		t.Fatal("detect --format json carries no stacks, want the node finding")
	}
	if payload.Stacks[0].Ecosystem != "node" {
		t.Errorf("detect stacks[0].ecosystem = %q, want %q", payload.Stacks[0].Ecosystem, "node")
	}
}

func TestDetectUnknownStackExit4(t *testing.T) {
	t.Parallel()

	root := t.TempDir() // empty: no detector fires
	if _, _, code := runCLI(t, []string{"detect", "--path", root}, ""); code != 4 {
		t.Errorf("detect on empty dir exit = %d, want 4 (unknown-stack)", code)
	}
	stdout, _, code := runCLI(t, []string{"detect", "--path", root, "--format", "json"}, "")
	if code != 4 {
		t.Errorf("detect --format json on empty dir exit = %d, want 4", code)
	}
	var payload struct {
		Stacks []any `json:"stacks"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Errorf("unknown-stack JSON is not parseable: %v\n%s", err, stdout)
	}
	stdout, _, code = runCLI(t, []string{"detect", "--path", root, "--allow-unknown"}, "")
	if code != 0 {
		t.Errorf("detect --allow-unknown exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "unknown allowed") {
		t.Errorf("detect --allow-unknown output missing the empty notice:\n%s", stdout)
	}
}

func TestApplyBlockedWithoutPromptInCI(t *testing.T) {
	// Not parallel: stubs the process-wide terminal probes.
	stubTTY(t, false) // CI: stdin is not a terminal
	stubStdoutTTY(t, false)
	root := nodeProject(t)
	seedFile(t, root, ".vscode/settings.json", "{}\n")
	before, err := os.ReadFile(filepath.Join(root, ".vscode", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}

	_, _, code := runCLI(t, []string{"apply", "--path", root}, "")
	if code != 3 {
		t.Fatalf("apply without --yes on non-TTY exit = %d, want 3 (blocked-non-interactive)", code)
	}
	after, err := os.ReadFile(filepath.Join(root, ".vscode", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Error("blocked apply modified the file, want zero writes")
	}
	baks, err := filepath.Glob(filepath.Join(root, ".vscode", "settings.json") + ".bak.*")
	if err != nil {
		t.Fatal(err)
	}
	if len(baks) != 0 {
		t.Errorf("blocked apply left %d backups, want zero writes", len(baks))
	}

	_, _, code = runCLI(t, []string{"apply", "--path", root, "--yes"}, "")
	if code != 0 {
		t.Fatalf("apply --yes exit = %d, want 0", code)
	}
	baks, err = filepath.Glob(filepath.Join(root, ".vscode", "settings.json") + ".bak.*")
	if err != nil {
		t.Fatal(err)
	}
	if len(baks) != 1 {
		t.Errorf("apply --yes left %d backups, want exactly one before the overwrite", len(baks))
	}
}

func TestApplyConfirmDialogOnTTY(t *testing.T) {
	// Not parallel: stubs the process-wide terminal probes and launcher.
	// The y/N line prompt from slice 4 is now the TUI confirm dialog;
	// the dialog itself is driven for real in internal/tui (teatest),
	// so here the stub stands in for each answer and the command must
	// honor it: decline aborts cleanly, confirm writes with a backup.
	stubTTY(t, true)
	stubStdoutTTY(t, true)
	root := nodeProject(t)
	seedFile(t, root, ".vscode/settings.json", "{}\n")

	stubLauncher(t, &tui.StubLauncher{Action: tui.ActionAbort})
	if _, _, code := runCLI(t, []string{"apply", "--path", root}, ""); code != 0 {
		t.Errorf("apply declining the dialog exit = %d, want 0 (clean abort)", code)
	}
	if raw, err := os.ReadFile(filepath.Join(root, ".vscode", "settings.json")); err != nil {
		t.Fatal(err)
	} else if string(raw) != "{}\n" {
		t.Errorf("declined apply modified the file to:\n%s", raw)
	}

	stubLauncher(t, &tui.StubLauncher{Action: tui.ActionApply})
	if _, _, code := runCLI(t, []string{"apply", "--path", root}, ""); code != 0 {
		t.Fatalf("apply confirming the dialog exit = %d, want 0", code)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".vscode", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "{}\n" {
		t.Error("confirmed apply wrote nothing, want the merged settings")
	}
}

// stubLauncher swaps the Bubble Tea program for a canned answer.
func stubLauncher(t *testing.T, stub *tui.StubLauncher) {
	t.Helper()
	old := newTUILauncher
	newTUILauncher = func(io.Reader, io.Writer) tui.Launcher { return stub }
	t.Cleanup(func() { newTUILauncher = old })
}

func TestApplyTUISwapKeepsCoreGreen(t *testing.T) {
	// Not parallel: stubs the process-wide terminal probes and launcher.
	// Spec scenario: with the TUI replaced by a stub, core command
	// behaviors remain unchanged.
	stubGates(t, true, true)

	stub := &tui.StubLauncher{Action: tui.ActionApply}
	stubLauncher(t, stub)
	root := nodeProject(t)
	seedFile(t, root, ".vscode/settings.json", "{}\n")

	if _, _, code := runCLI(t, []string{"apply", "--path", root}, ""); code != 0 {
		t.Fatalf("apply under apply-stub exit = %d, want 0", code)
	}
	// Always-present: confirm then result, two screens per apply.
	if stub.Calls != 2 {
		t.Errorf("stub calls = %d, want confirm + result (2) per apply", stub.Calls)
	}
	if len(stub.Inputs) != 2 || stub.Inputs[0].Screen != tui.ScreenConfirm || stub.Inputs[1].Screen != tui.ScreenResult {
		t.Errorf("stub screens = %v, want [confirm result]", stub.Inputs)
	}
	if len(stub.Plans) != 2 || len(stub.Plans[0].Files) != 16 {
		t.Errorf("stub saw %d plans, want 2 with the first carrying the 16-file plan", len(stub.Plans))
	}
	raw, err := os.ReadFile(filepath.Join(root, ".vscode", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "{}\n" {
		t.Error("stub-confirmed apply wrote nothing, want the merged settings")
	}

	aborter := &tui.StubLauncher{Action: tui.ActionAbort}
	stubLauncher(t, aborter)
	before := string(raw)
	seedFile(t, root, ".vscode/tasks.json", "{}\n")
	if _, _, code := runCLI(t, []string{"apply", "--path", root}, ""); code != 0 {
		t.Fatalf("apply under abort-stub exit = %d, want 0 (clean abort)", code)
	}
	// Abort shows only the confirm screen: no result, no writes.
	if aborter.Calls != 1 {
		t.Errorf("abort stub calls = %d, want exactly 1 confirm", aborter.Calls)
	}
	if len(aborter.Inputs) != 1 || aborter.Inputs[0].Screen != tui.ScreenConfirm {
		t.Errorf("abort stub screens = %v, want [confirm]", aborter.Inputs)
	}
	after, err := os.ReadFile(filepath.Join(root, ".vscode", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != before {
		t.Error("stub-aborted apply modified files, want zero writes")
	}
	other, err := os.ReadFile(filepath.Join(root, ".vscode", "tasks.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(other) != "{}\n" {
		t.Error("stub-aborted apply modified tasks.json, want zero writes")
	}
}

func TestDiffPreviewExit2WritesNothing(t *testing.T) {
	t.Parallel()

	root := nodeProject(t)
	stdout, _, code := runCLI(t, []string{"diff", "--path", root}, "")
	if code != 2 {
		t.Fatalf("diff with pending changes exit = %d, want 2 (preview-has-changes)", code)
	}
	if !strings.Contains(stdout, "--- a/") {
		t.Errorf("diff output missing the unified header:\n%s", stdout)
	}
	if got := treeFiles(t, root); len(got) != 1 || got[0] != "package.json" {
		t.Errorf("diff wrote files %v, want only the seeded package.json", got)
	}

	if _, _, code := runCLI(t, []string{"apply", "--path", root, "--yes"}, ""); code != 0 {
		t.Fatalf("apply --yes exit = %d, want 0", code)
	}
	stdout, _, code = runCLI(t, []string{"diff", "--path", root}, "")
	if code != 0 {
		t.Errorf("diff on a clean tree exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "no changes") {
		t.Errorf("clean diff output missing the no-change notice:\n%s", stdout)
	}
}

func TestGenerateDryRunJSONParseableWritesNothing(t *testing.T) {
	t.Parallel()

	root := nodeProject(t)
	stdout, _, code := runCLI(t, []string{"generate", "--path", root, "--dry-run", "--format", "json"}, "")
	if code != 2 {
		t.Fatalf("generate --dry-run exit = %d, want 2 (preview-has-changes)", code)
	}
	var payload struct {
		Stacks []struct {
			Ecosystem string `json:"ecosystem"`
		} `json:"stacks"`
		Files      []map[string]any `json:"files"`
		HasChanges bool             `json:"hasChanges"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("generate --dry-run JSON is not parseable: %v\n%s", err, stdout)
	}
	if !payload.HasChanges || len(payload.Files) == 0 {
		t.Error("generate --dry-run JSON carries no pending diff, want the fresh plan")
	}
	if got := treeFiles(t, root); len(got) != 1 || got[0] != "package.json" {
		t.Errorf("generate --dry-run wrote files %v, want zero writes", got)
	}
}

// TestGateMatrixBytesIdentical proves the gate exclusions stay text
// byte-identical with exits preserved. Not parallel: stubs both
// terminal probes and the launcher.
//
// Harness note: the task row names testdata/small, which does not exist
// in the repo; every leg below uses testdata/node via nodeProject, the
// same deterministic Node fixture slice 2 used for its harness.
func TestGateMatrixBytesIdentical(t *testing.T) {
	// Read-only text commands: deterministic, zero writes, so the same
	// root can serve the baseline and every exclusion leg.
	textCases := []struct {
		name string
		args func(root string) []string
		want int
	}{
		{"detect", func(root string) []string { return []string{"detect", "--path", root} }, 0},
		{"diff", func(root string) []string { return []string{"diff", "--path", root} }, 2},
		{"generate-dry-run", func(root string) []string { return []string{"generate", "--path", root, "--dry-run"} }, 2},
		{"apply-dry-run", func(root string) []string { return []string{"apply", "--path", root, "--dry-run"} }, 2},
	}
	for _, tc := range textCases {
		t.Run("text/"+tc.name, func(t *testing.T) {
			root := nodeProject(t)
			run := func(stdinTTY, stdoutTTY bool, extra ...string) (string, string, int, int) {
				stub := &tui.StubLauncher{Action: tui.ActionAbort}
				stubLauncher(t, stub)
				stubGates(t, stdinTTY, stdoutTTY)
				args := append(tc.args(root), extra...)
				stdout, stderr, code := runCLI(t, args, "")
				return stdout, stderr, code, stub.Calls
			}
			baseOut, baseErr, baseCode, baseCalls := run(false, false)
			if baseCode != tc.want {
				t.Fatalf("%s plain exit = %d, want %d", tc.name, baseCode, tc.want)
			}
			if baseCalls != 0 {
				t.Fatalf("%s plain stub calls = %d, want 0 (no TUI off-gate)", tc.name, baseCalls)
			}
			exclusions := []struct {
				name          string
				stdin, stdout bool
				extra         []string
			}{
				{"piped-stdout", true, false, nil},
				{"piped-stdin", false, true, nil},
				{"yes-flag", true, true, []string{"--yes"}},
				{"non-interactive", true, true, []string{"--non-interactive"}},
			}
			for _, ex := range exclusions {
				stdout, stderr, code, calls := run(ex.stdin, ex.stdout, ex.extra...)
				if calls != 0 {
					t.Errorf("%s %s stub calls = %d, want 0 (gate must skip TUI)", tc.name, ex.name, calls)
				}
				if code != baseCode {
					t.Errorf("%s %s exit = %d, want %d (baseline preserved)", tc.name, ex.name, code, baseCode)
				}
				if stdout != baseOut {
					t.Errorf("%s %s stdout differs from plain baseline:\nbaseline %q\ngot %q", tc.name, ex.name, baseOut, stdout)
				}
				if stderr != baseErr {
					t.Errorf("%s %s stderr differs from plain baseline:\nbaseline %q\ngot %q", tc.name, ex.name, baseErr, stderr)
				}
			}
		})
	}

	// JSON never presents, even on TTY: the TTY run must match the piped
	// run byte-identical with no launcher call.
	jsonCases := []struct {
		name string
		args func(root string) []string
		want int
	}{
		{"detect", func(root string) []string { return []string{"detect", "--path", root, "--format", "json"} }, 0},
		{"diff", func(root string) []string { return []string{"diff", "--path", root, "--format", "json"} }, 2},
		{"generate-dry-run", func(root string) []string {
			return []string{"generate", "--path", root, "--dry-run", "--format", "json"}
		}, 2},
	}
	for _, tc := range jsonCases {
		t.Run("json/"+tc.name, func(t *testing.T) {
			root := nodeProject(t)
			stub := &tui.StubLauncher{Action: tui.ActionAbort}
			stubLauncher(t, stub)
			stubGates(t, false, false)
			baseOut, _, baseCode := runCLI(t, tc.args(root), "")
			if baseCode != tc.want {
				t.Fatalf("%s json plain exit = %d, want %d", tc.name, baseCode, tc.want)
			}
			if stub.Calls != 0 {
				t.Fatalf("%s json plain stub calls = %d, want 0", tc.name, stub.Calls)
			}
			ttyStub := &tui.StubLauncher{Action: tui.ActionAbort}
			stubLauncher(t, ttyStub)
			stubGates(t, true, true)
			ttyOut, _, ttyCode := runCLI(t, tc.args(root), "")
			if ttyStub.Calls != 0 {
				t.Errorf("%s json TTY stub calls = %d, want 0 (json never presents)", tc.name, ttyStub.Calls)
			}
			if ttyCode != baseCode {
				t.Errorf("%s json TTY exit = %d, want %d", tc.name, ttyCode, baseCode)
			}
			if ttyOut != baseOut {
				t.Errorf("%s json TTY stdout differs from plain baseline:\nbaseline %q\ngot %q", tc.name, baseOut, ttyOut)
			}
			var js any
			if err := json.Unmarshal([]byte(ttyOut), &js); err != nil {
				t.Errorf("%s json TTY output is not parseable JSON: %v\n%s", tc.name, err, ttyOut)
			}
		})
	}

	// Apply blocked (exit 3, zero writes): piped and non-interactive legs
	// must match the plain baseline byte-identical. --yes unblocks, so it
	// stays out of this table.
	t.Run("text/apply-blocked", func(t *testing.T) {
		root := nodeProject(t)
		seedFile(t, root, ".vscode/settings.json", "{}\n")
		run := func(stdinTTY, stdoutTTY bool, extra ...string) (string, string, int, int) {
			stub := &tui.StubLauncher{Action: tui.ActionAbort}
			stubLauncher(t, stub)
			stubGates(t, stdinTTY, stdoutTTY)
			args := append([]string{"apply", "--path", root}, extra...)
			stdout, stderr, code := runCLI(t, args, "")
			return stdout, stderr, code, stub.Calls
		}
		baseOut, baseErr, baseCode, baseCalls := run(false, false)
		if baseCode != 3 {
			t.Fatalf("apply blocked plain exit = %d, want 3", baseCode)
		}
		if baseCalls != 0 {
			t.Fatalf("apply blocked plain stub calls = %d, want 0", baseCalls)
		}
		if !strings.Contains(baseErr, "apply blocked") {
			t.Fatalf("apply blocked stderr missing the notice:\n%s", baseErr)
		}
		for _, ex := range []struct {
			name          string
			stdin, stdout bool
			extra         []string
		}{
			{"piped-stdout", true, false, nil},
			{"piped-stdin", false, true, nil},
			{"non-interactive", true, true, []string{"--non-interactive"}},
		} {
			stdout, stderr, code, calls := run(ex.stdin, ex.stdout, ex.extra...)
			if calls != 0 {
				t.Errorf("apply blocked %s stub calls = %d, want 0", ex.name, calls)
			}
			if code != baseCode {
				t.Errorf("apply blocked %s exit = %d, want %d", ex.name, code, baseCode)
			}
			if stdout != baseOut || stderr != baseErr {
				t.Errorf("apply blocked %s bytes differ:\nbaseline out=%q err=%q\ngot out=%q err=%q",
					ex.name, baseOut, baseErr, stdout, stderr)
			}
		}
		if got := treeFiles(t, root); len(got) != 2 {
			t.Errorf("apply blocked wrote %v, want zero writes (package.json + settings.json)", got)
		}
	})
}

// TestLaunchFailureFallbackBytesIdentical proves the fallback rule: a
// qualifying TTY run whose launcher fails prints byte-identical stdout
// with a stderr warning and the engine exit preserved (never coerced to
// 1). Read-only commands give deterministic bytes, so they carry the
// strict equality; apply confirm/result legs assert the safe behavior
// without byte equality because backup timestamps vary per run.
func TestLaunchFailureFallbackBytesIdentical(t *testing.T) {
	// Not parallel: stubs both probes and the launcher.
	readOnly := []struct {
		name string
		args func(root string) []string
		want int
	}{
		{"detect", func(root string) []string { return []string{"detect", "--path", root} }, 0},
		{"diff", func(root string) []string { return []string{"diff", "--path", root} }, 2},
		{"generate-dry-run", func(root string) []string { return []string{"generate", "--path", root, "--dry-run"} }, 2},
		{"apply-dry-run", func(root string) []string { return []string{"apply", "--path", root, "--dry-run"} }, 2},
	}
	for _, tc := range readOnly {
		t.Run(tc.name, func(t *testing.T) {
			root := nodeProject(t)
			plainStub := &tui.StubLauncher{Action: tui.ActionAbort}
			stubLauncher(t, plainStub)
			stubGates(t, false, false)
			baseOut, _, baseCode := runCLI(t, tc.args(root), "")
			if baseCode != tc.want {
				t.Fatalf("%s plain exit = %d, want %d", tc.name, baseCode, tc.want)
			}
			if plainStub.Calls != 0 {
				t.Fatalf("%s plain stub calls = %d, want 0", tc.name, plainStub.Calls)
			}
			boomer := &tui.StubLauncher{Err: errors.New("boom")}
			stubLauncher(t, boomer)
			stubGates(t, true, true)
			fbOut, fbErr, fbCode := runCLI(t, tc.args(root), "")
			if boomer.Calls != 1 {
				t.Errorf("%s fallback stub calls = %d, want 1 (TUI attempted, then fallback)", tc.name, boomer.Calls)
			}
			if fbCode != baseCode {
				t.Errorf("%s fallback exit = %d, want engine exit %d (never 1)", tc.name, fbCode, baseCode)
			}
			if fbOut != baseOut {
				t.Errorf("%s fallback stdout differs from plain:\nplain %q\nfallback %q", tc.name, baseOut, fbOut)
			}
			if !strings.Contains(fbErr, "warning: interactive display unavailable") {
				t.Errorf("%s fallback stderr missing the warning:\n%s", tc.name, fbErr)
			}
			if got := treeFiles(t, root); len(got) != 1 || got[0] != "package.json" {
				t.Errorf("%s fallback wrote %v, want zero writes", tc.name, got)
			}
		})
	}

	t.Run("apply-confirm-failure-safe-abort", func(t *testing.T) {
		stubGates(t, true, true)
		stubLauncher(t, &tui.StubLauncher{Err: errors.New("boom")})
		root := nodeProject(t)
		seedFile(t, root, ".vscode/settings.json", "{}\n")
		before, err := os.ReadFile(filepath.Join(root, ".vscode", "settings.json"))
		if err != nil {
			t.Fatal(err)
		}
		stdout, stderr, code := runCLI(t, []string{"apply", "--path", root}, "")
		if code != 0 {
			t.Errorf("apply confirm fallback exit = %d, want 0 (safe abort, never 1)", code)
		}
		if stdout != "" {
			t.Errorf("apply confirm fallback stdout = %q, want empty (abort prints only to stderr)", stdout)
		}
		if !strings.Contains(stderr, "warning: interactive display unavailable") {
			t.Errorf("apply confirm fallback stderr missing the warning:\n%s", stderr)
		}
		if !strings.Contains(stderr, "apply aborted") {
			t.Errorf("apply confirm fallback stderr missing the abort notice:\n%s", stderr)
		}
		after, err := os.ReadFile(filepath.Join(root, ".vscode", "settings.json"))
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(before) {
			t.Error("apply confirm fallback modified the file, want zero writes")
		}
	})

	t.Run("apply-result-failure-falls-back-to-print", func(t *testing.T) {
		stubGates(t, true, true)
		stubLauncher(t, &tui.StubLauncher{Err: errors.New("boom")})
		root := nodeProject(t)
		stdout, stderr, code := runCLI(t, []string{"apply", "--path", root}, "")
		if code != 0 {
			t.Errorf("fresh apply result fallback exit = %d, want 0", code)
		}
		if !strings.Contains(stderr, "warning: interactive display unavailable") {
			t.Errorf("fresh apply result fallback stderr missing the warning:\n%s", stderr)
		}
		if !strings.Contains(stdout, "wrote ") {
			t.Errorf("fresh apply result fallback stdout missing the written summary:\n%s", stdout)
		}
		if got := treeFiles(t, root); len(got) <= 1 {
			t.Errorf("fresh apply result fallback wrote %v, want the fresh plan files", got)
		}
	})
}
