package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
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

	// Only expose store_id and channel_key, NOT api_secret
	config := map[string]string{
		"store_id":    settings.StoreID,
		"channel_key": settings.ChannelKey,
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

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call portone api: %w", err)
	}

	return resp, nil
}

// CompletePortonePayment verifies payment after browser completes PortOne payment flow
//
// @Summary      Complete PortOne payment
// @Description  Verify payment with PortOne API and update cart status
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Param        request body object{payment_id=string,cart_id=string} true "Payment completion request"
// @Success      200 {object} webutil.HTTPResponse "Payment verified"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/payment/portone/complete [post]
func CompletePortonePayment(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	var request struct {
		PaymentID string `json:"payment_id"`
		CartID    string `json:"cart_id"`
	}

	if err := c.Bind().JSON(&request); err != nil {
		return webutil.StatusBadRequest(c, "Invalid request")
	}

	if request.PaymentID == "" || request.CartID == "" {
		return webutil.StatusBadRequest(c, "Missing payment_id or cart_id")
	}

	// Load cart
	cart, err := db.Cart(c.Context(), request.CartID)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, "Cart not found")
	}

	// Load PortOne settings
	settings, err := queries.GetSettingByGroup[models.Portone](c.Context(), db)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Call PortOne API
	resp, err := callPortoneAPI("/payments/"+request.PaymentID, settings.ApiSecret)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return webutil.StatusBadRequest(c, "Payment not found")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Error("PortOne API error: %d %s", resp.StatusCode, string(body))
		return webutil.StatusInternalServerError(c)
	}

	// Parse response
	var payment struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Amount struct {
			Total    int    `json:"total"`
			Currency string `json:"currency"`
		} `json:"amount"`
		CustomData string `json:"customData"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payment); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, "Failed to decode payment")
	}

	// Verify payment status
	if payment.Status != "PAID" && payment.Status != "VIRTUAL_ACCOUNT_ISSUED" {
		return webutil.StatusBadRequest(c, fmt.Sprintf("Payment not completed: %s", payment.Status))
	}

	// Verify amount (PortOne uses smallest currency unit, e.g. cents)
	if payment.Amount.Total != int(cart.AmountTotal*100) {
		log.Error("Amount mismatch: expected %d, got %d", int(cart.AmountTotal*100), payment.Amount.Total)
		return webutil.StatusBadRequest(c, "Amount mismatch")
	}

	// Verify currency
	if payment.Amount.Currency != cart.Currency {
		log.Error("Currency mismatch: expected %s, got %s", cart.Currency, payment.Amount.Currency)
		return webutil.StatusBadRequest(c, "Currency mismatch")
	}

	// Verify cart_id in customData
	var customData struct {
		CartID string `json:"cart_id"`
	}
	if err := json.Unmarshal([]byte(payment.CustomData), &customData); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, "Failed to parse custom data")
	}
	if customData.CartID != request.CartID {
		log.Error("Cart ID mismatch: expected %s, got %s", request.CartID, customData.CartID)
		return webutil.StatusBadRequest(c, "Cart ID mismatch")
	}

	// Update cart status
	cart.PaymentID = request.PaymentID
	cart.PaymentStatus = "paid"
	cart.PaymentSystem = "portone"
	if err := db.UpdateCart(c.Context(), cart); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Payment verified", map[string]string{
		"status": payment.Status,
	})
}
