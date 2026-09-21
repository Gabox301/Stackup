package apply_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gabox301/Stackup/internal/apply"
	"github.com/Gabox301/Stackup/internal/diff"
	"github.com/Gabox301/Stackup/internal/generate"
)

// vscodePlan builds the deterministic generic vscode plan (4 JSON files).
func vscodePlan(t *testing.T) generate.Plan {
	t.Helper()
	p, err := generate.Build(nil, []string{generate.IDEVSCode})
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	return p
}

// seed writes content at root/rel, creating parent directories.
func seed(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// read returns the file at root/rel as a string.
func read(t *testing.T, root, rel string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// snapshot records every file under root as rel -> bytes.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(raw)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// backups returns the backup siblings recorded for rel.
func backups(t *testing.T, root, rel string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(rel)) + ".bak.*")
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func TestApplyCreatesFreshFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	plan := vscodePlan(t)
	res, err := apply.Apply(root, plan, apply.Options{})
	if err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}
	if len(res.Written) != len(plan.Files) {
		t.Errorf("Apply() wrote %d files, want %d", len(res.Written), len(plan.Files))
	}
	if len(res.Backups) != 0 {
		t.Errorf("Apply() took %d backups for fresh files, want zero", len(res.Backups))
	}
	for _, f := range plan.Files {
		if got := read(t, root, f.Path); got != string(f.Content) {
			t.Errorf("Apply() %s changed fresh bytes:\n got:\n%s\nwant:\n%s", f.Path, got, f.Content)
		}
	}
}

func TestApplyBackupBeforeWrite(t *testing.T) {
	t.Parallel()

	const existing = `{
  // Project editor settings; stackup must keep this comment.
  "editor.tabSize": 4,
  "custom.userKey": "keep-me",
}
`
	root := t.TempDir()
	seed(t, root, ".vscode/settings.json", existing)
	plan := generate.Plan{Files: []generate.FileOp{
		{Path: ".vscode/settings.json", Content: []byte("{\"editor.tabSize\": 2, \"files.eol\": \"\\n\"}"), JSON: true},
	}}
	res, err := apply.Apply(root, plan, apply.Options{})
	if err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}
	baks := backups(t, root, ".vscode/settings.json")
	if len(baks) != 1 {
		t.Fatalf("Apply() left %d backups, want exactly one timestamped .bak", len(baks))
	}
	if !strings.Contains(baks[0], ".bak.") {
		t.Errorf("Apply() backup %q misses the .bak.<ts> shape", baks[0])
	}
	raw, err := os.ReadFile(baks[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != existing {
		t.Errorf("Apply() backup is not the pre-write image:\n got:\n%s\nwant:\n%s", raw, existing)
	}
	got := read(t, root, ".vscode/settings.json")
	for _, want := range []string{
		"// Project editor settings; stackup must keep this comment.",
		`"custom.userKey": "keep-me"`,
		`"editor.tabSize": 4`, // default Merge path: existing scalars win
		`"files.eol"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Apply() merged result missing %q:\n%s", want, got)
		}
	}
	if res.Backups[".vscode/settings.json"] == "" {
		t.Error("Apply() result omits the backup entry for the overwritten file")
	}
}

func TestApplyNeverWritesWithoutBackup(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	plan := vscodePlan(t)
	before := map[string]string{}
	for _, f := range plan.Files {
		seed(t, root, f.Path, "{\"stale\": true}\n")
		before[f.Path] = "{\"stale\": true}\n"
	}
	res, err := apply.Apply(root, plan, apply.Options{})
	if err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}
	// Every overwritten file must carry a backup with its pre-image.
	for _, path := range res.Written {
		bak, ok := res.Backups[path]
		if !ok {
			t.Errorf("Apply() wrote %s with no backup entry: forbidden", path)
			continue
		}
		if got := read(t, root, bak); got != before[path] {
			t.Errorf("Apply() backup of %s is not the pre-write image", path)
		}
	}
}

func TestApplyDryRunWritesNothing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	plan := vscodePlan(t)
	seed(t, root, ".vscode/settings.json", "{\"stale\": true}\n")
	before := snapshot(t, root)
	res, err := apply.Apply(root, plan, apply.Options{DryRun: true})
	if err != nil {
		t.Fatalf("Apply() dry-run unexpected error: %v", err)
	}
	if !res.DryRun {
		t.Error("Apply() dry-run result does not echo DryRun")
	}
	if len(res.Written) == 0 {
		t.Error("Apply() dry-run reports nothing pending, want the would-write set")
	}
	if len(res.Backups) != 0 {
		t.Errorf("Apply() dry-run took %d backups, want zero writes", len(res.Backups))
	}
	after := snapshot(t, root)
	if len(before) != len(after) {
		t.Fatalf("Apply() dry-run changed file count %d -> %d, want zero writes", len(before), len(after))
	}
	for path, want := range before {
		if after[path] != want {
			t.Errorf("Apply() dry-run modified %s, want zero writes", path)
		}
	}
}

func TestApplyCommentPreservingEndToEnd(t *testing.T) {
	t.Parallel()

	const existing = `{
  // Project editor settings; stackup must keep this comment.
  "editor.tabSize": 4, // user prefers width 4
  "custom.userKey": "keep-me",
}
`
	root := t.TempDir()
	seed(t, root, ".vscode/settings.json", existing)
	plan := generate.Plan{Files: []generate.FileOp{
		{Path: ".vscode/settings.json", Content: []byte("{\"editor.tabSize\": 2, \"files.eol\": \"\\n\"}"), JSON: true},
		{Path: ".cursor/rules/stackup.mdc", Content: []byte("# rules\n"), JSON: false},
	}}
	if _, err := apply.Apply(root, plan, apply.Options{}); err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}
	got := read(t, root, ".vscode/settings.json")
	for _, want := range []string{
		"// Project editor settings; stackup must keep this comment.",
		"// user prefers width 4",
		`"custom.userKey": "keep-me"`,
		`"files.eol"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Apply() end-to-end result missing %q:\n%s", want, got)
		}
	}
	// Second apply is a no-op and the preview agrees: idempotent.
	again, err := apply.Apply(root, plan, apply.Options{})
	if err != nil {
		t.Fatalf("Apply() second run unexpected error: %v", err)
	}
	if len(again.Written) != 0 {
		t.Errorf("Apply() second run wrote %v, want idempotent no-op", again.Written)
	}
	preview, err := diff.Compute(root, plan, diff.Options{})
	if err != nil {
		t.Fatalf("Compute() unexpected error: %v", err)
	}
	if preview.HasChanges() {
		t.Error("Compute() reports changes right after apply, want clean")
	}
}

