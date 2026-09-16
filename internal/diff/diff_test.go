package diff_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"stackup/internal/diff"
	"stackup/internal/generate"
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

func TestComputeFreshPlanReportsAllNew(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	res, err := diff.Compute(root, vscodePlan(t), diff.Options{})
	if err != nil {
		t.Fatalf("Compute() unexpected error: %v", err)
	}
	if !res.HasChanges() {
		t.Fatal("Compute() HasChanges = false for a fresh plan, want true")
	}
	for _, f := range res.Files {
		if !f.IsNew || !f.Changed {
			t.Errorf("Compute() %s: IsNew=%v Changed=%v, want both true", f.Path, f.IsNew, f.Changed)
		}
		if !strings.HasPrefix(f.Unified, "--- a/"+f.Path) {
			t.Errorf("Compute() %s: unified diff missing header:\n%s", f.Path, f.Unified)
		}
		if len(f.Content) == 0 {
			t.Errorf("Compute() %s: empty desired content", f.Path)
		}
	}
	// A preview writes nothing: the tree must stay empty.
	if got := snapshot(t, root); len(got) != 0 {
		t.Errorf("Compute() wrote %d files, want zero writes", len(got))
	}
}

func TestComputeCleanTreeHasNoChanges(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	plan := vscodePlan(t)
	for _, f := range plan.Files {
		seed(t, root, f.Path, string(f.Content))
	}
	res, err := diff.Compute(root, plan, diff.Options{})
	if err != nil {
		t.Fatalf("Compute() unexpected error: %v", err)
	}
	if res.HasChanges() {
		t.Errorf("Compute() HasChanges = true on an up-to-date tree, want false")
	}
	for _, f := range res.Files {
		if f.Changed || f.Unified != "" {
			t.Errorf("Compute() %s: Changed=%v with diff output, want clean", f.Path, f.Changed)
		}
	}
}

func TestComputeDetectsPendingChange(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	plan := vscodePlan(t)
	seed(t, root, ".vscode/settings.json", "{\"editor.tabSize\": 4}\n")
	res, err := diff.Compute(root, plan, diff.Options{})
	if err != nil {
		t.Fatalf("Compute() unexpected error: %v", err)
	}
	if !res.HasChanges() {
		t.Fatal("Compute() HasChanges = false with a stale file, want true")
	}
	var found *diff.FileDiff
	for i := range res.Files {
		if res.Files[i].Path == ".vscode/settings.json" {
			found = &res.Files[i]
		}
	}
	if found == nil {
		t.Fatal("Compute() missing .vscode/settings.json in result")
	}
	if found.IsNew || !found.Changed {
		t.Errorf("Compute() settings: IsNew=%v Changed=%v, want false/true", found.IsNew, found.Changed)
	}
	if !strings.Contains(found.Unified, "\"editor.tabSize\": 4") {
		t.Errorf("Compute() settings: unified diff missing removed line:\n%s", found.Unified)
	}
	// Writes nothing even when changes are pending.
	if _, err := os.Stat(filepath.Join(root, ".vscode", "extensions.json")); !os.IsNotExist(err) {
		t.Error("Compute() created a missing file, want zero writes")
	}
}

func TestComputeMergeKeepsUserScalarsButForceOverwrites(t *testing.T) {
	t.Parallel()

	const existing = `{
  // Kept by stackup: user comment.
  "editor.tabSize": 4,
}
`
	const generated = `{"editor.tabSize": 2}`
	plan := generate.Plan{Files: []generate.FileOp{
		{Path: ".vscode/settings.json", Content: []byte(generated), JSON: true},
	}}
	root := t.TempDir()
	seed(t, root, ".vscode/settings.json", existing)

	def, err := diff.Compute(root, plan, diff.Options{})
	if err != nil {
		t.Fatalf("Compute() default unexpected error: %v", err)
	}
	if def.HasChanges() {
		t.Error("Compute() default reports changes, want none: existing scalars win the Merge path")
	}

	forced, err := diff.Compute(root, plan, diff.Options{Force: true})
	if err != nil {
		t.Fatalf("Compute() force unexpected error: %v", err)
	}
	if !forced.HasChanges() {
		t.Fatal("Compute() force reports no changes, want overwrite of the user scalar")
	}
	fd := forced.Files[0]
	if !strings.Contains(fd.Unified, `"editor.tabSize": 2`) {
		t.Errorf("Compute() force: unified diff missing generated value:\n%s", fd.Unified)
	}
	if !strings.Contains(string(fd.Content), `"editor.tabSize": 2`) {
		t.Errorf("Compute() force: desired bytes missing generated value:\n%s", fd.Content)
	}
	if !strings.Contains(string(fd.Content), "// Kept by stackup: user comment.") {
		t.Errorf("Compute() force: overwrite dropped the user comment:\n%s", fd.Content)
	}
}

func TestComputeDryRunWritesNothing(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	plan := vscodePlan(t)
	seed(t, root, ".vscode/settings.json", "{\"stale\": true}\n")
	before := snapshot(t, root)
	if _, err := diff.Compute(root, plan, diff.Options{}); err != nil {
		t.Fatalf("Compute() unexpected error: %v", err)
	}
	after := snapshot(t, root)
	if len(before) != len(after) {
		t.Fatalf("Compute() changed file count %d -> %d, want zero writes", len(before), len(after))
	}
	for path, want := range before {
		if after[path] != want {
			t.Errorf("Compute() modified %s, want zero writes", path)
		}
	}
}

func TestComputeInvalidExistingErrors(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	plan := generate.Plan{Files: []generate.FileOp{
		{Path: ".vscode/settings.json", Content: []byte(`{"a": 1}`), JSON: true},
	}}
	seed(t, root, ".vscode/settings.json", `{oops`)
	if _, err := diff.Compute(root, plan, diff.Options{}); err == nil {
		t.Fatal("Compute() expected error for invalid existing JSONC, got nil")
	}
}
