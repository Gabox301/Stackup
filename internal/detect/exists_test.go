package detect

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

// stubStat replaces statFile for the duration of a test. Callers must NOT
// use t.Parallel: sequential tests complete before parallel ones resume,
// so the stub never leaks into the parallel detector suite.
func stubStat(t *testing.T, fn func(string) (os.FileInfo, error)) {
	t.Helper()
	old := statFile
	statFile = fn
	t.Cleanup(func() { statFile = old })
}

func TestExistsTable(t *testing.T) {
	stubStat(t, func(path string) (os.FileInfo, error) {
		switch filepath.Base(path) {
		case "package.json":
			return nil, &os.PathError{Op: "stat", Path: path, Err: syscall.EACCES}
		case "go.mod":
			return nil, &os.PathError{Op: "stat", Path: path, Err: syscall.ENOENT}
		}
		return nil, &os.PathError{Op: "stat", Path: path, Err: syscall.ENOENT}
	})

	if _, err := exists(t.TempDir(), "package.json"); err == nil {
		t.Error("exists() on permission-denied manifest returned nil error, want it surfaced")
	}
	if ok, err := exists(t.TempDir(), "go.mod"); err != nil || ok {
		t.Errorf("exists() on absent manifest = (%v, %v), want (false, nil)", ok, err)
	}
}

func TestExistsRequiresRegularFile(t *testing.T) {
	// Real filesystem, no stub: parallel-safe.
	t.Parallel()

	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "package.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := exists(dir, "package.json"); err != nil || ok {
		t.Errorf("exists() on directory manifest = (%v, %v), want (false, nil)", ok, err)
	}
	if ok, err := exists(dir, "go.mod"); err != nil || !ok {
		t.Errorf("exists() on regular manifest = (%v, %v), want (true, nil)", ok, err)
	}
}

func TestDetectSurfacesPermissionDenied(t *testing.T) {
	stubStat(t, func(path string) (os.FileInfo, error) {
		return nil, &os.PathError{Op: "stat", Path: path, Err: syscall.EACCES}
	})

	if _, err := Detect(t.TempDir(), true); err == nil {
		t.Fatal("Detect() on an unreadable tree succeeded, want the permission error (never an empty scan)")
	} else if errors.Is(err, ErrUnknownStack) {
		t.Errorf("Detect() returned %v, want the permission failure instead of unknown-stack", err)
	}
}

func TestDetectUnreadableRootSurfacesError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod search-permission fixtures do not fail on Windows; the stub test above covers the contract")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"demo"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil || !os.IsPermission(err) {
		t.Skip("platform does not enforce directory search permission (e.g. running as root)")
	}
	if _, err := Detect(dir, true); err == nil {
		t.Error("Detect() on an unreadable root succeeded, want the permission error")
	} else if errors.Is(err, ErrUnknownStack) {
		t.Errorf("Detect() returned %v, want the permission failure instead of unknown-stack", err)
	}
}

func TestDetectDirectoryManifestYieldsNothing(t *testing.T) {
	// A directory wearing a manifest name must not yield evidence.
	// Parallel-safe: real filesystem only, no stat stub.
	t.Parallel()

	for _, manifest := range []string{
		"package.json", "go.mod", "Cargo.toml", "pyproject.toml",
		"Gemfile", "rebar.config", "pom.xml", "build.gradle",
	} {
		t.Run(manifest, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, manifest), 0o755); err != nil {
				t.Fatal(err)
			}
			got, err := Detect(dir, true)
			if err != nil {
				t.Fatalf("Detect() unexpected error: %v", err)
			}
			if len(got) != 0 {
				t.Errorf("Detect() with %s as a directory returned %v, want no evidence", manifest, got)
			}
		})
	}
}
