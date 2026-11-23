package handlers

import (
	"fmt"
	"image"
	"mime/multipart"
	"os"
	"slices"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/imageutil"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

// ProductImages returns a list of images for a product.
//
// @Summary      List product images
// @Description  Get all images for a product
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Success      200 {object} webutil.HTTPResponse{result=[]models.File} "Product images"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/image [get]
func ProductImages(c fiber.Ctx) error {
	productID := c.Params("product_id")
	db := queries.DB()
	log := logging.New()

	images, err := db.ProductImages(c.Context(), productID)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Product images", images)
}

// checkImageUpload validates the upload's real content type and extension. A
// non-empty msg is the reason to reject it with 400; err is set as well when the
// file could not be read and should be logged.
func checkImageUpload(file *multipart.FileHeader) (msg string, err error) {
	// Trust the file bytes, not the client-declared Content-Type header.
	mimeType, err := sniffMIMEType(file)
	if err != nil {
		return "cannot read uploaded file", err
	}
	if !validateImageMIME(mimeType) {
		return "file format not supported", nil
	}
	if !slices.Contains(validImageExtensions, normalizeExt(file.Filename)) {
		return "file extension not supported", nil
	}
	return "", nil
}

// saveImageVariants writes the sm and md resized copies next to the original and
// removes the files stored so far when one of them fails.
func saveImageVariants(src image.Image, fileUUID, fileExt, filePath string) error {
	sizes := []struct {
		size string
		dim  int
	}{
		{"sm", 147},
		{"md", 400},
	}

	for _, s := range sizes {
		resizedImage := imageutil.Fill(src, s.dim, s.dim)
		resizedPath := fmt.Sprintf("%s/%s_%s.%s", dirUploads, fileUUID, s.size, fileExt)
		if err := imageutil.Save(resizedImage, resizedPath); err != nil {
			_ = os.Remove(filePath)
			_ = os.Remove(fmt.Sprintf("%s/%s_sm.%s", dirUploads, fileUUID, fileExt))
			return err
		}
	}
	return nil
}

// AddProductImage uploads and adds an image to a product.
//
// @Summary      Upload product image
// @Description  Upload an image (PNG/JPEG) and create sm (147x147) and md (400x400) variants
// @Tags         Products
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Param        document   formData file true "Image file (PNG or JPEG)"
// @Success      200 {object} webutil.HTTPResponse{result=models.File} "Added image"
// @Failure      400 {object} webutil.HTTPResponse "Invalid file format"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/image [post]
func AddProductImage(c fiber.Ctx) error {
	productID := c.Params("product_id")
	db := queries.DB()
	log := logging.New()

	file, err := c.FormFile("document")
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	if msg, err := checkImageUpload(file); msg != "" {
		if err != nil {
			log.ErrorStack(err)
		}
		return webutil.StatusBadRequest(c, msg)
	}

	fileUUID, fileExt, fileName := generateFileName(file.Filename)
	filePath := fmt.Sprintf("%s/%s", dirUploads, fileName)
	fileOrigName := file.Filename

	if err := saveFile(file, filePath); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Validate that the payload really is a decodable image; otherwise remove
	// the stored file so invalid content never persists on disk.
	fileSource, err := imageutil.Open(filePath)
	if err != nil {
		log.ErrorStack(err)
		_ = os.Remove(filePath)
		return webutil.StatusBadRequest(c, "file is not a valid image")
	}

	if err := saveImageVariants(fileSource, fileUUID, fileExt, filePath); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// swag resolves models.File in this file's annotations through its imports,
	// so the type is named here instead of being inferred.
	var addedImage *models.File
	addedImage, err = db.AddImage(c.Context(), productID, fileUUID, fileExt, fileOrigName)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Image added", addedImage)
}

// DeleteProductImage deletes an image from a product.
//
// @Summary      Delete product image
// @Description  Delete a product image by product and image ID
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Param        image_id   path string true "Image ID"
// @Success      200 {object} webutil.HTTPResponse "Image deleted"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/image/{image_id} [delete]
func DeleteProductImage(c fiber.Ctx) error {
	productID := c.Params("product_id")
	imageID := c.Params("image_id")
	db := queries.DB()
	log := logging.New()

	if err := db.DeleteImage(c.Context(), productID, imageID); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Image deleted", nil)
}
