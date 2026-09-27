package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// PHPDetector recognizes PHP projects via composer.json and composer.lock.
type PHPDetector struct{}

// Name returns the ecosystem key.
func (PHPDetector) Name() string { return "php" }

// Detect returns PHP evidence: composer.json alone grades Medium with
// PackageManager composer, plus composer.lock grades High. A lone
// .php-version grades Low. VersionHint prefers .php-version, then the
// require.php constraint in composer.json.
func (PHPDetector) Detect(root string) ([]Evidence, error) {
	ok, err := exists(root, "composer.json")
	if err != nil {
		return nil, err
	}
	if !ok {
		if version, ok := readVersionFile(filepath.Join(root, ".php-version")); ok {
			return []Evidence{{
				Ecosystem:   "php",
				Confidence:  ConfidenceLow,
				Signals:     []string{".php-version"},
				VersionHint: version,
			}}, nil
		}
		return nil, nil
	}

	ev := Evidence{
		Ecosystem:      "php",
		Confidence:     ConfidenceMedium,
		Signals:        []string{"composer.json"},
		PackageManager: "composer",
	}
	if v := phpVersionHint(root); v != "" {
		ev.VersionHint = v
	}
	ok, err = exists(root, "composer.lock")
	if err != nil {
		return nil, err
	}
	if ok {
		ev.Signals = append(ev.Signals, "composer.lock")
		ev.Confidence = ConfidenceHigh
	}
	return []Evidence{ev}, nil
}

// phpVersionHint prefers .php-version, then require.php in composer.json.
// Reads are best-effort: any failure yields an empty hint, never an error.
func phpVersionHint(root string) string {
	if v, ok := readVersionFile(filepath.Join(root, ".php-version")); ok {
		return v
	}
	raw, err := os.ReadFile(filepath.Join(root, "composer.json"))
	if err != nil {
		return ""
	}
	var doc struct {
		Require map[string]string `json:"require"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return ""
	}
	if v, ok := doc.Require["php"]; ok {
		return v
	}
	return ""
}
