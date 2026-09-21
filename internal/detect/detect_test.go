package detect_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Gabox301/Stackup/internal/detect"
)

// copyFixture copies internal/detect/testdata/<name> into a fresh TempDir.
func copyFixture(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(filepath.Join("testdata", name))); err != nil {
		t.Fatalf("copy fixture %s: %v", name, err)
	}
	return dir
}

// writeFile creates a file with content under dir.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(name)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func ecosystems(evidences []detect.Evidence) []string {
	out := make([]string, 0, len(evidences))
	for _, ev := range evidences {
		out = append(out, ev.Ecosystem)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDetect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// setup builds the scanned directory and returns its path.
		setup func(t *testing.T) string
		// allowUnknown mirrors the --allow-unknown flag.
		allowUnknown bool
		// wantEcosystems lists expected ecosystems in rank order.
		wantEcosystems []string
		// wantConfidence maps ecosystem to expected grade.
		wantConfidence map[string]detect.Confidence
		// wantErr is the expected sentinel (nil for success).
		wantErr error
	}{
		{
			name:           "polyglot repo returns ranked list without single-winner collapse",
			setup:          func(t *testing.T) string { return copyFixture(t, "polyglot") },
			wantEcosystems: []string{"node", "go"},
			wantConfidence: map[string]detect.Confidence{
				"node": detect.ConfidenceMedium,
				"go":   detect.ConfidenceMedium,
			},
		},
		{
			name: "higher confidence outranks registry order",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, "package.json", `{"name":"demo"}`)
				writeFile(t, dir, "go.mod", "module example.com/demo\n\ngo 1.23.0\n")
				writeFile(t, dir, "go.sum", "example.com/dep v1.0.0 h1:AAA=\n")
				return dir
			},
			wantEcosystems: []string{"go", "node"},
			wantConfidence: map[string]detect.Confidence{
				"go":   detect.ConfidenceHigh,
				"node": detect.ConfidenceMedium,
			},
		},
		{
			name:           "unknown repo with allow-unknown returns empty list",
			setup:          func(t *testing.T) string { return t.TempDir() },
			allowUnknown:   true,
			wantEcosystems: []string{},
		},
		{
			name:    "empty detection without allow-unknown stops with unknown stack",
			setup:   func(t *testing.T) string { return t.TempDir() },
			wantErr: detect.ErrUnknownStack,
		},
		{
			name: "manifest alone grades medium",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, "package.json", `{"name":"demo"}`)
				return dir
			},
			wantEcosystems: []string{"node"},
			wantConfidence: map[string]detect.Confidence{
				"node": detect.ConfidenceMedium,
			},
		},
		{
			name:           "lockfile raises node confidence to high",
			setup:          func(t *testing.T) string { return copyFixture(t, "node") },
			wantEcosystems: []string{"node"},
			wantConfidence: map[string]detect.Confidence{
				"node": detect.ConfidenceHigh,
			},
		},
		{
			name:           "go lockfile grades high with version hint",
			setup:          func(t *testing.T) string { return copyFixture(t, "go") },
			wantEcosystems: []string{"go"},
			wantConfidence: map[string]detect.Confidence{
				"go": detect.ConfidenceHigh,
			},
		},
		{
			name:           "python poetry lockfile grades high",
			setup:          func(t *testing.T) string { return copyFixture(t, "python") },
			wantEcosystems: []string{"python"},
			wantConfidence: map[string]detect.Confidence{
				"python": detect.ConfidenceHigh,
			},
		},
		{
			name:           "rust lockfile grades high",
			setup:          func(t *testing.T) string { return copyFixture(t, "rust") },
			wantEcosystems: []string{"rust"},
			wantConfidence: map[string]detect.Confidence{
				"rust": detect.ConfidenceHigh,
			},
		},
		{
			name: "version-manager hint alone grades low",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, ".nvmrc", "20\n")
				return dir
			},
			wantEcosystems: []string{"node"},
			wantConfidence: map[string]detect.Confidence{
				"node": detect.ConfidenceLow,
			},
		},
		{
			name: "all four detectors fire on a full polyglot repo",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, "package.json", `{"name":"demo"}`)
				writeFile(t, dir, "package-lock.json", `{}`)
				writeFile(t, dir, "go.mod", "module example.com/demo\n")
				writeFile(t, dir, "pyproject.toml", "[project]\nname=\"demo\"\n")
				writeFile(t, dir, "Cargo.toml", "[package]\nname=\"demo\"\n")
				return dir
			},
			// node is high (lockfile); the rest are medium in registry order.
			wantEcosystems: []string{"node", "go", "python", "rust"},
			wantConfidence: map[string]detect.Confidence{
				"node":   detect.ConfidenceHigh,
				"go":     detect.ConfidenceMedium,
				"python": detect.ConfidenceMedium,
				"rust":   detect.ConfidenceMedium,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.setup(t)

			got, err := detect.Detect(dir, tt.allowUnknown)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Detect() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Detect() unexpected error: %v", err)
			}
			if got == nil {
				got = []detect.Evidence{}
			}
			if !equalStrings(ecosystems(got), tt.wantEcosystems) {
				t.Fatalf("Detect() ecosystems = %v, want %v", ecosystems(got), tt.wantEcosystems)
			}
			for _, ev := range got {
				want, ok := tt.wantConfidence[ev.Ecosystem]
				if !ok {
					continue
				}
				if ev.Confidence != want {
					t.Errorf("Detect() %s confidence = %v, want %v", ev.Ecosystem, ev.Confidence, want)
				}
				if len(ev.Signals) == 0 {
					t.Errorf("Detect() %s returned no signals", ev.Ecosystem)
				}
			}
		})
	}
}

