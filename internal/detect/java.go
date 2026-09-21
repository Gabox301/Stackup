package detect

import (
	"os"
	"path/filepath"
	"strings"
)

// JavaDetector recognizes Java projects via Maven or Gradle manifests.
type JavaDetector struct{}

// Name returns the ecosystem key.
func (JavaDetector) Name() string { return "java" }

// Detect returns a single Java evidence: pom.xml or build.gradle(.kts)
// grades Medium, plus a lockfile, wrapper, or toolchain pin grades High.
// When both Maven and Gradle manifests exist, Gradle wins the
// PackageManager but both manifests stay in Signals. Version-hint files
// alone grade Low.
func (JavaDetector) Detect(root string) ([]Evidence, error) {
	hasPom, err := exists(root, "pom.xml")
	if err != nil {
		return nil, err
	}
	hasGradleBuild, err := anyExists(root, "build.gradle", "build.gradle.kts")
	if err != nil {
		return nil, err
	}
	hasGradleSettings, err := anyExists(root, "settings.gradle", "settings.gradle.kts")
	if err != nil {
		return nil, err
	}
	hasGradle := hasGradleBuild || hasGradleSettings

	if !hasPom && !hasGradleBuild {
		v, signals, err := javaLowHint(root)
		if err != nil {
			return nil, err
		}
		if v != "" || signals != nil {
			ev := Evidence{
				Ecosystem:  "java",
				Confidence: ConfidenceLow,
				Signals:    signals,
			}
			ev.VersionHint = v
			return []Evidence{ev}, nil
		}
		return nil, nil
	}

	ev := Evidence{
		Ecosystem:  "java",
		Confidence: ConfidenceMedium,
	}
	if hasPom {
		ev.Signals = append(ev.Signals, "pom.xml")
	}
	for _, f := range []string{"build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts"} {
		ok, err := exists(root, f)
		if err != nil {
			return nil, err
		}
		if ok {
			ev.Signals = append(ev.Signals, f)
		}
	}
	if hasGradle {
		ev.PackageManager = "gradle"
	} else {
		ev.PackageManager = "maven"
	}
	if v := javaVersionHint(root); v != "" {
		ev.VersionHint = v
	}
	for _, pin := range []string{
		"gradle.lockfile",
		"verification-metadata.xml",
		"gradlew",
		"mvnw",
		".mvn",
		".java-version",
		".sdkmanrc",
		".tool-versions",
	} {
		ok, err := exists(root, pin)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if pin == ".tool-versions" {
			if _, ok := toolVersionsValue(root, "java"); !ok {
				continue
			}
		}
		ev.Signals = append(ev.Signals, pin)
		ev.Confidence = ConfidenceHigh
	}
	return []Evidence{ev}, nil
}

// javaVersionHint prefers .java-version, then .sdkmanrc java=, then .tool-versions.
func javaVersionHint(root string) string {
	if v, ok := readVersionFile(filepath.Join(root, ".java-version")); ok {
		return v
	}
	if raw, err := os.ReadFile(filepath.Join(root, ".sdkmanrc")); err == nil {
		for line := range strings.Lines(string(raw)) {
			line = strings.TrimSpace(line)
			if v, ok := strings.CutPrefix(line, "java="); ok {
				if v = strings.TrimSpace(v); v != "" {
					return v
				}
			}
		}
	}
	if v, ok := toolVersionsValue(root, "java"); ok {
		return v
	}
	return ""
}

// javaLowHint reports Low evidence from version-hint files alone.
func javaLowHint(root string) (string, []string, error) {
	var signals []string
	version := ""
	if v, ok := readVersionFile(filepath.Join(root, ".java-version")); ok {
		signals = append(signals, ".java-version")
		version = v
	}
	ok, err := exists(root, ".sdkmanrc")
	if err != nil {
		return "", nil, err
	}
	if ok {
		signals = append(signals, ".sdkmanrc")
		if version == "" {
			version = javaVersionHint(root)
		}
	}
	if v, ok := toolVersionsValue(root, "java"); ok {
		signals = append(signals, ".tool-versions")
		if version == "" {
			version = v
		}
	}
	if len(signals) == 0 {
		return "", nil, nil
	}
	return version, signals, nil
}

// toolVersionsValue returns the version for a plugin in .tool-versions.
func toolVersionsValue(root, plugin string) (string, bool) {
	raw, err := os.ReadFile(filepath.Join(root, ".tool-versions"))
	if err != nil {
		return "", false
	}
	for line := range strings.Lines(string(raw)) {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && fields[0] == plugin {
			return fields[1], true
		}
	}
	return "", false
}
