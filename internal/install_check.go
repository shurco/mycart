package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/webutil"
)

// InstallCheck checks the installation status and redirects to the installation page if necessary.
func InstallCheck(c fiber.Ctx) error {
	db := queries.DB()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := db.GetSettingByKey(ctx, "installed")
	if err != nil {
		return webutil.StatusInternalServerError(c)
	}

	install, _ := strconv.ParseBool(fmt.Sprint(response["installed"].Value))
	path := c.Path()

	if !install {
		if !isInstallPath(path) {
			if strings.HasPrefix(path, "/api/") {
				return webutil.StatusBadRequest(c, "application not installed")
			}
			return c.Redirect().To("/_/install")
		}
	} else if strings.HasPrefix(path, "/_/install") {
		return c.Redirect().To("/_")
	}

	return c.Next()
}

// isInstallPath reports paths that are reachable before the cart is installed.
func isInstallPath(path string) bool {
	if strings.HasPrefix(path, "/_/install") ||
		strings.HasPrefix(path, "/_/assets") ||
		strings.HasPrefix(path, "/_/_app") ||
		strings.HasPrefix(path, "/_app") ||
		strings.HasPrefix(path, "/uploads") {
		return true
	}
	if strings.HasPrefix(path, "/api/install") {
		return true
	}
	// Storefront public APIs stay available during first-time setup.
	return path == "/ping" ||
		strings.HasPrefix(path, "/api/settings") ||
		strings.HasPrefix(path, "/api/pages/") ||
		strings.HasPrefix(path, "/api/products") ||
		strings.HasPrefix(path, "/api/cart") ||
		strings.HasPrefix(path, "/cart/")
}