func TestDetectVersionHints(t *testing.T) {
	t.Parallel()

	dir := copyFixture(t, "node")
	got, err := detect.Detect(dir, false)
	if err != nil {
		t.Fatalf("Detect() unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Detect() returned %d evidences, want 1", len(got))
	}
	ev := got[0]
	if ev.VersionHint != ">=20" {
		t.Errorf("VersionHint = %q, want %q", ev.VersionHint, ">=20")
	}
	if ev.PackageManager != "npm" {
		t.Errorf("PackageManager = %q, want %q", ev.PackageManager, "npm")
	}
}

func TestDetectWave2(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// setup builds the scanned directory and returns its path.
		setup func(t *testing.T) string
		// wantEcosystem is the single expected ecosystem ("" means silent).
		wantEcosystem string
		// wantConfidence is the expected grade.
		wantConfidence detect.Confidence
		// wantPM is the expected package manager ("" skips the check).
		wantPM string
	}{
		{
			name:           "csharp manifest alone grades medium",
			setup:          func(t *testing.T) string { return copyFixture(t, "csharp") },
			wantEcosystem:  "csharp",
			wantConfidence: detect.ConfidenceMedium,
		},
		{
			name:           "csharp pin raises to high with version hint",
			setup:          func(t *testing.T) string { return copyFixture(t, "csharp-pin") },
			wantEcosystem:  "csharp",
			wantConfidence: detect.ConfidenceHigh,
		},
		{
			name: "csharp hint alone grades low",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, "NuGet.config", `<?xml version="1.0"?>`)
				return dir
			},
			wantEcosystem:  "csharp",
			wantConfidence: detect.ConfidenceLow,
		},
		{
			name:           "java maven grades medium",
			setup:          func(t *testing.T) string { return copyFixture(t, "java-maven") },
			wantEcosystem:  "java",
			wantConfidence: detect.ConfidenceMedium,
			wantPM:         "maven",
		},
		{
			name:           "java gradle grades medium",
			setup:          func(t *testing.T) string { return copyFixture(t, "java-gradle") },
			wantEcosystem:  "java",
			wantConfidence: detect.ConfidenceMedium,
			wantPM:         "gradle",
		},
		{
			name:           "java duel resolves gradle-wins single evidence",
			setup:          func(t *testing.T) string { return copyFixture(t, "java-duel") },
			wantEcosystem:  "java",
			wantConfidence: detect.ConfidenceMedium,
			wantPM:         "gradle",
		},
		{
			name: "java hint alone grades low",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, ".java-version", "17\n")
				return dir
			},
			wantEcosystem:  "java",
			wantConfidence: detect.ConfidenceLow,
		},
		{
			name:           "ruby manifest grades medium with bundler",
			setup:          func(t *testing.T) string { return copyFixture(t, "ruby") },
			wantEcosystem:  "ruby",
			wantConfidence: detect.ConfidenceMedium,
			wantPM:         "bundler",
		},
		{
			name:           "ruby lockfile grades high",
			setup:          func(t *testing.T) string { return copyFixture(t, "ruby-pin") },
			wantEcosystem:  "ruby",
			wantConfidence: detect.ConfidenceHigh,
			wantPM:         "bundler",
		},
		{
			name: "ruby hint alone grades low",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, ".ruby-version", "3.2.2\n")
				return dir
			},
			wantEcosystem:  "ruby",
			wantConfidence: detect.ConfidenceLow,
		},
		{
			name:           "erlang manifest grades medium with rebar3",
			setup:          func(t *testing.T) string { return copyFixture(t, "erlang") },
			wantEcosystem:  "erlang",
			wantConfidence: detect.ConfidenceMedium,
			wantPM:         "rebar3",
		},
		{
			name: "erlang lockfile grades high",
			setup: func(t *testing.T) string {
				dir := copyFixture(t, "erlang")
				writeFile(t, dir, "rebar.lock", `{}.`)
				return dir
			},
			wantEcosystem:  "erlang",
			wantConfidence: detect.ConfidenceHigh,
			wantPM:         "rebar3",
		},
		{
			name:           "erlang weak signals stay silent",
			setup:          func(t *testing.T) string { return copyFixture(t, "erlang-weak") },
			wantEcosystem:  "",
			wantConfidence: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.setup(t)

			got, err := detect.Detect(dir, false)
			if tt.wantEcosystem == "" {
				if err == nil {
					for _, ev := range got {
						if ev.Ecosystem == "erlang" {
							t.Fatalf("Detect() returned erlang evidence %v, want silence", ev)
						}
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Detect() unexpected error: %v", err)
			}
			var found *detect.Evidence
			for i := range got {
				if got[i].Ecosystem == tt.wantEcosystem {
					found = &got[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("Detect() ecosystems = %v, want %s", ecosystems(got), tt.wantEcosystem)
			}
			if found.Confidence != tt.wantConfidence {
				t.Errorf("Detect() %s confidence = %v, want %v", tt.wantEcosystem, found.Confidence, tt.wantConfidence)
			}
			if tt.wantPM != "" && found.PackageManager != tt.wantPM {
				t.Errorf("Detect() %s PackageManager = %q, want %q", tt.wantEcosystem, found.PackageManager, tt.wantPM)
			}
			if len(found.Signals) == 0 {
				t.Errorf("Detect() %s returned no signals", tt.wantEcosystem)
			}
		})
	}
}

func TestDetectWave2Contracts(t *testing.T) {
	t.Parallel()

	t.Run("java duel keeps both manifests in signals", func(t *testing.T) {
		t.Parallel()
		dir := copyFixture(t, "java-duel")
		got, err := detect.Detect(dir, false)
		if err != nil {
			t.Fatalf("Detect() unexpected error: %v", err)
		}
		count := 0
		for _, ev := range got {
			if ev.Ecosystem == "java" {
				count++
				hasPom, hasGradle := false, false
				for _, s := range ev.Signals {
					if s == "pom.xml" {
						hasPom = true
					}
					if s == "build.gradle" {
						hasGradle = true
					}
				}
				if !hasPom || !hasGradle {
					t.Errorf("java duel Signals = %v, want pom.xml and build.gradle", ev.Signals)
				}
			}
		}
		if count != 1 {
			t.Errorf("java duel returned %d evidences, want 1", count)
		}
	})

	t.Run("csharp pin prefers global.json version", func(t *testing.T) {
		t.Parallel()
		dir := copyFixture(t, "csharp-pin")
		got, err := detect.Detect(dir, false)
		if err != nil {
			t.Fatalf("Detect() unexpected error: %v", err)
		}
		for _, ev := range got {
			if ev.Ecosystem == "csharp" && ev.VersionHint != "8.0.100" {
				t.Errorf("VersionHint = %q, want %q", ev.VersionHint, "8.0.100")
			}
		}
	})
}

func TestDetectErlangWithoutRebar3(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping rebar3 integration path in short mode")
	}
	// Missing rebar3 never fails detection: file signals alone decide.
	if _, err := exec.LookPath("rebar3"); err == nil {
		t.Log("rebar3 present; file signals still decide")
	}
	dir := copyFixture(t, "erlang")
	got, err := detect.Detect(dir, false)
	if err != nil {
		t.Fatalf("Detect() unexpected error: %v", err)
	}
	found := false
	for _, ev := range got {
		if ev.Ecosystem != "erlang" {
			continue
		}
		found = true
		if ev.PackageManager != "rebar3" {
			t.Errorf("PackageManager = %q, want %q", ev.PackageManager, "rebar3")
		}
		if len(ev.Signals) == 0 {
			t.Errorf("Detect() erlang returned no signals")
		}
	}
	if !found {
		t.Fatalf("Detect() ecosystems = %v, want erlang", ecosystems(got))
	}
}

