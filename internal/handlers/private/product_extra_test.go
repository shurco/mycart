package handlers

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/shurco/mycart/internal/testutil"
)

func TestAddProduct_MalformedJSON(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()
	app.Post("/api/_/products", AddProduct)

	resp := testutil.DoRequest(t, app, http.MethodPost, "/api/_/products", "{not json", "")
	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestUpdateProduct_MalformedJSON(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()
	app.Patch("/api/_/products/:product_id", UpdateProduct)

	resp := testutil.DoRequest(t, app, http.MethodPatch,
		"/api/_/products/fv6c9s9cqzf36sc", "{not json", "")
	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestAddProductImage_NoFile(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()
	app.Post("/api/_/products/:product_id/image", AddProductImage)

	// FormFile returns error if no multipart part named "document".
	resp := testutil.DoRequest(t, app, http.MethodPost,
		"/api/_/products/fv6c9s9cqzf36sc/image", "", "")
	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestAddProductDigital_FileUpload(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()
	app.Post("/api/_/products/:product_id/digital", AddProductDigital)

	// Upload a tiny file to exercise the file branch of AddProductDigital.
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="document"; filename="data.bin"`)
	hdr.Set("Content-Type", "application/octet-stream")
	fw, err := w.CreatePart(hdr)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte("content"))
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost,
		"/api/_/products/fv6c9s9cqzf36sc/digital", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	testutil.AssertStatus(t, resp, http.StatusOK)
}
