package fsutil

import (
	"embed"
	"os"
	"path/filepath"
	"testing"
)

//go:embed testdata/sample/*
var embeddedSample embed.FS

func TestEmbedExtract(t *testing.T) {
	t.Parallel()

	// EmbedExtract writes to paths relative to process CWD, so chdir into
	// a temp dir to avoid polluting the repo.
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if err := EmbedExtract(embeddedSample, "testdata/sample"); err != nil {
		t.Fatalf("EmbedExtract: %v", err)
	}

	out := filepath.Join(tmp, "testdata", "sample", "hello.txt")
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("extracted file missing: %v", err)
	}
	if string(data) != "hello embed\n" {
		t.Errorf("content = %q, want %q", data, "hello embed\n")
	}
}
