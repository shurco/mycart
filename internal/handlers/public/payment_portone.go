package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

	client := &http.Client{Timeout: 10 * time.Second}
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

	// Call PortOne API and parse response
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
		log.Error().Msgf("PortOne API error: %d %s", resp.StatusCode, string(body))
		return webutil.StatusInternalServerError(c)
	}

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

	// Validate payment details
	if err := validatePaymentAmount(payment.Amount.Total, cart.AmountTotal, log); err != nil {
		return webutil.StatusBadRequest(c, err.Error())
	}

	if err := validatePaymentCurrency(payment.Amount.Currency, cart.Currency, log); err != nil {
		return webutil.StatusBadRequest(c, err.Error())
	}

	if err := validateCartID(payment.CustomData, request.CartID, log); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
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

// verifyWebhookSignature verifies webhook signature using HMAC-SHA256
func verifyWebhookSignature(body []byte, signature string, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// PortoneWebhook handles PortOne webhook notifications
//
// @Summary      PortOne webhook
// @Description  Handle PortOne webhook notifications for payment events
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Param        PortOne-Signature header string true "Webhook signature"
// @Success      200 {string} string "OK"
// @Failure      401 {string} string "Unauthorized"
// @Router       /api/payment/portone/webhook [post]
func PortoneWebhook(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	// Read raw body for signature verification
	body := c.Body()

	// Load PortOne settings
	settings, err := queries.GetSettingByGroup[models.Portone](c.Context(), db)
	if err != nil {
		log.ErrorStack(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	// Verify webhook signature
	signature := c.Get("PortOne-Signature")
	if signature == "" {
		log.Error().Msg("Missing webhook signature")
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	if !verifyWebhookSignature(body, signature, settings.ApiSecret) {
		log.Error().Msg("Invalid webhook signature")
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	// Parse webhook payload
	var webhook struct {
		Type string `json:"type"`
		Data struct {
			PaymentID string `json:"paymentId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &webhook); err != nil {
		log.ErrorStack(err)
		return c.SendStatus(fiber.StatusBadRequest)
	}

	// Extract payment_id
	paymentID := webhook.Data.PaymentID
	if paymentID == "" {
		log.Error().Msg("Missing payment_id in webhook")
		return c.SendStatus(fiber.StatusBadRequest)
	}

	// Get payment to extract cart_id
	resp, err := callPortoneAPI("/payments/"+paymentID, settings.ApiSecret)
	if err != nil {
		log.ErrorStack(err)
		return c.SendStatus(fiber.StatusOK) // Return 200 to prevent retry storms
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("Failed to get payment from PortOne API: %d", resp.StatusCode)
		return c.SendStatus(fiber.StatusOK) // Return 200 to prevent retry storms
	}

	var payment struct {
		Status     string `json:"status"`
		CustomData string `json:"customData"`
		Amount     struct {
			Total    int    `json:"total"`
			Currency string `json:"currency"`
		} `json:"amount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payment); err != nil {
		log.ErrorStack(err)
		return c.SendStatus(fiber.StatusOK)
	}

	var customData struct {
		CartID string `json:"cart_id"`
	}
	if err := json.Unmarshal([]byte(payment.CustomData), &customData); err != nil {
		log.ErrorStack(err)
		return c.SendStatus(fiber.StatusOK)
	}

	// Load cart
	cart, err := db.Cart(c.Context(), customData.CartID)
	if err != nil {
		log.ErrorStack(err)
		return c.SendStatus(fiber.StatusOK)
	}

	// Verify amount and currency
	if payment.Amount.Total != int(cart.AmountTotal*100) || payment.Amount.Currency != cart.Currency {
		log.Error().Msg("Amount/currency mismatch in webhook")
		return c.SendStatus(fiber.StatusOK)
	}

	// Update cart status
	if payment.Status == "PAID" || payment.Status == "VIRTUAL_ACCOUNT_ISSUED" {
		cart.PaymentID = paymentID
		cart.PaymentStatus = "paid"
		cart.PaymentSystem = "portone"
		if err := db.UpdateCart(c.Context(), cart); err != nil {
			log.ErrorStack(err)
		}
	}

	return c.SendStatus(fiber.StatusOK)
}
