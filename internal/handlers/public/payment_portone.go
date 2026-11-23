package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/httpclient"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

// portoneAPIURL can be overridden for testing
var portoneAPIURL = "https://api.portone.io"

// GetPortoneConfig returns public PortOne configuration (store_id, channel_key)
// API secret is NOT exposed to frontend for security
//
// @Summary      Get PortOne config
// @Description  Get public PortOne configuration for browser SDK
// @Tags         Cart
// @Produce      json
// @Success      200 {object} webutil.HTTPResponse "PortOne public config"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/cart/portone-config [get]
func GetPortoneConfig(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	settings, err := queries.GetSettingByGroup[models.Portone](c.Context(), db)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Only expose store_id, channel_key, and debug_enabled, NOT api_secret
	config := map[string]interface{}{
		"store_id":      settings.StoreID,
		"channel_key":   settings.ChannelKey,
		"debug_enabled": settings.DebugEnabled,
	}

	return webutil.Response(c, fiber.StatusOK, "PortOne config", config)
}

// callPortoneAPI makes authenticated HTTP request to PortOne API
func callPortoneAPI(endpoint string, apiSecret string) (*http.Response, error) {
	url := portoneAPIURL + endpoint

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "PortOne "+apiSecret)
	req.Header.Set("Content-Type", "application/json")

	client := httpclient.NewWithTimeout(10 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call portone api: %w", err)
	}

	return resp, nil
}

// validatePaymentAmount verifies payment amount matches cart total
func validatePaymentAmount(paymentTotal int, cartTotal int, log *logging.Log) error {
	expectedAmount := cartTotal * 100
	if paymentTotal != expectedAmount {
		log.Error().Msgf("Amount mismatch: expected %d, got %d", expectedAmount, paymentTotal)
		return fmt.Errorf("amount mismatch")
	}
	return nil
}

// validatePaymentCurrency verifies payment currency matches cart currency
func validatePaymentCurrency(paymentCurrency, cartCurrency string, log *logging.Log) error {
	if paymentCurrency != cartCurrency {
		log.Error().Msgf("Currency mismatch: expected %s, got %s", cartCurrency, paymentCurrency)
		return fmt.Errorf("currency mismatch")
	}
	return nil
}

// validateCartID verifies cart_id in payment custom data
func validateCartID(customDataJSON, expectedCartID string, log *logging.Log) error {
	var customData struct {
		CartID string `json:"cart_id"`
	}
	if err := json.Unmarshal([]byte(customDataJSON), &customData); err != nil {
		return fmt.Errorf("failed to parse custom data: %w", err)
	}
	if customData.CartID != expectedCartID {
		log.Error().Msgf("Cart ID mismatch: expected %s, got %s", expectedCartID, customData.CartID)
		return fmt.Errorf("cart ID mismatch")
	}
	return nil
}

// portonePayment is the part of a PortOne payment record that is checked
// before a cart is marked as paid.
type portonePayment struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Amount struct {
		Total    int    `json:"total"`
		Currency string `json:"currency"`
	} `json:"amount"`
	CustomData string `json:"customData"`
}

// isPaid reports whether PortOne considers the payment settled.
func (p *portonePayment) isPaid() bool {
	return p.Status == "PAID" || p.Status == "VIRTUAL_ACCOUNT_ISSUED"
}

// markPortonePaid records on the cart that PortOne payment paymentID paid for it.
func markPortonePaid(ctx context.Context, db *queries.Base, cart *models.Cart, paymentID string) error {
	cart.PaymentID = paymentID
	cart.PaymentStatus = "paid"
	cart.PaymentSystem = "portone"
	return db.UpdateCart(ctx, cart)
}
