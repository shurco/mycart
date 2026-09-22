package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/mailer"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/internal/webhook"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/litepay"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/security"
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

// PaymentList returns a list of available payment systems.
//
// @Summary      List payment providers
// @Description  Get active/inactive status of all payment providers
// @Tags         Cart
// @Produce      json
// @Success      200 {object} webutil.HTTPResponse "Payment provider statuses"
// @Failure      400 {object} webutil.HTTPResponse "Bad request"
// @Router       /api/cart/payment [get]
func PaymentList(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()
	paymentList, err := db.PaymentList(c.Context())
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	// Get current store currency for PortOne filtering
	currencySetting, err := db.GetSettingByKey(c.Context(), "currency")
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}
	mainCurrency := currencySetting["currency"].Value.(string)

	// Filter PortOne based on supported currencies
	if paymentList["portone"] {
		portoneSettings, err := queries.GetSettingByGroup[models.Portone](c.Context(), db)
		if err == nil && portoneSettings != nil && len(portoneSettings.SupportedCurrencies) > 0 {
			supported := false
			for _, curr := range portoneSettings.SupportedCurrencies {
				if curr == mainCurrency {
					supported = true
					break
				}
			}
			if !supported {
				paymentList["portone"] = false
			}
		}
	}

	return webutil.Response(c, fiber.StatusOK, "Payment list", paymentList)
}

// CreateCart creates a cart record in the database for PortOne payment flow.
// Unlike traditional providers that create cart during /cart/payment,
// PortOne needs cart_id upfront to pass to browser SDK.
//
// @Summary      Create cart
// @Description  Create a cart record and return its ID for browser-based payment (PortOne)
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Param        request body models.CartPayment true "Cart creation request"
// @Success      200 {object} webutil.HTTPResponse "Cart created"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/cart/create [post]
func CreateCart(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()
	payment := new(models.CartPayment)

	if err := c.Bind().Body(payment); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	setting, err := db.GetSettingByKey(c.Context(), "currency")
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}
	currency := setting["currency"].Value.(string)

	// Validate cart items before processing
	validationResult, err := queries.ValidateCartItems(c.Context(), db, payment.Products, currency)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	if !validationResult.Valid {
		return webutil.Response(c, fiber.StatusConflict, "Cart validation failed", map[string]any{
			"validation_errors": validationResult.Errors,
			"corrected_cart":    validationResult.CorrectedItems,
		})
	}

	// Calculate total amount using validated prices (includes variant surcharges)
	var amountTotal int
	for _, correctedItem := range validationResult.CorrectedItems {
		amountTotal += correctedItem.UnitPrice * correctedItem.Quantity
	}

	// Generate cart ID
	cartID := security.RandomString()

	// Create cart record
	if err := db.AddCart(c.Context(), &models.Cart{
		Core: models.Core{
			ID: cartID,
		},
		Email:         payment.Email,
		Cart:          payment.Products,
		AmountTotal:   amountTotal,
		Currency:      currency,
		PaymentStatus: litepay.NEW,
		PaymentSystem: payment.Provider,
	}); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Send webhook for cart initiation
	hook := &webhook.Payment{
		Event:     webhook.PAYMENT_INITIATION,
		TimeStamp: time.Now().Unix(),
		Data: webhook.Data{
			PaymentSystem: payment.Provider,
			PaymentStatus: litepay.NEW,
			CartID:        cartID,
			TotalAmount:   amountTotal,
			Currency:      currency,
		},
	}
	if err := webhook.SendPaymentHook(hook); err != nil {
		log.ErrorStack(err)
		// Don't fail cart creation if webhook fails
	}

	return webutil.Response(c, fiber.StatusOK, "Cart created", map[string]interface{}{
		"cart_id":      cartID,
		"amount_total": amountTotal,
		"currency":     currency,
	})
}

