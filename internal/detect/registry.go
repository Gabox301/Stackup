package detect

import (
	"errors"
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
// evidence sorted High to Low. Ties keep registry order (stable sort).
func (r *Registry) DetectAll(root string) []Evidence {
	var out []Evidence
	for _, d := range r.detectors {
		out = append(out, d.Detect(root)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Confidence > out[j].Confidence
	})
	return out
}

// Detect runs the default registry over root. When nothing fires it returns
// ErrUnknownStack unless allowUnknown is true, in which case it returns an
// empty list so callers can proceed without detection.
func Detect(root string, allowUnknown bool) ([]Evidence, error) {
	evidences := DefaultRegistry().DetectAll(root)
	if len(evidences) == 0 && !allowUnknown {
		return nil, ErrUnknownStack
	}
	return evidences, nil
}

// exists reports whether name exists directly under root.
func exists(root, name string) bool {
	_, err := os.Stat(filepath.Join(root, name))
	return err == nil
}
