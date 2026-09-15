package handlers

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/migrations"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

type installStatus struct {
	Installed bool            `json:"installed"`
	Database  installDatabase `json:"database"`
}

// installDatabase tells the wizard which database the process is running on and
// whether the operator has already decided. A locked database cannot be changed
// from the wizard: the operator set it with a flag or an environment variable,
// and quietly connecting somewhere else would be wrong.
type installDatabase struct {
	Driver string `json:"driver"`
	DSN    string `json:"dsn"`
	Source string `json:"source"`
	Locked bool   `json:"locked"`
}

// installDatabaseTest is the request of the "test connection" button.
type installDatabaseTest struct {
	Driver string `json:"driver"`
	DSN    string `json:"dsn"`
}

// installDatabaseTestResult is its response. ServerVersion and
// HasExistingSchema let the operator see what they are about to install into
// without having to read the server.
type installDatabaseTestResult struct {
	OK                bool   `json:"ok"`
	Driver            string `json:"driver"`
	ServerVersion     string `json:"server_version,omitempty"`
	HasExistingSchema bool   `json:"has_existing_schema"`
}

// InstallStatus reports whether first-time setup has been completed.
//
// @Summary      Installation status
// @Description  Returns whether the cart has been installed
// @Tags         Install
// @Produce      json
// @Success      200 {object} webutil.HTTPResponse{result=installStatus}
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/install/status [get]
func InstallStatus(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	installed, err := db.IsInstalled(c.Context())
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	cfg := database.Active()
	return webutil.Response(c, fiber.StatusOK, "Installation status", installStatus{
		Installed: installed,
		Database: installDatabase{
			Driver: cfg.Driver,
			// Never the raw DSN: it carries the database password.
			DSN:    cfg.Redacted(),
			Source: cfg.Source,
			Locked: cfg.Pinned(),
		},
	})
}

