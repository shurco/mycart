package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"existing dir", dir, true},
		{"file, not dir", file, false},
		{"missing", filepath.Join(dir, "missing"), false},
		{"empty path", "", false},
		{"overlong path", strings.Repeat("x", 500), false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsDir(tc.path); got != tc.want {
				t.Errorf("IsDir(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestIsEmptyDir(t *testing.T) {
	t.Parallel()

	empty := t.TempDir()
	nonEmpty := t.TempDir()
	if err := os.WriteFile(filepath.Join(nonEmpty, "x"), nil, 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if !IsEmptyDir(empty) {
		t.Error("empty dir reported non-empty")
	}
	if IsEmptyDir(nonEmpty) {
		t.Error("non-empty dir reported empty")
	}
	if IsEmptyDir(filepath.Join(empty, "missing")) {
		t.Error("missing dir should not report as empty (Open fails)")
	}
}

func TestWorkdir(t *testing.T) {
	t.Parallel()

	if dir := Workdir(); dir == "" {
		t.Error("Workdir returned empty string")
	}
}

func TestMkDirs(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b", "c")

	if err := MkDirs(0o755, a, b); err != nil {
		t.Fatalf("MkDirs: %v", err)
	}
	if !IsDir(a) || !IsDir(b) {
		t.Error("expected both dirs to exist")
	}

	// Idempotent second call must not error.
	if err := MkDirs(0o755, a, b); err != nil {
		t.Errorf("MkDirs (second call) returned %v, want nil", err)
	}
}

func TestMkSubDirs(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	if err := MkSubDirs(0o755, parent, "x", "y", "z"); err != nil {
		t.Fatalf("MkSubDirs: %v", err)
	}
	for _, sub := range []string{"x", "y", "z"} {
		if !IsDir(filepath.Join(parent, sub)) {
			t.Errorf("sub %q not created", sub)
		}
	}
}
