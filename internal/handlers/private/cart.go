package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/shurco/litecart/internal/mailer"
	"github.com/shurco/litecart/internal/queries"
	"github.com/shurco/litecart/pkg/errors"
	"github.com/shurco/litecart/pkg/logging"
	"github.com/shurco/litecart/pkg/webutil"
)

// Carts returns a list of all carts.
//
// @Summary      List carts
// @Description  Get paginated list of all carts
// @Tags         Carts
// @Security     BearerAuth
// @Produce      json
// @Param        page  query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200 {object} webutil.HTTPResponse "Carts list with pagination"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/carts [get]
func Carts(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	page := fiber.Query[int](c, "page", 1)
	limit := fiber.Query[int](c, "limit", 20)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	carts, total, err := db.Carts(c.Context(), limit, offset)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Carts", map[string]any{
		"carts": carts,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// Cart returns detailed cart information by cart_id.
//
// @Summary      Get cart
// @Description  Get detailed cart information including product items
// @Tags         Carts
// @Security     BearerAuth
// @Produce      json
// @Param        cart_id path string true "Cart ID"
// @Success      200 {object} webutil.HTTPResponse "Cart details"
// @Failure      400 {object} webutil.HTTPResponse "Missing cart_id"
// @Failure      404 {object} webutil.HTTPResponse "Cart not found"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/carts/{cart_id} [get]
func Cart(c fiber.Ctx) error {
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
		"payment_id":     cart.PaymentID,
		"created":        cart.Created,
		"updated":        cart.Updated,
		"items":          cartItems,
	})
}

// CartSendMail sends an email notification for a cart.
//
// @Summary      Send cart email
// @Description  Re-send purchase confirmation email for a cart
// @Tags         Carts
// @Security     BearerAuth
// @Produce      json
// @Param        cart_id path string true "Cart ID"
// @Success      200 {object} webutil.HTTPResponse "Email sent"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/carts/{cart_id}/mail [post]
func CartSendMail(c fiber.Ctx) error {
	cartID := c.Params("cart_id")
	log := logging.New()

	if err := mailer.SendCartLetter(cartID); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Mail sended", nil)
}
