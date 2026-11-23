package handlers

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/mailer"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/internal/webhook"
	"github.com/shurco/mycart/pkg/litepay"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

// verifyCartAmount ensures the provider-reported charge matches the stored
// cart total and currency before the cart may be marked as paid.
func verifyCartAmount(payment *litepay.Payment, cart *models.Cart) error {
	if payment.AmountTotal != cart.AmountTotal {
		return fmt.Errorf("amount mismatch: provider=%d cart=%d", payment.AmountTotal, cart.AmountTotal)
	}
	if !strings.EqualFold(payment.Currency, cart.Currency) {
		return fmt.Errorf("currency mismatch: provider=%q cart=%q", payment.Currency, cart.Currency)
	}
	return nil
}

// PaymentSuccess handles successful payment redirects.
//
// @Summary      Payment success
// @Description  Handle successful payment redirect, verify with provider, and update cart
// @Tags         Cart
// @Produce      json
// @Param        cart_id        query string true  "Cart ID"
// @Param        payment_system query string true  "Payment system"
// @Param        session        query string false "Stripe session ID"
// @Param        token          query string false "PayPal token"
// @Param        charge_id      query string false "Coinbase charge ID"
// @Success      200 "Passes to SPA handler"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /cart/payment/success [get]
func PaymentSuccess(c fiber.Ctx) error {
	// Only process GET requests
	if c.Method() != fiber.MethodGet {
		return c.Next()
	}

	log := logging.New()
	if c.Query("cart_id") == "" {
		return webutil.StatusBadRequest(c, nil)
	}

	payment := &litepay.Payment{
		CartID:        c.Query("cart_id"),
		PaymentSystem: litepay.PaymentSystem(c.Query("payment_system")),
	}

	if err := payment.Validate(); err != nil {
		return c.Redirect().To("/")
	}

	db := queries.DB()
	cartInfo, err := db.Cart(c.Context(), c.Query("cart_id"))
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Validate dummy provider usage: only allowed for free carts (amountTotal = 0)
	if payment.PaymentSystem == litepay.DUMMY && cartInfo.AmountTotal > 0 {
		log.Error().Msgf("Attempt to use dummy provider for paid cart (cart_id: %s, amount: %d)", payment.CartID, cartInfo.AmountTotal)
		return webutil.StatusBadRequest(c, "Dummy payment provider can only be used for free items")
	}

	// If already paid, pass control to SPA handler
	if cartInfo.PaymentStatus == "paid" {
		return c.Next()
	}

	if done, err := checkoutProvider(c, log, db, payment, cartInfo); done {
		return err
	}

	return settlePaidRedirect(c, log, db, payment)
}

// settlePaidRedirect records what the provider said about the payment, then
// tells the buyer and the shop's webhooks about it and hands the request on.
func settlePaidRedirect(c fiber.Ctx, log *logging.Log, db *queries.Base, payment *litepay.Payment) error {
	err := db.UpdateCart(c.Context(), &models.Cart{
		Core: models.Core{
			ID: payment.CartID,
		},
		PaymentID:     payment.MerchantID,
		PaymentStatus: payment.Status,
		PaymentSystem: payment.PaymentSystem,
	})
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// send email. A failure here is logged and not returned: the buyer has
	// already been charged and is standing on the page the provider sent them
	// back to, so answering with a server error would take the confirmation
	// away from them without putting the guide back. The letter is what the
	// shop owes them, not what this request is — the provider's callback
	// retries it, and the panel's resend does when that never comes.
	if payment.Status == litepay.PAID {
		if err := mailer.SendCartLetter(payment.CartID); err != nil {
			log.ErrorStack(err)
		}
	}

	// send hook (don't block process on webhook error)
	sendPaymentWebhook(webhook.PAYMENT_SUCCESS, payment.PaymentSystem, payment.Status, payment.CartID, log, false)

	// After processing payment, pass control to SPA handler
	// The SPA will display the success page with cart information
	return c.Next()
}

// checkoutProvider asks the provider whether the payment the buyer comes back
// from is real, and records what it says on payment. done is true when the
// request has been answered, with err being that answer.
func checkoutProvider(c fiber.Ctx, log *logging.Log, db *queries.Base, payment *litepay.Payment, cartInfo *models.Cart) (done bool, err error) {
	switch payment.PaymentSystem {
	case litepay.STRIPE:
		return checkoutByReference(c, log, db, payment, cartInfo, "session", c.Query("session"))

	case litepay.PAYPAL:
		return checkoutByReference(c, log, db, payment, cartInfo, "token", c.Query("token"))

	case litepay.SPECTROCOIN:
		// Spectrocoin payment processing handled in callback

	case litepay.COINBASE:
		return checkoutByReference(c, log, db, payment, cartInfo, "charge", c.Query("charge_id"))

	case litepay.DUMMY:
		// Dummy provider is always active and only for free carts (already validated above)
		response, err := litepay.New("", "", "").Dummy().Checkout(payment, "")
		if err != nil {
			log.ErrorStack(err)
			return true, webutil.StatusInternalServerError(c)
		}
		payment.MerchantID = response.MerchantID
		payment.Status = response.Status
	}
	return false, nil
}

// checkoutByReference is checkoutProvider for the providers that send the buyer
// back with a reference (session, token or charge) to their own record of the
// payment: it must be the one the cart was opened with, and the amount the
// provider reports must be the cart's.
func checkoutByReference(c fiber.Ctx, log *logging.Log, db *queries.Base, payment *litepay.Payment, cartInfo *models.Cart, kind, reference string) (done bool, err error) {
	if reference == "" || (cartInfo.PaymentID != "" && reference != cartInfo.PaymentID) {
		log.Error().Msgf("%s %s does not match cart %s", payment.PaymentSystem, kind, payment.CartID)
		return true, webutil.StatusBadRequest(c, fmt.Sprintf("payment %s does not match cart", kind))
	}
	session, active, err := providerSession(c.Context(), db, litepay.New("", "", ""), payment.PaymentSystem)
	if err != nil {
		log.ErrorStack(err)
		return true, webutil.StatusInternalServerError(c)
	}

	if !active {
		return true, webutil.StatusNotFound(c)
	}
	response, err := session.Checkout(payment, reference)
	if err != nil {
		log.ErrorStack(err)
		return true, webutil.StatusInternalServerError(c)
	}
	if err := verifyCartAmount(response, cartInfo); err != nil {
		log.Error().Msgf("%s verification failed for cart %s: %v", payment.PaymentSystem, payment.CartID, err)
		return true, webutil.StatusBadRequest(c, "payment verification failed")
	}
	payment.MerchantID = response.MerchantID
	payment.Status = response.Status
	return false, nil
}
