package handlers

import (
	"math"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/mailer"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/internal/webhook"
	"github.com/shurco/mycart/pkg/litepay"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

// sendPaymentWebhook sends a payment webhook notification.
// If blockOnError is true, returns error on webhook failure (for API endpoints).
// If blockOnError is false, logs error but doesn't block (for user-facing pages).
func sendPaymentWebhook(event webhook.Event, paymentSystem litepay.PaymentSystem, paymentStatus litepay.Status, cartID string, log *logging.Log, blockOnError bool) error {
	hook := &webhook.Payment{
		Event:     event,
		TimeStamp: time.Now().Unix(),
		Data: webhook.Data{
			PaymentSystem: paymentSystem,
			PaymentStatus: paymentStatus,
			CartID:        cartID,
		},
	}

	if err := webhook.SendPaymentHook(hook); err != nil {
		log.ErrorStack(err)
		if blockOnError {
			return err
		}
	}
	return nil
}

// PaymentCallback handles payment callback from payment providers.
//
// @Summary      Payment callback
// @Description  Webhook endpoint for payment providers to report status changes
// @Tags         Cart
// @Accept       json
// @Produce      plain
// @Param        cart_id        query string true "Cart ID"
// @Param        payment_system query string true "Payment system"
// @Success      200 {string} string "*ok*"
// @Failure      400 {object} webutil.HTTPResponse "Bad request"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /cart/payment/callback [post]
func PaymentCallback(c fiber.Ctx) error {
	return paymentCallback(c, litepay.VerifySpectrocoinCallback)
}

// paymentCallback is PaymentCallback with the SpectroCoin signature verifier
// injected, so that the tests can reach the checks the signature guards.
func paymentCallback(c fiber.Ctx, verifySpectrocoin spectrocoinVerifier) error {
	log := logging.New()
	payment := &litepay.Payment{
		CartID:        c.Query("cart_id"),
		PaymentSystem: litepay.PaymentSystem(c.Query("payment_system")),
	}

	switch payment.PaymentSystem {
	case litepay.SPECTROCOIN:
		response := new(litepay.CallbackSpectrocoin)
		if err := c.Bind().Body(response); err != nil {
			log.ErrorStack(err)
			return webutil.StatusBadRequest(c, err.Error())
		}

		setting, err := queries.GetSettingByGroup[models.Spectrocoin](c.Context(), queries.DB())
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		if !setting.Active {
			return webutil.StatusNotFound(c)
		}

		// The callback is unauthenticated input: its RSA signature must be
		// verified against the official SpectroCoin public key before any
		// field (status, amount) may be trusted.
		if err := verifySpectrocoin(response); err != nil {
			log.Error().Msgf("spectrocoin callback signature verification failed for cart %s: %v", payment.CartID, err)
			return webutil.StatusBadRequest(c, "invalid callback signature")
		}

		settled, err := spectrocoinPayment(log, response, setting, payment.CartID)
		if err != nil {
			return webutil.StatusBadRequest(c, err.Error())
		}
		payment = settled
	default:
		return webutil.StatusBadRequest(c, "unsupported payment system")
	}

	return settleCart(c, log, payment)
}

// settleCart records the payment a provider's callback describes, then tells
// the buyer and the shop's webhooks about it. It is the pipeline every provider
// shares; paymentCallback only decides how a callback becomes a Payment.
func settleCart(c fiber.Ctx, log *logging.Log, payment *litepay.Payment) error {
	db := queries.DB()

	cartInfo, err := db.Cart(c.Context(), payment.CartID)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Idempotency guard: repeated PAID callbacks must not re-send the
	// purchase letter or re-fire webhooks.
	if cartInfo.PaymentStatus == litepay.PAID {
		return c.Status(fiber.StatusOK).SendString("*ok*")
	}

	if payment.Status == litepay.PAID && !callbackAmountMatches(log, payment, cartInfo) {
		return webutil.StatusBadRequest(c, "callback amount does not match cart")
	}

	err = db.UpdateCart(c.Context(), &models.Cart{
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

	// send email. Unlike the redirect below, a failure here is returned on
	// purpose: the caller is the payment provider, and a provider that is told
	// the callback failed sends it again — which is the retry that gets the
	// buyer their file. The buyer is not waiting on this request.
	if payment.Status == litepay.PAID {
		if err := mailer.SendCartLetter(payment.CartID); err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
	}

	// send hook
	if err := sendPaymentWebhook(webhook.PAYMENT_CALLBACK, payment.PaymentSystem, payment.Status, payment.CartID, log, true); err != nil {
		return webutil.StatusInternalServerError(c)
	}

	return c.Status(fiber.StatusOK).SendString("*ok*")
}

// callbackAmountMatches reports whether the charge a PAID callback describes is
// the cart's own, and logs the mismatch when it is not.
func callbackAmountMatches(log *logging.Log, payment *litepay.Payment, cartInfo *models.Cart) bool {
	gotCents := int(math.Round(payment.Coin.AmountTotal * 100))
	// EqualFold, not !=: the provider's spelling of a currency code may
	// differ in case from the stored one, and verifyCartAmount already
	// treats it that way.
	if !strings.EqualFold(payment.Coin.Currency, cartInfo.Currency) || math.Abs(float64(gotCents-cartInfo.AmountTotal)) > 1 {
		log.Error().Msgf("%s callback amount mismatch for cart %s: got %s %.2f, want %s %.2f",
			payment.PaymentSystem, payment.CartID, payment.Coin.Currency, payment.Coin.AmountTotal, cartInfo.Currency, float64(cartInfo.AmountTotal)/100)
		return false
	}
	return true
}
