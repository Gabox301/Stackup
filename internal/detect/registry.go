package detect

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ErrUnknownStack is returned by Detect when no detector fires and
// allowUnknown is false. Generation must stop in that case.
var ErrUnknownStack = errors.New("detect: unknown stack (no supported signals found)")

// Registry runs detectors in registration order.
type Registry struct {
	detectors []Detector
}

// NewRegistry returns a Registry running the given detectors in order.
func NewRegistry(detectors ...Detector) *Registry {
	return &Registry{detectors: detectors}
}

// DefaultRegistry returns detectors in wave-2 order: node, go, python,
// rust, csharp, java, ruby, erlang. New ecosystems append after rust so
// existing tie order never shifts.
func DefaultRegistry() *Registry {
	return NewRegistry(
		NodeDetector{},
		GoDetector{},
		PythonDetector{},
		RustDetector{},
		CSharpDetector{},
		JavaDetector{},
		RubyDetector{},
		ErlangDetector{},
	)
}

// DetectAll runs every detector in registry order and returns the combined
// evidence sorted High to Low. Ties keep registry order (stable sort). A
// filesystem error (e.g. a permission-denied manifest) aborts the run:
// an unreadable tree must never scan as an empty one.
func (r *Registry) DetectAll(root string) ([]Evidence, error) {
	var out []Evidence
	for _, d := range r.detectors {
		ev, err := d.Detect(root)
		if err != nil {
			return nil, err
		}
		out = append(out, ev...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Confidence > out[j].Confidence
	})
	return out, nil
}

// Detect runs the default registry over root. When nothing fires it returns
// ErrUnknownStack unless allowUnknown is true, in which case it returns an
// empty list so callers can proceed without detection. Filesystem errors
// propagate unchanged instead of collapsing into unknown-stack.
func Detect(root string, allowUnknown bool) ([]Evidence, error) {
	evidences, err := DefaultRegistry().DetectAll(root)
	if err != nil {
		return nil, err
	}
	if len(evidences) == 0 && !allowUnknown {
		return nil, ErrUnknownStack
	}
	return evidences, nil
}

// statFile stats one filesystem path. It is a variable so tests can inject
// permission failures deterministically on platforms where chmod fixtures
// do not enforce (e.g. Windows); production always uses os.Stat.
var statFile = os.Stat

// exists reports whether name is a regular file directly under root.
//
// Absent files report (false, nil). Permission failures surface as errors
// so callers never mistake an unreadable tree for an empty one. Any other
// stat failure keeps the historical absent verdict. Non-regular files (a
// directory named package.json, a fifo, ...) report (false, nil): only
// regular files yield evidence.
func exists(root, name string) (bool, error) {
	fi, err := statFile(filepath.Join(root, name))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		if os.IsPermission(err) {
			return false, fmt.Errorf("detect: stat %s: %w", name, err)
		}
		return false, nil
	}
	if !fi.Mode().IsRegular() {
		return false, nil
	}
	return true, nil
}

// anyExists reports whether any of names is a regular file under root.
// The first permission failure aborts with an error; see exists.
func anyExists(root string, names ...string) (bool, error) {
	for _, name := range names {
		ok, err := exists(root, name)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}
