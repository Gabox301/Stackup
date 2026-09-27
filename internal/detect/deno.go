package detect

import (
	"path/filepath"
)

// DenoDetector recognizes Deno projects via deno.json[c] and deno.lock.
type DenoDetector struct{}

// Name returns the ecosystem key.
func (DenoDetector) Name() string { return "deno" }

// Detect returns Deno evidence: deno.json or deno.jsonc alone grades Medium
// with PackageManager deno, plus deno.lock grades High. A lone
// .node-version grades Low. VersionHint comes from .node-version, which is
// Deno-owned: the node detector only reads .nvmrc, so the two never overlap.
func (DenoDetector) Detect(root string) ([]Evidence, error) {
	var signals []string
	for _, manifest := range []string{"deno.json", "deno.jsonc"} {
		ok, err := exists(root, manifest)
		if err != nil {
			return nil, err
		}
		if ok {
			signals = append(signals, manifest)
		}
	}
	if len(signals) == 0 {
		if version, ok := readVersionFile(filepath.Join(root, ".node-version")); ok {
			return []Evidence{{
				Ecosystem:   "deno",
				Confidence:  ConfidenceLow,
				Signals:     []string{".node-version"},
				VersionHint: version,
			}}, nil
		}
		return nil, nil
	}

	ev := Evidence{
		Ecosystem:      "deno",
		Confidence:     ConfidenceMedium,
		Signals:        signals,
		PackageManager: "deno",
	}
	if version, ok := readVersionFile(filepath.Join(root, ".node-version")); ok {
		ev.VersionHint = version
	}
	ok, err := exists(root, "deno.lock")
	if err != nil {
		return nil, err
	}
	if ok {
		ev.Signals = append(ev.Signals, "deno.lock")
		ev.Confidence = ConfidenceHigh
	}
	return []Evidence{ev}, nil
}
