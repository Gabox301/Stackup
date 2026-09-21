package detect

import (
	"os"
	"path/filepath"
	"strings"
)

// GoDetector recognizes Go projects via go.mod and go.sum.
type GoDetector struct{}

// Name returns the ecosystem key.
func (GoDetector) Name() string { return "go" }

// Detect returns Go evidence: go.mod alone grades Medium, plus go.sum
// grades High. The go directive line feeds VersionHint.
func (GoDetector) Detect(root string) ([]Evidence, error) {
	ok, err := exists(root, "go.mod")
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	ev := Evidence{
		Ecosystem:  "go",
		Confidence: ConfidenceMedium,
		Signals:    []string{"go.mod"},
	}
	if raw, err := os.ReadFile(filepath.Join(root, "go.mod")); err == nil {
		ev.VersionHint = goDirectiveVersion(string(raw))
	}
	ok, err = exists(root, "go.sum")
	if err != nil {
		return nil, err
	}
	if ok {
		ev.Signals = append(ev.Signals, "go.sum")
		ev.Confidence = ConfidenceHigh
	}
	return []Evidence{ev}, nil
}

// goDirectiveVersion scans go.mod for the `go <version>` directive.
func goDirectiveVersion(mod string) string {
	for line := range strings.Lines(mod) {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "go "); ok {
			if v = strings.TrimSpace(v); v != "" {
				return v
			}
		}
	}
	return ""
}
