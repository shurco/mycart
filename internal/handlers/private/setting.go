package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/mailer"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/seo"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/update"
	"github.com/shurco/mycart/pkg/webutil"
)

// versionCacheTTL is how long the fetched release info is cached in the session store.
const versionCacheTTL = 24 * time.Hour

// Version returns the current application version and update information.
//
// @Summary      Get version
// @Description  Get current app version and available updates (cached 24h)
// @Tags         Settings
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} webutil.HTTPResponse "Version info"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/version [get]
func Version(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	if cached, err := loadCachedVersion(c.Context(), db); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	} else if cached != nil {
		return webutil.Response(c, fiber.StatusOK, "Version", cached)
	}

	version := currentVersion()
	if release, fetchErr := update.FetchLatestRelease(c.Context(), "shurco", "mycart"); fetchErr != nil {
		log.ErrorStack(fetchErr)
	} else if release != nil && version.CurrentVersion != release.Name {
		version.NewVersion = release.Name
		version.ReleaseURL = release.GetUrl()
	}

	if err := cacheVersion(c.Context(), db, version); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Version", version)
}

// currentVersion returns a safe copy of the compiled-in version info.
// Returns an empty Version if SetVersion was not called (e.g. in tests).
func currentVersion() *update.Version {
	if info := update.VersionInfo(); info != nil {
		v := *info
		return &v
	}
	return &update.Version{}
}

// loadCachedVersion returns non-nil Version if the value is present in the session cache.
func loadCachedVersion(ctx context.Context, db *queries.Base) (*update.Version, error) {
	session, err := db.GetSession(ctx, "update")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if session == "" {
		return nil, nil
	}
	v := &update.Version{}
	if err := json.Unmarshal([]byte(session), v); err != nil {
		return nil, err
	}
	return v, nil
}

// cacheVersion stores the version info in the session cache with a TTL.
func cacheVersion(ctx context.Context, db *queries.Base, v *update.Version) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	// AddSession is idempotent for the same key (INSERT … ON CONFLICT … DO
	// UPDATE), so we don't need an explicit DeleteSession here.
	expires := time.Now().Add(versionCacheTTL).Unix()
	return db.AddSession(ctx, "update", string(data), expires)
}

// secretSettingKeys are standalone keys whose stored value is never read back
// over the API.
//
// The admin password is write-only because the panel has no use for the hash.
// account_jwt_secret is the key the cabinet signs its session cookies with: it
// is generated and read only by queries.CustomerQueries, an operator never sets
// it, and the group it belongs to (settingModelFor("account")) deliberately
// leaves it out. Reaching it through the raw key/value fallback would hand it to
// anyone holding an admin token, so the fallback refuses it as well.
var secretSettingKeys = []string{"password", "account_jwt_secret"}

// GetSetting returns a setting value by key.
//
// @Summary      Get setting
// @Description  Get a setting group or individual setting by key
// @Tags         Settings
// @Security     BearerAuth
// @Produce      json
// @Param        setting_key path string true "Setting key (main, social, auth, jwt, webhook, payment, stripe, paypal, spectrocoin, coinbase, dummy, mail, or custom key)"
// @Success      200 {object} webutil.HTTPResponse "Setting value"
// @Failure      404 {object} webutil.HTTPResponse "Setting not found"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/settings/{setting_key} [get]
func GetSetting(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()
	settingKey := c.Params("setting_key")

	if slices.Contains(secretSettingKeys, settingKey) {
		// Secrets are write-only — never hand the stored value back over the API.
		return webutil.StatusNotFound(c)
	}

	var section any
	var err error
	if model := settingModelFor(settingKey); model != nil {
		section, err = db.GetSettingByGroup(c.Context(), model)
	} else {
		// The key/value query answers an unknown key with an empty map rather
		// than an error, so "there is no such setting" is decided here.
		var values map[string]models.SettingName
		if values, err = db.GetSettingByKey(c.Context(), settingKey); err == nil && len(values) == 0 {
			return webutil.StatusNotFound(c)
		}
		section = values
	}

	if err != nil {
		if errors.Is(err, errors.ErrSettingNotFound) {
			return webutil.StatusNotFound(c)
		}
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}
	return webutil.Response(c, fiber.StatusOK, "Setting", section)
}

