package handlers

import (
	"mime/multipart"
	"slices"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/digitalfiles"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

// ProductDigital returns digital content for a product.
//
// @Summary      Get product digital content
// @Description  Get digital files or license keys for a product
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Success      200 {object} webutil.HTTPResponse{result=models.Digital} "Digital content"
// @Failure      404 {object} webutil.HTTPResponse "Product not found"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/digital [get]
func ProductDigital(c fiber.Ctx) error {
	productID := c.Params("product_id")
	db := queries.DB()
	log := logging.New()

	digital, err := db.ProductDigital(c.Context(), productID)
	if err != nil {
		if errors.Is(err, errors.ErrProductNotFound) {
			return webutil.StatusNotFound(c)
		}
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Product digital", digital)
}

// addDigitalFile stores an uploaded digital file and links it to the product.
func addDigitalFile(c fiber.Ctx, productID string, fileTmp *multipart.FileHeader) error {
	db := queries.DB()
	log := logging.New()

	// Defense in depth: never accept active-content file types as
	// digital products (they are delivered by email and downloadable
	// by admins).
	if slices.Contains(blockedDigitalExtensions, normalizeExt(fileTmp.Filename)) {
		return webutil.StatusBadRequest(c, "file type not allowed")
	}

	fileUUID, fileExt, _ := generateFileName(fileTmp.Filename)
	filePath := digitalfiles.Path(fileUUID, fileExt)
	fileOrigName := fileTmp.Filename

	if err := saveFile(fileTmp, filePath); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	file, err := db.AddDigitalFile(c.Context(), productID, fileUUID, fileExt, fileOrigName)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Digital added", file)
}

// AddProductDigital adds digital content (file or data) to a product.
//
// @Summary      Add product digital content
// @Description  Upload a digital file or create a license key entry
// @Tags         Products
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Param        document   formData file false "Digital file"
// @Success      200 {object} webutil.HTTPResponse "Digital content added"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/digital [post]
func AddProductDigital(c fiber.Ctx) error {
	productID := c.Params("product_id")
	db := queries.DB()
	log := logging.New()

	fileTmp, _ := c.FormFile("document")
	if fileTmp != nil {
		return addDigitalFile(c, productID, fileTmp)
	}

	data, err := db.AddDigitalData(c.Context(), productID, "")
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Digital added", data)
}

// DownloadProductDigital streams a digital file to an authenticated admin.
// Replaces the former public /secrets static mount.
//
// @Summary      Download digital file
// @Description  Stream a product digital file as an attachment
// @Tags         Products
// @Security     BearerAuth
// @Param        product_id  path string true "Product ID"
// @Param        digital_id  path string true "Digital file ID"
// @Success      200 {file} file "File content"
// @Failure      404 {object} webutil.HTTPResponse "File not found"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/digital/{digital_id}/download [get]
func DownloadProductDigital(c fiber.Ctx) error {
	productID := c.Params("product_id")
	fileID := c.Params("digital_id")
	db := queries.DB()
	log := logging.New()

	file, err := db.DigitalFile(c.Context(), productID, fileID)
	if err != nil {
		if errors.Is(err, errors.ErrProductNotFound) {
			return webutil.StatusNotFound(c)
		}
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	if err := digitalfiles.Serve(c, file); err != nil {
		log.ErrorStack(err)
		return webutil.StatusNotFound(c)
	}

	return nil
}

// UpdateProductDigital updates digital content for a product.
//
// @Summary      Update product digital content
// @Description  Update the content of a digital data entry (license key)
// @Tags         Products
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Param        digital_id path string true "Digital ID"
// @Param        request    body models.Data true "Digital data"
// @Success      200 {object} webutil.HTTPResponse "Digital content updated"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/digital/{digital_id} [patch]
func UpdateProductDigital(c fiber.Ctx) error {
	request := new(models.Data)
	request.ID = c.Params("digital_id")
	// request.Content = c.Params("digital_id")
	db := queries.DB()
	log := logging.New()

	if err := c.Bind().Body(request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	if err := db.UpdateDigital(c.Context(), request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Digital updated", nil)
}

// DeleteProductDigital deletes digital content from a product.
//
// @Summary      Delete product digital content
// @Description  Delete a digital content entry by product and digital ID
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Param        digital_id path string true "Digital ID"
// @Success      200 {object} webutil.HTTPResponse "Digital content deleted"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/digital/{digital_id} [delete]
func DeleteProductDigital(c fiber.Ctx) error {
	productID := c.Params("product_id")
	digitalID := c.Params("digital_id")
	db := queries.DB()
	log := logging.New()

	if err := db.DeleteDigital(c.Context(), productID, digitalID); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Digital deleted", nil)
}