func TestDetectBun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// setup builds the scanned directory and returns its path.
		setup func(t *testing.T) string
		// wantPM is the expected package manager.
		wantPM string
		// wantRuntime is the expected runtime.
		wantRuntime string
	}{
		{
			name:        "bun coexistence keeps single evidence with packageManager-first",
			setup:       func(t *testing.T) string { return copyFixture(t, "node-bun") },
			wantPM:      "npm",
			wantRuntime: "bun",
		},
		{
			name: "bun lock alone resolves bun when field absent",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, "package.json", `{"name":"demo"}`)
				writeFile(t, dir, "bun.lock", `{}`)
				return dir
			},
			wantPM:      "bun",
			wantRuntime: "bun",
		},
		{
			name: "bun.lockb binary presence-only sets runtime without parsing",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, "package.json", `{"name":"demo"}`)
				writeFile(t, dir, "bun.lockb", "\x00bun-binary\xff")
				return dir
			},
			wantPM:      "bun",
			wantRuntime: "bun",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.setup(t)

			got, err := detect.Detect(dir, false)
			if err != nil {
				t.Fatalf("Detect() unexpected error: %v", err)
			}
			count := 0
			for _, ev := range got {
				if ev.Ecosystem != "node" {
					continue
				}
				count++
				if ev.Runtime != tt.wantRuntime {
					t.Errorf("Runtime = %q, want %q", ev.Runtime, tt.wantRuntime)
				}
				if ev.PackageManager != tt.wantPM {
					t.Errorf("PackageManager = %q, want %q", ev.PackageManager, tt.wantPM)
				}
				if ev.Confidence != detect.ConfidenceHigh {
					t.Errorf("Confidence = %v, want %v", ev.Confidence, detect.ConfidenceHigh)
				}
				if len(ev.Signals) == 0 {
					t.Errorf("Detect() node returned no signals")
				}
			}
			if count != 1 {
				t.Errorf("Detect() returned %d node evidences, want 1 (single evidence)", count)
			}
			if tt.name == "bun coexistence keeps single evidence with packageManager-first" {
				for _, ev := range got {
					if ev.Ecosystem != "node" {
						continue
					}
					hasPackageLock, hasBunLock := false, false
					for _, s := range ev.Signals {
						if s == "package-lock.json" {
							hasPackageLock = true
						}
						if s == "bun.lock" {
							hasBunLock = true
						}
					}
					if !hasPackageLock || !hasBunLock {
						t.Errorf("coexistence Signals = %v, want package-lock.json and bun.lock", ev.Signals)
					}
				}
			}
		})
	}
}

