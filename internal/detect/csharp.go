package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	csharpTFM  = regexp.MustCompile(`<TargetFramework>([^<]+)</TargetFramework>`)
	csharpTFMs = regexp.MustCompile(`<TargetFrameworks>([^<]+)</TargetFrameworks>`)
)

// CSharpDetector recognizes C# projects via root *.csproj and solution files.
type CSharpDetector struct{}

// Name returns the ecosystem key.
func (CSharpDetector) Name() string { return "csharp" }

// Detect returns C# evidence: a root *.csproj or *.sln[x] alone grades
// Medium, plus packages.lock.json or global.json grades High. A lone
// dotnet-tools hint, NuGet.config, or Directory.Build.props grades Low.
func (CSharpDetector) Detect(root string) ([]Evidence, error) {
	matches, _ := filepath.Glob(filepath.Join(root, "*.csproj"))
	var signals []string
	for _, m := range matches {
		signals = append(signals, filepath.Base(m))
	}
	for _, sln := range []string{"*.sln", "*.slnx"} {
		if found, _ := filepath.Glob(filepath.Join(root, sln)); len(found) > 0 {
			for _, m := range found {
				signals = append(signals, filepath.Base(m))
			}
		}
	}
	if len(signals) > 0 {
		ev := Evidence{
			Ecosystem:  "csharp",
			Confidence: ConfidenceMedium,
			Signals:    signals,
		}
		if v := csharpVersionHint(root, matches); v != "" {
			ev.VersionHint = v
		}
		for _, pin := range []string{"packages.lock.json", "global.json"} {
			ok, err := exists(root, pin)
			if err != nil {
				return nil, err
			}
			if ok {
				ev.Signals = append(ev.Signals, pin)
				ev.Confidence = ConfidenceHigh
			}
		}
		return []Evidence{ev}, nil
	}

	var hints []string
	for _, hint := range []string{
		filepath.Join(".config", "dotnet-tools.json"),
		"NuGet.config",
		"Directory.Build.props",
	} {
		ok, err := exists(root, hint)
		if err != nil {
			return nil, err
		}
		if ok {
			hints = append(hints, hint)
		}
	}
	if len(hints) > 0 {
		return []Evidence{{
			Ecosystem:  "csharp",
			Confidence: ConfidenceLow,
			Signals:    hints,
		}}, nil
	}
	return nil, nil
}

// csharpVersionHint prefers global.json sdk.version, then the first TFM.
func csharpVersionHint(root string, matches []string) string {
	if raw, err := os.ReadFile(filepath.Join(root, "global.json")); err == nil {
		var doc struct {
			SDK struct {
				Version string `json:"version"`
			} `json:"sdk"`
		}
		if json.Unmarshal(raw, &doc) == nil && doc.SDK.Version != "" {
			return doc.SDK.Version
		}
	}
	for _, m := range matches {
		raw, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		if found := csharpTFM.FindStringSubmatch(string(raw)); found != nil {
			if v := strings.TrimSpace(found[1]); v != "" {
				return v
			}
		}
		if found := csharpTFMs.FindStringSubmatch(string(raw)); found != nil {
			if first := strings.TrimSpace(strings.Split(found[1], ";")[0]); first != "" {
				return first
			}
		}
	}
	return ""
}