// GetCart returns cart information by cart_id.
//
// @Summary      Get cart (public)
// @Description  Get cart details including product items by cart ID
// @Tags         Cart
// @Produce      json
// @Param        cart_id path string true "Cart ID"
// @Success      200 {object} webutil.HTTPResponse "Cart details"
// @Failure      400 {object} webutil.HTTPResponse "Missing cart_id"
// @Failure      404 {object} webutil.HTTPResponse "Cart not found"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/cart/{cart_id} [get]
func GetCart(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()
	cartID := c.Params("cart_id")

	if cartID == "" {
		return webutil.StatusBadRequest(c, "cart_id is required")
	}

	cart, err := db.Cart(c.Context(), cartID)
	if err != nil {
		log.ErrorStack(err)
		if errors.Is(err, errors.ErrCartNotFound) {
			return webutil.StatusNotFound(c)
		}
		return webutil.StatusInternalServerError(c)
	}

	// Load full product information for cart items
	// Pass cartID to include digital products purchased in this cart
	var cartItems []map[string]any
	if len(cart.Cart) > 0 {
		products, err := db.ListProducts(c.Context(), false, 0, 0, cartID, cart.Cart...)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		cartItems = queries.BuildCartItems(cart, products)
	}

	return webutil.Response(c, fiber.StatusOK, "Cart", map[string]any{
		"id":             cart.ID,
		"email":          cart.Email,
		"amount_total":   cart.AmountTotal,
		"currency":       cart.Currency,
		"payment_status": cart.PaymentStatus,
		"payment_system": cart.PaymentSystem,
		"items":          cartItems,
	})
}

