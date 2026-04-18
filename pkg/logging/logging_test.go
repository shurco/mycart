package logging

import (
	"errors"
	"testing"
)

// Logging is a thin wrapper over zerolog that writes to os.Stderr. There is
// no configurable writer, so the tests below focus on invariants rather than
// exact output: New() must never return nil, and ErrorStack must tolerate
// nil errors and wrapped errors without panicking.

func TestNew_ReturnsUsableLogger(t *testing.T) {
	t.Parallel()

	log := New()
	if log == nil || log.Logger == nil {
		t.Fatal("New returned nil logger")
	}
	// A second call must also succeed (no init-time state we cannot share).
	if New() == nil {
		t.Fatal("second New() returned nil")
	}
}

func TestErrorStack_SafeWithRealError(t *testing.T) {
	t.Parallel()

	log := New()
	// If this panics, the test fails. Output goes to stderr and is not
	// asserted — zerolog's own tests cover the marshalling.
	log.ErrorStack(errors.New("boom"))
}

func TestErrorStack_SafeWithNilError(t *testing.T) {
	t.Parallel()

	log := New()
	// zerolog permits Err(nil); we just verify no panic escapes.
	log.ErrorStack(nil)
}
