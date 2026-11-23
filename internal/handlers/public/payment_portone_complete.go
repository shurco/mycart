package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

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

	payment, err := fetchPortonePaymentForComplete(c, log, request.PaymentID, settings.ApiSecret)
	if payment == nil {
		return err
	}

	if err := checkPortonePayment(payment, cart, request.CartID, log); err != nil {
		return webutil.StatusBadRequest(c, err.Error())
	}

	// Update cart status
	if err := markPortonePaid(c.Context(), db, cart, request.PaymentID); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Payment verified", map[string]string{
		"status": payment.Status,
	})
}

// fetchPortonePaymentForComplete asks PortOne for the payment and checks it is
// settled. When it cannot be had it answers the request itself and returns a nil payment together with
// that answer.
func fetchPortonePaymentForComplete(c fiber.Ctx, log *logging.Log, paymentID, apiSecret string) (*portonePayment, error) {
	// Call PortOne API and parse response
	resp, err := callPortoneAPI("/payments/"+paymentID, apiSecret)
	if err != nil {
		log.ErrorStack(err)
		return nil, webutil.StatusInternalServerError(c)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, webutil.StatusBadRequest(c, "Payment not found")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Error().Msgf("PortOne API error: %d %s", resp.StatusCode, string(body))
		return nil, webutil.StatusInternalServerError(c)
	}

	payment := new(portonePayment)
	if err := json.NewDecoder(resp.Body).Decode(payment); err != nil {
		log.ErrorStack(err)
		return nil, webutil.StatusBadRequest(c, "Failed to decode payment")
	}

	// Verify payment status
	if !payment.isPaid() {
		return nil, webutil.StatusBadRequest(c, fmt.Sprintf("Payment not completed: %s", payment.Status))
	}
	return payment, nil
}

// checkPortonePayment verifies the payment is the one the cart asked for. The
// error is worded for the buyer.
func checkPortonePayment(payment *portonePayment, cart *models.Cart, cartID string, log *logging.Log) error {
	// Validate payment details
	if err := validatePaymentAmount(payment.Amount.Total, cart.AmountTotal, log); err != nil {
		return err
	}

	if err := validatePaymentCurrency(payment.Amount.Currency, cart.Currency, log); err != nil {
		return err
	}

	if err := validateCartID(payment.CustomData, cartID, log); err != nil {
		log.ErrorStack(err)
		return err
	}
	return nil
}
