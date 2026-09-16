package detect

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PythonDetector recognizes Python projects via manifests and lockfiles.
type PythonDetector struct{}

// Name returns the ecosystem key.
func (PythonDetector) Name() string { return "python" }

// pythonLock maps a lockfile to its package manager.
var pythonLock = []struct {
	file string
	pm   string
}{
	{"poetry.lock", "poetry"},
	{"Pipfile.lock", "pipenv"},
	{"uv.lock", "uv"},
	{"pdm.lock", "pdm"},
}

var requiresPython = regexp.MustCompile(`requires-python\s*=\s*"([^"]+)"`)

// Detect returns Python evidence: a manifest alone grades Medium, plus a
// lockfile grades High. A lone .python-version grades Low.
func (PythonDetector) Detect(root string) []Evidence {
	var signals []string
	pm := ""

	for _, manifest := range []string{"pyproject.toml", "Pipfile", "setup.py"} {
		if exists(root, manifest) {
			signals = append(signals, manifest)
		}
	}
	if reqs, _ := filepath.Glob(filepath.Join(root, "requirements*.txt")); len(reqs) > 0 {
		for _, r := range reqs {
			signals = append(signals, filepath.Base(r))
		}
		if pm == "" {
			pm = "pip"
		}
	}
	if len(signals) == 0 {
		if version, ok := readVersionFile(filepath.Join(root, ".python-version")); ok {
			return []Evidence{{
				Ecosystem:   "python",
				Confidence:  ConfidenceLow,
				Signals:     []string{".python-version"},
				VersionHint: version,
			}}
		}
		return nil
	}

	if exists(root, "Pipfile") && pm == "" {
		pm = "pipenv"
	}
	ev := Evidence{
		Ecosystem:      "python",
		Confidence:     ConfidenceMedium,
		Signals:        signals,
		PackageManager: pm,
	}
	for _, lock := range pythonLock {
		if exists(root, lock.file) {
			ev.Signals = append(ev.Signals, lock.file)
			ev.Confidence = ConfidenceHigh
			ev.PackageManager = lock.pm
		}
	}
	if ev.VersionHint == "" {
		ev.VersionHint = pythonVersionHint(root)
	}
	return []Evidence{ev}
}

// pythonVersionHint prefers .python-version, then requires-python.
func pythonVersionHint(root string) string {
	if version, ok := readVersionFile(filepath.Join(root, ".python-version")); ok {
		return version
	}
	raw, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		return ""
	}
	if m := requiresPython.FindStringSubmatch(string(raw)); m != nil {
		return m[1]
	}
	return ""
}

// readVersionFile returns the trimmed first line of a version file.
func readVersionFile(path string) (string, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	if v := strings.TrimSpace(strings.SplitN(string(raw), "\n", 2)[0]); v != "" {
		return v, true
	}
	return "", false
}
