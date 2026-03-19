package handlers

import (
	"net/http"
	"testing"

	"github.com/shurco/litecart/internal/testutil"
)

func TestPage(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/pages/:page_slug", Page)

	tests := []struct {
		name       string
		slug       string
		wantStatus []int
	}{
		{"terms page from fixtures", "terms", []int{http.StatusOK}},
		{"privacy page from fixtures", "privacy", []int{http.StatusOK}},
		{"cookies page from fixtures", "cookies", []int{http.StatusOK}},
		{"non-existent page", "nonexistent", []int{http.StatusNotFound}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodGet, "/api/pages/"+tt.slug, "", "")
			testutil.AssertStatus(t, resp, tt.wantStatus...)
		})
	}
}