// UpdateSetting updates a setting value by key.
//
// @Summary      Update setting
// @Description  Update a setting group or individual setting by key
// @Tags         Settings
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        setting_key path string true "Setting key"
// @Param        request     body object true "Setting value (structure depends on key)"
// @Success      200 {object} webutil.HTTPResponse "Setting updated"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/settings/{setting_key} [patch]
// protectedSettingKeys are raw setting keys that must never be writable via
// the generic key/value PATCH fallback: flipping them would let an attacker
// re-open the unauthenticated install endpoint (installed), or rotate the
// token-signing secret (jwt_secret) for a persistent backdoor.
// account_jwt_secret is the cabinet's counterpart to jwt_secret and is refused
// for the same reason.
var protectedSettingKeys = []string{"installed", "jwt_secret", "account_jwt_secret"}

// groupOnlySettingKeys are keys a group carries and validates. Writing one of
// them on its own through the raw key/value fallback skips that validation,
// which for the branding marks is the check that a value is a bare file name
// rather than a path — so the fallback refuses them and the group endpoint,
// which validates, is the way in.
var groupOnlySettingKeys = []string{
	"branding_logo",
	"branding_favicon",
	"branding_tagline",
}

// UpdateSetting updates a setting value by key.
//
// @Summary      Update setting
// @Description  Update a setting group or individual setting by key
// @Tags         Settings
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        setting_key path string true "Setting key"
// @Param        request     body object true "Setting value (structure depends on key)"
// @Success      200 {object} webutil.HTTPResponse "Setting updated"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/settings/{setting_key} [patch]
func UpdateSetting(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()
	settingKey := c.Params("setting_key")

	if slices.Contains(protectedSettingKeys, settingKey) {
		return webutil.StatusBadRequest(c, "setting key is protected")
	}

	if slices.Contains(groupOnlySettingKeys, settingKey) {
		return webutil.StatusBadRequest(c, "setting is written through its group")
	}

	var request any
	switch {
	case settingKey == "password":
		request = &models.Password{}
	default:
		if model := settingModelFor(settingKey); model != nil {
			request = model
		} else {
			request = &models.SettingName{}
		}
	}

	// Parse the request body into the appropriate struct
	if err := c.Bind().Body(request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	// Raw key/value writes get their key from the URL path before validation.
	if settingName, ok := request.(*models.SettingName); ok {
		settingName.Key = settingKey
	}

	// Run model validation when the bound struct defines it, so stored
	// settings always satisfy their declared constraints.
	if v, ok := request.(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			log.ErrorStack(err)
			return webutil.StatusBadRequest(c, err.Error())
		}
	}

	// Handle the password update separately if that's the case
	if settingKey == "password" {
		password := request.(*models.Password)
		if err := db.UpdatePassword(c.Context(), password); err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		return webutil.Response(c, fiber.StatusOK, "Password updated", nil)
	}

	if settingName, ok := request.(*models.SettingName); ok {
		if err := db.UpdateSettingByKey(c.Context(), settingName); err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		return webutil.Response(c, fiber.StatusOK, "Setting key updated", nil)
	}

	// Update setting for all other cases
	if err := db.UpdateSettingByGroup(c.Context(), request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Reload CDN rules when SEO settings change.
	if settingKey == "seo" {
		if s, ok := request.(*models.SEO); ok {
			seo.StoreCDN(s)
		}
	}

	return webutil.Response(c, fiber.StatusOK, "Setting group updated", nil)
}

// TestLetter sends a test email letter.
//
// @Summary      Send test letter
// @Description  Send a test email using configured SMTP settings
// @Tags         Settings
// @Security     BearerAuth
// @Produce      json
// @Param        letter_name path string true "Letter template name"
// @Success      200 {object} webutil.HTTPResponse{result=string} "Message sent"
// @Failure      400 {object} webutil.HTTPResponse "Sending failed"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/test/letter/{letter_name} [get]
func TestLetter(c fiber.Ctx) error {
	letter := c.Params("letter_name")
	log := logging.New()

	if err := mailer.SendTestLetter(letter); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	return webutil.Response(c, fiber.StatusOK, "Test letter", "Message sent to your mailbox")
}
