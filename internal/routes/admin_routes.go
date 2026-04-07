package routes

import (
	"io/fs"

	"github.com/gofiber/fiber/v3"
	"github.com/shurco/mycart/web"
)

func AdminRoutes(c *fiber.App) {
	embedAdmin, _ := fs.Sub(web.EmbedAdmin(), web.AdminBuildPath)

	c.Use("/_", setupSPAHandler(embedAdmin, func(string) bool { return false }, "/_"))
}
