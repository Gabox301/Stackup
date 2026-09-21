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

// maxHunkLines caps the body of any single emitted hunk. Denser change
// groups split into sequential hunks, so no hunk ever grows into a
// whole-file dump.
const maxHunkLines = 200

// maxExactLines bounds the exact Myers path: inputs with more combined
// lines take the coarser trimmed path, so a pathological preview cannot
// blow up time or memory. Config files never approach this bound.
const maxExactLines = 10000

// maxMyersDistance bails an exact alignment back to the coarse path once
// the edit distance proves the inputs are mostly unrelated.
const maxMyersDistance = 2000

// noNewlineMarker flags a hunk line that lacks its trailing newline.
const noNewlineMarker = "\\ No newline at end of file"

// op is one aligned line in the edit script.
type op struct {
	kind   byte // ' ' (unchanged), '-' (old only), '+' (new only)
	oldIdx int  // line index for ' ' and '-'
	newIdx int  // line index for ' ' and '+'
}

// unified renders oldBytes vs newBytes as a bounded multi-hunk unified
// diff. Nearby changes share one hunk with diffContext lines of context;
// changes separated by more than twice that split into hunks, so distant
// edits never merge into a giant single hunk.
func unified(path string, oldBytes, newBytes []byte) string {
	oldLines := splitLines(string(oldBytes))
	newLines := splitLines(string(newBytes))
	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", path, path)
	oldNL := hasTrailingNewline(oldBytes)
	newNL := hasTrailingNewline(newBytes)
	render := func(ops []op) bool {
		hunks := groupHunks(ops)
		if len(hunks) == 0 {
			return false
		}
		for _, h := range hunks {
			writeHunk(&b, ops, h[0], h[1], oldLines, newLines, oldNL, newNL)
		}
		return true
	}
	if len(oldLines)+len(newLines) <= maxExactLines {
		if ops, ok := diffOps(oldLines, newLines); ok {
			if render(ops) {
				return b.String()
			}
			// No line-level difference, but the caller only renders
			// changed files: the bytes differ in trailing newline
			// alone. Show the final line with an explicit marker
			// instead of a whole-file hunk.
			writeEOFNewlineHunk(&b, oldLines, oldNL, newNL)
			return b.String()
		}
	}
	// Enormous or mostly-unrelated inputs: trim the common prefix and
	// suffix, then render the remainder as sequential bounded hunks.
	// Coarser than Myers hunks, but every line stays covered exactly
	// once in order, so the hunks still apply cleanly.
	if render(coarseOps(oldLines, newLines)) {
		return b.String()
	}
	writeEOFNewlineHunk(&b, oldLines, oldNL, newNL)
	return b.String()
}

// diffOps aligns oldLines with newLines via the Myers greedy algorithm
// and returns the edit script. It reports false when the edit distance
// exceeds maxMyersDistance so the caller can take the coarse path.
func diffOps(oldLines, newLines []string) ([]op, bool) {
	n, m := len(oldLines), len(newLines)
	if n == 0 && m == 0 {
		return nil, true
	}
	max := n + m
	off := max
	v := make([]int, 2*max+1)
	var trace [][]int
	done := false
	var d int
	for d = 0; d <= max; d++ {
		if d > maxMyersDistance {
			return nil, false
		}
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[k-1+off] < v[k+1+off]) {
				x = v[k+1+off]
			} else {
				x = v[k-1+off] + 1
			}
			y := x - k
			for x < n && y < m && oldLines[x] == newLines[y] {
				x++
				y++
			}
			v[k+off] = x
			if x >= n && y >= m {
				done = true
				break
			}
		}
		snap := make([]int, len(v))
		copy(snap, v)
		trace = append(trace, snap)
		if done {
			break
		}
	}
	ops := make([]op, 0, n+m)
	x, y := n, m
	for dd := len(trace) - 1; dd > 0; dd-- {
		pv := trace[dd-1]
		k := x - y
		var prevK int
		if k == -dd || (k != dd && pv[k-1+off] < pv[k+1+off]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}
		prevX := pv[prevK+off]
		prevY := prevX - prevK
		for x > prevX && y > prevY {
			x--
			y--
			ops = append(ops, op{kind: ' ', oldIdx: x, newIdx: y})
		}
		if x == prevX {
			y--
			ops = append(ops, op{kind: '+', newIdx: y})
		} else {
			x--
			ops = append(ops, op{kind: '-', oldIdx: x})
		}
	}
	for x > 0 && y > 0 {
		x--
		y--
		ops = append(ops, op{kind: ' ', oldIdx: x, newIdx: y})
	}
	for i, j := 0, len(ops)-1; i < j; i, j = i+1, j-1 {
		ops[i], ops[j] = ops[j], ops[i]
	}
	return ops, true
}

