package detect

import (
	"os"
	"path/filepath"
	"regexp"
)

var terraformRequiredVersion = regexp.MustCompile(`required_version\s*=\s*"([^"]+)"`)

// TerraformDetector recognizes Terraform projects via *.tf and lockfiles.
type TerraformDetector struct{}

// Name returns the ecosystem key.
func (TerraformDetector) Name() string { return "terraform" }

// Detect returns Terraform evidence: a root *.tf file alone grades Medium,
// plus .terraform.lock.hcl grades High. A lone .terraform-version grades
// Low. VersionHint prefers .terraform-version, then required_version in
// *.tf files.
func (TerraformDetector) Detect(root string) ([]Evidence, error) {
	matches, _ := filepath.Glob(filepath.Join(root, "*.tf"))
	var signals []string
	for _, m := range matches {
		signals = append(signals, filepath.Base(m))
	}
	if len(signals) == 0 {
		if version, ok := readVersionFile(filepath.Join(root, ".terraform-version")); ok {
			return []Evidence{{
				Ecosystem:   "terraform",
				Confidence:  ConfidenceLow,
				Signals:     []string{".terraform-version"},
				VersionHint: version,
			}}, nil
		}
		return nil, nil
	}

	ev := Evidence{
		Ecosystem:  "terraform",
		Confidence: ConfidenceMedium,
		Signals:    signals,
	}
	if v := terraformVersionHint(root, matches); v != "" {
		ev.VersionHint = v
	}
	ok, err := exists(root, ".terraform.lock.hcl")
	if err != nil {
		return nil, err
	}
	if ok {
		ev.Signals = append(ev.Signals, ".terraform.lock.hcl")
		ev.Confidence = ConfidenceHigh
	}
	return []Evidence{ev}, nil
}

// terraformVersionHint prefers .terraform-version, then required_version in
// the matched *.tf files. Reads are best-effort: any failure yields an
// empty hint, never an error.
func terraformVersionHint(root string, matches []string) string {
	if v, ok := readVersionFile(filepath.Join(root, ".terraform-version")); ok {
		return v
	}
	for _, m := range matches {
		raw, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		if found := terraformRequiredVersion.FindStringSubmatch(string(raw)); found != nil {
			return found[1]
		}
	}
	return ""
}
