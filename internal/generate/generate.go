// Package generate renders per-IDE configuration files from detection evidence.
//
// Build produces a Plan of relative-path file operations with deterministic
// bytes: identical evidence and IDE selection always yields identical output.
// JSON targets are rendered as standard JSON for fresh files. Merging into
// existing user files (comment-preserving, union-no-delete) belongs to
// internal/merge and the apply flow, never to this package.
package generate

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"stackup/internal/detect"
)

//go:embed templates
var templateFS embed.FS

// Supported IDE identifiers. Keep these exact strings; they are user-facing
// flag values and directory names.
const (
	IDEVSCode = "vscode"
	IDECursor = "cursor"
	IDEDevin  = "devin"
	IDEKiro   = "kiro"
)

// SupportedIDEs returns the canonical IDE order.
func SupportedIDEs() []string {
	return []string{IDEVSCode, IDECursor, IDEDevin, IDEKiro}
}

// FileOp is a single rendered file with a slash-separated relative path
// (e.g. ".vscode/settings.json"). JSON marks targets that must be merged
// with merge.Merge when the file already exists; other files are plain text
// (markdown) written verbatim.
type FileOp struct {
	Path    string
	Content []byte
	JSON    bool
}

// Plan is the ordered set of files to create or merge.
type Plan struct {
	Files []FileOp
}

// Paths returns the relative paths in plan order.
func (p Plan) Paths() []string {
	out := make([]string, 0, len(p.Files))
	for _, f := range p.Files {
		out = append(out, f.Path)
	}
	return out
}

// stackInfo is the template view of one detection finding.
type stackInfo struct {
	Ecosystem      string
	Confidence     string
	Signals        string
	VersionHint    string
	PackageManager string
}

// templateData is the model passed to every embedded template.
type templateData struct {
	Stacks      []stackInfo
	StackList   string
	Generic     bool
	HasNode     bool
	HasGo       bool
	HasPython   bool
	HasRust     bool
	HasCSharp   bool
	HasJava     bool
	HasRuby     bool
	HasErlang   bool
	HasBun      bool
	Frameworks  []string
	Extensions  []string
	StackCounts int
}

// templateTarget binds an embedded template to a plan output path.
type templateTarget struct {
	template string
	path     string
	json     bool
}

var ideTargets = map[string][]templateTarget{
	IDEVSCode: {
		{"templates/vscode/settings.json.tmpl", ".vscode/settings.json", true},
		{"templates/vscode/extensions.json.tmpl", ".vscode/extensions.json", true},
		{"templates/vscode/launch.json.tmpl", ".vscode/launch.json", true},
		{"templates/vscode/tasks.json.tmpl", ".vscode/tasks.json", true},
	},
	IDECursor: {
		{"templates/cursor/rules.mdc.tmpl", ".cursor/rules/stackup.mdc", false},
		{"templates/cursor/mcp.json.tmpl", ".cursor/mcp.json", true},
	},
	IDEDevin: {
		{"templates/devin/rules.md.tmpl", ".devin/rules/stackup.md", false},
		{"templates/devin/mcp_config.json.tmpl", ".devin/mcp_config.json", true},
		{"templates/windsurf/rules.md.tmpl", ".windsurf/rules/stackup.md", false},
	},
	IDEKiro: {
		{"templates/kiro/steering-product.md.tmpl", ".kiro/steering/product.md", false},
		{"templates/kiro/steering-tech.md.tmpl", ".kiro/steering/tech.md", false},
		{"templates/kiro/steering-structure.md.tmpl", ".kiro/steering/structure.md", false},
		{"templates/kiro/specs-requirements.md.tmpl", ".kiro/specs/stackup/requirements.md", false},
		{"templates/kiro/specs-design.md.tmpl", ".kiro/specs/stackup/design.md", false},
		{"templates/kiro/specs-tasks.md.tmpl", ".kiro/specs/stackup/tasks.md", false},
		{"templates/kiro/mcp.json.tmpl", ".kiro/settings/mcp.json", true},
	},
}

// vscodeExtensionRecommendations maps ecosystems to marketplace IDs.
// Exact verified casings required; forbidden IDs (rebornix.Ruby,
// octref.vetur, rust-lang.rust, JamesBirtles.svelte-vscode) never enter.
var vscodeExtensionRecommendations = map[string][]string{
	"node":   {"dbaeumer.vscode-eslint"},
	"go":     {"golang.go"},
	"python": {"ms-python.python"},
	"rust":   {"rust-lang.rust-analyzer"},
	"csharp": {"ms-dotnettools.csharp"},
	"java":   {"redhat.java"},
	"ruby":   {"Shopify.ruby-lsp"},
	"erlang": {"pgourlain.erlang"},
}

