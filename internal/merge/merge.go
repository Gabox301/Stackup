// Package merge patches JSONC user files through the hujson AST so comments,
// trailing commas, and unknown keys survive generation.
//
// Two entry points exist. Merge deep-merges a generated document into an
// existing one: objects merge recursively, arrays union without deleting
// user entries, and existing scalars always win. Apply executes an explicit
// list of EditOps for callers that need targeted overwrites (SetKey),
// array unions (UnionArray), or explicit no-ops (Noop).
//
// This package never imports encoding/json for user files and never calls
// Format on the write path: output is Value.Pack, which round-trips input
// bytes unchanged when nothing was merged.
package merge

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/tailscale/hujson"
)

// OpKind names the operation an EditOp performs.
type OpKind string

// Supported edit operations.
const (
	// OpSetKey overwrites the value at Path, creating missing objects
	// along the way. It is the only operation that replaces user data.
	OpSetKey OpKind = "SetKey"
	// OpUnionArray appends Value elements missing from the array at Path,
	// never removing existing entries.
	OpUnionArray OpKind = "UnionArray"
	// OpNoop changes nothing. It exists so callers can record an
	// explicitly reviewed decision to leave a path alone.
	OpNoop OpKind = "Noop"
)

// EditOp is one targeted patch against a JSONC document. Path is an
// RFC 6901 JSON pointer ("" addresses the document root). Value holds a JSON
// literal for SetKey, or a JSON array (or single JSON value) whose elements
// are unioned for UnionArray. Value is unused for Noop.
type EditOp struct {
	Path  string
	Kind  OpKind
	Value []byte
}

// Merge deep-merges generated into existing and returns the patched bytes.
// An empty existing document yields generated unchanged. Objects merge key
// by key, arrays union without deleting user entries, and existing scalars
// win over generated ones. Comments and trailing commas are preserved.
// Either input must be valid JSONC; an invalid existing document errors
// instead of risking user data.
func Merge(existing, generated []byte) ([]byte, error) {
	if len(bytes.TrimSpace(existing)) == 0 {
		return generated, nil
	}
	if len(bytes.TrimSpace(generated)) == 0 {
		return existing, nil
	}
	dst, err := hujson.Parse(existing)
	if err != nil {
		return nil, fmt.Errorf("merge: parse existing: %w", err)
	}
	src, err := hujson.Parse(generated)
	if err != nil {
		return nil, fmt.Errorf("merge: parse generated: %w", err)
	}
	deepMerge(&dst, &src)
	return dst.Pack(), nil
}

// Apply executes ops in order against existing and returns the patched
// bytes. An empty existing document starts from an empty object. The result
// is byte-identical to the input when every op is a Noop.
func Apply(existing []byte, ops []EditOp) ([]byte, error) {
	var doc hujson.Value
	if len(bytes.TrimSpace(existing)) == 0 {
		var err error
		doc, err = hujson.Parse([]byte("{}"))
		if err != nil {
			return nil, fmt.Errorf("merge: parse empty base: %w", err)
		}
	} else {
		var err error
		doc, err = hujson.Parse(existing)
		if err != nil {
			return nil, fmt.Errorf("merge: parse existing: %w", err)
		}
	}
	for i, op := range ops {
		if err := applyOne(&doc, op); err != nil {
			return nil, fmt.Errorf("merge: op %d (%s %q): %w", i, op.Kind, op.Path, err)
		}
	}
	return doc.Pack(), nil
}

