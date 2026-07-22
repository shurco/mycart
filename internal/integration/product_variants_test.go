package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/testutil"
	"github.com/shurco/mycart/pkg/csvimport"

	handlers "github.com/shurco/mycart/internal/handlers/private"
)

func TestIntegration_ProductWithVariants_FullLifecycle(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	// Register routes
	app.Post("/api/_/products", handlers.AddProduct)
	app.Get("/api/_/products/:product_id", handlers.Product)

	// Step 1: Create product with variants
	productID := createTestProductWithVariants(t, app, cookie)

	// Step 2: Retrieve product with variants
	getResp := testutil.DoRequest(t, app, http.MethodGet, "/api/_/products/"+productID, "", cookie)

	var getResult struct {
		Result models.Product `json:"result"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&getResult); err != nil {
		t.Fatalf("Failed to parse get response: %v", err)
	}
	getResp.Body.Close()

	testutil.AssertStatus(t, getResp, http.StatusOK)

	// Step 3: Verify product data
	verifyProductVariantData(t, getResult.Result)
}

// createTestProductWithVariants creates a test product with variants and returns its ID
func createTestProductWithVariants(t *testing.T, app *fiber.App, cookie string) string {
	createPayload := `{
		"name": "Integration Test T-Shirt",
		"slug": "integration-test-tshirt",
		"description": "Full lifecycle test",
		"amount": 2500,
		"has_variants": true,
		"digital": {"type": "file"},
		"options": [
			{
				"name": "Size",
				"values": [
					{"value": "Small"},
					{"value": "Medium"},
					{"value": "Large"}
				]
			},
			{
				"name": "Color",
				"values": [
					{"value": "Red"},
					{"value": "Blue"}
				]
			}
		],
		"variants": [
			{"sku": "TS-S-R", "option_values": {"Size": "Small", "Color": "Red"}, "price_surcharge": 0, "quantity": 10},
			{"sku": "TS-S-B", "option_values": {"Size": "Small", "Color": "Blue"}, "price_surcharge": 0, "quantity": 10},
			{"sku": "TS-M-R", "option_values": {"Size": "Medium", "Color": "Red"}, "price_surcharge": 200, "quantity": 15},
			{"sku": "TS-M-B", "option_values": {"Size": "Medium", "Color": "Blue"}, "price_surcharge": 200, "quantity": 15},
			{"sku": "TS-L-R", "option_values": {"Size": "Large", "Color": "Red"}, "price_surcharge": 500, "quantity": 8},
			{"sku": "TS-L-B", "option_values": {"Size": "Large", "Color": "Blue"}, "price_surcharge": 500, "quantity": 8}
		]
	}`

	createResp := testutil.DoRequest(t, app, http.MethodPost, "/api/_/products", createPayload, cookie)

	var createResult struct {
		Result models.Product `json:"result"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&createResult); err != nil {
		t.Fatalf("Failed to parse create response: %v", err)
	}
	createResp.Body.Close()

	testutil.AssertStatus(t, createResp, http.StatusOK)

	productID := createResult.Result.ID
	if productID == "" {
		t.Fatal("Product ID is empty")
	}

	return productID
}

// verifyProductVariantData verifies product variant structure and data
func verifyProductVariantData(t *testing.T, product models.Product) {
	if product.Name != "Integration Test T-Shirt" {
		t.Errorf("Expected name 'Integration Test T-Shirt', got %s", product.Name)
	}

	if !product.HasVariants {
		t.Error("Expected has_variants to be true")
	}

	if len(product.Options) != 2 {
		t.Errorf("Expected 2 options, got %d", len(product.Options))
	}

	if len(product.Variants) != 6 {
		t.Errorf("Expected 6 variants, got %d", len(product.Variants))
	}

	// Verify option structure
	sizeOption := product.Options[0]
	if sizeOption.Name != "Size" {
		t.Errorf("Expected first option to be 'Size', got %s", sizeOption.Name)
	}
	if len(sizeOption.Values) != 3 {
		t.Errorf("Expected 3 size values, got %d", len(sizeOption.Values))
	}

	// Verify variant structure
	foundVariant := false
	for _, variant := range product.Variants {
		if variant.SKU == "TS-M-R" {
			foundVariant = true
			if variant.PriceSurcharge != 200 {
				t.Errorf("Expected price surcharge 200, got %d", variant.PriceSurcharge)
			}
			if variant.Quantity != 15 {
				t.Errorf("Expected quantity 15, got %d", variant.Quantity)
			}
			if variant.OptionValues["Size"] != "Medium" {
				t.Errorf("Expected Size=Medium, got %s", variant.OptionValues["Size"])
			}
			if variant.OptionValues["Color"] != "Red" {
				t.Errorf("Expected Color=Red, got %s", variant.OptionValues["Color"])
			}
		}
	}

	if !foundVariant {
		t.Error("Failed to find expected variant TS-M-R")
	}
}

