package webutil

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// runHandler wires a one-off Fiber app to exercise ParsePagination against a
// real request URL. Keeping it local avoids dragging in the heavier
// internal/testutil harness, which is tuned for DB-backed tests.
func runHandler(t *testing.T, rawQuery string) Pagination {
	t.Helper()

	app := fiber.New()
	var got Pagination
	app.Get("/", func(c fiber.Ctx) error {
		got = ParsePagination(c)
		return c.SendStatus(fiber.StatusOK)
	})

	target := "/"
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	req := httptest.NewRequest(fiber.MethodGet, target, nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	t.Cleanup(func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	})
	return got
}

func TestParsePagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      string
		wantPage   int
		wantLimit  int
		wantOffset int
	}{
		{"defaults", "", 1, 20, 0},
		{"valid page and limit", "page=3&limit=10", 3, 10, 20},
		{"zero page falls back", "page=0&limit=15", 1, 15, 0},
		{"negative page falls back", "page=-5&limit=15", 1, 15, 0},
		{"zero limit falls back", "page=2&limit=0", 2, 20, 20},
		{"negative limit falls back", "page=2&limit=-3", 2, 20, 20},
		{"limit above cap is clamped", "page=2&limit=5000", 2, MaxLimit, MaxLimit},
		{"garbage falls back to defaults", "page=abc&limit=xyz", 1, 20, 0},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := runHandler(t, tc.query)
			if got.Page != tc.wantPage {
				t.Errorf("Page: got %d, want %d", got.Page, tc.wantPage)
			}
			if got.Limit != tc.wantLimit {
				t.Errorf("Limit: got %d, want %d", got.Limit, tc.wantLimit)
			}
			if got.Offset != tc.wantOffset {
				t.Errorf("Offset: got %d, want %d", got.Offset, tc.wantOffset)
			}
		})
	}
}
