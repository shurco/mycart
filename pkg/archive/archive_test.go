package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTar_Zipslip(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	payload := []byte("evil")
	if err := tw.WriteHeader(&tar.Header{
		Name: "../../../etc/evil",
		Mode: 0o600,
		Size: int64(len(payload)),
	}); err != nil {
		t.Fatalf("WriteHeader: %v", err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatalf("Write: %v", err)
	}
	_ = tw.Close()
	_ = gzw.Close()

	src := filepath.Join(t.TempDir(), "a.tar.gz")
	if err := os.WriteFile(src, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := ExtractTar(src, t.TempDir()); err == nil {
		t.Fatal("expected zip-slip detection to return an error")
	}
}

func TestExtractTar_NonGzipReturnsError(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "notgz.bin")
	if err := os.WriteFile(src, []byte("not a gzip"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := ExtractTar(src, t.TempDir()); err == nil {
		t.Fatal("expected gzip header error")
	}
}

func TestExtractTar_MissingSource(t *testing.T) {
	t.Parallel()
	if err := ExtractTar(filepath.Join(t.TempDir(), "missing"), t.TempDir()); err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestTarArchive_WriteAndRead(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tarPath := filepath.Join(dir, "out.tar.gz")
	f, err := os.Create(tarPath)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	ar := NewTarArchive(f)
	if err := ar.Directory("bundle"); err != nil {
		t.Fatalf("Directory: %v", err)
	}

	payload := filepath.Join(dir, "p.txt")
	if err := os.WriteFile(payload, []byte("tar body"), 0o644); err != nil {
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
	if _, err := w.Write([]byte("tar body")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := ar.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Round-trip.
	dest := filepath.Join(dir, "out")
	if err := ExtractTar(tarPath, dest); err != nil {
		t.Fatalf("ExtractTar: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "bundle", "p.txt"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "tar body" {
		t.Errorf("round-trip mismatch: %q", got)
	}
}
