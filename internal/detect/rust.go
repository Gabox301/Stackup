package detect

import (
	"os"
	"path/filepath"
	"regexp"
)

var (
	cargoRustVersion = regexp.MustCompile(`(?m)^rust-version\s*=\s*"([^"]+)"`)
	toolchainChannel = regexp.MustCompile(`(?m)^channel\s*=\s*"([^"]+)"`)
)

// RustDetector recognizes Rust projects via Cargo.toml and Cargo.lock.
type RustDetector struct{}

// Name returns the ecosystem key.
func (RustDetector) Name() string { return "rust" }

// Detect returns Rust evidence: Cargo.toml alone grades Medium, plus
// Cargo.lock grades High. A lone rust-toolchain.toml grades Low.
func (RustDetector) Detect(root string) ([]Evidence, error) {
	ok, err := exists(root, "Cargo.toml")
	if err != nil {
		return nil, err
	}
	if !ok {
		if raw, err := os.ReadFile(filepath.Join(root, "rust-toolchain.toml")); err == nil {
			ev := Evidence{
				Ecosystem:  "rust",
				Confidence: ConfidenceLow,
				Signals:    []string{"rust-toolchain.toml"},
			}
			if m := toolchainChannel.FindStringSubmatch(string(raw)); m != nil {
				ev.VersionHint = m[1]
			}
			return []Evidence{ev}, nil
		}
		return nil, nil
	}

	ev := Evidence{
		Ecosystem:  "rust",
		Confidence: ConfidenceMedium,
		Signals:    []string{"Cargo.toml"},
	}
	if raw, err := os.ReadFile(filepath.Join(root, "Cargo.toml")); err == nil {
		if m := cargoRustVersion.FindStringSubmatch(string(raw)); m != nil {
			ev.VersionHint = m[1]
		}
	}
	ok, err = exists(root, "Cargo.lock")
	if err != nil {
		return nil, err
	}
	if ok {
		ev.Signals = append(ev.Signals, "Cargo.lock")
		ev.Confidence = ConfidenceHigh
	}
	return []Evidence{ev}, nil
}
