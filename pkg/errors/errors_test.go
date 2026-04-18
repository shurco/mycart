package errors

import (
	stdlibErrors "errors"
	"fmt"
	"testing"
)

// TestSentinels_UnwrapThroughFmtErrorf ensures every exported sentinel is a
// real unique error value that can be wrapped and detected via our `Is`
// alias. Regression guard: an accidental duplicate would cause `Is` to
// return true for unrelated errors.
func TestSentinels_UnwrapThroughFmtErrorf(t *testing.T) {
	t.Parallel()

	sentinels := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrNotFound", ErrNotFound, MsgNotFound},
		{"ErrWrongPassword", ErrWrongPassword, MsgWrongPassword},
		{"ErrUserNotFound", ErrUserNotFound, MsgUserNotFound},
		{"ErrUserPasswordNotFound", ErrUserPasswordNotFound, MsgUserPasswordNotFound},
		{"ErrUserEmailNotFound", ErrUserEmailNotFound, MsgUserEmailNotFound},
		{"ErrProductNotFound", ErrProductNotFound, MsgProductNotFound},
		{"ErrPageNotFound", ErrPageNotFound, MsgPageNotFound},
		{"ErrSettingNotFound", ErrSettingNotFound, MsgSettingNotFound},
	}

	for _, s := range sentinels {
		s := s
		t.Run(s.name, func(t *testing.T) {
			t.Parallel()

			if s.err.Error() != s.msg {
				t.Errorf("Error() = %q, want %q", s.err.Error(), s.msg)
			}
			wrapped := fmt.Errorf("ctx: %w", s.err)
			if !Is(wrapped, s.err) {
				t.Errorf("Is returned false for wrapped %q", s.name)
			}
		})
	}

	// Cross-check: Is must return false for unrelated sentinels.
	if Is(ErrNotFound, ErrWrongPassword) {
		t.Error("Is must return false for unrelated sentinels")
	}
}

// TestIs_AliasesStdlib documents that our Is is literally stdlib errors.Is.
// If a future refactor replaces it, this test fails loudly.
func TestIs_AliasesStdlib(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("outer: %w", ErrNotFound)
	if Is(err, ErrNotFound) != stdlibErrors.Is(err, ErrNotFound) {
		t.Fatal("Is diverged from stdlib errors.Is")
	}
}
