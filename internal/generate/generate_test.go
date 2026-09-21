package generate_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gabox301/Stackup/internal/detect"
	"github.com/Gabox301/Stackup/internal/generate"
)

var update = flag.Bool("update", false, "update golden files")

// golden compares got against testdata/name, writing it back under -update.
func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update to create)", name, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("golden %s mismatch:\n got:\n%s\nwant:\n%s", name, got, want)
	}
}

func nodeEvidence() []detect.Evidence {
	return []detect.Evidence{
		{
			Ecosystem:      "node",
			Confidence:     detect.ConfidenceHigh,
			Signals:        []string{"package.json", "package-lock.json"},
			VersionHint:    ">=20",
			PackageManager: "npm",
		},
	}
}

func findFile(t *testing.T, plan generate.Plan, path string) []byte {
	t.Helper()
	for _, f := range plan.Files {
		if f.Path == path {
			return f.Content
		}
	}
	t.Fatalf("plan missing file %q (have %v)", path, plan.Paths())
	return nil
}

func TestBuildAllIDEsCoversSpecTargets(t *testing.T) {
	t.Parallel()

	plan, err := generate.Build(nodeEvidence(), nil)
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	want := []string{
		".vscode/settings.json",
		".vscode/extensions.json",
		".vscode/launch.json",
		".vscode/tasks.json",
		".cursor/rules/stackup.mdc",
		".cursor/mcp.json",
		".devin/rules/stackup.md",
		".devin/mcp_config.json",
		".windsurf/rules/stackup.md",
		".kiro/steering/product.md",
		".kiro/steering/tech.md",
		".kiro/steering/structure.md",
		".kiro/specs/stackup/requirements.md",
		".kiro/specs/stackup/design.md",
		".kiro/specs/stackup/tasks.md",
		".kiro/settings/mcp.json",
	}
	if len(plan.Files) != len(want) {
		t.Fatalf("Build() files = %v, want %d files", plan.Paths(), len(want))
	}
	for i, path := range want {
		if plan.Files[i].Path != path {
			t.Errorf("Build() file %d = %q, want %q", i, plan.Files[i].Path, path)
		}
		if len(plan.Files[i].Content) == 0 {
			t.Errorf("Build() file %q rendered empty", path)
		}
	}
}

func TestBuildIDESelection(t *testing.T) {
	t.Parallel()

	plan, err := generate.Build(nodeEvidence(), []string{"vscode"})
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if len(plan.Files) != 4 {
		t.Fatalf("Build(vscode) files = %v, want 4 targets", plan.Paths())
	}
	for _, f := range plan.Files {
		if !f.JSON {
			t.Errorf("Build(vscode) file %q should be a JSON merge target", f.Path)
		}
	}
}

func TestBuildUnknownIDERejected(t *testing.T) {
	t.Parallel()

	if _, err := generate.Build(nodeEvidence(), []string{"sublime"}); err == nil {
		t.Fatal("Build() expected error for unknown IDE, got nil")
	}
}

func TestBuildEmptyEvidenceRendersGeneric(t *testing.T) {
	t.Parallel()

	plan, err := generate.Build(nil, []string{"cursor", "kiro"})
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if len(plan.Files) == 0 {
		t.Fatal("Build() with empty evidence rendered no files")
	}
	mdc := findFile(t, plan, ".cursor/rules/stackup.mdc")
	if !bytes.Contains(mdc, []byte("unknown")) {
		t.Errorf("generic cursor rules should name the unknown stack, got:\n%s", mdc)
	}
}

func TestFreshRenderGoldens(t *testing.T) {
	t.Parallel()

	plan, err := generate.Build(nodeEvidence(), nil)
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	cases := map[string]string{
		".vscode/settings.json":     "vscode-settings-node.golden",
		".vscode/extensions.json":   "vscode-extensions-node.golden",
		".vscode/launch.json":       "vscode-launch-node.golden",
		".vscode/tasks.json":        "vscode-tasks-node.golden",
		".cursor/rules/stackup.mdc": "cursor-rules-node.golden",
		".devin/rules/stackup.md":   "devin-rules-node.golden",
		".kiro/steering/tech.md":    "kiro-steering-tech-node.golden",
	}
	for path, gold := range cases {
		golden(t, gold, findFile(t, plan, path))
	}
}

func TestRerenderIsIdempotent(t *testing.T) {
	t.Parallel()

	first, err := generate.Build(nodeEvidence(), nil)
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	second, err := generate.Build(nodeEvidence(), nil)
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if len(first.Files) != len(second.Files) {
		t.Fatalf("re-render file count = %d, want %d", len(second.Files), len(first.Files))
	}
	for i := range first.Files {
		if first.Files[i].Path != second.Files[i].Path {
			t.Fatalf("re-render path %d = %q, want %q", i, second.Files[i].Path, first.Files[i].Path)
		}
		if !bytes.Equal(first.Files[i].Content, second.Files[i].Content) {
			t.Errorf("re-render of %q changed bytes", first.Files[i].Path)
		}
	}
}