// groupHunks splits ops into [lo,hi) ranges: runs of changes joined while
// separated by at most twice the context, each flanked by up to
// diffContext unchanged lines.
func groupHunks(ops []op) [][2]int {
	var changeIdx []int
	for i, o := range ops {
		if o.kind != ' ' {
			changeIdx = append(changeIdx, i)
		}
	}
	if len(changeIdx) == 0 {
		return nil
	}
	span := func(start, end int) [2]int {
		lo := start - diffContext
		if lo < 0 {
			lo = 0
		}
		hi := end + diffContext + 1
		if hi > len(ops) {
			hi = len(ops)
		}
		return [2]int{lo, hi}
	}
	var hunks [][2]int
	start, prev := changeIdx[0], changeIdx[0]
	for _, idx := range changeIdx[1:] {
		if idx-prev-1 > 2*diffContext {
			hunks = append(hunks, span(start, prev))
			start = idx
		}
		prev = idx
	}
	return append(hunks, span(start, prev))
}

// coarseOps aligns the common prefix/suffix exactly and marks the whole
// remainder changed, without an inner alignment pass.
func coarseOps(oldLines, newLines []string) []op {
	pre := 0
	for pre < len(oldLines) && pre < len(newLines) && oldLines[pre] == newLines[pre] {
		pre++
	}
	suf := 0
	for suf < len(oldLines)-pre && suf < len(newLines)-pre &&
		oldLines[len(oldLines)-1-suf] == newLines[len(newLines)-1-suf] {
		suf++
	}
	ops := make([]op, 0, len(oldLines)+len(newLines))
	for i := 0; i < pre; i++ {
		ops = append(ops, op{kind: ' ', oldIdx: i, newIdx: i})
	}
	for i := pre; i < len(oldLines)-suf; i++ {
		ops = append(ops, op{kind: '-', oldIdx: i})
	}
	for i := pre; i < len(newLines)-suf; i++ {
		ops = append(ops, op{kind: '+', newIdx: i})
	}
	for i := 0; i < suf; i++ {
		ops = append(ops, op{kind: ' ', oldIdx: len(oldLines) - suf + i, newIdx: len(newLines) - suf + i})
	}
	return ops
}

// writeHunk renders ops[lo:hi] as one or more hunks, splitting bodies
// longer than maxHunkLines into sequential pieces so no emitted hunk is
// giant. Splits preserve order and cover every line exactly once, so the
// pieces still apply cleanly.
func writeHunk(b *strings.Builder, ops []op, lo, hi int, oldLines, newLines []string, oldNL, newNL bool) {
	for lo < hi {
		end := lo + maxHunkLines
		if end > hi {
			end = hi
		}
		writeHunkPiece(b, ops, lo, end, oldLines, newLines, oldNL, newNL)
		lo = end
	}
}

// writeHunkPiece renders ops[lo:end] as a single @@ hunk.
func writeHunkPiece(b *strings.Builder, ops []op, lo, end int, oldLines, newLines []string, oldNL, newNL bool) {
	oldStart, newStart := 0, 0
	for _, o := range ops[:lo] {
		if o.kind != '+' {
			oldStart++
		}
		if o.kind != '-' {
			newStart++
		}
	}
	oldCount, newCount := 0, 0
	for _, o := range ops[lo:end] {
		if o.kind != '+' {
			oldCount++
		}
		if o.kind != '-' {
			newCount++
		}
	}
	// Empty ranges address the line before the hunk, not the line after.
	oldRef, newRef := oldStart+1, newStart+1
	if oldCount == 0 {
		oldRef = oldStart
	}
	if newCount == 0 {
		newRef = newStart
	}
	fmt.Fprintf(b, "@@ -%d,%d +%d,%d @@\n", oldRef, oldCount, newRef, newCount)
	for _, o := range ops[lo:end] {
		switch o.kind {
		case ' ':
			b.WriteString(" " + oldLines[o.oldIdx] + "\n")
		case '-':
			b.WriteString("-" + oldLines[o.oldIdx] + "\n")
			if o.oldIdx == len(oldLines)-1 && !oldNL {
				b.WriteString(noNewlineMarker + "\n")
			}
		case '+':
			b.WriteString("+" + newLines[o.newIdx] + "\n")
			if o.newIdx == len(newLines)-1 && !newNL {
				b.WriteString(noNewlineMarker + "\n")
			}
		}
	}
}

// writeEOFNewlineHunk renders a trailing-newline-only change: the final
// line on both sides with a marker on the side missing the newline.
func writeEOFNewlineHunk(b *strings.Builder, oldLines []string, oldNL, newNL bool) {
	if len(oldLines) == 0 {
		return
	}
	n := len(oldLines)
	fmt.Fprintf(b, "@@ -%d,1 +%d,1 @@\n", n, n)
	last := oldLines[n-1]
	b.WriteString("-" + last + "\n")
	if !oldNL {
		b.WriteString(noNewlineMarker + "\n")
	}
	b.WriteString("+" + last + "\n")
	if !newNL {
		b.WriteString(noNewlineMarker + "\n")
	}
}

// hasTrailingNewline reports whether raw ends in a newline. Empty input
// counts as missing, matching splitLines rendering it as zero lines.
func hasTrailingNewline(raw []byte) bool {
	return len(raw) > 0 && raw[len(raw)-1] == '\n'
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