// applyOne executes a single edit operation against the document root.
func applyOne(doc *hujson.Value, op EditOp) error {
	segs, err := splitPointer(op.Path)
	if err != nil {
		return err
	}
	switch op.Kind {
	case OpNoop:
		return nil
	case OpSetKey:
		if len(segs) == 0 {
			return fmt.Errorf("SetKey needs a member path, got document root")
		}
		frag, err := parseFragment(op.Value)
		if err != nil {
			return fmt.Errorf("parse value: %w", err)
		}
		parent, err := navigate(doc, segs[:len(segs)-1], true)
		if err != nil {
			return err
		}
		obj, ok := parent.Value.(*hujson.Object)
		if !ok {
			return fmt.Errorf("parent of %q is not an object", op.Path)
		}
		setKey(obj, segs[len(segs)-1], frag)
		return nil
	case OpUnionArray:
		frag, err := parseFragment(op.Value)
		if err != nil {
			return fmt.Errorf("parse value: %w", err)
		}
		items := unionItems(frag)
		if len(segs) == 0 {
			arr, ok := doc.Value.(*hujson.Array)
			if !ok {
				return fmt.Errorf("UnionArray at document root needs an array document")
			}
			unionArray(arr, items)
			return nil
		}
		parent, err := navigate(doc, segs[:len(segs)-1], true)
		if err != nil {
			return err
		}
		obj, ok := parent.Value.(*hujson.Object)
		if !ok {
			return fmt.Errorf("parent of %q is not an object", op.Path)
		}
		last := segs[len(segs)-1]
		if idx := findMember(obj, last); idx >= 0 {
			arr, ok := obj.Members[idx].Value.Value.(*hujson.Array)
			if !ok {
				return fmt.Errorf("target %q is not an array", op.Path)
			}
			unionArray(arr, items)
			return nil
		}
		arr := &hujson.Array{}
		unionArray(arr, items)
		appendMember(obj, last, hujson.Value{Value: arr})
		return nil
	default:
		return fmt.Errorf("unknown operation %q", op.Kind)
	}
}

// deepMerge folds src into dst: missing keys are added, nested objects
// recurse, arrays union, and existing scalars are left untouched.
func deepMerge(dst, src *hujson.Value) {
	dstObj, dstOk := dst.Value.(*hujson.Object)
	srcObj, srcOk := src.Value.(*hujson.Object)
	if !dstOk || !srcOk {
		if dstArr, ok := dst.Value.(*hujson.Array); ok {
			if srcArr, ok := src.Value.(*hujson.Array); ok {
				unionArray(dstArr, srcArr.Elements)
			}
		}
		return
	}
	for i := range srcObj.Members {
		sm := &srcObj.Members[i]
		name := memberName(sm)
		idx := findMember(dstObj, name)
		if idx < 0 {
			appendMemberClone(dstObj, sm)
			continue
		}
		dv := &dstObj.Members[idx].Value
		deepMerge(dv, &sm.Value)
	}
}

// unionItems normalizes a UnionArray payload: an array contributes its
// elements, any other value contributes itself as a single item.
func unionItems(frag hujson.Value) []hujson.Value {
	if arr, ok := frag.Value.(*hujson.Array); ok {
		return arr.Elements
	}
	return []hujson.Value{frag}
}

// unionArray appends items missing from arr, preserving existing entries,
// order, comments, and trailing commas.
func unionArray(arr *hujson.Array, items []hujson.Value) {
	multiline := isMultilineArray(arr)
	wasEmpty := len(arr.Elements) == 0
	var indent string
	if multiline {
		indent = inferArrayIndent(arr)
	}
	for _, item := range items {
		if containsValue(arr.Elements, item) {
			continue
		}
		next := item.Clone()
		if multiline && !hasNewline(next.BeforeExtra) {
			next.BeforeExtra = append([]byte{'\n'}, []byte(indent)...)
		}
		arr.Elements = append(arr.Elements, next)
	}
	if wasEmpty && len(arr.Elements) > 0 && len(arr.AfterExtra) == 0 {
		arr.AfterExtra = []byte{'\n'}
	}
}

// containsValue reports whether list already holds a value equal to want.
// Comparison runs on Standardized clones, which is a read-only bridge:
// the originals are never reformatted.
func containsValue(list []hujson.Value, want hujson.Value) bool {
	for _, have := range list {
		if valuesEqual(have, want) {
			return true
		}
	}
	return false
}

// valuesEqual compares two values by their standardized JSON form.
func valuesEqual(a, b hujson.Value) bool {
	ac := a.Clone()
	bc := b.Clone()
	ac.Standardize()
	bc.Standardize()
	return bytes.Equal(ac.Pack(), bc.Pack())
}

// setKey replaces the named member or appends it when missing. Replacing
// keeps the existing member position and surrounding comments.
func setKey(obj *hujson.Object, name string, frag hujson.Value) {
	trimmed := frag.Clone().Value
	if idx := findMember(obj, name); idx >= 0 {
		obj.Members[idx].Value.Value = trimmed
		return
	}
	appendMember(obj, name, hujson.Value{Value: trimmed})
}

