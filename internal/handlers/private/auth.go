package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/jwtutil"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/security"
	"github.com/shurco/mycart/pkg/webutil"
)

// SignIn authenticates a user and sets an HttpOnly JWT session cookie.
//
// @Summary      Sign in
// @Description  Authenticate an admin user with email and password and set the session cookie
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body models.SignIn true "Credentials"
// @Success      200 {object} webutil.HTTPResponse "Signed in"
// @Failure      400 {object} webutil.HTTPResponse "Invalid credentials or validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/sign/in [post]
func SignIn(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()
	request := new(models.SignIn)

	if err := c.Bind().Body(request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	if err := request.Validate(); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	passwordHash, err := db.GetPasswordByEmail(c.Context(), request.Email)
	if err != nil {
		// Unknown email: run the same bcrypt comparison against a dummy hash
		// and return the same generic error as a wrong password.
		security.ComparePasswords(security.DummyPasswordHash, request.Password)
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, "wrong user email address or password")
	}

	compareUserPassword := security.ComparePasswords(passwordHash, request.Password)
	if !compareUserPassword {
		return webutil.StatusBadRequest(c, "wrong user email address or password")
	}

	// Create a server-revocable first-party session token.
	settingJWT, err := queries.GetSettingByGroup[models.JWT](c.Context(), db)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	userID := uuid.New()
	expires := time.Now().Add(time.Hour * time.Duration(settingJWT.ExpireHours)).Unix()
	token, err := jwtutil.GenerateNewToken(settingJWT.Secret, userID.String(), expires, nil)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Add session record. Written through SessionValue so the row carries the
	// role the admin middleware checks for rather than a second copy of it.
	if err := db.AddSession(c.Context(),
		userID.String(), queries.SessionValue(queries.SessionRoleAdmin, ""), expires); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Unix(expires, 0),
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Strict",
	})

	return webutil.StatusOK(c, "Signed in", nil)
}

// SignOut invalidates the user session and clears the authentication token.
//
// @Summary      Sign out
// @Description  Invalidate current session and clear token cookie
// @Tags         Auth
// @Security     BearerAuth
// @Success      204 "Session invalidated"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/sign/out [post]
func SignOut(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	settingJWT, err := queries.GetSettingByGroup[models.JWT](c.Context(), db)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	claims, err := jwtutil.ExtractTokenMetadata(c, settingJWT.Secret)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	if err := db.DeleteSession(c.Context(), claims.ID); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Expires:  time.Now().Add(-(time.Hour * 2)),
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Strict",
	})

	return c.SendStatus(fiber.StatusNoContent)
}
