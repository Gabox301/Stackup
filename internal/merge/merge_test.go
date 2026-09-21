package merge_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gabox301/Stackup/internal/merge"
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

const commentedSettings = `{
  // Project editor settings; stackup must keep this comment.
  "editor.formatOnSave": false, // user prefers manual formatting
  "editor.tabSize": 4,
  "custom.userKey": "keep-me",
}
`

const generatedSettings = `{
  "editor.formatOnSave": true,
  "editor.tabSize": 2,
  "files.eol": "\n",
  "files.trimTrailingWhitespace": true
}
`

func TestMergePreservesCommentsAndUserKeys(t *testing.T) {
	t.Parallel()

	got, err := merge.Merge([]byte(commentedSettings), []byte(generatedSettings))
	if err != nil {
		t.Fatalf("Merge() unexpected error: %v", err)
	}
	for _, want := range []string{
		"// Project editor settings; stackup must keep this comment.",
		"// user prefers manual formatting",
		`"custom.userKey": "keep-me"`,
		`"files.eol"`,
		`"files.trimTrailingWhitespace"`,
	} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("Merge() result missing %q:\n%s", want, got)
		}
	}
	// Existing scalars win: the user disabled formatOnSave and uses width 4.
	for _, want := range []string{`"editor.formatOnSave": false`, `"editor.tabSize": 4`} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("Merge() overwrote a user scalar, missing %q:\n%s", want, got)
		}
	}
	golden(t, "merge-comment.golden", got)
}

func TestMergeUnionArraysKeepUserEntries(t *testing.T) {
	t.Parallel()

	existing := `{
  "recommendations": [
    "user.only-extension",
    "dbaeumer.vscode-eslint"
  ]
}
`
	generated := `{
  "recommendations": [
    "dbaeumer.vscode-eslint",
    "golang.go"
  ]
}
`
	got, err := merge.Merge([]byte(existing), []byte(generated))
	if err != nil {
		t.Fatalf("Merge() unexpected error: %v", err)
	}
	for _, want := range []string{`"user.only-extension"`, `"dbaeumer.vscode-eslint"`, `"golang.go"`} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("Merge() result missing %q:\n%s", want, got)
		}
	}
	if n := bytes.Count(got, []byte(`"dbaeumer.vscode-eslint"`)); n != 1 {
		t.Errorf("Merge() duplicated union entry %d times:\n%s", n, got)
	}
	golden(t, "merge-union.golden", got)
}

func TestMergeFreshFileReturnsGenerated(t *testing.T) {
	t.Parallel()

	got, err := merge.Merge(nil, []byte(generatedSettings))
	if err != nil {
		t.Fatalf("Merge() unexpected error: %v", err)
	}
	if !bytes.Equal(got, []byte(generatedSettings)) {
		t.Errorf("Merge() fresh result changed bytes:\n got:\n%s\nwant:\n%s", got, generatedSettings)
	}
}

func TestMergeIsIdempotent(t *testing.T) {
	t.Parallel()

	once, err := merge.Merge([]byte(commentedSettings), []byte(generatedSettings))
	if err != nil {
		t.Fatalf("Merge() unexpected error: %v", err)
	}
	twice, err := merge.Merge(once, []byte(generatedSettings))
	if err != nil {
		t.Fatalf("Merge() unexpected error: %v", err)
	}
	if !bytes.Equal(once, twice) {
		t.Errorf("re-merge changed bytes:\n once:\n%s\ntwice:\n%s", once, twice)
	}
}

func TestMergeInvalidExistingErrors(t *testing.T) {
	t.Parallel()

	if _, err := merge.Merge([]byte(`{oops`), []byte(generatedSettings)); err == nil {
		t.Fatal("Merge() expected error for invalid existing JSONC, got nil")
	}
}

