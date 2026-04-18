package handlers

import (
	"net/http"
	"testing"

	"github.com/shurco/mycart/internal/testutil"
)

func TestAddPage_MalformedJSON(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()
	app.Post("/api/_/pages", AddPage)

	resp := testutil.DoRequest(t, app, http.MethodPost, "/api/_/pages",
		"{not json", cookie)
	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestUpdatePage_MalformedJSON(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()
	app.Patch("/api/_/pages/:page_id", UpdatePage)

	resp := testutil.DoRequest(t, app, http.MethodPatch,
		"/api/_/pages/any-page-id-0x", "{not json", cookie)
	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestUpdatePageContent_MalformedJSON(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()
	app.Patch("/api/_/pages/:page_id/content", UpdatePageContent)

	resp := testutil.DoRequest(t, app, http.MethodPatch,
		"/api/_/pages/any-page-id-0x/content", "{not json", cookie)
	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}
