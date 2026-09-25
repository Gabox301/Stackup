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
// Backup retention: apply never prunes on its own. Same-second reruns
// dedupe with a numeric suffix, so repeated applies accumulate
// <name>.bak.<ts>[-N] siblings next to the target. Call PruneBackups to
// drop old siblings and RestoreBackups to recover the pre-write images
// recorded in a Result (or ApplyError) Backups manifest. Overwrites preserve the
// source file mode on both the target and its backup, so restrictive
// permissions never widen through the backup copy; fresh files are
// created 0644 (umask applies).
package apply

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	mkdirAllFunc   = os.MkdirAll
	writeFileFunc  = os.WriteFile
	chmodFunc      = os.Chmod
	removeFileFunc = os.Remove
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
// the copy. Backups accumulate: apply keeps every sibling until the caller
// drops old ones with PruneBackups.
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

// isBackupSuffix reports whether suffix is exactly the stamp shape
// backupFile appends after ".bak.": YYYYMMDD-HHMMSS with an optional -N
// dedupe counter. Anything else (notes, partial dates, trailing text) is
// a user file that must never be pruned or restored.
func isBackupSuffix(suffix string) bool {
	const stampLen = len("20060102-150405")
	if len(suffix) < stampLen {
		return false
	}
	stamp, rest := suffix[:stampLen], suffix[stampLen:]
	for i := 0; i < stampLen; i++ {
		c := stamp[i]
		if i == 8 {
			if c != '-' {
				return false
			}
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	if rest == "" {
		return true
	}
	if len(rest) < 2 || rest[0] != '-' {
		return false
	}
	for i := 1; i < len(rest); i++ {
		if rest[i] < '0' || rest[i] > '9' {
			return false
		}
	}
	return true
}

// ListBackups returns the timestamped backup siblings of target as
// root-relative slash paths, oldest first. Only siblings matching exactly
// the backupFile pattern qualify; user files that merely share the
// ".bak." prefix and non-regular files are skipped.
func ListBackups(root, target string) ([]string, error) {
	abs := filepath.Join(root, filepath.FromSlash(target))
	matches, err := filepath.Glob(abs + ".bak.*")
	if err != nil {
		return nil, fmt.Errorf("apply: list backups of %s: %w", target, err)
	}
	base := filepath.Base(abs)
	var out []string
	for _, m := range matches {
		suffix, ok := strings.CutPrefix(filepath.Base(m), base+".bak.")
		if !ok || !isBackupSuffix(suffix) {
			continue
		}
		fi, err := os.Stat(m)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("apply: stat backup of %s: %w", target, err)
		}
		if !fi.Mode().IsRegular() {
			continue
		}
		rel, err := filepath.Rel(root, m)
		if err != nil {
			return nil, fmt.Errorf("apply: relativize backup: %w", err)
		}
		out = append(out, filepath.ToSlash(rel))
	}
	sort.Strings(out)
	return out, nil
}

// PruneBackups deletes backup siblings of target, keeping the newest keep.
// A negative keep is an error; keep zero removes every backup sibling.
// Only exact-pattern siblings qualify, so user files are never deleted.
// It returns the deleted root-relative slash paths, oldest first; on a
// remove failure it returns the paths deleted so far with the error.
func PruneBackups(root, target string, keep int) ([]string, error) {
	if keep < 0 {
		return nil, fmt.Errorf("apply: prune %s: keep must be >= 0, got %d", target, keep)
	}
	listed, err := ListBackups(root, target)
	if err != nil {
		return nil, err
	}
	if len(listed) <= keep {
		return nil, nil
	}
	var deleted []string
	for _, rel := range listed[:len(listed)-keep] {
		if err := removeFileFunc(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			return deleted, fmt.Errorf("apply: prune %s: %w", rel, err)
		}
		deleted = append(deleted, rel)
	}
	return deleted, nil
}

// RestoreFile copies the backup sibling back over target, preserving the
// backup mode on the target. The backup must be an exact-pattern sibling
// of the target; anything else fails instead of copying an arbitrary file
// over user data.
func RestoreFile(root, target, backup string) error {
	targetAbs := filepath.Join(root, filepath.FromSlash(target))
	backupAbs := filepath.Join(root, filepath.FromSlash(backup))
	suffix, ok := strings.CutPrefix(filepath.Base(backupAbs), filepath.Base(targetAbs)+".bak.")
	if !ok || !isBackupSuffix(suffix) {
		return fmt.Errorf("apply: restore %s: %s is not a backup of %s", target, backup, target)
	}
	if filepath.Dir(backupAbs) != filepath.Dir(targetAbs) {
		return fmt.Errorf("apply: restore %s: backup %s is not a sibling", target, backup)
	}
	content, mode, ok, err := readExisting(backupAbs, backup)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("apply: restore %s: backup %s not found", target, backup)
	}
	if err := writeFileFunc(targetAbs, content, mode); err != nil {
		return fmt.Errorf("apply: restore %s: %w", target, err)
	}
	if err := chmodFunc(targetAbs, mode); err != nil {
		return fmt.Errorf("apply: preserve mode of %s: %w", target, err)
	}
	return nil
}

// RestoreBackups recovers every entry of a Backups manifest (Result or
// ApplyError) in deterministic target order, stopping at the first
// failure. The manifest itself is unchanged, so a failure can be retried
// after fixing the cause. A nil or empty manifest is a no-op.
func RestoreBackups(root string, backups map[string]string) error {
	targets := make([]string, 0, len(backups))
	for target := range backups {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	for _, target := range targets {
		if err := RestoreFile(root, target, backups[target]); err != nil {
			return err
		}
	}
	return nil
}