func TestApplySetKey(t *testing.T) {
	t.Parallel()

	got, err := merge.Apply([]byte(commentedSettings), []merge.EditOp{
		{Path: "/editor.tabSize", Kind: merge.OpSetKey, Value: []byte(`2`)},
		{Path: "/files.eol", Kind: merge.OpSetKey, Value: []byte("\"\\n\"")},
		{Path: "/nested/deep/key", Kind: merge.OpSetKey, Value: []byte(`true`)},
	})
	if err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}
	for _, want := range []string{
		`"editor.tabSize": 2`,
		`"custom.userKey": "keep-me"`,
		"// Project editor settings; stackup must keep this comment.",
		`"key": true`,
	} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("Apply() result missing %q:\n%s", want, got)
		}
	}
}

func TestApplyUnionArray(t *testing.T) {
	t.Parallel()

	existing := `{"recommendations": ["a"]}`
	got, err := merge.Apply([]byte(existing), []merge.EditOp{
		{Path: "/recommendations", Kind: merge.OpUnionArray, Value: []byte(`["a", "b"]`)},
	})
	if err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}
	for _, want := range []string{`"a"`, `"b"`} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("Apply() result missing %q:\n%s", want, got)
		}
	}
	if n := bytes.Count(got, []byte(`"a"`)); n != 1 {
		t.Errorf("Apply() duplicated union entry %d times:\n%s", n, got)
	}
	// Union onto a missing path creates the array.
	created, err := merge.Apply([]byte(`{}`), []merge.EditOp{
		{Path: "/recommendations", Kind: merge.OpUnionArray, Value: []byte(`["b"]`)},
	})
	if err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}
	if !bytes.Contains(created, []byte(`"b"`)) {
		t.Errorf("Apply() did not create missing array:\n%s", created)
	}
}

func TestApplyNoopLeavesBytesUntouched(t *testing.T) {
	t.Parallel()

	got, err := merge.Apply([]byte(commentedSettings), []merge.EditOp{
		{Path: "/editor.tabSize", Kind: merge.OpNoop},
	})
	if err != nil {
		t.Fatalf("Apply() unexpected error: %v", err)
	}
	if !bytes.Equal(got, []byte(commentedSettings)) {
		t.Errorf("Noop changed bytes:\n got:\n%s\nwant:\n%s", got, commentedSettings)
	}
}

func TestApplyRejectsUnknownOp(t *testing.T) {
	t.Parallel()

	if _, err := merge.Apply([]byte(`{}`), []merge.EditOp{
		{Path: "/a", Kind: merge.OpKind("Delete")},
	}); err == nil {
		t.Fatal("Apply() expected error for unknown op, got nil")
	}
}

func TestApplyErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// existing is the document the ops run against.
		existing string
		// ops is the failing edit list.
		ops []merge.EditOp
		// wantSub must appear in the returned error.
		wantSub string
	}{
		{
			name:     "pointer without leading slash",
			existing: `{"a": 1}`,
			ops:      []merge.EditOp{{Path: "a", Kind: merge.OpSetKey, Value: []byte(`2`)}},
			wantSub:  `invalid path "a"`,
		},
		{
			name:     "setkey at document root",
			existing: `{"a": 1}`,
			ops:      []merge.EditOp{{Path: "", Kind: merge.OpSetKey, Value: []byte(`2`)}},
			wantSub:  "SetKey needs a member path",
		},
		{
			name:     "union array onto scalar",
			existing: `{"a": 1}`,
			ops:      []merge.EditOp{{Path: "/a", Kind: merge.OpUnionArray, Value: []byte(`[1, 2]`)}},
			wantSub:  `target "/a" is not an array`,
		},
		{
			name:     "union array onto object",
			existing: `{"a": {"b": 1}}`,
			ops:      []merge.EditOp{{Path: "/a", Kind: merge.OpUnionArray, Value: []byte(`[1]`)}},
			wantSub:  `target "/a" is not an array`,
		},
		{
			name:     "union array at root onto non-array document",
			existing: `{"a": 1}`,
			ops:      []merge.EditOp{{Path: "", Kind: merge.OpUnionArray, Value: []byte(`[1]`)}},
			wantSub:  "needs an array document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := merge.Apply([]byte(tt.existing), tt.ops); err == nil {
				t.Fatalf("Apply() expected error containing %q, got nil", tt.wantSub)
			} else if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("Apply() error = %q, want it to contain %q", err.Error(), tt.wantSub)
			}
		})
	}
}
