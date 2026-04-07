package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/shurco/mycart/internal/testutil"
)

func TestCarts(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/_/carts", Carts)

	tests := []struct {
		name      string
		query     string
		wantPage  int
		wantLimit int
	}{
		{"default pagination", "", 1, 20},
		{"custom pagination", "?page=2&limit=5", 2, 5},
		{"page=0 resets to 1", "?page=0&limit=10", 1, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodGet, "/api/_/carts"+tt.query, "", "")
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", resp.StatusCode)
			}

			var res struct {
				Result struct {
					Page  int `json:"page"`
					Limit int `json:"limit"`
				} `json:"result"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&res)

			if res.Result.Page != tt.wantPage {
				t.Errorf("page = %d, want %d", res.Result.Page, tt.wantPage)
			}
			if res.Result.Limit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", res.Result.Limit, tt.wantLimit)
			}
		})
	}
}

func TestCart(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/_/carts/:cart_id", Cart)

	tests := []struct {
		name       string
		cartID     string
		wantStatus []int
	}{
		{"existing cart from fixtures", "iodz4ibf5h5zmov", []int{http.StatusOK}},
		{"non-existent cart", "nonexistent12345", []int{http.StatusNotFound, http.StatusInternalServerError}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodGet, "/api/_/carts/"+tt.cartID, "", "")
			testutil.AssertStatus(t, resp, tt.wantStatus...)
		})
	}
}

func TestCartSendMail(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Post("/api/_/carts/:cart_id/mail", CartSendMail)

	tests := []struct {
		name       string
		cartID     string
		wantStatus []int
	}{
		{"existing cart", "iodz4ibf5h5zmov", []int{http.StatusOK, http.StatusInternalServerError}},
		{"non-existent cart", "nonexistent12345", []int{http.StatusInternalServerError}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodPost, "/api/_/carts/"+tt.cartID+"/mail", "", "")
			testutil.AssertStatus(t, resp, tt.wantStatus...)
		})
	}
}
