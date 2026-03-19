package testutil

import (
	"os"
	"testing"
)

// WithCmdTestDir changes CWD to a temp directory managed by t.TempDir().
// Go automatically cleans it up when the test finishes.
// All relative artifacts (e.g. ./lc_uploads) will be created inside it.
func WithCmdTestDir(t *testing.T) func() {
	t.Helper()
	oldwd, _ := os.Getwd()
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir to %s: %v", tmpDir, err)
	}
	return func() {
		_ = os.Chdir(oldwd)
	}
}
