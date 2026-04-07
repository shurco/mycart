package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/internal/testutil"
)

func TestPaymentList(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/cart/payment", PaymentList)

	resp := testutil.DoRequest(t, app, http.MethodGet, "/api/cart/payment", "", "")
	testutil.AssertStatus(t, resp, http.StatusOK)
}

func TestGetCart(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/cart/:cart_id", GetCart)

	tests := []struct {
		name       string
		cartID     string
		wantStatus []int
	}{
		{"existing cart from fixtures", "iodz4ibf5h5zmov", []int{http.StatusOK}},
		{"cancelled cart from fixtures", "efzs4xayz43f226", []int{http.StatusOK}},
		{"non-existent cart", "nonexistent12345", []int{http.StatusNotFound, http.StatusInternalServerError}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodGet, "/api/cart/"+tt.cartID, "", "")
			testutil.AssertStatus(t, resp, tt.wantStatus...)
		})
	}
}

func TestPaymentCancel(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	clearWebhookURL(t)

	app.Get("/cart/payment/cancel", PaymentCancel)

	tests := []struct {
		name       string
		query      string
		wantStatus []int
	}{
		{
			"cancel existing cart",
			"?cart_id=efzs4xayz43f226&payment_system=stripe",
			[]int{http.StatusSeeOther, http.StatusFound, http.StatusOK, http.StatusInternalServerError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/cart/payment/cancel"+tt.query, nil)
			resp, err := app.Test(req, fiber.TestConfig{Timeout: 5 * time.Second})
			if err != nil {
				t.Fatalf("GET /cart/payment/cancel: %v", err)
			}
			testutil.AssertStatus(t, resp, tt.wantStatus...)
		})
	}
}

func TestPaymentCallback(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Post("/cart/payment/callback", PaymentCallback)

	tests := []struct {
		name       string
		query      string
		wantStatus []int
	}{
		{
			"spectrocoin callback",
			"?cart_id=iodz4ibf5h5zmov&payment_system=spectrocoin",
			[]int{http.StatusOK, http.StatusBadRequest, http.StatusInternalServerError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodPost, "/cart/payment/callback"+tt.query, "", "")
			testutil.AssertStatus(t, resp, tt.wantStatus...)
		})
	}
}

func clearWebhookURL(t *testing.T) {
	t.Helper()
	db := queries.DB()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = db.UpdateSettingByKey(ctx, &models.SettingName{Key: "webhook_url", Value: ""})
}
