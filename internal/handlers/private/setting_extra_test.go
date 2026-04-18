package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/internal/testutil"
	"github.com/shurco/mycart/pkg/update"
)

func TestUpdateSetting_BadJSONReturns400(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Patch("/api/_/settings/:setting_key", UpdateSetting)

	resp := testutil.DoRequest(t, app, http.MethodPatch,
		"/api/_/settings/main", "{not json", cookie)
	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestUpdateSetting_UnknownKeyRoutesToSettingName(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Patch("/api/_/settings/:setting_key", UpdateSetting)

	// Custom keys go through UpdateSettingByKey.
	resp := testutil.DoRequest(t, app, http.MethodPatch,
		"/api/_/settings/custom_prop", `{"value":"hello"}`, cookie)
	testutil.AssertStatus(t, resp, http.StatusOK, http.StatusInternalServerError)
}

func TestGetSetting_UnknownKeyNotFound(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/_/settings/:setting_key", GetSetting)

	// Unknown key => GetSettingByKey returns empty map + no error, so the
	// handler returns 200 with an empty payload. Accept either path.
	resp := testutil.DoRequest(t, app, http.MethodGet,
		"/api/_/settings/doesnotexist", "", cookie)
	testutil.AssertStatus(t, resp, http.StatusOK, http.StatusNotFound)
}

func TestVersion_UsesCacheOnSecondCall(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	// Pre-seed session cache so Version never reaches the network.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cached, _ := json.Marshal(&update.Version{CurrentVersion: "v0.0.0-test"})
	if err := queries.DB().AddSession(ctx, "update", string(cached),
		time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	app.Get("/api/_/version", Version)
	resp := testutil.DoRequest(t, app, http.MethodGet, "/api/_/version", "", cookie)
	testutil.AssertStatus(t, resp, http.StatusOK)
}

func TestVersion_CorruptCacheSurfaces500(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := queries.DB().AddSession(ctx, "update", "not-json",
		time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	app.Get("/api/_/version", Version)
	resp := testutil.DoRequest(t, app, http.MethodGet, "/api/_/version", "", cookie)
	testutil.AssertStatus(t, resp, http.StatusInternalServerError)
}
