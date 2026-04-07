package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/mailer"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/update"
	"github.com/shurco/mycart/pkg/webutil"
)

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

	session, err := db.GetSession(c.Context(), "update")
	if err != nil && err != sql.ErrNoRows {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	version := (*update.Version)(nil)
	if err == sql.ErrNoRows {
		version = update.VersionInfo()

		release, fetchErr := update.FetchLatestRelease(context.Background(), "shurco", "mycart")
		if fetchErr != nil {
			log.ErrorStack(fetchErr)
		} else if version.CurrentVersion != release.Name {
			version.NewVersion = release.Name
			version.ReleaseURL = release.GetUrl()
		}

		if err := db.DeleteSession(c.Context(), "update"); err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}

		data, err := json.Marshal(version)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		expires := time.Now().Add(24 * time.Hour).Unix()
		if err := db.AddSession(c.Context(), "update", string(data), expires); err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
	}

	if session != "" {
		version = new(update.Version)
		if err := json.Unmarshal([]byte(session), version); err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
	}

	return webutil.Response(c, fiber.StatusOK, "Version", version)
}

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

	var section any
	var err error

	switch settingKey {
	case "password":
		return webutil.StatusNotFound(c)
	case "main":
		section, err = db.GetSettingByGroup(c.Context(), &models.Main{})
	case "social":
		section, err = db.GetSettingByGroup(c.Context(), &models.Social{})
	case "auth":
		section, err = db.GetSettingByGroup(c.Context(), &models.Auth{})
	case "jwt":
		section, err = db.GetSettingByGroup(c.Context(), &models.JWT{})
	case "webhook":
		section, err = db.GetSettingByGroup(c.Context(), &models.Webhook{})
	case "payment":
		section, err = db.GetSettingByGroup(c.Context(), &models.Payment{})
	case "stripe":
		section, err = db.GetSettingByGroup(c.Context(), &models.Stripe{})
	case "paypal":
		section, err = db.GetSettingByGroup(c.Context(), &models.Paypal{})
	case "spectrocoin":
		section, err = db.GetSettingByGroup(c.Context(), &models.Spectrocoin{})
	case "coinbase":
		section, err = db.GetSettingByGroup(c.Context(), &models.Coinbase{})
	case "dummy":
		section, err = db.GetSettingByGroup(c.Context(), &models.Dummy{})
	case "mail":
		section, err = db.GetSettingByGroup(c.Context(), &models.Mail{})
	default:
		section, err = db.GetSettingByKey(c.Context(), settingKey)
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
func UpdateSetting(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()
	settingKey := c.Params("setting_key")
	var request any

	switch settingKey {
	case "password":
		request = &models.Password{}
	case "main":
		request = &models.Main{}
	case "auth":
		request = &models.Auth{}
	case "jwt":
		request = &models.JWT{}
	case "social":
		request = &models.Social{}
	case "payment":
		request = &models.Payment{}
	case "stripe":
		request = &models.Stripe{}
	case "paypal":
		request = &models.Paypal{}
	case "spectrocoin":
		request = &models.Spectrocoin{}
	case "coinbase":
		request = &models.Coinbase{}
	case "dummy":
		request = &models.Dummy{}
	case "webhook":
		request = &models.Webhook{}
	case "mail":
		request = &models.Mail{}
	default:
		request = &models.SettingName{}
	}

	// Parse the request body into the appropriate struct
	if err := c.Bind().Body(request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
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
		settingName.Key = settingKey
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
