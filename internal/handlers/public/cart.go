package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/internal/webhook"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/litepay"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/security"
	"github.com/shurco/mycart/pkg/webutil"
)

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
	mainCurrency, err := storeCurrency(c.Context(), db)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}
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

	currency, err := storeCurrency(c.Context(), db)
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
		return cartValidationFailed(c, validationResult)
	}

	amountTotal := cartTotal(validationResult.CorrectedItems)

	cartID, err := storeNewCart(c.Context(), db, payment, currency, amountTotal)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Send webhook for cart initiation
	if err := sendInitiationHook(payment.Provider, cartID, amountTotal, currency, nil); err != nil {
		log.ErrorStack(err)
		// Don't fail cart creation if webhook fails
	}

	return webutil.Response(c, fiber.StatusOK, "Cart created", map[string]interface{}{
		"cart_id":      cartID,
		"amount_total": amountTotal,
		"currency":     currency,
	})
}

// cartTotal totals the server-corrected cart lines.
func cartTotal(items []models.CorrectedCartItem) int {
	// Calculate total amount using validated prices (includes variant surcharges)
	var amountTotal int
	for _, correctedItem := range items {
		amountTotal += correctedItem.UnitPrice * correctedItem.Quantity
	}
	return amountTotal
}

// storeNewCart creates the cart record for payment and returns its ID.
func storeNewCart(ctx context.Context, db *queries.Base, payment *models.CartPayment, currency string, amountTotal int) (string, error) {
	// Generate cart ID
	cartID := security.RandomString()

	// Create cart record
	err := db.AddCart(ctx, &models.Cart{
		Core: models.Core{
			ID: cartID,
		},
		Email:         payment.Email,
		Cart:          payment.Products,
		AmountTotal:   amountTotal,
		Currency:      currency,
		PaymentStatus: litepay.NEW,
		PaymentSystem: payment.Provider,
	})
	return cartID, err
}

// storeCurrency returns the currency the shop sells in.
func storeCurrency(ctx context.Context, db *queries.Base) (string, error) {
	setting, err := db.GetSettingByKey(ctx, "currency")
	if err != nil {
		return "", err
	}
	return setting["currency"].Value.(string), nil
}

// cartValidationFailed answers a cart that does not match the shop's current
// stock and prices, with the server's corrected version of it.
func cartValidationFailed(c fiber.Ctx, result *models.CartValidationResult) error {
	return webutil.Response(c, fiber.StatusConflict, "Cart validation failed", map[string]any{
		"validation_errors": result.Errors,
		"corrected_cart":    result.CorrectedItems,
	})
}

// sendInitiationHook tells the shop's webhooks a payment has been initiated.
func sendInitiationHook(system litepay.PaymentSystem, cartID string, amountTotal int, currency string, items []litepay.Item) error {
	return webhook.SendPaymentHook(&webhook.Payment{
		Event:     webhook.PAYMENT_INITIATION,
		TimeStamp: time.Now().Unix(),
		Data: webhook.Data{
			PaymentSystem: system,
			PaymentStatus: litepay.NEW,
			CartID:        cartID,
			TotalAmount:   amountTotal,
			Currency:      currency,
			CartItems:     items,
		},
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
