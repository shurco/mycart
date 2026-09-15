package handlers

import (
	"net/http"
	"testing"

	"github.com/shurco/mycart/internal/testutil"
)

func TestReorderProductImages(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Post("/api/_/products/:product_id/images/reorder", ReorderProductImages)

	tests := []struct {
		name       string
		productID  string
		body       string
		wantStatus int
	}{
		{
			name:       "successful batch position update",
			productID:  "fv6c9s9cqzf36sc",
			body:       `{"updates":[{"imageId":"dj9bae53oob0ukj","position":0}]}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty updates array",
			productID:  "fv6c9s9cqzf36sc",
			body:       `{"updates":[]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid request format",
			productID:  "fv6c9s9cqzf36sc",
			body:       `{invalid json}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing updates field",
			productID:  "fv6c9s9cqzf36sc",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodPost, "/api/_/products/"+tt.productID+"/images/reorder", tt.body, "")
			testutil.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