func TestIntegration_CSV_ImportExport_Roundtrip(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	// Register routes
	app.Post("/api/_/products/import/preview", handlers.ImportPreview)
	app.Post("/api/_/products/import", handlers.ImportProducts)
	app.Get("/api/_/products/export", handlers.ExportProducts)

	// Step 1: Import CSV with variants
	csvContent := `name,slug,description,amount,digital,option1_name,option1_values,option2_name,option2_values,variant_prices,variant_quantities,variant_skus
Test Product,test-product,Test description,1000,file,Size,S;M;L,Color,Red;Blue,0;0;100;100;200;200,10;10;15;15;20;20,TP-S-R;TP-S-B;TP-M-R;TP-M-B;TP-L-R;TP-L-B
Simple Product,simple-product,No variants,500,file,,,,,,,`

	// Create multipart form with CSV file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "products.csv")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write([]byte(csvContent))
	writer.Close()

	// Step 2: Preview import
	req := httptest.NewRequest(http.MethodPost, "/api/_/products/import/preview", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Cookie", cookie)

	resp, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("Preview request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, respBody)
	}

	var previewResult struct {
		Result csvimport.ImportResult `json:"result"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &previewResult); err != nil {
		t.Fatalf("Failed to parse preview response: %v", err)
	}

	// Verify preview
	if previewResult.Result.TotalRows != 2 {
		t.Errorf("Expected 2 total rows, got %d", previewResult.Result.TotalRows)
	}

	if previewResult.Result.ToAdd != 2 {
		t.Errorf("Expected 2 rows to add, got %d", previewResult.Result.ToAdd)
	}

	if len(previewResult.Result.Errors) > 0 {
		t.Errorf("Expected no errors, got %d: %+v", len(previewResult.Result.Errors), previewResult.Result.Errors)
	}

	// Step 3: Execute import
	body2 := &bytes.Buffer{}
	writer2 := multipart.NewWriter(body2)
	part2, _ := writer2.CreateFormFile("file", "products.csv")
	part2.Write([]byte(csvContent))
	writer2.Close()

	req2 := httptest.NewRequest(http.MethodPost, "/api/_/products/import", body2)
	req2.Header.Set("Content-Type", writer2.FormDataContentType())
	req2.Header.Set("Cookie", cookie)

	resp2, err := app.Test(req2, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("Import request failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		respBody2, _ := io.ReadAll(resp2.Body)
		t.Fatalf("Expected status 200, got %d. Body: %s", resp2.StatusCode, respBody2)
	}

	var importResult struct {
		Result csvimport.ImportResult `json:"result"`
	}
	respBody2, _ := io.ReadAll(resp2.Body)
	if err := json.Unmarshal(respBody2, &importResult); err != nil {
		t.Fatalf("Failed to parse import response: %v", err)
	}

	// Verify import
	if importResult.Result.Imported != 2 {
		t.Errorf("Expected 2 imported, got %d", importResult.Result.Imported)
	}

	// Step 4: Export and verify
	exportReq := httptest.NewRequest(http.MethodGet, "/api/_/products/export", nil)
	exportReq.Header.Set("Cookie", cookie)

	exportResp, err := app.Test(exportReq, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("Export request failed: %v", err)
	}
	defer exportResp.Body.Close()

	if exportResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", exportResp.StatusCode)
	}

	exportBody, _ := io.ReadAll(exportResp.Body)
	exportCSV := string(exportBody)

	// Verify CSV contains imported products
	if !strings.Contains(exportCSV, "test-product") {
		t.Error("Export CSV should contain 'test-product'")
	}

	if !strings.Contains(exportCSV, "simple-product") {
		t.Error("Export CSV should contain 'simple-product'")
	}

	// Verify CSV header
	if !strings.HasPrefix(exportCSV, "name,slug,brief,description,images,attributes,amount") {
		t.Error("Export CSV should have correct header")
	}
}

func TestIntegration_SlugGeneration_UniquenessDuringImport(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	// Register routes
	app.Post("/api/_/products", handlers.AddProduct)
	app.Post("/api/_/products/slug/generate", handlers.GenerateSlug)

	// Step 1: Create product with specific slug
	createPayload := `{
		"name": "Original Product",
		"slug": "test-slug",
		"description": "First product",
		"amount": 1000,
		"digital": {"type": "file"}
	}`

	createResp := testutil.DoRequest(t, app, http.MethodPost, "/api/_/products", createPayload, cookie)
	testutil.AssertStatus(t, createResp, http.StatusOK)

	// Step 2: Generate slug for duplicate name
	slugPayload := `{"name": "Test Slug"}`
	slugResp := testutil.DoRequest(t, app, http.MethodPost, "/api/_/products/slug/generate", slugPayload, cookie)

	var slugResult struct {
		Result struct {
			Slug string `json:"slug"`
		} `json:"result"`
	}
	if err := json.NewDecoder(slugResp.Body).Decode(&slugResult); err != nil {
		t.Fatalf("Failed to parse slug response: %v", err)
	}
	slugResp.Body.Close()

	testutil.AssertStatus(t, slugResp, http.StatusOK)

	// Should generate test-slug-2 since test-slug exists
	if slugResult.Result.Slug != "test-slug-2" {
		t.Errorf("Expected slug 'test-slug-2', got '%s'", slugResult.Result.Slug)
	}
}

func TestIntegration_VariantQuantityTracking(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	// Register routes
	app.Post("/api/_/products", handlers.AddProduct)
	app.Get("/api/_/products/:product_id", handlers.Product)

	// Create product with variants and specific quantities
	createPayload := `{
		"name": "Quantity Test Product",
		"slug": "quantity-test",
		"description": "Testing quantity tracking",
		"amount": 1000,
		"has_variants": true,
		"digital": {"type": "file"},
		"options": [
			{
				"name": "Size",
				"values": [{"value": "Small"}, {"value": "Large"}]
			}
		],
		"variants": [
			{"sku": "QT-S", "option_values": {"Size": "Small"}, "price_surcharge": 0, "quantity": 5},
			{"sku": "QT-L", "option_values": {"Size": "Large"}, "price_surcharge": 0, "quantity": 0}
		]
	}`

	createResp := testutil.DoRequest(t, app, http.MethodPost, "/api/_/products", createPayload, cookie)

	var createResult struct {
		Result models.Product `json:"result"`
	}
	json.NewDecoder(createResp.Body).Decode(&createResult)
	createResp.Body.Close()

	testutil.AssertStatus(t, createResp, http.StatusOK)

	// Retrieve and verify quantities
	getResp := testutil.DoRequest(t, app, http.MethodGet, "/api/_/products/"+createResult.Result.ID, "", cookie)

	var getResult struct {
		Result models.Product `json:"result"`
	}
	json.NewDecoder(getResp.Body).Decode(&getResult)
	getResp.Body.Close()

	testutil.AssertStatus(t, getResp, http.StatusOK)

	// Verify variant quantities
	for _, variant := range getResult.Result.Variants {
		if variant.SKU == "QT-S" && variant.Quantity != 5 {
			t.Errorf("Expected QT-S quantity 5, got %d", variant.Quantity)
		}
		if variant.SKU == "QT-L" && variant.Quantity != 0 {
			t.Errorf("Expected QT-L quantity 0, got %d", variant.Quantity)
		}
	}
}