// frameworkExtensionRecommendations maps presence-only framework signals to
// marketplace IDs. React and Next intentionally map to nothing (React=NONE;
// dbaeumer.vscode-eslint covers them).
var frameworkExtensionRecommendations = map[string][]string{
	"vue":       {"Vue.volar"},
	"nuxt":      {"Vue.volar"},
	"svelte":    {"svelte.svelte-vscode"},
	"sveltekit": {"svelte.svelte-vscode"},
	"astro":     {"astro-build.astro-vscode"},
}

// Build renders a Plan for the given evidence and IDE selection.
// An empty ides list selects every supported IDE in canonical order.
// Unknown IDE names return an error. Empty evidence renders generic
// stack-agnostic content so --allow-unknown flows still produce a plan.
func Build(evidences []detect.Evidence, ides []string) (Plan, error) {
	if len(ides) == 0 {
		ides = SupportedIDEs()
	}
	seen := map[string]bool{}
	ordered := make([]string, 0, len(ides))
	for _, ide := range ides {
		if _, ok := ideTargets[ide]; !ok {
			return Plan{}, fmt.Errorf("generate: unknown IDE %q (supported: %s)", ide, strings.Join(SupportedIDEs(), ", "))
		}
		if !seen[ide] {
			seen[ide] = true
			ordered = append(ordered, ide)
		}
	}

	data := buildData(evidences)
	var plan Plan
	for _, ide := range ordered {
		for _, target := range ideTargets[ide] {
			content, err := render(target.template, data)
			if err != nil {
				return Plan{}, err
			}
			plan.Files = append(plan.Files, FileOp{
				Path:    target.path,
				Content: content,
				JSON:    target.json,
			})
		}
	}
	return plan, nil
}

// buildData converts ranked evidence into the deterministic template model.
func buildData(evidences []detect.Evidence) templateData {
	data := templateData{}
	frameworkSet := map[string]bool{}
	for _, ev := range evidences {
		data.Stacks = append(data.Stacks, stackInfo{
			Ecosystem:      ev.Ecosystem,
			Confidence:     ev.Confidence.String(),
			Signals:        strings.Join(ev.Signals, ", "),
			VersionHint:    ev.VersionHint,
			PackageManager: ev.PackageManager,
		})
		switch ev.Ecosystem {
		case "node":
			data.HasNode = true
		case "go":
			data.HasGo = true
		case "python":
			data.HasPython = true
		case "rust":
			data.HasRust = true
		case "csharp":
			data.HasCSharp = true
		case "java":
			data.HasJava = true
		case "ruby":
			data.HasRuby = true
		case "erlang":
			data.HasErlang = true
		}
		if ev.Runtime == "bun" {
			data.HasBun = true
		}
		for _, fw := range ev.Frameworks {
			if !frameworkSet[fw] {
				frameworkSet[fw] = true
				data.Frameworks = append(data.Frameworks, fw)
			}
		}
	}
	names := make([]string, 0, len(data.Stacks))
	extSet := map[string]bool{}
	addExts := func(ids []string) {
		for _, ext := range ids {
			if !extSet[ext] {
				extSet[ext] = true
				data.Extensions = append(data.Extensions, ext)
			}
		}
	}
	for _, s := range data.Stacks {
		names = append(names, s.Ecosystem)
		addExts(vscodeExtensionRecommendations[s.Ecosystem])
	}
	if data.HasBun {
		addExts([]string{"oven.bun-vscode"})
	}
	for _, fw := range data.Frameworks {
		addExts(frameworkExtensionRecommendations[fw])
	}
	sort.Strings(data.Extensions)
	if len(names) == 0 {
		data.StackList = "unknown (no stacks detected)"
		data.Generic = true
	} else {
		data.StackList = strings.Join(names, ", ")
	}
	data.StackCounts = len(data.Stacks)
	return data
}

// containsString reports whether list holds s. It backs the "contains"
// template function used by prose targets for framework mentions.
func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// render executes one embedded template with the plan data.
func render(name string, data templateData) ([]byte, error) {
	raw, err := templateFS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("generate: read template %s: %w", name, err)
	}
	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"contains": containsString,
	}).Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("generate: parse template %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("generate: render template %s: %w", name, err)
	}
	return buf.Bytes(), nil
}