func TestDetectFrameworks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// setup builds the scanned directory and returns its path.
		setup func(t *testing.T) string
		// wantFrameworks is the expected Frameworks list in order (nil means none).
		wantFrameworks []string
	}{
		{
			name:           "next maps to react and next regardless of range",
			setup:          func(t *testing.T) string { return copyFixture(t, "node-next") },
			wantFrameworks: []string{"react", "next"},
		},
		{
			name: "react alone maps to react",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, "package.json", `{"name":"demo","dependencies":{"react":"^18.0.0"}}`)
				return dir
			},
			wantFrameworks: []string{"react"},
		},
		{
			name: "nuxt maps to vue and nuxt",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, "package.json", `{"name":"demo","dependencies":{"nuxt":"^3.0.0"}}`)
				return dir
			},
			wantFrameworks: []string{"vue", "nuxt"},
		},
		{
			name: "sveltekit maps to svelte and sveltekit",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, "package.json", `{"name":"demo","dependencies":{"@sveltejs/kit":"^2.0.0"}}`)
				return dir
			},
			wantFrameworks: []string{"svelte", "sveltekit"},
		},
		{
			name: "devDependencies count for presence-only scan",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, "package.json", `{"name":"demo","devDependencies":{"vue":"^3.0.0"}}`)
				return dir
			},
			wantFrameworks: []string{"vue"},
		},
		{
			name: "astro and svelte presence without semver",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, "package.json", `{"name":"demo","dependencies":{"astro":"4","svelte":"4"}}`)
				return dir
			},
			wantFrameworks: []string{"svelte", "astro"},
		},
		{
			name: "nested manifest ignored root-only",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				writeFile(t, dir, "package.json", `{"name":"demo"}`)
				writeFile(t, dir, "sub/package.json", `{"name":"nested","dependencies":{"next":"^14.0.0"}}`)
				return dir
			},
			wantFrameworks: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := tt.setup(t)

			got, err := detect.Detect(dir, false)
			if err != nil {
				t.Fatalf("Detect() unexpected error: %v", err)
			}
			count := 0
			for _, ev := range got {
				if ev.Ecosystem != "node" {
					continue
				}
				count++
				if !equalStrings(ev.Frameworks, tt.wantFrameworks) {
					t.Errorf("Frameworks = %v, want %v", ev.Frameworks, tt.wantFrameworks)
				}
			}
			if count != 1 {
				t.Errorf("Detect() returned %d node evidences, want 1 (single evidence)", count)
			}
		})
	}
}
