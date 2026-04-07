package handlers

import (
	"fmt"
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

	return webutil.Response(c, fiber.StatusOK, "Payment list", paymentList)
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
		if errors.Is(err, errors.ErrProductNotFound) {
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

	items := make([]litepay.Item, len(products.Products))
	for i, product := range products.Products {
		images := []string{}
		for _, image := range product.Images {
			path := fmt.Sprintf("https://%s/uploads/%s_md.%s", domain, image.Name, image.Ext)
			images = append(images, path)
		}

		quantity := 1
		for _, cartProduct := range payment.Products {
			if cartProduct.ProductID == product.ID {
				quantity = cartProduct.Quantity
			}
		}

		items[i] = litepay.Item{
			PriceData: litepay.Price{
				UnitAmount: product.Amount,
				Product: litepay.Product{
					Name:   product.Name,
					Images: images,
				},
			},
			Quantity: quantity,
		}

		if product.Description != "" {
			items[i].PriceData.Product.Description = product.Description
		}
	}

	cart := litepay.Cart{
		ID:       security.RandomString(),
		Currency: currency,
		Items:    items,
	}

	// Calculate total amount before processing payment
	var amountTotal int
	for _, item := range cart.Items {
		amountTotal += item.PriceData.UnitAmount * item.Quantity
	}

	// Validate dummy provider usage: only allowed for free carts (amountTotal = 0)
	paymentSystem := payment.Provider
	if paymentSystem == litepay.DUMMY && amountTotal > 0 {
		log.Error().Msg("Attempt to use dummy provider for paid cart")
		return webutil.StatusBadRequest(c, "Dummy payment provider can only be used for free items")
	}

	callbackURL := fmt.Sprintf("https://%s/cart/payment/callback", domain)
	successURL := fmt.Sprintf("https://%s/cart/payment/success", domain)
	cancelURL := fmt.Sprintf("https://%s/cart/payment/cancel", domain)
	pay := litepay.New(callbackURL, successURL, cancelURL)

	paymentURL := fmt.Sprintf("https://%s/cart", domain)
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
	log := logging.New()
	payment := &litepay.Payment{
		CartID:        c.Query("cart_id"),
		PaymentSystem: litepay.PaymentSystem(c.Query("payment_system")),
	}

	switch payment.PaymentSystem {
	// case litepay.STRIPE:
	//	return webutil.Response(c, fiber.StatusOK, "Callback", payment)
	case litepay.SPECTROCOIN:
		response := new(litepay.CallbackSpectrocoin)
		if err := c.Bind().Body(response); err != nil {
			log.ErrorStack(err)
			return webutil.StatusBadRequest(c, err.Error())
		}
		payment.Status = litepay.StatusPayment(litepay.SPECTROCOIN, string(rune(response.Status)))
		payment.MerchantID = response.MerchantApiID
		payment.Coin = &litepay.Coin{
			AmountTotal: response.ReceiveAmount,
			Currency:    response.ReceiveCurrency,
		}
	}

	db := queries.DB()
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

	// send email
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
		payment.MerchantID = response.MerchantID
		payment.Status = response.Status

	case litepay.PAYPAL:
		tokenPaypal := c.Query("token")
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
		payment.MerchantID = response.MerchantID
		payment.Status = response.Status

	case litepay.SPECTROCOIN:
		// Spectrocoin payment processing handled in callback

	case litepay.COINBASE:
		chargeID := c.Query("charge_id")
		if chargeID == "" {
			chargeID = payment.CartID // Fallback to cart ID if charge_id not provided
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

	// send email
	if payment.Status == litepay.PAID {
		if err := mailer.SendCartLetter(payment.CartID); err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
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
	payment := &litepay.Payment{
		CartID:        c.Query("cart_id"),
		PaymentSystem: litepay.PaymentSystem(c.Query("payment_system")),
	}

	db := queries.DB()
	err := db.UpdateCart(c.Context(), &models.Cart{
		Core: models.Core{
			ID: payment.CartID,
		},
		PaymentStatus: litepay.CANCELED,
		PaymentSystem: payment.PaymentSystem,
	})
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// send hook (don't block process on webhook error)
	sendPaymentWebhook(webhook.PAYMENT_CANCEL, payment.PaymentSystem, litepay.CANCELED, payment.CartID, log, false)

	// Redirect to SPA cancel page with query parameters
	redirectURL := "/cart/payment/cancel"
	if payment.CartID != "" {
		redirectURL += "?cart_id=" + payment.CartID
		if string(payment.PaymentSystem) != "" {
			redirectURL += "&payment_system=" + string(payment.PaymentSystem)
		}
	}
	return c.Redirect().To(redirectURL)
}
