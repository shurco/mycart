package fsutil

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"existing file", filePath, true},
		{"existing directory", dir, false},
		{"missing", filepath.Join(dir, "missing"), false},
		{"empty path", "", false},
		{"path way too long", strings.Repeat("x", 500), false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsFile(tc.path); got != tc.want {
				t.Errorf("IsFile(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestOpenFile_CreatesParentDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "deeply", "nested", "file.txt")

	f, err := OpenFile(target, FsCWFlags, 0o644)
	if err != nil {
		t.Fatalf("OpenFile: %v", err)
	}
	if _, err := f.WriteString("hello"); err != nil {
		t.Fatalf("WriteString: %v", err)
	}
	_ = f.Close()

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("got %q, want %q", data, "hello")
	}
}

func TestWriteOSFile(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		t.Parallel()
		fp := filepath.Join(t.TempDir(), "s.txt")
		f, err := OpenFile(fp, FsCWFlags, 0o644)
		if err != nil {
			t.Fatalf("OpenFile: %v", err)
		}
		n, err := WriteOSFile(f, "abc")
		if err != nil || n != 3 {
			t.Fatalf("WriteOSFile: n=%d err=%v", n, err)
		}
	})

	t.Run("bytes", func(t *testing.T) {
		t.Parallel()
		fp := filepath.Join(t.TempDir(), "b.txt")
		f, err := OpenFile(fp, FsCWFlags, 0o644)
		if err != nil {
			t.Fatalf("OpenFile: %v", err)
		}
		if _, err := WriteOSFile(f, []byte{1, 2, 3}); err != nil {
			t.Fatalf("WriteOSFile: %v", err)
		}
	})

	t.Run("reader", func(t *testing.T) {
		t.Parallel()
		fp := filepath.Join(t.TempDir(), "r.txt")
		f, err := OpenFile(fp, FsCWFlags, 0o644)
		if err != nil {
			t.Fatalf("OpenFile: %v", err)
		}
		if _, err := WriteOSFile(f, bytes.NewReader([]byte("data"))); err != nil {
			t.Fatalf("WriteOSFile: %v", err)
		}
	})

	t.Run("unsupported type panics", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if recover() == nil {
				t.Error("expected panic for unsupported data type")
			}
		}()
		fp := filepath.Join(t.TempDir(), "p.txt")
		f, err := OpenFile(fp, FsCWFlags, 0o644)
		if err != nil {
			t.Fatalf("OpenFile: %v", err)
		}
		_, _ = WriteOSFile(f, 42) // int is not supported
	})
}

func TestExtName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want string
	}{
		{"foo.txt", "txt"},
		{"archive.tar.gz", "gz"},
		{"/path/to/file.PNG", "PNG"},
		{"no_extension", ""},
		{"", ""},
		{".hidden", "hidden"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := ExtName(tc.in); got != tc.want {
				t.Errorf("ExtName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
