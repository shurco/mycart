package strutil

import (
	"reflect"
	"testing"
)

func TestToSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		sep  []string
		want []string
	}{
		{"default comma", "a,b,c", nil, []string{"a", "b", "c"}},
		{"custom separator", "a|b|c", []string{"|"}, []string{"a", "b", "c"}},
		{"empty string keeps single empty element", "", nil, []string{""}},
		{"no separator present", "abc", nil, []string{"abc"}},
		{"trailing separator creates empty tail", "a,b,", nil, []string{"a", "b", ""}},
		{"multi-char separator", "a::b::c", []string{"::"}, []string{"a", "b", "c"}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ToSlice(tc.in, tc.sep...)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ToSlice(%q, %v) = %v, want %v", tc.in, tc.sep, got, tc.want)
			}
		})
	}
}

func TestToAny(t *testing.T) {
	t.Parallel()

	t.Run("empty input", func(t *testing.T) {
		t.Parallel()
		got := ToAny()
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("preserves order and values", func(t *testing.T) {
		t.Parallel()
		got := ToAny("a", "b", "c")
		want := []any{"a", "b", "c"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("handles single element", func(t *testing.T) {
		t.Parallel()
		got := ToAny("only")
		if len(got) != 1 || got[0].(string) != "only" {
			t.Errorf("unexpected result: %v", got)
		}
	})
}
