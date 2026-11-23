package queries

import (
	"reflect"
	"testing"
)

func TestPaginate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		limit      int
		offset     int
		wantQuery  string
		wantParams []any
	}{
		{"no limit leaves query unpaged", 0, 5, "SELECT 1", []any{"a"}},
		{"limit only", 10, 0, "SELECT 1 LIMIT ?", []any{"a", 10}},
		{"limit and offset", 10, 20, "SELECT 1 LIMIT ? OFFSET ?", []any{"a", 10, 20}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			query, params := paginate("SELECT 1", []any{"a"}, tt.limit, tt.offset)
			if query != tt.wantQuery || !reflect.DeepEqual(params, tt.wantParams) {
				t.Fatalf("got %q %v, want %q %v", query, params, tt.wantQuery, tt.wantParams)
			}
		})
	}
}
