package handlers

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/mailer"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/litepay"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/security"
	"github.com/shurco/mycart/pkg/webutil"
)

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

	checkout, err := loadCheckoutCart(c.Context(), db, payment)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	if !checkout.validation.Valid {
		return cartValidationFailed(c, checkout.validation)
	}

	// Use request scheme (http/https) for URLs
	protocol := c.Scheme()

	cart, amountTotal := paymentCart(checkout, protocol)

	// Validate dummy provider usage: only allowed for free carts (amountTotal = 0)
	if payment.Provider == litepay.DUMMY && amountTotal > 0 {
		log.Error().Msg("Attempt to use dummy provider for paid cart")
		return webutil.StatusBadRequest(c, "Dummy payment provider can only be used for free items")
	}

	paymentURL, providerRef, active, err := openPayment(c.Context(), db, protocol, checkout.domain, payment.Provider, cart)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}
	if !active {
		return webutil.Response(c, fiber.StatusOK, "Payment url", paymentURL)
	}

	if err := recordPayment(c.Context(), db, payment, cart, amountTotal, providerRef, paymentURL); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Payment url", map[string]string{"url": paymentURL})
}

// paymentCart is the cart sent to the provider, built from the server-corrected
// lines, and its total.
func paymentCart(checkout *checkoutCart, protocol string) (litepay.Cart, int) {
	items, amountTotal := paymentItems(checkout.products, checkout.validation.CorrectedItems, protocol, checkout.domain)
	return litepay.Cart{
		ID:       security.RandomString(),
		Currency: checkout.currency,
		Items:    items,
	}, amountTotal
}

// paymentItems turns the server-corrected cart lines into the items sent to the
// provider, and totals them.
func paymentItems(products *models.Products, corrected []models.CorrectedCartItem, protocol, domain string) ([]litepay.Item, int) {
	// Build product map for quick lookup
	productMap := make(map[string]*models.Product)
	for i := range products.Products {
		productMap[products.Products[i].ID] = &products.Products[i]
	}

	// Build cart items using validated prices
	items := make([]litepay.Item, 0, len(corrected))
	var amountTotal int
	for _, correctedItem := range corrected {
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
	return items, amountTotal
}

// checkoutCart is what Payment needs to know before it opens a payment: where
// the shop lives, what it sells in, and the cart as the shop sees it now.
type checkoutCart struct {
	domain, currency string
	products         *models.Products
	validation       *models.CartValidationResult
}

// loadCheckoutCart reads the shop settings and validates the requested items
// against current stock and prices.
func loadCheckoutCart(ctx context.Context, db *queries.Base, payment *models.CartPayment) (*checkoutCart, error) {
	setting, err := db.GetSettingByKey(ctx, "domain", "currency")
	if err != nil {
		return nil, err
	}
	checkout := &checkoutCart{
		domain:   setting["domain"].Value.(string),
		currency: setting["currency"].Value.(string),
	}

	checkout.products, err = db.ListProducts(ctx, false, 0, 0, "", payment.Products...)
	if err != nil {
		return nil, err
	}

	// Validate cart items before processing
	checkout.validation, err = queries.ValidateCartItems(ctx, db, payment.Products, checkout.currency)
	if err != nil {
		return nil, err
	}
	return checkout, nil
}

// openPayment opens the payment with the provider. It returns the URL to send
// the buyer to and the provider's own reference for the payment. active is
// false when the provider is switched off, and the URL is then the shop's cart page.
func openPayment(ctx context.Context, db *queries.Base, protocol, domain string, system litepay.PaymentSystem, cart litepay.Cart) (paymentURL, providerRef string, active bool, err error) {
	fallbackURL := fmt.Sprintf("%s://%s/cart", protocol, domain)
	callbackURL := fmt.Sprintf("%s://%s/cart/payment/callback", protocol, domain)
	successURL := fmt.Sprintf("%s://%s/cart/payment/success", protocol, domain)

	// The cancel URL carries an HMAC capability token: only the buyer who was
	// redirected through the provider flow can actually cancel the cart.
	settingJWT, err := queries.GetSettingByGroup[models.JWT](ctx, db)
	if err != nil {
		return "", "", true, err
	}
	cancelURL := fmt.Sprintf("%s://%s/cart/payment/cancel?cancel_token=%s", protocol, domain, cancelToken(settingJWT.Secret, cart.ID))

	session, active, err := providerSession(ctx, db, litepay.New(callbackURL, successURL, cancelURL), system)
	if err != nil || !active || session == nil {
		return fallbackURL, "", active, err
	}
	response, err := session.Pay(cart)
	if err != nil {
		return "", "", true, err
	}
	switch system {
	case litepay.STRIPE, litepay.PAYPAL, litepay.COINBASE:
		providerRef = response.MerchantID
	}
	return response.URL, providerRef, true, nil
}

// recordPayment stores the new cart, mails the buyer the payment link and tells
// the shop's webhooks the payment has been initiated.
func recordPayment(ctx context.Context, db *queries.Base, payment *models.CartPayment, cart litepay.Cart, amountTotal int, providerRef, paymentURL string) error {
	if err := db.AddCart(ctx, &models.Cart{
		Core: models.Core{
			ID: cart.ID,
		},
		Email:         payment.Email,
		Cart:          payment.Products,
		AmountTotal:   amountTotal,
		Currency:      cart.Currency,
		PaymentID:     providerRef,
		PaymentStatus: litepay.NEW,
		PaymentSystem: payment.Provider,
	}); err != nil {
		return err
	}

	// send email
	if err := mailer.SendPrepaymentLetter(payment.Email, fmt.Sprintf("%.2f %s", float64(amountTotal)/100, cart.Currency), paymentURL); err != nil {
		return err
	}

	// send hook
	return sendInitiationHook(payment.Provider, cart.ID, amountTotal, cart.Currency, cart.Items)
}