// Payment initiates a payment process for a cart.
//
// @Summary      Initiate payment
// @Description  Create a payment session and return a redirect URL for the selected provider
// @Tags         Cart
// @Accept       json
// @Produce      json
// @Param        request body models.CartPayment true "Payment request"
// @Success      200 {object} webutil.HTTPResponse "Payment URL"
// @Failure      400 {object} webutil.HTTPResponse "Validation error or dummy provider for paid cart"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /cart/payment [post]
func Payment(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()
	payment := new(models.CartPayment)

	if err := c.Bind().Body(payment); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	setting, err := db.GetSettingByKey(c.Context(), "domain", "currency")
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}
	domain := setting["domain"].Value.(string)
	currency := setting["currency"].Value.(string)

	products, err := db.ListProducts(c.Context(), false, 0, 0, "", payment.Products...)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Validate cart items before processing
	validationResult, err := queries.ValidateCartItems(c.Context(), db, payment.Products, currency)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	if !validationResult.Valid {
		return webutil.Response(c, fiber.StatusConflict, "Cart validation failed", map[string]any{
			"validation_errors": validationResult.Errors,
			"corrected_cart":    validationResult.CorrectedItems,
		})
	}

	// Use request scheme (http/https) for URLs
	protocol := c.Scheme()

	// Build product map for quick lookup
	productMap := make(map[string]*models.Product)
	for i := range products.Products {
		productMap[products.Products[i].ID] = &products.Products[i]
	}

	// Build cart items using validated prices
	items := make([]litepay.Item, 0, len(validationResult.CorrectedItems))
	var amountTotal int
	for _, correctedItem := range validationResult.CorrectedItems {
		product, exists := productMap[correctedItem.ProductID]
		if !exists {
			continue
		}

		images := []string{}
		for _, image := range product.Images {
			path := fmt.Sprintf("%s://%s/uploads/%s_md.%s", protocol, domain, image.Name, image.Ext)
			images = append(images, path)
		}

		items = append(items, litepay.Item{
			PriceData: litepay.Price{
				UnitAmount: correctedItem.UnitPrice, // Uses validated price with variant surcharge
				Product: litepay.Product{
					Name:        product.Name,
					Description: product.Description,
					Images:      images,
				},
			},
			Quantity: correctedItem.Quantity,
		})

		amountTotal += correctedItem.UnitPrice * correctedItem.Quantity
	}

	cart := litepay.Cart{
		ID:       security.RandomString(),
		Currency: currency,
		Items:    items,
	}

	// Validate dummy provider usage: only allowed for free carts (amountTotal = 0)
	paymentSystem := payment.Provider
	if paymentSystem == litepay.DUMMY && amountTotal > 0 {
		log.Error().Msg("Attempt to use dummy provider for paid cart")
		return webutil.StatusBadRequest(c, "Dummy payment provider can only be used for free items")
	}

	callbackURL := fmt.Sprintf("%s://%s/cart/payment/callback", protocol, domain)
	successURL := fmt.Sprintf("%s://%s/cart/payment/success", protocol, domain)

	// The cancel URL carries an HMAC capability token: only the buyer who was
	// redirected through the provider flow can actually cancel the cart.
	settingJWT, err := queries.GetSettingByGroup[models.JWT](c.Context(), db)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}
	cancelURL := fmt.Sprintf("%s://%s/cart/payment/cancel?cancel_token=%s", protocol, domain, cancelToken(settingJWT.Secret, cart.ID))

	pay := litepay.New(callbackURL, successURL, cancelURL)

	paymentURL := fmt.Sprintf("%s://%s/cart", protocol, domain)
	providerRef := ""
	switch paymentSystem {
	case litepay.STRIPE:
		setting, err := queries.GetSettingByGroup[models.Stripe](c.Context(), db)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}

		if !setting.Active {
			return webutil.Response(c, fiber.StatusOK, "Payment url", paymentURL)
		}
		session := pay.Stripe(setting.SecretKey)
		response, err := session.Pay(cart)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		paymentURL = response.URL
		providerRef = response.MerchantID

	case litepay.PAYPAL:
		setting, err := queries.GetSettingByGroup[models.Paypal](c.Context(), db)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}

		if !setting.Active {
			return webutil.Response(c, fiber.StatusOK, "Payment url", paymentURL)
		}
		session := pay.Paypal(setting.ClientID, setting.SecretKey)
		response, err := session.Pay(cart)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		paymentURL = response.URL
		providerRef = response.MerchantID

	case litepay.SPECTROCOIN:
		setting, err := queries.GetSettingByGroup[models.Spectrocoin](c.Context(), db)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}

		if !setting.Active {
			return webutil.Response(c, fiber.StatusOK, "Payment url", paymentURL)
		}
		session := pay.Spectrocoin(setting.MerchantID, setting.ProjectID, setting.PrivateKey)
		response, err := session.Pay(cart)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		paymentURL = response.URL

	case litepay.COINBASE:
		setting, err := queries.GetSettingByGroup[models.Coinbase](c.Context(), db)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}

		if !setting.Active {
			return webutil.Response(c, fiber.StatusOK, "Payment url", paymentURL)
		}
		session := pay.Coinbase(setting.ApiKey)
		response, err := session.Pay(cart)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		paymentURL = response.URL
		providerRef = response.MerchantID

	case litepay.DUMMY:
		// Dummy provider is always active and only for free carts (already validated above)
		session := pay.Dummy()
		response, err := session.Pay(cart)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		paymentURL = response.URL
	}

	if err := db.AddCart(c.Context(), &models.Cart{
		Core: models.Core{
			ID: cart.ID,
		},
		Email:         payment.Email,
		Cart:          payment.Products,
		AmountTotal:   amountTotal,
		Currency:      cart.Currency,
		PaymentID:     providerRef,
		PaymentStatus: litepay.NEW,
		PaymentSystem: paymentSystem,
	}); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// send email
	if err := mailer.SendPrepaymentLetter(payment.Email, fmt.Sprintf("%.2f %s", float64(amountTotal)/100, cart.Currency), paymentURL); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// send hook
	hook := &webhook.Payment{
		Event:     webhook.PAYMENT_INITIATION,
		TimeStamp: time.Now().Unix(),
		Data: webhook.Data{
			PaymentSystem: paymentSystem,
			PaymentStatus: litepay.NEW,
			CartID:        cart.ID,
			TotalAmount:   amountTotal,
			Currency:      cart.Currency,
			CartItems:     items,
		},
	}
	if err := webhook.SendPaymentHook(hook); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Payment url", map[string]string{"url": paymentURL})
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

	if payment.Status == litepay.PAID {
		gotCents := int(math.Round(payment.Coin.AmountTotal * 100))
		// EqualFold, not !=: the provider's spelling of a currency code may
		// differ in case from the stored one, and verifyCartAmount already
		// treats it that way.
		if !strings.EqualFold(payment.Coin.Currency, cartInfo.Currency) || math.Abs(float64(gotCents-cartInfo.AmountTotal)) > 1 {
			log.Error().Msgf("%s callback amount mismatch for cart %s: got %s %.2f, want %s %.2f",
				payment.PaymentSystem, payment.CartID, payment.Coin.Currency, payment.Coin.AmountTotal, cartInfo.Currency, float64(cartInfo.AmountTotal)/100)
			return webutil.StatusBadRequest(c, "callback amount does not match cart")
		}
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

	switch payment.PaymentSystem {
	case litepay.STRIPE:
		sessionStripe := c.Query("session")
		if sessionStripe == "" || (cartInfo.PaymentID != "" && sessionStripe != cartInfo.PaymentID) {
			log.Error().Msgf("stripe session does not match cart %s", payment.CartID)
			return webutil.StatusBadRequest(c, "payment session does not match cart")
		}
		setting, err := queries.GetSettingByGroup[models.Stripe](c.Context(), db)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}

		if !setting.Active {
			return webutil.StatusNotFound(c)
		}
		response, err := litepay.New("", "", "").Stripe(setting.SecretKey).Checkout(payment, sessionStripe)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		if err := verifyCartAmount(response, cartInfo); err != nil {
			log.Error().Msgf("stripe verification failed for cart %s: %v", payment.CartID, err)
			return webutil.StatusBadRequest(c, "payment verification failed")
		}
		payment.MerchantID = response.MerchantID
		payment.Status = response.Status

	case litepay.PAYPAL:
		tokenPaypal := c.Query("token")
		if tokenPaypal == "" || (cartInfo.PaymentID != "" && tokenPaypal != cartInfo.PaymentID) {
			log.Error().Msgf("paypal token does not match cart %s", payment.CartID)
			return webutil.StatusBadRequest(c, "payment token does not match cart")
		}
		setting, err := queries.GetSettingByGroup[models.Paypal](c.Context(), db)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}

		if !setting.Active {
			return webutil.StatusNotFound(c)
		}
		response, err := litepay.New("", "", "").Paypal(setting.ClientID, setting.SecretKey).Checkout(payment, tokenPaypal)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		if err := verifyCartAmount(response, cartInfo); err != nil {
			log.Error().Msgf("paypal verification failed for cart %s: %v", payment.CartID, err)
			return webutil.StatusBadRequest(c, "payment verification failed")
		}
		payment.MerchantID = response.MerchantID
		payment.Status = response.Status

	case litepay.SPECTROCOIN:
		// Spectrocoin payment processing handled in callback

	case litepay.COINBASE:
		chargeID := c.Query("charge_id")
		if chargeID == "" || (cartInfo.PaymentID != "" && chargeID != cartInfo.PaymentID) {
			log.Error().Msgf("coinbase charge does not match cart %s", payment.CartID)
			return webutil.StatusBadRequest(c, "payment charge does not match cart")
		}
		setting, err := queries.GetSettingByGroup[models.Coinbase](c.Context(), db)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}

		if !setting.Active {
			return webutil.StatusNotFound(c)
		}
		response, err := litepay.New("", "", "").Coinbase(setting.ApiKey).Checkout(payment, chargeID)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		if err := verifyCartAmount(response, cartInfo); err != nil {
			log.Error().Msgf("coinbase verification failed for cart %s: %v", payment.CartID, err)
			return webutil.StatusBadRequest(c, "payment verification failed")
		}
		payment.MerchantID = response.MerchantID
		payment.Status = response.Status

	case litepay.DUMMY:
		// Dummy provider is always active and only for free carts (already validated above)
		response, err := litepay.New("", "", "").Dummy().Checkout(payment, "")
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		payment.MerchantID = response.MerchantID
		payment.Status = response.Status
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

	// Redirect to SPA cancel page with query parameters
	redirectURL := "/cart/payment/cancel"
	if cartID != "" {
		redirectURL += "?cart_id=" + cartID
		if paymentSystem := c.Query("payment_system"); paymentSystem != "" {
			redirectURL += "&payment_system=" + paymentSystem
		}
	}

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
