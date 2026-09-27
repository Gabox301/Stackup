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
// rust, csharp, java, ruby, erlang, php, terraform, deno. New ecosystems
// append at the end so existing tie order never shifts.
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
		PHPDetector{},
		TerraformDetector{},
		DenoDetector{},
	)
}

// MaxScanDepth bounds nested monorepo scanning below root. Root itself is
// depth 0; subdirectories down to this depth are scanned. The default is a
// fixed constant on purpose: no CLI flag grows the surface for a heuristic
// that must stay predictable.
const MaxScanDepth = 3

// DetectAll runs every detector in registry order and returns the combined
// evidence sorted High to Low. Ties keep registry order (stable sort). Each
// detector scans root plus nested subdirectories down to MaxScanDepth, and
// findings dedupe to one evidence per ecosystem: the highest confidence
// wins, root wins ties. A filesystem error (e.g. a permission-denied
// manifest) aborts the run: an unreadable tree must never scan as an empty
// one.
func (r *Registry) DetectAll(root string) ([]Evidence, error) {
	dirs, err := scanDirs(root, MaxScanDepth)
	if err != nil {
		return nil, err
	}
	var out []Evidence
	for _, d := range r.detectors {
		var best *Evidence
		for _, dir := range dirs {
			evs, err := d.Detect(dir)
			if err != nil {
				return nil, err
			}
			for i := range evs {
				ev := evs[i]
				if dir != root {
					ev.Signals = prefixSignals(root, dir, ev.Signals)
				}
				// Strictly greater wins: dirs arrive root-first, so root
				// keeps ties and the first scan wins deeper ties.
				if best == nil || ev.Confidence > best.Confidence {
					dup := ev
					best = &dup
				}
			}
		}
		if best != nil {
			out = append(out, *best)
		}
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

// scanDirs lists root plus nested subdirectories breadth-first down to
// maxDepth (root is depth 0). Entries arrive root-first, shallower-first,
// and lexical within a level for deterministic dedupe. Symlinks are
// skipped, never followed. A missing root scans as root alone so callers
// keep the historical unknown-stack verdict; any other filesystem error
// aborts the run so an unreadable tree never scans as an empty one.
func scanDirs(root string, maxDepth int) ([]string, error) {
	dirs := []string{root}
	type queueItem struct {
		dir   string
		depth int
	}
	queue := []queueItem{{dir: root, depth: 0}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.depth >= maxDepth {
			continue
		}
		entries, err := os.ReadDir(cur.dir)
		if err != nil {
			if os.IsNotExist(err) {
				if cur.depth == 0 {
					return []string{root}, nil
				}
				continue
			}
			return nil, err
		}
		for _, e := range entries {
			if e.Type()&os.ModeSymlink != 0 {
				continue
			}
			if !e.IsDir() {
				continue
			}
			sub := filepath.Join(cur.dir, e.Name())
			dirs = append(dirs, sub)
			queue = append(queue, queueItem{dir: sub, depth: cur.depth + 1})
		}
	}
	return dirs, nil
}

// prefixSignals rewrites bare manifest names with their slash-separated path
// relative to root (package.json under frontend becomes
// frontend/package.json). Root evidence keeps bare names.
func prefixSignals(root, dir string, signals []string) []string {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "." {
		return signals
	}
	out := make([]string, 0, len(signals))
	for _, s := range signals {
		out = append(out, filepath.ToSlash(filepath.Join(rel, s)))
	}
	return out
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
