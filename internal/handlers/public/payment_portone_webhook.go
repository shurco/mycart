package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/logging"
)

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

	// Load PortOne settings
	settings, err := queries.GetSettingByGroup[models.Portone](c.Context(), db)
	if err != nil {
		log.ErrorStack(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	paymentID, status := portoneWebhookPaymentID(c, log, settings.ApiSecret)
	if status != fiber.StatusOK {
		return c.SendStatus(status)
	}

	// Get payment to extract cart_id
	payment := fetchPortoneWebhookPayment(log, paymentID, settings.ApiSecret)
	if payment == nil {
		return c.SendStatus(fiber.StatusOK)
	}

	cart := portoneWebhookCart(c, db, log, payment)
	if cart != nil && payment.isPaid() {
		// Update cart status
		if err := markPortonePaid(c.Context(), db, cart, paymentID); err != nil {
			log.ErrorStack(err)
		}
	}

	return c.SendStatus(fiber.StatusOK)
}

// portoneWebhookPaymentID authenticates the webhook request and returns the ID
// of the payment it announces. status is the HTTP status to answer with when it
// cannot be trusted or read, and fiber.StatusOK when paymentID is good.
func portoneWebhookPaymentID(c fiber.Ctx, log *logging.Log, apiSecret string) (paymentID string, status int) {
	// Read raw body for signature verification
	body := c.Body()

	// Verify webhook signature
	signature := c.Get("PortOne-Signature")
	if signature == "" {
		log.Error().Msg("Missing webhook signature")
		return "", fiber.StatusUnauthorized
	}

	if !verifyWebhookSignature(body, signature, apiSecret) {
		log.Error().Msg("Invalid webhook signature")
		return "", fiber.StatusUnauthorized
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
		return "", fiber.StatusBadRequest
	}

	// Extract payment_id
	if webhook.Data.PaymentID == "" {
		log.Error().Msg("Missing payment_id in webhook")
		return "", fiber.StatusBadRequest
	}
	return webhook.Data.PaymentID, fiber.StatusOK
}

// fetchPortoneWebhookPayment reads the announced payment back from PortOne. It
// returns nil on any failure, which the webhook answers with 200 anyway to
// prevent retry storms.
func fetchPortoneWebhookPayment(log *logging.Log, paymentID, apiSecret string) *portonePayment {
	resp, err := callPortoneAPI("/payments/"+paymentID, apiSecret)
	if err != nil {
		log.ErrorStack(err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("Failed to get payment from PortOne API: %d", resp.StatusCode)
		return nil
	}

	payment := new(portonePayment)
	if err := json.NewDecoder(resp.Body).Decode(payment); err != nil {
		log.ErrorStack(err)
		return nil
	}
	return payment
}

// portoneWebhookCart finds the cart the payment was made for and checks the
// payment against it. It returns nil when the webhook cannot be matched to a
// cart that the payment covers.
func portoneWebhookCart(c fiber.Ctx, db *queries.Base, log *logging.Log, payment *portonePayment) *models.Cart {
	var customData struct {
		CartID string `json:"cart_id"`
	}
	if err := json.Unmarshal([]byte(payment.CustomData), &customData); err != nil {
		log.ErrorStack(err)
		return nil
	}

	// Load cart
	cart, err := db.Cart(c.Context(), customData.CartID)
	if err != nil {
		log.ErrorStack(err)
		return nil
	}

	// Verify amount and currency
	if payment.Amount.Total != int(cart.AmountTotal*100) || payment.Amount.Currency != cart.Currency {
		log.Error().Msg("Amount/currency mismatch in webhook")
		return nil
	}
	return cart
}
