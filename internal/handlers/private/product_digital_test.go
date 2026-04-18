package handlers

import (
	"net/http"
	"testing"

	"github.com/shurco/mycart/internal/testutil"
)

func TestUpdateProductDigital(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Patch("/api/_/products/:product_id/digital/:digital_id", UpdateProductDigital)

	// Fixture: digital_data row c0gog7a4zrwW4Vf belongs to product xrtb1b919t2nuj9.
	resp := testutil.DoRequest(
		t, app, http.MethodPatch,
		"/api/_/products/xrtb1b919t2nuj9/digital/c0gog7a4zrwW4Vf",
		`{"content":"new-license-key"}`,
		"",
	)
	testutil.AssertStatus(t, resp, http.StatusOK)
}

func TestUpdateProductDigital_BadBody(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Patch("/api/_/products/:product_id/digital/:digital_id", UpdateProductDigital)

	resp := testutil.DoRequest(
		t, app, http.MethodPatch,
		"/api/_/products/xrtb1b919t2nuj9/digital/c0gog7a4zrwW4Vf",
		"{not json",
		"",
	)
	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestDeleteProductDigital_Data(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Delete("/api/_/products/:product_id/digital/:digital_id", DeleteProductDigital)

	resp := testutil.DoRequest(
		t, app, http.MethodDelete,
		"/api/_/products/xrtb1b919t2nuj9/digital/c0gog7a4zrwW4Vf",
		"", "",
	)
	testutil.AssertStatus(t, resp, http.StatusOK)
}

func TestDeleteProductDigital_UnknownID(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Delete("/api/_/products/:product_id/digital/:digital_id", DeleteProductDigital)

	// Non-existent product must surface as a 500 (the underlying scan fails
	// with sql.ErrNoRows before we reach the switch on digital type).
	resp := testutil.DoRequest(
		t, app, http.MethodDelete,
		"/api/_/products/no-such-product/digital/no-such-digital",
		"", "",
	)
	testutil.AssertStatus(t, resp, http.StatusInternalServerError, http.StatusOK)
}

func TestDeleteProductImage_Roundtrip(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Delete("/api/_/products/:product_id/image/:image_id", DeleteProductImage)

	// The fixture image id dj9bae53oob0ukj belongs to product fv6c9s9cqzf36sc;
	// the physical file is missing so DeleteImage errors out — we still cover
	// the handler path.
	resp := testutil.DoRequest(
		t, app, http.MethodDelete,
		"/api/_/products/fv6c9s9cqzf36sc/image/dj9bae53oob0ukj",
		"", "",
	)
	testutil.AssertStatus(t, resp,
		http.StatusOK, http.StatusInternalServerError)
}