// appendMember adds a new member with indentation matching its siblings.
func appendMember(obj *hujson.Object, name string, val hujson.Value) {
	if len(val.BeforeExtra) == 0 {
		val.BeforeExtra = append([]byte{'\n'}, []byte(inferObjectIndent(obj))...)
	}
	member := hujson.ObjectMember{
		Name:  hujson.Value{Value: hujson.String(name)},
		Value: val,
	}
	member.Name.BeforeExtra = val.BeforeExtra
	member.Value.BeforeExtra = nil
	if len(member.Value.BeforeExtra) == 0 {
		member.Value.BeforeExtra = []byte(" ")
	}
	obj.Members = append(obj.Members, member)
	if len(obj.AfterExtra) == 0 {
		obj.AfterExtra = []byte{'\n'}
	}
}

// appendMemberClone copies a generated member into dst, keeping the
// generated formatting for the new key.
func appendMemberClone(dst *hujson.Object, src *hujson.ObjectMember) {
	next := hujson.ObjectMember{
		Name:  src.Name.Clone(),
		Value: src.Value.Clone(),
	}
	dst.Members = append(dst.Members, next)
	if len(dst.AfterExtra) == 0 && len(src.Value.AfterExtra) == 0 {
		dst.AfterExtra = []byte{'\n'}
	}
}

// navigate walks segs from root, optionally creating missing objects.
func navigate(root *hujson.Value, segs []string, create bool) (*hujson.Value, error) {
	cur := root
	for _, seg := range segs {
		obj, ok := cur.Value.(*hujson.Object)
		if !ok {
			return nil, fmt.Errorf("segment %q: not an object", seg)
		}
		idx := findMember(obj, seg)
		if idx < 0 {
			if !create {
				return nil, fmt.Errorf("segment %q: not found", seg)
			}
			appendMember(obj, seg, hujson.Value{Value: &hujson.Object{}})
			idx = len(obj.Members) - 1
		}
		cur = &obj.Members[idx].Value
	}
	return cur, nil
}

// splitPointer parses an RFC 6901 JSON pointer into path segments.
func splitPointer(ptr string) ([]string, error) {
	if ptr == "" {
		return nil, nil
	}
	if !strings.HasPrefix(ptr, "/") {
		return nil, fmt.Errorf("invalid path %q: must be empty or start with '/'", ptr)
	}
	parts := strings.Split(ptr[1:], "/")
	for i, p := range parts {
		p = strings.ReplaceAll(p, "~1", "/")
		parts[i] = strings.ReplaceAll(p, "~0", "~")
	}
	return parts, nil
}

// parseFragment parses one JSON value from an EditOp payload.
func parseFragment(raw []byte) (hujson.Value, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return hujson.Value{}, fmt.Errorf("empty value")
	}
	return hujson.Parse(raw)
}

// findMember returns the index of the named member or -1.
func findMember(obj *hujson.Object, name string) int {
	for i := range obj.Members {
		if memberName(&obj.Members[i]) == name {
			return i
		}
	}
	return -1
}

// memberName returns the unescaped name of an object member.
func memberName(m *hujson.ObjectMember) string {
	lit, ok := m.Name.Value.(hujson.Literal)
	if !ok {
		return ""
	}
	return lit.String()
}

// isMultilineArray reports whether the array spans multiple lines.
func isMultilineArray(arr *hujson.Array) bool {
	if hasNewline(arr.AfterExtra) {
		return true
	}
	for i := range arr.Elements {
		if hasNewline(arr.Elements[i].BeforeExtra) || hasNewline(arr.Elements[i].AfterExtra) {
			return true
		}
	}
	return len(arr.Elements) == 0
}

// inferObjectIndent reuses the sibling indent or defaults to two spaces.
func inferObjectIndent(obj *hujson.Object) string {
	if len(obj.Members) > 0 {
		if indent, ok := trailingIndent(obj.Members[len(obj.Members)-1].Name.BeforeExtra); ok {
			return indent
		}
	}
	return "  "
}

// inferArrayIndent reuses the element indent or defaults to two spaces.
func inferArrayIndent(arr *hujson.Array) string {
	if len(arr.Elements) > 0 {
		if indent, ok := trailingIndent(arr.Elements[len(arr.Elements)-1].BeforeExtra); ok {
			return indent
		}
	}
	return "  "
}

// trailingIndent extracts the indent after the last newline.
func trailingIndent(extra hujson.Extra) (string, bool) {
	s := string(extra)
	idx := strings.LastIndexByte(s, '\n')
	if idx < 0 {
		return "", false
	}
	return s[idx+1:], true
}

// hasNewline reports whether extra contains a line break.
func hasNewline(extra []byte) bool {
	return bytes.IndexByte(extra, '\n') >= 0
}