func TestApplyForceOverwritesScalars(t *testing.T) {
	t.Parallel()

	plan := generate.Plan{Files: []generate.FileOp{
		{Path: ".vscode/settings.json", Content: []byte(`{"editor.tabSize": 2}`), JSON: true},
	}}
	root := t.TempDir()
	seed(t, root, ".vscode/settings.json", "{\"editor.tabSize\": 4}\n")

	if _, err := apply.Apply(root, plan, apply.Options{}); err != nil {
		t.Fatalf("Apply() default unexpected error: %v", err)
	}
	if got := read(t, root, ".vscode/settings.json"); !strings.Contains(got, `"editor.tabSize": 4`) {
		t.Errorf("Apply() default overwrote the user scalar:\n%s", got)
	}

	if _, err := apply.Apply(root, plan, apply.Options{Force: true}); err != nil {
		t.Fatalf("Apply() force unexpected error: %v", err)
	}
	if got := read(t, root, ".vscode/settings.json"); !strings.Contains(got, `"editor.tabSize": 2`) {
		t.Errorf("Apply() force did not overwrite the user scalar:\n%s", got)
	}
	if len(backups(t, root, ".vscode/settings.json")) == 0 {
		t.Error("Apply() force left no backup, want backup before every overwrite")
	}
}

func TestApplyFailsSafeOnInvalidJSON(t *testing.T) {
	t.Parallel()

	plan := generate.Plan{Files: []generate.FileOp{
		{Path: ".vscode/settings.json", Content: []byte(`{"a": 1}`), JSON: true},
	}}
	root := t.TempDir()
	seed(t, root, ".vscode/settings.json", `{oops`)
	if _, err := apply.Apply(root, plan, apply.Options{}); err == nil {
		t.Fatal("Apply() expected error for invalid existing JSONC, got nil")
	}
	if got := read(t, root, ".vscode/settings.json"); got != `{oops` {
		t.Errorf("Apply() touched the file on error:\n%s", got)
	}
	if len(backups(t, root, ".vscode/settings.json")) != 0 {
		t.Error("Apply() left a backup for a write that never happened")
	}
}

func TestApplyPreservesFileMode(t *testing.T) {
	t.Parallel()

	const rel = ".vscode/settings.json"
	root := t.TempDir()
	seed(t, root, rel, "{\"editor.tabSize\": 4}\n")
	abs := filepath.Join(root, filepath.FromSlash(rel))
	// Restrict the mode best-effort: platforms without POSIX bits keep
	// 0666/0444 and the equality checks below still hold vacuously.
	if err := os.Chmod(abs, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	plan := generate.Plan{Files: []generate.FileOp{
		{Path: rel, Content: []byte("{\"editor.tabSize\": 2, \"files.eol\": \"\\n\"}"), JSON: true},
	}}
	res, err := apply.Apply(root, plan, apply.Options{})
	if err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}
	if got := read(t, root, rel); !strings.Contains(got, `"files.eol"`) {
		t.Fatalf("Apply() did not overwrite %s, want the merged write:\n%s", rel, got)
	}
	after, err := os.Stat(abs)
	if err != nil {
		t.Fatal(err)
	}
	if after.Mode().Perm() != before.Mode().Perm() {
		t.Errorf("Apply() mode = %o, want the preserved source mode %o", after.Mode().Perm(), before.Mode().Perm())
	}
	bak, ok := res.Backups[rel]
	if !ok {
		t.Fatalf("Apply() result omits the backup entry for %s", rel)
	}
	bakInfo, err := os.Stat(filepath.Join(root, filepath.FromSlash(bak)))
	if err != nil {
		t.Fatal(err)
	}
	if bakInfo.Mode().Perm() != before.Mode().Perm() {
		t.Errorf("Apply() backup mode = %o, want the source mode %o (permissions must never widen through the copy)",
			bakInfo.Mode().Perm(), before.Mode().Perm())
	}
}
