// Package apply writes a generate.Plan to disk with mandatory pre-write backups.
//
// Every overwritten file is first copied to a timestamped sibling
// (<name>.bak.<ts>); fresh files need no backup. There is no exported
// write path that skips the backup: an apply without backup is
// structurally forbidden. The default write resolves existing JSON
// targets through merge.Merge (union-no-delete); the overwrite path
// (merge.Apply with SetKey ops, the future --force behavior) is selected
// only via Options.Force. DryRun resolves the same bytes but writes
// nothing, backing the --dry-run preview.
//
// Backup retention: stackup never prunes backups. Same-second reruns
// dedupe with a numeric suffix, so repeated applies accumulate
// <name>.bak.<ts>[-N] siblings next to the target; delete them yourself
// when the pre-write image no longer matters. Overwrites preserve the
// source file mode on both the target and its backup, so restrictive
// permissions never widen through the backup copy; fresh files are
// created 0644 (umask applies).
package apply

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Gabox301/Stackup/internal/diff"
	"github.com/Gabox301/Stackup/internal/generate"
)

// Options tunes an apply run.
type Options struct {
	// Force selects the overwrite resolution path (generated values win).
	Force bool
	// DryRun resolves pending writes and reports them without touching
	// disk: no files created, no backups taken.
	DryRun bool
}

// Result describes one apply run.
type Result struct {
	// Written lists relative paths created or updated, in plan order.
	Written []string
	// Backups maps each overwritten relative path to its timestamped
	// backup relative path. Fresh files have no entry.
	Backups map[string]string
	// DryRun echoes Options.DryRun: when true, nothing was written.
	DryRun bool
}

// timeFormat stamps backups without characters illegal on Windows.
const timeFormat = "20060102-150405"

// Apply resolves p against root and writes pending files. Existing files
// are backed up before any write; a backup failure aborts the run before
// the corresponding write. DryRun returns the pending set and writes
// nothing.
func Apply(root string, p generate.Plan, opts Options) (Result, error) {
	preview, err := diff.Compute(root, p, diff.Options{Force: opts.Force})
	if err != nil {
		return Result{}, err
	}
	res := Result{Backups: map[string]string{}, DryRun: opts.DryRun}
	for _, fd := range preview.Files {
		if !fd.Changed {
			continue
		}
		if opts.DryRun {
			res.Written = append(res.Written, fd.Path)
			continue
		}
		backup, err := writeOne(root, fd)
		if err != nil {
			return Result{}, err
		}
		res.Written = append(res.Written, fd.Path)
		if backup != "" {
			res.Backups[fd.Path] = backup
		}
	}
	return res, nil
}

// writeOne backs up an existing target and writes the resolved bytes,
// returning the backup relative path ("" for fresh files). The source is
// opened once and its mode and pre-write image come from that single
// open, so no stat-then-read race can swap the bytes between the check
// and the backup. Overwrites preserve the source mode on the target and
// the backup; fresh files are created 0644.
func writeOne(root string, fd diff.FileDiff) (string, error) {
	abs := filepath.Join(root, filepath.FromSlash(fd.Path))
	original, mode, ok, err := readExisting(abs, fd.Path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", fmt.Errorf("apply: create parent of %s: %w", fd.Path, err)
	}
	var backup string
	if ok {
		backup, err = backupFile(root, abs, original, mode)
		if err != nil {
			return "", err
		}
	}
	perm := os.FileMode(0o644)
	if ok {
		perm = mode
	}
	if err := os.WriteFile(abs, fd.Content, perm); err != nil {
		return "", fmt.Errorf("apply: write %s: %w", fd.Path, err)
	}
	if ok {
		// WriteFile applies perm only on creation; truncating an
		// existing file keeps whatever mode it had, so enforce the
		// captured source mode explicitly.
		if err := os.Chmod(abs, mode); err != nil {
			return "", fmt.Errorf("apply: preserve mode of %s: %w", fd.Path, err)
		}
	}
	return backup, nil
}

// readExisting opens an existing target once and returns its bytes, its
// permission bits, and true. A missing target reports ok=false with no
// error so the caller takes the fresh-file path; any other open, stat,
// or read failure aborts before anything is written. Non-regular targets
// (e.g. a directory where a file is planned) fail instead of being
// backed up or truncated.
func readExisting(abs, rel string) (content []byte, mode os.FileMode, ok bool, err error) {
	f, err := os.Open(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, false, nil
		}
		return nil, 0, false, fmt.Errorf("apply: open %s: %w", rel, err)
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil {
		return nil, 0, false, fmt.Errorf("apply: stat %s: %w", rel, err)
	}
	if !fi.Mode().IsRegular() {
		return nil, 0, false, fmt.Errorf("apply: %s: not a regular file", rel)
	}
	raw, err := io.ReadAll(f)
	if err != nil {
		return nil, 0, false, fmt.Errorf("apply: read %s: %w", rel, err)
	}
	return raw, fi.Mode().Perm(), true, nil
}

// backupFile copies original (already read from abs under one open; see
// writeOne) to a timestamped sibling and returns the root-relative backup
// path. Same-second reruns dedupe with a counter suffix. The backup
// inherits the source mode so restrictive permissions never widen through
// the copy. Backups accumulate: stackup keeps every sibling and never
// prunes; remove <name>.bak.* yourself when the pre-write image no longer
// matters.
func backupFile(root, abs string, original []byte, mode os.FileMode) (string, error) {
	stamp := time.Now().UTC().Format(timeFormat)
	candidate := abs + ".bak." + stamp
	for i := 2; ; i++ {
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			break
		} else if err != nil {
			return "", fmt.Errorf("apply: stat backup: %w", err)
		}
		candidate = fmt.Sprintf("%s.bak.%s-%d", abs, stamp, i)
	}
	if err := os.WriteFile(candidate, original, mode); err != nil {
		return "", fmt.Errorf("apply: write backup: %w", err)
	}
	if err := os.Chmod(candidate, mode); err != nil {
		return "", fmt.Errorf("apply: preserve mode of backup: %w", err)
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return "", fmt.Errorf("apply: relativize backup: %w", err)
	}
	return filepath.ToSlash(rel), nil
}
