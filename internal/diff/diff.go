// Package diff previews a generate.Plan against the files on disk.
//
// Compute reads each planned file, resolves the desired bytes
// (comment-preserving merge.Merge for existing JSON targets, generated
// bytes verbatim otherwise), and returns a typed Result with one unified
// diff per changed file. It writes nothing: the pending-changes signal
// travels as Result.HasChanges so the Cobra layer (a later slice) can map
// it to its preview exit status.
package diff

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"

	"github.com/Gabox301/Stackup/internal/generate"
	"github.com/Gabox301/Stackup/internal/merge"
)

// Options tunes how desired bytes are resolved.
type Options struct {
	// Force resolves existing JSON targets through the overwrite path
	// (merge.Apply with one SetKey op per generated top-level key, so
	// generated values win) instead of the default union-no-delete merge.
	// It previews the future --force behavior; the default stays Merge.
	Force bool
}

// FileDiff is the preview of one planned file against disk.
type FileDiff struct {
	Path    string // slash-separated relative path, as in generate.FileOp
	IsNew   bool   // file does not exist on disk yet
	Changed bool   // desired bytes differ from disk
	Unified string // unified diff, empty when unchanged
	Content []byte // desired bytes, set only when Changed
}

// Result is the preview of a whole plan. It carries no exit codes: the
// CLI layer maps HasChanges to its preview status.
type Result struct {
	Files []FileDiff
}

// HasChanges reports whether any planned file differs from disk.
func (r Result) HasChanges() bool {
	for _, f := range r.Files {
		if f.Changed {
			return true
		}
	}
	return false
}

// Compute resolves every file in p against root and returns the preview.
// It performs filesystem reads only and never writes.
func Compute(root string, p generate.Plan, opts Options) (Result, error) {
	res := Result{}
	for _, op := range p.Files {
		fd, err := diffOne(root, op, opts)
		if err != nil {
			return Result{}, err
		}
		res.Files = append(res.Files, fd)
	}
	return res, nil
}

// diffOne previews a single planned file against disk.
func diffOne(root string, op generate.FileOp, opts Options) (FileDiff, error) {
	abs := filepath.Join(root, filepath.FromSlash(op.Path))
	existing, err := os.ReadFile(abs)
	if err != nil {
		if !os.IsNotExist(err) {
			return FileDiff{}, fmt.Errorf("diff: read %s: %w", op.Path, err)
		}
		return FileDiff{
			Path:    op.Path,
			IsNew:   true,
			Changed: true,
			Unified: unified(op.Path, nil, op.Content),
			Content: op.Content,
		}, nil
	}
	desired, err := resolve(existing, op, opts)
	if err != nil {
		return FileDiff{}, err
	}
	if bytes.Equal(existing, desired) {
		return FileDiff{Path: op.Path}, nil
	}
	return FileDiff{
		Path:    op.Path,
		Changed: true,
		Unified: unified(op.Path, existing, desired),
		Content: desired,
	}, nil
}

// resolve returns the bytes apply would write for op over existing.
func resolve(existing []byte, op generate.FileOp, opts Options) ([]byte, error) {
	if !op.JSON {
		return op.Content, nil
	}
	if opts.Force {
		return overwrite(existing, op)
	}
	merged, err := merge.Merge(existing, op.Content)
	if err != nil {
		return nil, fmt.Errorf("diff: %s: %w", op.Path, err)
	}
	return merged, nil
}

// overwrite resolves through merge.Apply with one SetKey op per top-level
// generated key, so generated values replace user scalars while unknown
// user keys and comments survive. A non-object generated document falls
// back to verbatim bytes; every path stays backed up by the apply layer.
func overwrite(existing []byte, op generate.FileOp) ([]byte, error) {
	doc, err := hujson.Parse(op.Content)
	if err != nil {
		return nil, fmt.Errorf("diff: %s: parse generated: %w", op.Path, err)
	}
	obj, ok := doc.Value.(*hujson.Object)
	if !ok {
		return op.Content, nil
	}
	ops := make([]merge.EditOp, 0, len(obj.Members))
	for i := range obj.Members {
		m := &obj.Members[i]
		name, ok := memberName(m)
		if !ok {
			return nil, fmt.Errorf("diff: %s: generated member %d has no string name", op.Path, i)
		}
		ops = append(ops, merge.EditOp{
			Path:  "/" + escapePointer(name),
			Kind:  merge.OpSetKey,
			Value: m.Value.Pack(),
		})
	}
	out, err := merge.Apply(existing, ops)
	if err != nil {
		return nil, fmt.Errorf("diff: %s: %w", op.Path, err)
	}
	return out, nil
}

// memberName returns the unescaped name of an object member.
func memberName(m *hujson.ObjectMember) (string, bool) {
	lit, ok := m.Name.Value.(hujson.Literal)
	if !ok {
		return "", false
	}
	return lit.String(), true
}

// escapePointer escapes a single RFC 6901 path segment.
func escapePointer(seg string) string {
	seg = strings.ReplaceAll(seg, "~", "~0")
	return strings.ReplaceAll(seg, "/", "~1")
}

// diffContext is the number of unchanged lines kept around a change.
const diffContext = 3

// unified renders oldBytes vs newBytes as a single-hunk unified diff.
func unified(path string, oldBytes, newBytes []byte) string {
	oldLines := splitLines(string(oldBytes))
	newLines := splitLines(string(newBytes))
	pre := 0
	for pre < len(oldLines) && pre < len(newLines) && oldLines[pre] == newLines[pre] {
		pre++
	}
	suf := 0
	for suf < len(oldLines)-pre && suf < len(newLines)-pre &&
		oldLines[len(oldLines)-1-suf] == newLines[len(newLines)-1-suf] {
		suf++
	}
	if len(oldLines) == pre+0 && len(newLines) == pre+0 && suf == 0 {
		// No line-level difference (whitespace-only change): show the
		// whole file so the preview never renders an empty change.
		pre, suf = 0, 0
	}
	cb := min(diffContext, pre)
	ca := min(diffContext, suf)
	oldStart := pre - cb
	oldCount := len(oldLines) - suf + ca - oldStart
	newStart := pre - cb
	newCount := len(newLines) - suf + ca - newStart
	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", path, path)
	fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", oldStart+1, oldCount, newStart+1, newCount)
	for _, l := range oldLines[oldStart:pre] {
		b.WriteString(" ")
		b.WriteString(l)
		b.WriteString("\n")
	}
	for _, l := range oldLines[pre : len(oldLines)-suf] {
		b.WriteString("-")
		b.WriteString(l)
		b.WriteString("\n")
	}
	for _, l := range newLines[pre : len(newLines)-suf] {
		b.WriteString("+")
		b.WriteString(l)
		b.WriteString("\n")
	}
	for _, l := range oldLines[len(oldLines)-suf : len(oldLines)-suf+ca] {
		b.WriteString(" ")
		b.WriteString(l)
		b.WriteString("\n")
	}
	return b.String()
}

// splitLines splits s into lines without keeping a trailing empty element
// for a final newline.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
