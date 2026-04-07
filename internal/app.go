package app

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	"github.com/shurco/mycart/internal/middleware"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/internal/routes"
	"github.com/shurco/mycart/migrations"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

const (
	// DefaultBodyLimit is the maximum request body size (50MB)
	DefaultBodyLimit = 50 * 1024 * 1024
	// DefaultHTTPSPort is the default HTTPS port
	DefaultHTTPSPort = ":443"
)

var (
	DevMode bool
	log     *logging.Log
)

// NewApp initializes and starts the web application
func NewApp(httpAddr, httpsAddr string, noSite, appDev bool) error {
	DevMode = appDev
	log = logging.New()

	schema, mainAddr := determineSchemaAndAddr(httpAddr, httpsAddr)

	if err := queries.New(migrations.Embed()); err != nil {
		log.Err(err).Send()
		return err
	}

	app, err := setupFiberApp(noSite)
	if err != nil {
		return err
	}

	if err := Init(); err != nil {
		log.Err(err).Send()
		os.Exit(1)
	}

	setupRoutes(app, noSite)
	printStartupInfo(schema, mainAddr, noSite)

	if schema == "https" {
		return startHTTPS(app, mainAddr, httpsAddr)
	}

	return startHTTP(mainAddr, app)
}

// determineSchemaAndAddr determines the schema and main address based on the provided parameters.
func determineSchemaAndAddr(httpAddr, httpsAddr string) (schema, mainAddr string) {
	if httpsAddr != "" {
		return "https", httpsAddr
	}
	return "http", httpAddr
}

// setupFiberApp configures and returns a Fiber application instance.
func setupFiberApp(noSite bool) (*fiber.App, error) {
	config := fiber.Config{
		BodyLimit: DefaultBodyLimit,
	}

	// Site is now a SPA, no need for HTML templates

	app := fiber.New(config)
	middleware.Fiber(app, log.Logger)

	return app, nil
}

// setupRoutes configures application routes.
func setupRoutes(app *fiber.App, noSite bool) {
	app.Use("/uploads", static.New("./lc_uploads"))
	app.Use("/secrets", static.New("./lc_digitals"))

	// Register API routes before SPA routes to ensure they are processed first
	routes.ApiPrivateRoutes(app)
	if !noSite {
		routes.ApiPublicRoutes(app)
	}

	// Setup SPA routes before InstallCheck to allow static assets to be served
	if !noSite {
		routes.SiteRoutes(app)
	}
	routes.AdminRoutes(app)

	// InstallCheck runs after SPA routes to allow static assets (_app, etc.)
	app.Use(InstallCheck)

	routes.NotFoundRoute(app, noSite)
}

// printStartupInfo prints application startup information.
func printStartupInfo(schema, mainAddr string, noSite bool) {
	fmt.Print("🛒 myCart - open source shopping-cart in 1 file\n")
	if !noSite {
		fmt.Printf("├─ Cart UI: %s://%s/\n", schema, mainAddr)
	}
	fmt.Printf("└─ Admin UI: %s://%s/_/\n", schema, mainAddr)
}

// startHTTPS starts the server with HTTPS support and automatic TLS.
func startHTTPS(app *fiber.App, mainAddr, httpsAddr string) error {
	hostOnly := extractHostOnly(mainAddr)
	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(hostOnly),
		Cache:      autocert.DirCache("./lc_certs"),
	}

	cfgTLS := &tls.Config{
		GetCertificate: manager.GetCertificate,
		NextProtos:     []string{"http/1.1", "acme-tls/1"},
	}

	listenAddr := DefaultHTTPSPort
	if httpsAddr != "" {
		listenAddr = httpsAddr
	}

	ln, err := tls.Listen("tcp", listenAddr, cfgTLS)
	if err != nil {
		log.Err(err).Send()
		os.Exit(1)
	}

	if err := app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}); err != nil {
		log.Err(err).Send()
		os.Exit(1)
	}

	return nil
}

// extractHostOnly extracts only the host from the address, removing the port.
func extractHostOnly(addr string) string {
	if !strings.Contains(addr, ":") {
		return addr
	}

	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}

	return addr
}

// startHTTP starts the HTTP server with graceful shutdown support.
func startHTTP(mainAddr string, app *fiber.App) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if DevMode {
		return StartServer(ctx, mainAddr, app)
	}

	idleConnsClosed := make(chan struct{})

	go handleShutdown(ctx, app, idleConnsClosed)
	go func() {
		if err := StartServer(ctx, mainAddr, app); err != nil {
			log.Err(err).Send()
		}
	}()

	<-idleConnsClosed
	return nil
}

// handleShutdown handles application shutdown signals.
func handleShutdown(ctx context.Context, app *fiber.App, idleConnsClosed chan struct{}) {
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	<-sigint

	if err := app.Shutdown(); err != nil {
		log.Err(err).Send()
	}

	close(idleConnsClosed)
}

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
			return c.Redirect().To("/_/install")
		}
	} else if strings.HasPrefix(path, "/_/install") {
		return c.Redirect().To("/_")
	}

	return c.Next()
}

// isInstallPath checks if the path is related to installation or static assets.
func isInstallPath(path string) bool {
	return strings.HasPrefix(path, "/_/install") ||
		strings.HasPrefix(path, "/_/assets") ||
		strings.HasPrefix(path, "/_/_app") ||
		strings.HasPrefix(path, "/_app") ||
		strings.HasPrefix(path, "/api") ||
		strings.HasPrefix(path, "/uploads")
}

// StartServer starts the server and handles graceful shutdown.
func StartServer(ctx context.Context, addr string, a *fiber.App) error {
	errCh := make(chan error)

	go func() {
		if err := a.Listen(addr, fiber.ListenConfig{DisableStartupMessage: true}); err != nil {
			log.Err(err).Send()
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		err := errors.New("shutdown signal received, closing server")
		log.Err(err).Send()
		return a.Shutdown()
	case err := <-errCh:
		return err
	}
}
