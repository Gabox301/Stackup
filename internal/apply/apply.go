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
package apply

import (
	"fmt"
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
// returning the backup relative path ("" for fresh files).
func writeOne(root string, fd diff.FileDiff) (string, error) {
	abs := filepath.Join(root, filepath.FromSlash(fd.Path))
	var backup string
	if _, err := os.Stat(abs); err == nil {
		var err error
		backup, err = backupFile(root, abs)
		if err != nil {
			return "", err
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("apply: stat %s: %w", fd.Path, err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", fmt.Errorf("apply: create parent of %s: %w", fd.Path, err)
	}
	if err := os.WriteFile(abs, fd.Content, 0o644); err != nil {
		return "", fmt.Errorf("apply: write %s: %w", fd.Path, err)
	}
	return backup, nil
}

// backupFile copies abs to a timestamped sibling and returns the
// root-relative backup path. Same-second reruns dedupe with a counter.
func backupFile(root, abs string) (string, error) {
	original, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("apply: read for backup: %w", err)
	}
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
	if err := os.WriteFile(candidate, original, 0o644); err != nil {
		return "", fmt.Errorf("apply: write backup: %w", err)
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return "", fmt.Errorf("apply: relativize backup: %w", err)
	}
	return filepath.ToSlash(rel), nil
}
