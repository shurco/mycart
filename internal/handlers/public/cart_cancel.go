package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/internal/webhook"
	"github.com/shurco/mycart/pkg/litepay"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

// cancelToken derives an HMAC-SHA256 capability token that authorizes the
// cancellation of a specific cart. The token is embedded into the cancel URL
// built during payment initiation; without it, anyone could flip an arbitrary
// cart to "canceled" by hitting the public redirect endpoint.
func cancelToken(secret, cartID string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(cartID))
	return hex.EncodeToString(mac.Sum(nil))
}

// PaymentCancel handles canceled payment redirects.
//
// @Summary      Payment cancel
// @Description  Handle canceled payment, update cart status, and redirect to SPA
// @Tags         Cart
// @Produce      json
// @Param        cart_id        query string false "Cart ID"
// @Param        payment_system query string false "Payment system"
// @Success      302 "Redirect to cancel page"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /cart/payment/cancel [get]
func PaymentCancel(c fiber.Ctx) error {
	// Only process GET requests
	if c.Method() != fiber.MethodGet {
		return c.Next()
	}

	log := logging.New()
	cartID := c.Query("cart_id")
	redirectURL := cancelRedirectURL(c, cartID)

	// Cancellation mutates order state, so it requires the capability token
	// that was embedded into the cancel URL during payment initiation.
	token := c.Query("cancel_token")
	if cartID == "" || token == "" {
		return c.Redirect().To(redirectURL)
	}

	db := queries.DB()
	settingJWT, err := queries.GetSettingByGroup[models.JWT](c.Context(), db)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	expected := cancelToken(settingJWT.Secret, cartID)
	if !hmac.Equal([]byte(expected), []byte(token)) {
		log.Error().Msgf("cancel token mismatch for cart %s", cartID)
		return c.Redirect().To(redirectURL)
	}

	err = db.UpdateCart(c.Context(), &models.Cart{
		Core: models.Core{
			ID: cartID,
		},
		PaymentStatus: litepay.CANCELED,
		PaymentSystem: litepay.PaymentSystem(c.Query("payment_system")),
	})
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// send hook (don't block process on webhook error)
	sendPaymentWebhook(webhook.PAYMENT_CANCEL, litepay.PaymentSystem(c.Query("payment_system")), litepay.CANCELED, cartID, log, false)

	return c.Redirect().To(redirectURL)
}

// cancelRedirectURL is the SPA cancel page the buyer is sent to, with the
// cart and payment system carried along as query parameters.
func cancelRedirectURL(c fiber.Ctx, cartID string) string {
	// Redirect to SPA cancel page with query parameters
	redirectURL := "/cart/payment/cancel"
	if cartID != "" {
		redirectURL += "?cart_id=" + cartID
		if paymentSystem := c.Query("payment_system"); paymentSystem != "" {
			redirectURL += "&payment_system=" + paymentSystem
		}
	}
	return redirectURL
}