// Install performs the initial installation of the application.
//
// @Summary      Install application
// @Description  Perform initial setup with admin credentials and domain
// @Tags         Install
// @Accept       json
// @Produce      json
// @Param        request body models.Install true "Installation data"
// @Success      200 {object} webutil.HTTPResponse "Cart installed"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/install [post]
func Install(c fiber.Ctx) error {
	log := logging.New()
	request := new(models.Install)

	if err := c.Bind().Body(request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	if err := request.Validate(); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	current := database.Active()
	target := current
	if request.Database != nil {
		target = choiceConfig(*request.Database, current)
	}

	// No switch requested: the ordinary path, unchanged.
	if target.Driver == current.Driver && target.DSN == current.DSN {
		if err := queries.DB().Install(c.Context(), request); err != nil {
			if errors.Is(err, queries.ErrAlreadyInstalled) {
				return webutil.StatusBadRequest(c, err.Error())
			}
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		return webutil.Response(c, fiber.StatusOK, "Cart installed", nil)
	}

	// Switching is only for first-time setup. The endpoint is reachable without
	// a session, so once the running database holds an installation, moving the
	// process to another one would hand the shop to whoever asks.
	//
	// A failure to read the current state refuses the switch as well: a security
	// decision may not be skipped because the check itself went wrong.
	installed, err := queries.DB().IsInstalled(c.Context())
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}
	if installed {
		return webutil.StatusBadRequest(c, "application already installed")
	}

	if current.Pinned() {
		return webutil.StatusBadRequest(c, fmt.Sprintf(
			"the database is fixed by the %s configuration; restart the process with a different one to install elsewhere",
			current.Source))
	}

	if err := installInto(c.Context(), target, request); err != nil {
		log.ErrorStack(err)
		if status, ok := installFailureStatus(err); ok {
			return webutil.Response(c, status, err.Error(), nil)
		}
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Cart installed", nil)
}

// installInto installs the cart into a database other than the one the process
// started on, then makes it the active one.
//
// The order matters: the new database is fully prepared and installed before
// anything about the running process changes, and config.json is written last,
// so a failure at any point leaves the previous installation untouched.
func installInto(ctx context.Context, target database.Config, request *models.Install) error {
	conn, err := database.Open(target, migrations.Embed())
	if err != nil {
		// The wizard calls this before anyone has authenticated, so the server's
		// own words — host, user, database name — must not travel back.
		// However, distinguish between connection failures and migration failures.
		detail := err.Error()

		// Check for connection-related errors first, even if wrapped in "run migrations:"
		// Connection errors should be reported as connection failures, not migration failures
		if strings.Contains(detail, "connection refused") ||
			strings.Contains(detail, "no such host") ||
			strings.Contains(detail, "i/o timeout") ||
			strings.Contains(detail, "context deadline exceeded") ||
			strings.Contains(detail, "network is unreachable") ||
			strings.Contains(detail, "dial tcp") ||
			strings.Contains(detail, "dial error") {
			return &installError{status: fiber.StatusBadRequest,
				err: fmt.Errorf("connect to the selected database: %s", describeConnectFailure(err))}
		}

		// If it contains "run migrations:" but isn't a connection error, it's a migration failure
		if strings.Contains(detail, "run migrations:") {
			return &installError{status: fiber.StatusBadRequest,
				err: fmt.Errorf("apply database schema: migrations failed; check server logs for details")}
		}

		// Default to connection failure for other errors
		return &installError{status: fiber.StatusBadRequest,
			err: fmt.Errorf("connect to the selected database: %s", describeConnectFailure(err))}
	}

	candidate := queries.NewBase(conn)

	// Refuse to overwrite a cart that is already running somewhere else.
	installed, err := candidate.IsInstalled(ctx)
	if err != nil {
		_ = conn.Close()
		return &installError{status: fiber.StatusBadRequest, err: fmt.Errorf("inspect the selected database: %w", err)}
	}
	if installed {
		_ = conn.Close()
		return &installError{status: fiber.StatusConflict, err: fmt.Errorf("the selected database already holds an installed cart")}
	}

	if err := candidate.Install(ctx, request); err != nil {
		_ = conn.Close()
		if errors.Is(err, queries.ErrAlreadyInstalled) {
			return &installError{status: fiber.StatusConflict, err: err}
		}
		return &installError{status: fiber.StatusInternalServerError, err: err}
	}

	previous := queries.Swap(conn)
	previousConfig := database.Active()
	database.SetActive(target)

	if err := database.WriteConfig(target); err != nil {
		// Roll back both halves: the queries and what the process reports as
		// its database. Leaving Active on the target would make the wizard and
		// the status endpoint describe an installation that a restart would not
		// return to.
		queries.Swap(previous)
		database.SetActive(previousConfig)
		_ = conn.Close()
		return err
	}

	if previous != nil && previous != conn {
		_ = previous.Close()
	}

	return nil
}

// installError carries the HTTP status an install failure should be reported
// with. Everything the operator can fix — an unreachable database, a cart that
// already exists there — is a 4xx; anything else is a server error.
type installError struct {
	status int
	err    error
}

func (e *installError) Error() string { return e.err.Error() }
func (e *installError) Unwrap() error { return e.err }

// installFailureStatus reports the HTTP status stored in err. The second result
// is false when the error carries no status of its own, which is a server error
// and must be reported without exposing the error text.
func installFailureStatus(err error) (int, bool) {
	var failure *installError
	if stderrors.As(err, &failure) {
		return failure.status, true
	}
	return 0, false
}

// choiceConfig turns the wizard's selection into a full configuration. A
// selection equal to the current one is left alone, so "SQLite (default)" on an
// installation that already uses it stays a no-op.
func choiceConfig(choice models.DatabaseChoice, current database.Config) database.Config {
	cfg := database.Config{
		Driver: choice.Driver,
		DSN:    choice.DSN,
		Source: database.SourceWizard,
	}

	switch cfg.Driver {
	case database.DriverSQLite:
		if cfg.DSN == "" {
			cfg.DSN = database.DefaultSQLiteDSN
		}
	case database.DriverPostgres:
		// The DSN is required and validated by models.DatabaseChoice.
	}

	if cfg.Driver == current.Driver && cfg.DSN == current.DSN {
		return current
	}
	return cfg
}

// InstallDBTest checks that the wizard's database selection is reachable,
// without changing anything. It is the "test connection" button.
//
// @Summary      Test database connection
// @Description  Connects to the selected database and reports what was found
// @Tags         Install
// @Accept       json
// @Produce      json
// @Param        request body installDatabaseTest true "Database to test"
// @Success      200 {object} webutil.HTTPResponse{result=installDatabaseTestResult}
// @Failure      400 {object} webutil.HTTPResponse "Connection failed"
// @Router       /api/install/db/test [post]
func InstallDBTest(c fiber.Ctx) error {
	log := logging.New()
	request := new(installDatabaseTest)

	if err := c.Bind().Body(request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	// Reachable without a session, so it must not survive the installation:
	// afterwards it would let anyone probe arbitrary addresses from the server.
	if installed, err := queries.DB().IsInstalled(c.Context()); err == nil && installed {
		return webutil.StatusBadRequest(c, "application already installed")
	}

	choice := models.DatabaseChoice{Driver: request.Driver, DSN: request.DSN}
	if choice.Driver == "" {
		choice.Driver = database.DriverSQLite
	}
	if err := choice.Validate(); err != nil {
		return webutil.StatusBadRequest(c, err.Error())
	}

	target := choiceConfig(choice, database.Active())

	conn, err := database.Connect(target)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, describeConnectFailure(err))
	}
	defer func() { _ = conn.Close() }()

	result := installDatabaseTestResult{OK: true, Driver: target.Driver}
	result.HasExistingSchema = tableExists(c.Context(), conn, "setting")

	if target.Driver == database.DriverPostgres {
		if err := conn.QueryRowContext(c.Context(), "SELECT version()").Scan(&result.ServerVersion); err != nil {
			log.ErrorStack(err)
		}
	}

	return webutil.Response(c, fiber.StatusOK, "Database connection OK", result)
}

// describeConnectFailure turns a connection error into something the operator
// can act on, without echoing the address, user or database back to a caller
// that has not authenticated yet.
func describeConnectFailure(err error) string {
	detail := err.Error()

	switch {
	case strings.Contains(detail, "password authentication failed"),
		strings.Contains(detail, "role \"") && strings.Contains(detail, "does not exist"),
		strings.Contains(detail, "no password supplied"):
		return "authentication failed: check the user and password"
	case strings.Contains(detail, "does not exist") && strings.Contains(detail, "database"):
		return "the database does not exist on that server"
	case strings.Contains(detail, "connection refused"),
		strings.Contains(detail, "no such host"),
		strings.Contains(detail, "i/o timeout"),
		strings.Contains(detail, "context deadline exceeded"),
		strings.Contains(detail, "network is unreachable"):
		return "the server is not reachable at that address"
	case strings.Contains(detail, "no such file or directory"),
		strings.Contains(detail, "unable to open database file"),
		strings.Contains(detail, "permission denied"):
		return "the database file cannot be opened at that path"
	case strings.Contains(detail, "timezone"):
		// Our own message: it names the problem and the fix, and nothing else.
		return detail
	case strings.Contains(detail, "is a pgxpool option"):
		return detail
	default:
		return "could not connect to the selected database: check the connection settings"
	}
}

// tableExists probes for a table by selecting from it. Both engines report a
// missing table as an error, which keeps the probe free of dialect-specific
// catalogue queries; the wizard only uses it for a warning, so an error caused
// by something else is not worth distinguishing.
func tableExists(ctx context.Context, conn *database.Conn, table string) bool {
	var count int
	return conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count) == nil
}
