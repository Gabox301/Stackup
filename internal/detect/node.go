package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// NodeDetector recognizes Node.js projects via package.json and lockfiles.
type NodeDetector struct{}

// Name returns the ecosystem key.
func (NodeDetector) Name() string { return "node" }

// nodeLock maps a lockfile to its package manager.
var nodeLock = []struct {
	file string
	pm   string
}{
	{"package-lock.json", "npm"},
	{"yarn.lock", "yarn"},
	{"pnpm-lock.yaml", "pnpm"},
}

// Detect returns Node evidence: package.json alone grades Medium, plus a
// lockfile grades High. A lone .nvmrc grades Low. Bun lockfiles
// (bun.lock/bun.lockb) are presence-only and set Runtime=bun; the
// packageManager field wins over lockfiles, otherwise bun wins when its
// lockfile is present. Frameworks come from a root-only presence-only scan.
func (NodeDetector) Detect(root string) ([]Evidence, error) {
	ok, err := exists(root, "package.json")
	if err != nil {
		return nil, err
	}
	if !ok {
		nvm, err := exists(root, ".nvmrc")
		if err != nil {
			return nil, err
		}
		if nvm {
			return []Evidence{{
				Ecosystem:  "node",
				Confidence: ConfidenceLow,
				Signals:    []string{".nvmrc"},
			}}, nil
		}
		return nil, nil
	}

	ev := Evidence{
		Ecosystem:  "node",
		Confidence: ConfidenceMedium,
		Signals:    []string{"package.json"},
	}
	version, pm, frameworks := readPackageJSON(filepath.Join(root, "package.json"))
	ev.VersionHint = version
	ev.PackageManager = pm
	fieldPM := pm
	if len(frameworks) > 0 {
		ev.Frameworks = frameworks
	}

	for _, lock := range nodeLock {
		ok, err := exists(root, lock.file)
		if err != nil {
			return nil, err
		}
		if ok {
			ev.Signals = append(ev.Signals, lock.file)
			ev.Confidence = ConfidenceHigh
			if ev.PackageManager == "" {
				ev.PackageManager = lock.pm
			}
		}
	}

	// Bun lockfiles are presence-only: check existence, never read or parse
	// (bun.lockb is binary). Either file upgrades to High and marks the
	// runtime. PackageManager resolves field-first, else bun.
	hasBun := false
	for _, bunLock := range []string{"bun.lock", "bun.lockb"} {
		ok, err := exists(root, bunLock)
		if err != nil {
			return nil, err
		}
		if ok {
			ev.Signals = append(ev.Signals, bunLock)
			ev.Confidence = ConfidenceHigh
			hasBun = true
		}
	}
	if hasBun {
		ev.Runtime = "bun"
		if fieldPM == "" {
			ev.PackageManager = "bun"
		} else {
			ev.PackageManager = fieldPM
		}
	} else if fieldPM == "bun" {
		ev.Runtime = "bun"
	}
	return []Evidence{ev}, nil
}

// nodeFrameworks maps root dependencies to framework signals, presence-only.
// Meta-frameworks record both base and meta (next->react+next,
// nuxt->vue+nuxt, sveltekit->svelte+sveltekit). No semver, no recursion.
func nodeFrameworks(deps, devDeps map[string]any) []string {
	has := func(name string) bool {
		if deps != nil {
			if _, ok := deps[name]; ok {
				return true
			}
		}
		if devDeps != nil {
			if _, ok := devDeps[name]; ok {
				return true
			}
		}
		return false
	}
	var frameworks []string
	if has("next") {
		frameworks = append(frameworks, "react", "next")
	} else if has("react") {
		frameworks = append(frameworks, "react")
	}
	if has("nuxt") {
		frameworks = append(frameworks, "vue", "nuxt")
	} else if has("vue") {
		frameworks = append(frameworks, "vue")
	}
	if has("@sveltejs/kit") || has("sveltekit") {
		frameworks = append(frameworks, "svelte", "sveltekit")
	} else if has("svelte") {
		frameworks = append(frameworks, "svelte")
	}
	if has("astro") {
		frameworks = append(frameworks, "astro")
	}
	return frameworks
}

// readPackageJSON extracts engines.node, the packageManager field, and
// presence-only framework signals on a best-effort basis; any failure
// yields empty hints, never an error.
func readPackageJSON(path string) (version, pm string, frameworks []string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", "", nil
	}
	var manifest struct {
		Engines struct {
			Node string `json:"node"`
		} `json:"engines"`
		PackageManager  string         `json:"packageManager"`
		Dependencies    map[string]any `json:"dependencies"`
		DevDependencies map[string]any `json:"devDependencies"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return "", "", nil
	}
	if manifest.PackageManager != "" {
		// Format is "<name>@<version>"; keep the manager name only.
		if name, _, ok := strings.Cut(manifest.PackageManager, "@"); ok {
			pm = name
		} else {
			pm = manifest.PackageManager
		}
	}
	return manifest.Engines.Node, pm, nodeFrameworks(manifest.Dependencies, manifest.DevDependencies)
}
