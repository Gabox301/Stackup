package apply

// Failure-manifest tests for apply atomicity (T3).
//
// Both tests force a failure inside a multi-file plan and prove the two
// halves of the contract: detectable failures fail with ZERO writes
// (pre-validation), while failures that only surface mid-plan abort with
// an *ApplyError manifest (applied[] / failed / pending[]) and intact
// backups.
//
// The mid-plan failure is injected through the writeFileFunc seam rather
// than OS permission bits on purpose: chmod-based read-only targets are
// not reliable on Windows (ACLs vs mode bits) and are silently ignored
// when the test process runs as root/administrator, so a permission test
// would need platform skips and still stay flaky. The seam fails one
// exact target path deterministically on every platform.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gabox301/Stackup/internal/generate"
)

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

// anyBackup reports whether root holds any timestamped backup sibling.
func anyBackup(t *testing.T, root string) bool {
	t.Helper()
	var found bool
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.Contains(filepath.ToSlash(path), ".bak.") {
			found = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// TestApplyPrevalidationWritesNothing proves T1: a failure detectable
// before the first write (here a directory standing where the second
// planned file goes) aborts the whole plan with zero writes. The first
// file is fresh on purpose: the old sequential loop would have created
// it before hitting the directory, so its absence proves validation ran
// first.
func TestApplyPrevalidationWritesNothing(t *testing.T) {
	// Not parallel with the seam test below (shares the package), and
	// the assertion is order-sensitive by design.
	root := t.TempDir()
	seedFile(t, root, "keep.txt", "sentinel\n")
	// A directory wearing the second target's name: readExisting rejects
	// non-regular targets during validation on every platform.
	if err := os.MkdirAll(filepath.Join(root, "sub", "file.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	plan := generate.Plan{Files: []generate.FileOp{
		{Path: "a.txt", Content: []byte("new-a\n"), JSON: false},
		{Path: "sub/file.txt", Content: []byte("new-sub\n"), JSON: false},
	}}

	_, err := Apply(root, plan, Options{})
	if err == nil {
		t.Fatal("Apply() expected the directory-conflict error, got nil")
	}
	var manifest *ApplyError
	if errors.As(err, &manifest) {
		t.Fatalf("Apply() returned a mid-plan manifest for a detectable failure, want a plain zero-write error: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "a.txt")); !os.IsNotExist(statErr) {
		t.Errorf("Apply() created a.txt before the detectable failure, want zero writes")
	}
	if got, _ := os.ReadFile(filepath.Join(root, "keep.txt")); string(got) != "sentinel\n" {
		t.Errorf("Apply() touched the sentinel file, want zero writes")
	}
	if anyBackup(t, root) {
		t.Error("Apply() left backup siblings for writes that never happened, want zero writes")
	}
	if fi, statErr := os.Stat(filepath.Join(root, "sub", "file.txt")); statErr != nil || !fi.IsDir() {
		t.Errorf("Apply() disturbed the conflicting directory, want it intact: %v", statErr)
	}
}

// TestApplyFailureManifestListsAppliedFailedPending proves T2: a write
// that only fails mid-plan (injected disk-full on the second target)
// aborts with an *ApplyError whose applied[]/failed/pending[] lists are
// exact, and the backup of the already-applied overwrite is intact.
func TestApplyFailureManifestListsAppliedFailedPending(t *testing.T) {
	// Not parallel: stubs the package-wide writeFileFunc seam.
	root := t.TempDir()
	seedFile(t, root, "a.txt", "old-a\n")
	plan := generate.Plan{Files: []generate.FileOp{
		{Path: "a.txt", Content: []byte("new-a\n"), JSON: false},
		{Path: "b.txt", Content: []byte("new-b\n"), JSON: false},
		{Path: "c.txt", Content: []byte("new-c\n"), JSON: false},
	}}

	orig := writeFileFunc
	failTarget := filepath.Join(root, filepath.FromSlash("b.txt"))
	writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
		if path == failTarget {
			return errors.New("boom: disk full")
		}
		return orig(path, data, perm)
	}
	t.Cleanup(func() { writeFileFunc = orig })

	_, err := Apply(root, plan, Options{})
	if err == nil {
		t.Fatal("Apply() expected the injected write failure, got nil")
	}
	var manifest *ApplyError
	if !errors.As(err, &manifest) {
		t.Fatalf("Apply() error type = %T, want *ApplyError with the manifest", err)
	}
	if len(manifest.Applied) != 1 || manifest.Applied[0] != "a.txt" {
		t.Errorf("Apply() applied = %v, want [a.txt]", manifest.Applied)
	}
	if manifest.Failed != "b.txt" {
		t.Errorf("Apply() failed = %q, want b.txt", manifest.Failed)
	}
	if manifest.Cause == nil || !strings.Contains(manifest.Cause.Error(), "boom") {
		t.Errorf("Apply() cause = %v, want it to carry the injected failure", manifest.Cause)
	}
	if len(manifest.Pending) != 1 || manifest.Pending[0] != "c.txt" {
		t.Errorf("Apply() pending = %v, want [c.txt]", manifest.Pending)
	}
	if !strings.Contains(manifest.Error(), "b.txt") {
		t.Errorf("Apply() Error() = %q, want it to name the failed file", manifest.Error())
	}
	// Applied file holds the new bytes; failed and pending were never
	// created.
	if got, rerr := os.ReadFile(filepath.Join(root, "a.txt")); rerr != nil || string(got) != "new-a\n" {
		t.Errorf("Apply() a.txt = %q, %v; want the applied new bytes", got, rerr)
	}
	if _, statErr := os.Stat(filepath.Join(root, "b.txt")); !os.IsNotExist(statErr) {
		t.Error("Apply() left b.txt behind the failed write, want it absent")
	}
	if _, statErr := os.Stat(filepath.Join(root, "c.txt")); !os.IsNotExist(statErr) {
		t.Error("Apply() wrote pending c.txt after the abort, want it absent")
	}
	// The overwrite backup is the recovery story: it exists and holds
	// the pre-write image.
	bak, ok := manifest.Backups["a.txt"]
	if !ok || bak == "" {
		t.Fatalf("Apply() manifest omits the backup entry for applied a.txt: %v", manifest.Backups)
	}
	raw, rerr := os.ReadFile(filepath.Join(root, filepath.FromSlash(bak)))
	if rerr != nil {
		t.Fatal(rerr)
	}
	if string(raw) != "old-a\n" {
		t.Errorf("Apply() backup of a.txt = %q, want the pre-write image", raw)
	}
}
