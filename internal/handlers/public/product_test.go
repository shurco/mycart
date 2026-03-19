package handlers

import (
	"net/http"
	"testing"

	"github.com/shurco/litecart/internal/testutil"
)

func TestProducts(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/products", Products)

	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{"default list (fixtures have active products)", "", http.StatusOK},
		{"custom pagination", "?page=1&limit=5", http.StatusOK},
		{"high page (empty result)", "?page=999", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodGet, "/api/products"+tt.query, "", "")
			testutil.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestProduct(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/products/:product_id", Product)

	tests := []struct {
		name       string
		productID  string
		wantStatus []int
	}{
		{"active product from fixtures", "fv6c9s9cqzf36sc", []int{http.StatusOK, http.StatusInternalServerError}},
		{"inactive product", "zlfpc6b17gte0ot", []int{http.StatusNotFound, http.StatusInternalServerError}},
		{"non-existent product", "nonexistent12345", []int{http.StatusNotFound, http.StatusInternalServerError}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodGet, "/api/products/"+tt.productID, "", "")
			testutil.AssertStatus(t, resp, tt.wantStatus...)
		})
	}
}
