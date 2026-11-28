package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/internal/middleware"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/internal/routes"
	"github.com/shurco/mycart/internal/seo"
	"github.com/shurco/mycart/pkg/logging"
)

const (
	// DefaultBodyLimit is the maximum request body size (50MB)
	DefaultBodyLimit = 50 * 1024 * 1024
	// DefaultHTTPSPort is the default HTTPS port
	DefaultHTTPSPort = ":443"
)

// Process-wide state that NewApp installs and the rest of the package reads.
//
// Both are written by NewApp and read from goroutines that outlive the call it
// makes them in: the Fiber logger middleware holds the logger, and the listen
// goroutine inside StartServer reads both. As plain variables that is a data
// race — reported under -race between NewApp and a goroutine left running by an
// earlier NewApp in the same process, which is what a test binary does — so
// access is atomic and the ordering is defined rather than merely unlikely.
var (
	devMode atomic.Bool
	appLog  atomic.Pointer[logging.Log]
)

// DevMode reports whether the application is in development mode. It is false
// until NewApp has run.
func DevMode() bool { return devMode.Load() }

// setDevMode sets the development mode flag.
func setDevMode(dev bool) { devMode.Store(dev) }

// logger returns the process-wide logger, or nil before NewApp has run.
func logger() *logging.Log { return appLog.Load() }

// setLogger installs the process-wide logger.
func setLogger(l *logging.Log) { appLog.Store(l) }

// NewApp initializes and starts the web application
func NewApp(dbCfg database.Config, httpAddr, httpsAddr string, noSite, appDev bool) error {
	lg := logging.New()
	setDevMode(appDev)
	setLogger(lg)

	schema, mainAddr := determineSchemaAndAddr(httpAddr, httpsAddr)

	if err := Init(dbCfg); err != nil {
		return err
	}

	// Load CDN rules on startup.
	loadCDNRules()

	app, err := setupFiberApp(noSite)
	if err != nil {
		return err
	}

	setupRoutes(app, noSite)
	printStartupInfo(os.Stdout, schema, mainAddr, noSite, dbCfg)

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
		// Trust loopback/private proxies so c.Scheme() (and therefore the
		// Secure cookie flag and generated payment URLs) honors
		// X-Forwarded-Proto from a TLS-terminating reverse proxy, while
		// spoofed headers from direct clients are ignored.
		TrustProxy: true,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Loopback: true,
			Private:  true,
		},
	}

	// Site is now a SPA, no need for HTML templates

	app := fiber.New(config)
	middleware.Fiber(app, logger().Logger)

	return app, nil
}

// setupRoutes configures application routes.
func setupRoutes(app *fiber.App, noSite bool) {
	// Public image uploads only. Digital products (lc_digitals) are
	// intentionally NOT served statically: purchased files are delivered by
	// email and admins download them via an authenticated endpoint
	// (/api/_/products/:id/digital/:file_id/download).
	app.Use("/uploads", static.New("./lc_uploads"))

	// CDN middleware (runs on every request; must be before routes).
	app.Use(seo.CDNMiddleware())

	// Register gofiber-swagger for OpenAPI spec generation.
	// Serves UI at /_/swagger (admin-only), generates spec at startup.
	if err := routes.RegisterSwagger(app); err != nil {
		logging.New().ErrorStack(fmt.Errorf("register swagger: %w", err))
	}

	// Register API routes before InstallCheck so /api/install is reachable on first boot.
	routes.ApiPrivateRoutes(app)
	if !noSite {
		routes.ApiPublicRoutes(app)
	}

	// InstallCheck must run before SPA handlers: the SPA middleware serves
	// index.html without calling c.Next(), so a guard registered after it
	// never executes for /_/ paths.
	app.Use(InstallCheck)

	if !noSite {
		routes.SiteRoutes(app)
	}
	routes.AdminRoutes(app)

	routes.NotFoundRoute(app, noSite)
}

// loadCDNRules loads SEO CDN rules from the database on startup.
func loadCDNRules() {
	db := queries.DB()
	if db == nil {
		return // Database not yet initialized.
	}
	ctx := context.Background()
	s, err := queries.GetSettingByGroup[models.SEO](ctx, db)
	if err == nil {
		seo.StoreCDN(s)
	}
}

// printStartupInfo writes the application startup information to w.
//
// The writer is a parameter rather than stdout directly so the banner can be
// read back without swapping os.Stdout process-wide, which is a global the
// server goroutines also write to.
func printStartupInfo(w io.Writer, schema, mainAddr string, noSite bool, dbCfg database.Config) {
	fmt.Fprint(w, "🛒 myCart - open source shopping-cart in 1 file\n")
	if !noSite {
		fmt.Fprintf(w, "├─ Cart UI: %s://%s/\n", schema, mainAddr)
	}
	fmt.Fprintf(w, "├─ Admin UI: %s://%s/_/\n", schema, mainAddr)
	fmt.Fprintf(w, "├─ Database: %s (%s)\n", dbCfg.Driver, dbCfg.Redacted())
	fmt.Fprintf(w, "└─ API Docs: %s://%s/_/swagger\n", schema, mainAddr)
}
