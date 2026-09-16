package detect

import (
	"path/filepath"
)

// RubyDetector recognizes Ruby projects via Gemfile and gemspecs.
type RubyDetector struct{}

// Name returns the ecosystem key.
func (RubyDetector) Name() string { return "ruby" }

// Detect returns Ruby evidence: Gemfile or a root *.gemspec alone grades
// Medium, plus Gemfile.lock or a version pin grades High with
// PackageManager bundler. Version files alone grade Low.
func (RubyDetector) Detect(root string) []Evidence {
	var signals []string
	if exists(root, "Gemfile") {
		signals = append(signals, "Gemfile")
	}
	if matches, _ := filepath.Glob(filepath.Join(root, "*.gemspec")); len(matches) > 0 {
		for _, m := range matches {
			signals = append(signals, filepath.Base(m))
		}
	}
	if len(signals) > 0 {
		ev := Evidence{
			Ecosystem:      "ruby",
			Confidence:     ConfidenceMedium,
			Signals:        signals,
			PackageManager: "bundler",
		}
		if v := rubyVersionHint(root); v != "" {
			ev.VersionHint = v
		}
		if exists(root, "Gemfile.lock") {
			ev.Signals = append(ev.Signals, "Gemfile.lock")
			ev.Confidence = ConfidenceHigh
		}
		for _, pin := range []string{".ruby-version", ".rbenv-version"} {
			if exists(root, pin) {
				ev.Signals = append(ev.Signals, pin)
				ev.Confidence = ConfidenceHigh
			}
		}
		if v, ok := toolVersionsValue(root, "ruby"); ok {
			ev.Signals = append(ev.Signals, ".tool-versions")
			ev.Confidence = ConfidenceHigh
			if ev.VersionHint == "" {
				ev.VersionHint = v
			}
		}
		return []Evidence{ev}
	}

	var hints []string
	version := ""
	for _, pin := range []string{".ruby-version", ".rbenv-version"} {
		if v, ok := readVersionFile(filepath.Join(root, pin)); ok {
			hints = append(hints, pin)
			if version == "" {
				version = v
			}
		}
	}
	if v, ok := toolVersionsValue(root, "ruby"); ok {
		hints = append(hints, ".tool-versions")
		if version == "" {
			version = v
		}
	}
	if len(hints) > 0 {
		return []Evidence{{
			Ecosystem:   "ruby",
			Confidence:  ConfidenceLow,
			Signals:     hints,
			VersionHint: version,
		}}
	}
	return nil
}

// rubyVersionHint prefers .ruby-version, then .rbenv-version, then .tool-versions.
func rubyVersionHint(root string) string {
	for _, pin := range []string{".ruby-version", ".rbenv-version"} {
		if v, ok := readVersionFile(filepath.Join(root, pin)); ok {
			return v
		}
	}
	if v, ok := toolVersionsValue(root, "ruby"); ok {
		return v
	}
	return ""
}
