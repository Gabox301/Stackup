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

// ApplyError reports a mid-plan write failure with the failure manifest:
// what was already written, what failed and why, and what never started.
// Pre-validation (see validatePlan) catches every detectable failure with
// zero writes; ApplyError is the backstop for failures that only surface
// mid-plan (I/O errors, vanished directories, permission changes between
// validation and write). There is no auto-rollback: the per-file .bak
// backups recorded in Backups are the recovery story.
type ApplyError struct {
	// Applied lists relative paths written before the failure, plan order.
	Applied []string
	// Backups maps each overwritten path in Applied to its timestamped
	// backup relative path. Fresh files have no entry.
	Backups map[string]string
	// Failed is the relative path whose write aborted the run.
	Failed string
	// Cause is the underlying write failure.
	Cause error
	// Pending lists relative paths never attempted, in plan order.
	Pending []string
}

// Error preserves the underlying write failure text so callers matching
// on today's messages see no change; the manifest travels on the typed
// error itself (use errors.As to read it).
func (e *ApplyError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return "apply: write " + e.Failed + ": unknown error"
}

// Unwrap exposes the underlying write failure.
func (e *ApplyError) Unwrap() error { return e.Cause }

// timeFormat stamps backups without characters illegal on Windows.
const timeFormat = "20060102-150405"

// Filesystem seams for failure-injection tests. Production always uses
// the os defaults; white-box tests stub writeFileFunc to force a
// deterministic mid-plan failure without relying on OS permission bits
// (unreliable on Windows and when running as root/administrator).
var (
	mkdirAllFunc  = os.MkdirAll
	writeFileFunc = os.WriteFile
	chmodFunc     = os.Chmod
)

// Apply resolves p against root and writes pending files. The whole plan
// is resolved (diff.Compute) and pre-validated (validatePlan) BEFORE the
// first write, so anything detectable fails with zero writes. A failure
// that still surfaces mid-plan aborts the run and returns an *ApplyError
// carrying the applied[]/failed/pending[] manifest. Existing files are
// backed up before any write; a backup failure aborts the run before
// the corresponding write. DryRun returns the pending set and writes
// nothing (validation is skipped: there is nothing to protect).
func Apply(root string, p generate.Plan, opts Options) (Result, error) {
	preview, err := diff.Compute(root, p, diff.Options{Force: opts.Force})
	if err != nil {
		return Result{}, err
	}
	res := Result{Backups: map[string]string{}, DryRun: opts.DryRun}
	var changed []diff.FileDiff
	for _, fd := range preview.Files {
		if !fd.Changed {
			continue
		}
		changed = append(changed, fd)
	}
	if opts.DryRun {
		for _, fd := range changed {
			res.Written = append(res.Written, fd.Path)
		}
		return res, nil
	}
	// Resolve-then-commit: the ENTIRE plan validates before the first
	// byte is written. A validation failure returns a plain error with
	// zero writes (no manifest: nothing was applied, nothing is pending
	// beyond the whole plan).
	if err := validatePlan(root, changed); err != nil {
		return Result{}, err
	}
	for i, fd := range changed {
		backup, err := writeOne(root, fd)
		if err != nil {
			applied := append([]string{}, res.Written...)
			backups := make(map[string]string, len(res.Backups))
			for k, v := range res.Backups {
				backups[k] = v
			}
			var pending []string
			for _, rest := range changed[i+1:] {
				pending = append(pending, rest.Path)
			}
			return Result{}, &ApplyError{
				Applied: applied,
				Backups: backups,
				Failed:  fd.Path,
				Cause:   err,
				Pending: pending,
			}
		}
		res.Written = append(res.Written, fd.Path)
		if backup != "" {
			res.Backups[fd.Path] = backup
		}
	}
	return res, nil
}

// validatePlan checks every pending write before the first one happens:
// each target must be readable (or absent) and regular, and each parent
// directory must exist as a writable directory or be creatable under a
// writable existing ancestor. Any failure aborts with zero writes.
func validatePlan(root string, changed []diff.FileDiff) error {
	if len(changed) == 0 {
		return nil
	}
	if fi, err := os.Stat(root); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("apply: stat %s: %w", root, err)
		}
		// Missing root is creatable: the per-file parent checks below
		// verify the nearest existing ancestor is a writable directory.
	} else if !fi.IsDir() {
		return fmt.Errorf("apply: %s: not a directory", root)
	}
	for _, fd := range changed {
		abs := filepath.Join(root, filepath.FromSlash(fd.Path))
		if _, _, _, err := readExisting(abs, fd.Path); err != nil {
			return err
		}
		if err := validateParentDir(abs, fd.Path); err != nil {
			return err
		}
	}
	return nil
}

// validateParentDir ensures the parent of a pending target either exists
// as a writable directory or can be created under the nearest existing
// ancestor (which must itself be a writable directory). A file blocking
// the path fails instead of surfacing later as a half-written plan.
// Writability is judged by permission bits (any of owner/group/other
// write): a portable, side-effect-free check. ACLs, quotas, and disk-full
// cannot be detected this way and remain mid-plan manifest cases.
func validateParentDir(abs, rel string) error {
	cur := filepath.Dir(abs)
	for {
		fi, err := os.Stat(cur)
		if err == nil {
			if !fi.IsDir() {
				return fmt.Errorf("apply: parent of %s: %s is not a directory", rel, cur)
			}
			if fi.Mode().Perm()&0o222 == 0 {
				return fmt.Errorf("apply: parent of %s: directory %s is not writable", rel, cur)
			}
			return nil
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("apply: stat %s: %w", cur, err)
		}
		up := filepath.Dir(cur)
		if up == cur {
			return fmt.Errorf("apply: parent of %s: no creatable directory", rel)
		}
		cur = up
	}
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
	if err := mkdirAllFunc(filepath.Dir(abs), 0o755); err != nil {
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
	if err := writeFileFunc(abs, fd.Content, perm); err != nil {
		return "", fmt.Errorf("apply: write %s: %w", fd.Path, err)
	}
	if ok {
		// WriteFile applies perm only on creation; truncating an
		// existing file keeps whatever mode it had, so enforce the
		// captured source mode explicitly.
		if err := chmodFunc(abs, mode); err != nil {
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
	if err := writeFileFunc(candidate, original, mode); err != nil {
		return "", fmt.Errorf("apply: write backup: %w", err)
	}
	if err := chmodFunc(candidate, mode); err != nil {
		return "", fmt.Errorf("apply: preserve mode of backup: %w", err)
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return "", fmt.Errorf("apply: relativize backup: %w", err)
	}
	return filepath.ToSlash(rel), nil
}
