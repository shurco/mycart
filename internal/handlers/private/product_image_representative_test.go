package handlers

import (
	"net/http"
	"testing"

	"github.com/shurco/mycart/internal/testutil"
)

func TestSetProductRepresentativeImage(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Post("/api/_/products/:product_id/images/:image_id/representative", SetProductRepresentativeImage)

	tests := []struct {
		name       string
		productID  string
		imageID    string
		wantStatus int
	}{
		{
			name:       "successful representative image update",
			productID:  "fv6c9s9cqzf36sc",
			imageID:    "dj9bae53oob0ukj",
			wantStatus: http.StatusOK,
		},
		{
			name:       "image not found",
			productID:  "fv6c9s9cqzf36sc",
			imageID:    "nonexistent123",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "image belongs to different product",
			productID:  "different123456",
			imageID:    "dj9bae53oob0ukj",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "/api/_/products/" + tt.productID + "/images/" + tt.imageID + "/representative"
			resp := testutil.DoRequest(t, app, http.MethodPost, path, "", "")
			testutil.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}
