package handlers

import (
	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/digitalfiles"
	"github.com/shurco/mycart/internal/middleware"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

// CustomerPurchases lists what the signed-in customer has bought.
//
// @Summary      Purchases of the signed-in customer
// @Description  Paid orders of the cabinet session, with the products in them and the files and keys each one hands over
// @Tags         Account
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} webutil.HTTPResponse "Purchases"
// @Failure      401 {object} webutil.HTTPResponse "Not signed in"
// @Failure      404 {object} webutil.HTTPResponse "The cabinet is disabled"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/customer/purchases [get]
func CustomerPurchases(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	purchases, err := db.CustomerPurchases(c.Context(), middleware.CustomerEmail(c))
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.StatusOK(c, "Purchases", map[string]any{"purchases": purchases})
}

// CustomerDownload streams a digital file the signed-in customer has paid for.
//
// @Summary      Download a purchased digital file
// @Description  Stream a product digital file as an attachment, if the cabinet session's address has paid for the product
// @Tags         Account
// @Security     BearerAuth
// @Param        file_id path string true "Digital file ID"
// @Success      200 {file} file "File content"
// @Failure      401 {object} webutil.HTTPResponse "Not signed in"
// @Failure      404 {object} webutil.HTTPResponse "The cabinet is disabled, or the file is not one this customer bought"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/customer/purchases/{file_id}/download [get]
func CustomerDownload(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	file, err := db.EntitledDigitalFile(c.Context(), middleware.CustomerEmail(c), c.Params("file_id"))
	if err != nil {
		// "No such file" and "not yours" arrive here as the same error and
		// leave as the same answer. An operator who deleted the account is
		// told nothing either: 404 is also what a customer of another shop
		// would see.
		if errors.Is(err, errors.ErrProductNotFound) {
			return webutil.StatusNotFound(c)
		}
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	if err := digitalfiles.Serve(c, file); err != nil {
		// The row is there and the buyer is entitled to it, but the bytes are
		// not. That is the shop's problem to notice — the error is logged — and
		// the buyer can do nothing with it, so the answer is the same as for a
		// file that never existed.
		log.ErrorStack(err)
		return webutil.StatusNotFound(c)
	}

	return nil
}
