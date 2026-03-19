package webutil

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
)

type HTTPResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Result  any    `json:"result,omitzero"`
}

func Response(c fiber.Ctx, status int, message string, data any) error {
	if len(message) > 0 {
		return c.Status(status).JSON(HTTPResponse{
			Success: status == fiber.StatusOK,
			Message: message,
			Result:  data,
		})
	}

	return c.Status(status).JSON(data)
}

func StatusOK(c fiber.Ctx, message string, data any) error {
	return Response(c, fiber.StatusOK, message, data)
}

func StatusNotFound(c fiber.Ctx) error {
	return Response(c, fiber.StatusNotFound, http.StatusText(fiber.StatusNotFound), nil)
}

func StatusBadRequest(c fiber.Ctx, data any) error {
	return Response(c, fiber.StatusBadRequest, http.StatusText(fiber.StatusBadRequest), data)
}

func StatusInternalServerError(c fiber.Ctx) error {
	return Response(c, fiber.StatusInternalServerError, http.StatusText(fiber.StatusInternalServerError), nil)
}
