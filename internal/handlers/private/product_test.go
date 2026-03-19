package handlers

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"github.com/shurco/litecart/internal/testutil"
)

func TestProducts(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/_/products", Products)

	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{"default list", "", http.StatusOK},
		{"custom pagination", "?page=1&limit=5", http.StatusOK},
		{"high page (empty result)", "?page=999", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodGet, "/api/_/products"+tt.query, "", "")
			testutil.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestAddProduct(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Post("/api/_/products", AddProduct)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			"valid file product",
			`{"name":"NewProd","slug":"newprod","amount":500,"digital":{"type":"file"}}`,
			http.StatusOK,
		},
		{
			"valid data product",
			`{"name":"DataProd","slug":"dataprod","amount":300,"digital":{"type":"data"}}`,
			http.StatusOK,
		},
		{
			"missing digital type",
			`{"name":"NoDig","slug":"nodig","amount":100}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodPost, "/api/_/products", tt.body, "")
			testutil.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestProduct(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/_/products/:product_id", Product)

	tests := []struct {
		name       string
		productID  string
		wantStatus []int
	}{
		{"existing product from fixtures", "fv6c9s9cqzf36sc", []int{http.StatusOK}},
		{"non-existent product", "nonexistent12345", []int{http.StatusNotFound, http.StatusInternalServerError}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodGet, "/api/_/products/"+tt.productID, "", "")
			testutil.AssertStatus(t, resp, tt.wantStatus...)
		})
	}
}

func TestUpdateProduct(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Patch("/api/_/products/:product_id", UpdateProduct)

	resp := testutil.DoRequest(t, app, http.MethodPatch, "/api/_/products/fv6c9s9cqzf36sc",
		`{"name":"Updated Name"}`, "")
	testutil.AssertStatus(t, resp, http.StatusOK)
}

func TestDeleteProduct(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Delete("/api/_/products/:product_id", DeleteProduct)

	resp := testutil.DoRequest(t, app, http.MethodDelete, "/api/_/products/fv6c9s9cqzf36sc", "", "")
	testutil.AssertStatus(t, resp, http.StatusOK)
}

func TestUpdateProductActive(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Patch("/api/_/products/:product_id/active", UpdateProductActive)

	resp := testutil.DoRequest(t, app, http.MethodPatch, "/api/_/products/fv6c9s9cqzf36sc/active", "", "")
	testutil.AssertStatus(t, resp, http.StatusOK)
}

func TestProductImages(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/_/products/:product_id/image", ProductImages)

	tests := []struct {
		name       string
		productID  string
		wantStatus int
	}{
		{"product with images", "fv6c9s9cqzf36sc", http.StatusOK},
		{"product without images", "7mweb67t8xv9pzx", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodGet, "/api/_/products/"+tt.productID+"/image", "", "")
			testutil.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestAddProductImage(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Post("/api/_/products/:product_id/image", AddProductImage)

	productID := "fv6c9s9cqzf36sc"

	t.Run("valid png upload", func(t *testing.T) {
		body, contentType := createTestImage(t, "image/png", "test.png")
		req := httptest.NewRequest(http.MethodPost, "/api/_/products/"+productID+"/image", body)
		req.Header.Set("Content-Type", contentType)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		testutil.AssertStatus(t, resp, http.StatusOK)

		entries, err := os.ReadDir("./lc_uploads")
		if err != nil {
			t.Fatalf("read lc_uploads: %v", err)
		}
		var haveOrig, haveSm, haveMd bool
		for _, e := range entries {
			name := e.Name()
			ext := filepath.Ext(name)
			if ext == ".png" {
				switch {
				case len(name) > 7 && name[len(name)-7:] == "_sm.png":
					haveSm = true
				case len(name) > 7 && name[len(name)-7:] == "_md.png":
					haveMd = true
				default:
					haveOrig = true
				}
			}
		}
		if !haveOrig || !haveSm || !haveMd {
			t.Fatal("expected orig + sm + md images")
		}
	})

	t.Run("bad mime type", func(t *testing.T) {
		body, contentType := createTestImageBadMIME(t)
		req := httptest.NewRequest(http.MethodPost, "/api/_/products/"+productID+"/image", body)
		req.Header.Set("Content-Type", contentType)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		testutil.AssertStatus(t, resp, http.StatusBadRequest)
	})
}

func TestProductDigital(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/_/products/:product_id/digital", ProductDigital)
	app.Post("/api/_/products/:product_id/digital", AddProductDigital)

	tests := []struct {
		name       string
		method     string
		productID  string
		wantStatus []int
	}{
		{"get digital for data product", http.MethodGet, "xrtb1b919t2nuj9", []int{http.StatusOK}},
		{"get digital for file product", http.MethodGet, "fv6c9s9cqzf36sc", []int{http.StatusOK}},
		{"get digital non-existent", http.MethodGet, "nonexistent12345", []int{http.StatusOK, http.StatusNotFound, http.StatusInternalServerError}},
		{"add digital data to data product", http.MethodPost, "xrtb1b919t2nuj9", []int{http.StatusOK}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, tt.method, "/api/_/products/"+tt.productID+"/digital", "", "")
			testutil.AssertStatus(t, resp, tt.wantStatus...)
		})
	}
}

// --- test helpers (DRY) ---

func createTestImage(t *testing.T, mime, filename string) (*bytes.Buffer, string) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 400, 400))
	for y := range 400 {
		for x := range 400 {
			img.Set(x, y, color.RGBA{R: 255, A: 255})
		}
	}

	var imgBuf bytes.Buffer
	if err := png.Encode(&imgBuf, img); err != nil {
		t.Fatal(err)
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="document"; filename="`+filename+`"`)
	hdr.Set("Content-Type", mime)
	fw, err := w.CreatePart(hdr)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(imgBuf.Bytes()); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	return &body, w.FormDataContentType()
}

func createTestImageBadMIME(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="document"; filename="test.gif"`)
	hdr.Set("Content-Type", "image/gif")
	fw, _ := w.CreatePart(hdr)
	_, _ = fw.Write([]byte("GIF89a fake"))
	_ = w.Close()

	return &body, w.FormDataContentType()
}
