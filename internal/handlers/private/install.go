package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
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
		return installCurrent(c, request)
	}

	return installSwitched(c, current, target, request)
}

// installCurrent installs into the database the process already runs on.
func installCurrent(c fiber.Ctx, request *models.Install) error {
	if err := queries.DB().Install(c.Context(), request); err != nil {
		if errors.Is(err, queries.ErrAlreadyInstalled) {
			return webutil.StatusBadRequest(c, err.Error())
		}
		logging.New().ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}
	return webutil.Response(c, fiber.StatusOK, "Cart installed", nil)
}

// installSwitched installs into a database chosen in the wizard, after checking
// that switching is still allowed.
func installSwitched(c fiber.Ctx, current, target database.Config, request *models.Install) error {
	log := logging.New()

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
