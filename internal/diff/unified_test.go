package diff

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

// golden compares got against testdata/name, writing it back under -update.
func golden(t *testing.T, name string, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update to create)", name, err)
	}
	if got != string(want) {
		t.Errorf("golden %s mismatch:\n got:\n%s\nwant:\n%s", name, got, want)
	}
}

// hunkBodies splits a unified diff into per-hunk body line counts,
// skipping the ---/+++/@@ headers.
func hunkBodies(t *testing.T, ud string) []int {
	t.Helper()
	var bodies []int
	count := -1
	for line := range strings.Lines(ud) {
		switch {
		case strings.HasPrefix(line, "@@ "):
			count++
			bodies = append(bodies, 0)
		case strings.HasPrefix(line, "--- "), strings.HasPrefix(line, "+++ "):
		default:
			if count < 0 {
				t.Fatalf("diff body line outside any hunk: %q", line)
			}
			bodies[count]++
		}
	}
	return bodies
}

// numberedLines builds n deterministic "line %04d" lines (1-based).
func numberedLines(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("line %04d", i+1)
	}
	return out
}

// joinLines joins ls with newlines, adding a trailing newline when nl.
func joinLines(ls []string, nl bool) string {
	s := strings.Join(ls, "\n")
	if nl {
		s += "\n"
	}
	return s
}

func TestUnifiedBoundedHunks(t *testing.T) {
	t.Parallel()

	hugeOld := numberedLines(600)
	hugeNew := append([]string(nil), hugeOld...)
	hugeNew[9] = "line 0010 EDITED"
	hugeNew[589] = "line 0590 EDITED"

	wsOld := numberedLines(60)
	wsNew := append([]string(nil), wsOld...)
	wsNew[29] += "   "

	denseOld := numberedLines(300)
	denseNew := make([]string, len(denseOld))
	for i := range denseNew {
		denseNew[i] = fmt.Sprintf("replaced %04d", i+1)
	}

	tests := []struct {
		name string
		old  string
		new  string
		// wantHunks is the exact hunk count.
		wantHunks int
		// maxBody caps every hunk body in the case.
		maxBody int
		// golden pins the exact diff; "" skips the golden leg.
		golden string
		// mustContain lists snippets the diff must carry.
		mustContain []string
	}{
		{
			name:        "two distant changes split into two hunks",
			old:         joinLines(hugeOld, true),
			new:         joinLines(hugeNew, true),
			wantHunks:   2,
			maxBody:     2*diffContext + 2,
			golden:      "unified-huge-two-changes.golden",
			mustContain: []string{"line 0010 EDITED", "line 0590 EDITED"},
		},
		{
			name:        "trailing whitespace only stays one small hunk",
			old:         joinLines(wsOld, true),
			new:         joinLines(wsNew, true),
			wantHunks:   1,
			maxBody:     2*diffContext + 2,
			golden:      "unified-trailing-whitespace.golden",
			mustContain: []string{"-line 0030\n", "+line 0030   \n"},
		},
		{
			name:        "missing trailing newline marks the plus side",
			old:         "a\nb\n",
			new:         "a\nb",
			wantHunks:   1,
			maxBody:     4,
			golden:      "unified-missing-newline.golden",
			mustContain: []string{"+b\n" + noNewlineMarker},
		},
		{
			name:        "added trailing newline marks the minus side",
			old:         "a\nb",
			new:         "a\nb\n",
			wantHunks:   1,
			maxBody:     4,
			golden:      "unified-added-newline.golden",
			mustContain: []string{"-b\n" + noNewlineMarker},
		},
		{
			name:        "dense changes split at the hunk cap",
			old:         joinLines(denseOld, true),
			new:         joinLines(denseNew, true),
			wantHunks:   3,
			maxBody:     maxHunkLines,
			golden:      "",
			mustContain: []string{"-line 0001", "+replaced 0300"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := unified("f.txt", []byte(tt.old), []byte(tt.new))
			bodies := hunkBodies(t, got)
			if len(bodies) != tt.wantHunks {
				t.Fatalf("unified() hunks = %d, want %d:\n%s", len(bodies), tt.wantHunks, got)
			}
			for i, n := range bodies {
				if n > tt.maxBody {
					t.Errorf("unified() hunk %d body = %d lines, want at most %d", i, n, tt.maxBody)
				}
			}
			for _, want := range tt.mustContain {
				if !strings.Contains(got, want) {
					t.Errorf("unified() diff missing %q:\n%s", want, got)
				}
			}
			if tt.golden != "" {
				golden(t, tt.golden, got)
			}
		})
	}
}
