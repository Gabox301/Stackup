// Package detect identifies project stacks from filesystem signals.
//
// Detectors run in registry order and return confidence-ranked Evidence.
// Grade thresholds: a manifest alone grades Medium, a manifest plus a
// lockfile or toolchain pin grades High, and a version-manager hint
// alone grades Low. Results sort High to Low; registry order breaks ties.
package detect

// Confidence grades how strongly filesystem signals support an ecosystem.
type Confidence int

const (
	// ConfidenceLow marks a weak hint (e.g. a version-manager file alone).
	ConfidenceLow Confidence = iota + 1
	// ConfidenceMedium marks a manifest without a lockfile or pin.
	ConfidenceMedium
	// ConfidenceHigh marks a manifest plus a lockfile or toolchain pin.
	ConfidenceHigh
)

// String returns the lowercase grade name.
func (c Confidence) String() string {
	switch c {
	case ConfidenceLow:
		return "low"
	case ConfidenceMedium:
		return "medium"
	case ConfidenceHigh:
		return "high"
	default:
		return "unknown"
	}
}

// Evidence is a single detector finding that drives config generation.
type Evidence struct {
	// Ecosystem is the registry key of the detector (node, go, python, rust).
	Ecosystem string
	// Confidence grades signal strength; see package docs for thresholds.
	Confidence Confidence
	// Signals lists the files that fired, in detection order.
	Signals []string
	// VersionHint is a best-effort version (engines field, toolchain line).
	VersionHint string
	// PackageManager names the resolved manager (npm, poetry, ...) or "".
	PackageManager string
	// Runtime names the resolved runtime (bun, ...) or "" when not applicable.
	Runtime string `json:",omitempty"`
	// Frameworks lists presence-only framework signals (react, next, ...) or nil.
	Frameworks []string `json:",omitempty"`
}

// Detector recognizes one ecosystem from files under root.
type Detector interface {
	// Name returns the ecosystem key used in Evidence.Ecosystem.
	Name() string
	// Detect returns zero or more findings for root. Missing files and
	// non-regular files simply yield no evidence; permission-denied
	// manifests return an error so an unreadable tree never scans as
	// an empty one.
	Detect(root string) ([]Evidence, error)
}
