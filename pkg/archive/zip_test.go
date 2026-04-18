package archive

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// zipIntoFile writes a minimal deflate zip archive containing the provided
// entries and returns its on-disk path.
func zipIntoFile(t *testing.T, entries map[string]string) string {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("Create %q: %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("Write %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	fp := filepath.Join(t.TempDir(), "a.zip")
	if err := os.WriteFile(fp, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return fp
}

func TestExtractZip_Basic(t *testing.T) {
	t.Parallel()

	src := zipIntoFile(t, map[string]string{
		"a/b.txt":   "body-b",
		"top.txt":   "top",
		"nested/c/": "", // directory entry, skipped
	})
	dest := t.TempDir()

	if err := ExtractZip(src, dest); err != nil {
		t.Fatalf("ExtractZip: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "a", "b.txt"))
	if err != nil {
		t.Fatalf("ReadFile a/b.txt: %v", err)
	}
	if string(got) != "body-b" {
		t.Errorf("body wrong: %q", got)
	}
	if _, err := os.ReadFile(filepath.Join(dest, "top.txt")); err != nil {
		t.Errorf("top.txt missing: %v", err)
	}
}

func TestExtractZip_Zipslip(t *testing.T) {
	t.Parallel()

	src := zipIntoFile(t, map[string]string{
		"../../evil.txt": "should not be written",
	})
	err := ExtractZip(src, t.TempDir())
	if err == nil {
		t.Fatal("expected zip-slip detection to return an error")
	}
}

func TestExtractZip_MissingSource(t *testing.T) {
	t.Parallel()

	if err := ExtractZip(filepath.Join(t.TempDir(), "missing.zip"), t.TempDir()); err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestZipArchive_RoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	zipPath := filepath.Join(dir, "out.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ar := NewZipArchive(f)
	if err := ar.Directory("root"); err != nil {
		t.Fatalf("Directory: %v", err)
	}

	// Write one file by re-using the Header API with a real os.FileInfo.
	payload := filepath.Join(dir, "payload.txt")
	if err := os.WriteFile(payload, []byte("hello zip"), 0o644); err != nil {
		t.Fatalf("payload: %v", err)
	}
	info, err := os.Stat(payload)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	w, err := ar.Header(info)
	if err != nil {
		t.Fatalf("Header: %v", err)
	}
	if _, err := w.Write([]byte("hello zip")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := ar.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Extract and verify.
	dest := filepath.Join(dir, "out")
	if err := ExtractZip(zipPath, dest); err != nil {
		t.Fatalf("ExtractZip: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "root", "payload.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "hello zip" {
		t.Errorf("round-trip content mismatch: %q", got)
	}
}