func TestReactEmitsEslintOnly(t *testing.T) {
	t.Parallel()

	ev := []detect.Evidence{{
		Ecosystem:  "node",
		Confidence: detect.ConfidenceMedium,
		Signals:    []string{"package.json"},
		Frameworks: []string{"react", "next"},
	}}
	plan, err := generate.Build(ev, []string{"vscode"})
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	ext := findFile(t, plan, ".vscode/extensions.json")
	if !bytes.Contains(ext, []byte("dbaeumer.vscode-eslint")) {
		t.Errorf("react extensions should contain eslint, got:\n%s", ext)
	}
	for _, unwanted := range []string{"Vue.volar", "svelte.svelte-vscode", "astro-build.astro-vscode", "oven.bun-vscode"} {
		if bytes.Contains(ext, []byte(unwanted)) {
			t.Errorf("react extensions should not contain %q, got:\n%s", unwanted, ext)
		}
	}
}

func TestForbiddenIDsNeverEmitted(t *testing.T) {
	t.Parallel()

	ev := []detect.Evidence{
		{
			Ecosystem:      "ruby",
			Confidence:     detect.ConfidenceHigh,
			Signals:        []string{"Gemfile", "Gemfile.lock"},
			PackageManager: "bundler",
		},
		{
			Ecosystem:  "node",
			Confidence: detect.ConfidenceMedium,
			Signals:    []string{"package.json"},
			Frameworks: []string{"svelte"},
		},
	}
	plan, err := generate.Build(ev, nil)
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	var all bytes.Buffer
	for _, f := range plan.Files {
		all.Write(f.Content)
	}
	for _, want := range []string{"Shopify.ruby-lsp", "svelte.svelte-vscode"} {
		if !bytes.Contains(all.Bytes(), []byte(want)) {
			t.Errorf("output should contain %q", want)
		}
	}
	for _, forbidden := range []string{"rebornix.Ruby", "octref.vetur", "JamesBirtles.svelte-vscode"} {
		if bytes.Contains(all.Bytes(), []byte(forbidden)) {
			t.Errorf("output must never contain forbidden %q", forbidden)
		}
	}
	// rust-lang.rust-analyzer is allowed; bare rust-lang.rust must not appear
	// as a standalone recommendation. Check the vscode array explicitly.
	ext := findFile(t, plan, ".vscode/extensions.json")
	if strings.Contains(string(ext), `"rust-lang.rust"`) {
		t.Errorf("extensions must not contain bare rust-lang.rust, got:\n%s", ext)
	}
}

func TestUnionIdempotentRerender(t *testing.T) {
	t.Parallel()

	ev := []detect.Evidence{
		{
			Ecosystem:      "node",
			Confidence:     detect.ConfidenceHigh,
			Signals:        []string{"package.json", "bun.lock"},
			PackageManager: "bun",
			Runtime:        "bun",
			Frameworks:     []string{"react", "next", "vue", "svelte", "astro"},
		},
		{Ecosystem: "csharp", Confidence: detect.ConfidenceHigh, Signals: []string{"app.csproj", "global.json"}},
		{Ecosystem: "java", Confidence: detect.ConfidenceMedium, Signals: []string{"pom.xml"}},
		{
			Ecosystem:      "ruby",
			Confidence:     detect.ConfidenceHigh,
			Signals:        []string{"Gemfile", "Gemfile.lock"},
			PackageManager: "bundler",
		},
		{Ecosystem: "erlang", Confidence: detect.ConfidenceMedium, Signals: []string{"rebar.config"}},
	}
	first, err := generate.Build(ev, nil)
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	second, err := generate.Build(ev, nil)
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	if len(first.Files) != len(second.Files) {
		t.Fatalf("re-render file count = %d, want %d", len(second.Files), len(first.Files))
	}
	for i := range first.Files {
		if !bytes.Equal(first.Files[i].Content, second.Files[i].Content) {
			t.Errorf("re-render of %q changed bytes", first.Files[i].Path)
		}
	}
	ext := findFile(t, first, ".vscode/extensions.json")
	for _, want := range []string{
		"dbaeumer.vscode-eslint",
		"oven.bun-vscode",
		"Vue.volar",
		"svelte.svelte-vscode",
		"astro-build.astro-vscode",
		"ms-dotnettools.csharp",
		"redhat.java",
		"Shopify.ruby-lsp",
		"pgourlain.erlang",
	} {
		if count := bytes.Count(ext, []byte(want)); count != 1 {
			t.Errorf("extensions should contain %q exactly once, found %d in:\n%s", want, count, ext)
		}
	}
}

func TestCursorProseMentionsCSharp(t *testing.T) {
	t.Parallel()

	ev := []detect.Evidence{{
		Ecosystem:  "csharp",
		Confidence: detect.ConfidenceHigh,
		Signals:    []string{"app.csproj", "global.json"},
	}}
	plan, err := generate.Build(ev, []string{"cursor"})
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}
	mdc := findFile(t, plan, ".cursor/rules/stackup.mdc")
	if !bytes.Contains(mdc, []byte("ms-dotnettools.csharp")) {
		t.Errorf("cursor rules should mention ms-dotnettools.csharp in prose, got:\n%s", mdc)
	}
	if bytes.Contains(mdc, []byte(`"recommendations"`)) {
		t.Errorf("cursor rules must not contain a recommendations array, got:\n%s", mdc)
	}
}
