package queries

import (
	"strings"
	"testing"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/security"
)

func validProductInput() *models.Product {
	return &models.Product{
		Name:        "T-shirt",
		Brief:       "brief",
		Description: "desc",
		Slug:        "t-shirt",
		Amount:      1000,
		Digital:     models.Digital{Type: "data"},
	}
}

func TestProduct_FullLifecycle(t *testing.T) {
	db, ctx := bootstrap(t)

	p, err := db.AddProduct(ctx, validProductInput())
	if err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	if p.ID == "" || p.Created == 0 {
		t.Fatalf("AddProduct did not populate ID/Created: %+v", p)
	}

	// Private listing should include the new (inactive) product.
	list, err := db.ListProducts(ctx, true, 10, 0, "")
	if err != nil {
		t.Fatalf("ListProducts private: %v", err)
	}
	if list.Total == 0 {
		t.Error("expected at least one product")
	}

	// Activate so public queries can see it.
	if err := db.UpdateActive(ctx, p.ID); err != nil {
		t.Fatalf("UpdateActive: %v", err)
	}

	if !db.IsProduct(ctx, "t-shirt") {
		t.Error("IsProduct must return true for active product")
	}

	if _, err := db.AddDigitalData(ctx, p.ID, "initial-key"); err != nil {
		t.Fatalf("AddDigitalData: %v", err)
	}

	p.Name = "T-shirt v2"
	p.Slug = "t-shirt-v2"
	p.Description = "desc v2"
	if err := db.UpdateProduct(ctx, p); err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}

	if !db.IsProduct(ctx, "t-shirt-v2") {
		t.Error("IsProduct must return true for active product")
	}
	if db.IsProduct(ctx, "missing") {
		t.Error("IsProduct must return false for unknown slug")
	}

	// Public fetch by slug.
	pub, err := db.Product(ctx, false, "t-shirt-v2")
	if err != nil {
		t.Fatalf("Product public: %v", err)
	}
	if pub.Slug != "t-shirt-v2" {
		t.Errorf("unexpected slug: %s", pub.Slug)
	}

	// Private fetch by ID.
	priv, err := db.Product(ctx, true, p.ID)
	if err != nil {
		t.Fatalf("Product private: %v", err)
	}
	if priv.ID != p.ID {
		t.Errorf("unexpected ID: %s", priv.ID)
	}

	// Product not found.
	if _, err := db.Product(ctx, true, "unknown-id"); err == nil {
		t.Error("expected ErrProductNotFound")
	}

	if err := db.DeleteProduct(ctx, p.ID); err != nil {
		t.Fatalf("DeleteProduct: %v", err)
	}
}

func TestProductImages_Lifecycle(t *testing.T) {
	db, ctx := bootstrap(t)
	p, err := db.AddProduct(ctx, validProductInput())
	if err != nil {
		t.Fatalf("AddProduct: %v", err)
	}

	file, err := db.AddImage(ctx, p.ID, "00000000-0000-0000-0000-000000000000", "jpg", "photo.jpg")
	if err != nil {
		t.Fatalf("AddImage: %v", err)
	}

	images, err := db.ProductImages(ctx, p.ID)
	if err != nil {
		t.Fatalf("ProductImages: %v", err)
	}
	if images == nil || len(*images) != 1 {
		t.Fatalf("expected 1 image, got %+v", images)
	}

	// DeleteImage removes the DB row even when physical files are missing;
	// the deletion of /lc_uploads paths swallows os.IsNotExist.
	if err := db.DeleteImage(ctx, p.ID, file.ID); err != nil {
		t.Fatalf("DeleteImage: %v", err)
	}
	images, _ = db.ProductImages(ctx, p.ID)
	if images != nil && len(*images) != 0 {
		t.Errorf("expected empty images, got %+v", images)
	}
}

func TestProductDigital_DataLifecycle(t *testing.T) {
	db, ctx := bootstrap(t)

	p, _ := db.AddProduct(ctx, validProductInput())

	dd, err := db.AddDigitalData(ctx, p.ID, "serial-key")
	if err != nil {
		t.Fatalf("AddDigitalData: %v", err)
	}

	digital, err := db.ProductDigital(ctx, p.ID)
	if err != nil {
		t.Fatalf("ProductDigital: %v", err)
	}
	if len(digital.Data) == 0 {
		t.Fatalf("expected data row, got %+v", digital)
	}

	dd.Content = "updated-key"
	if err := db.UpdateDigital(ctx, dd); err != nil {
		t.Fatalf("UpdateDigital: %v", err)
	}
	got, _ := db.ProductDigital(ctx, p.ID)
	found := false
	for _, d := range got.Data {
		if d.Content == "updated-key" {
			found = true
		}
	}
	if !found {
		t.Error("UpdateDigital did not persist content")
	}

	// Delete via "data" branch — no filesystem side effect.
	if err := db.DeleteDigital(ctx, p.ID, dd.ID); err != nil {
		t.Fatalf("DeleteDigital (data): %v", err)
	}
}

func TestProductDigital_AddDigitalFile(t *testing.T) {
	db, ctx := bootstrap(t)
	p, _ := db.AddProduct(ctx, validProductInput())

	f, err := db.AddDigitalFile(ctx, p.ID, "uuid-file", "zip", "bundle.zip")
	if err != nil {
		t.Fatalf("AddDigitalFile: %v", err)
	}
	if f.ID == "" {
		t.Error("AddDigitalFile did not return ID")
	}

	got, err := db.ProductDigital(ctx, p.ID)
	if err != nil {
		t.Fatalf("ProductDigital: %v", err)
	}
	if len(got.Files) != 1 {
		t.Fatalf("expected 1 file, got %+v", got.Files)
	}
}

func TestListProducts_WithIDFilter(t *testing.T) {
	db, ctx := bootstrap(t)

	p1, _ := db.AddProduct(ctx, validProductInput())
	p2Input := validProductInput()
	p2Input.Slug = "mug"
	p2Input.Name = "Mug"
	p2, _ := db.AddProduct(ctx, p2Input)

	if err := db.UpdateActive(ctx, p1.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateActive(ctx, p2.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.AddDigitalData(ctx, p1.ID, "k1"); err != nil {
		t.Fatal(err)
	}

	// NB: idList filtering only works in public mode — in private mode the
	// query builder appends "AND product.id IN (...)" with no preceding
	// WHERE, which the SQLite driver rejects. This test pins the working
	// path (public + idList) to catch regressions.
	filtered, err := db.ListProducts(ctx, false, 0, 0, "", models.CartProduct{ProductID: p1.ID})
	if err != nil {
		t.Fatalf("ListProducts idList: %v", err)
	}
	if len(filtered.Products) != 1 || filtered.Products[0].ID != p1.ID {
		t.Errorf("idList filter failed: %+v", filtered.Products)
	}
}

func TestListProducts_PublicShowsActiveWithoutDigitalContent(t *testing.T) {
	db, ctx := bootstrap(t)

	p, err := db.AddProduct(ctx, validProductInput())
	if err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	if err := db.UpdateActive(ctx, p.ID); err != nil {
		t.Fatalf("UpdateActive: %v", err)
	}

	pub, err := db.ListProducts(ctx, false, 10, 0, "")
	if err != nil {
		t.Fatalf("ListProducts public: %v", err)
	}

	found := false
	for _, product := range pub.Products {
		if product.ID == p.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected active product without digital content in public list: %+v", pub.Products)
	}

	got, err := db.Product(ctx, false, p.Slug)
	if err != nil {
		t.Fatalf("Product public by slug: %v", err)
	}
	if got.ID != p.ID {
		t.Fatalf("unexpected product: %+v", got)
	}
}

func TestListProducts_PublicShowsAPIProduct(t *testing.T) {
	db, ctx := bootstrap(t)

	in := validProductInput()
	in.Slug = "api-product"
	in.Name = "API Product"
	in.Digital = models.Digital{Type: "api"}

	p, err := db.AddProduct(ctx, in)
	if err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	if err := db.UpdateActive(ctx, p.ID); err != nil {
		t.Fatalf("UpdateActive: %v", err)
	}

	pub, err := db.ListProducts(ctx, false, 10, 0, "")
	if err != nil {
		t.Fatalf("ListProducts public: %v", err)
	}

	for _, product := range pub.Products {
		if product.ID == p.ID {
			return
		}
	}
	t.Fatalf("expected active api product in public list: %+v", pub.Products)
}

func TestListProducts_PublicWithCartID(t *testing.T) {
	db, ctx := bootstrap(t)

	p, _ := db.AddProduct(ctx, validProductInput())
	if err := db.UpdateActive(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.AddDigitalData(ctx, p.ID, "public-key"); err != nil {
		t.Fatal(err)
	}

	pub, err := db.ListProducts(ctx, false, 10, 0, "cart-stub-123")
	if err != nil {
		t.Fatalf("ListProducts public: %v", err)
	}
	if pub == nil {
		t.Fatal("nil public listing")
	}
}

func TestBuildCartItems(t *testing.T) {
	t.Parallel()
	products := &models.Products{
		Products: []models.Product{
			{Core: models.Core{ID: "p1"}, Name: "A", Slug: "a", Amount: 100,
				Images: []models.File{{ID: "i1", Name: "img", Ext: "jpg"}}},
			{Core: models.Core{ID: "p2"}, Name: "B", Slug: "b", Amount: 200},
		},
	}
	cart := &models.Cart{Cart: []models.CartProduct{
		{ProductID: "p1", Quantity: 2},
		{ProductID: "p-missing", Quantity: 1},
	}}

	items := BuildCartItems(cart, products)
	if len(items) != 1 {
		t.Fatalf("expected 1 item (missing product skipped), got %d: %+v", len(items), items)
	}
	if items[0]["id"].(string) != "p1" {
		t.Errorf("unexpected item: %+v", items[0])
	}
	if _, ok := items[0]["image"]; !ok {
		t.Error("image field missing")
	}

	if BuildCartItems(&models.Cart{}, products) != nil {
		t.Error("empty cart must return nil")
	}
}

func TestPaymentList(t *testing.T) {
	db, ctx := bootstrap(t)

	list, err := db.PaymentList(ctx)
	if err != nil {
		t.Fatalf("PaymentList: %v", err)
	}
	if !list["dummy"] {
		t.Error("dummy should be active by default")
	}
	for _, k := range []string{"stripe", "paypal", "coinbase"} {
		if _, ok := list[k]; !ok {
			t.Errorf("missing key %s in PaymentList: %+v", k, list)
		}
	}
}

func TestAddProduct_PersistsMetadataAndAttributes(t *testing.T) {
	db, ctx := bootstrap(t)

	in := validProductInput()
	in.Metadata = []models.Metadata{{Key: "color", Value: "red"}}
	in.Attributes = []string{"size-M", "size-L"}
	p, err := db.AddProduct(ctx, in)
	if err != nil {
		t.Fatalf("AddProduct: %v", err)
	}

	if err := db.UpdateActive(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.AddDigitalData(ctx, p.ID, "k"); err != nil {
		t.Fatal(err)
	}

	priv, err := db.Product(ctx, true, p.ID)
	if err != nil {
		t.Fatalf("Product: %v", err)
	}
	if len(priv.Metadata) != 1 || priv.Metadata[0].Key != "color" {
		t.Errorf("metadata not persisted: %+v", priv.Metadata)
	}
	if !strings.Contains(strings.Join(priv.Attributes, ","), "size-M") {
		t.Errorf("attributes not persisted: %+v", priv.Attributes)
	}
}

func TestAddProductWithVariants(t *testing.T) {
	db, ctx := bootstrap(t)

	product := &models.Product{
		Core:        models.Core{ID: security.RandomString()},
		Name:        "Test T-Shirt",
		Description: "A test shirt",
		Slug:        "test-tshirt",
		Amount:      2500,
		HasVariants: true,
		Digital:     models.Digital{Type: "file"},
		Options: []models.ProductOption{
			{
				ID:       security.RandomString(),
				Name:     "Size",
				Position: 0,
				Values: []models.ProductOptionValue{
					{ID: security.RandomString(), Value: "Small", Position: 0},
					{ID: security.RandomString(), Value: "Medium", Position: 1},
				},
			},
		},
		Variants: []models.ProductVariant{
			{
				ID:             security.RandomString(),
				SKU:            "TEST-SMALL",
				OptionValues:   map[string]string{"Size": "Small"},
				PriceSurcharge: 0,
				Quantity:       10,
				Active:         true,
			},
			{
				ID:             security.RandomString(),
				SKU:            "TEST-MEDIUM",
				OptionValues:   map[string]string{"Size": "Medium"},
				PriceSurcharge: 500,
				Quantity:       5,
				Active:         true,
			},
		},
	}

	result, err := db.AddProductWithVariants(ctx, product)
	if err != nil {
		t.Fatalf("AddProductWithVariants() error = %v", err)
	}

	if result.ID != product.ID {
		t.Errorf("Expected product ID %s, got %s", product.ID, result.ID)
	}

	// Verify options were created
	var optionCount int
	err = db.ProductQueries.QueryRowContext(ctx, "SELECT COUNT(*) FROM product_option WHERE product_id = ?", product.ID).Scan(&optionCount)
	if err != nil {
		t.Fatalf("Failed to count options: %v", err)
	}
	if optionCount != 1 {
		t.Errorf("Expected 1 option, got %d", optionCount)
	}

	// Verify option values were created
	var valueCount int
	err = db.ProductQueries.QueryRowContext(ctx, "SELECT COUNT(*) FROM product_option_value WHERE option_id = ?", product.Options[0].ID).Scan(&valueCount)
	if err != nil {
		t.Fatalf("Failed to count option values: %v", err)
	}
	if valueCount != 2 {
		t.Errorf("Expected 2 option values, got %d", valueCount)
	}

	// Verify variants were created
	var variantCount int
	err = db.ProductQueries.QueryRowContext(ctx, "SELECT COUNT(*) FROM product_variant WHERE product_id = ?", product.ID).Scan(&variantCount)
	if err != nil {
		t.Fatalf("Failed to count variants: %v", err)
	}
	if variantCount != 2 {
		t.Errorf("Expected 2 variants, got %d", variantCount)
	}
}

func TestGetProductWithVariants(t *testing.T) {
	db, ctx := bootstrap(t)

	// First create a product with variants
	product := &models.Product{
		Core:        models.Core{ID: security.RandomString()},
		Name:        "Test Product",
		Description: "Test description",
		Slug:        "test-product",
		Amount:      1000,
		HasVariants: true,
		Digital:     models.Digital{Type: "file"},
		Options: []models.ProductOption{
			{
				ID:       security.RandomString(),
				Name:     "Color",
				Position: 0,
				Values: []models.ProductOptionValue{
					{ID: security.RandomString(), Value: "Red", Position: 0},
				},
			},
		},
		Variants: []models.ProductVariant{
			{
				ID:             security.RandomString(),
				SKU:            "TEST-RED",
				OptionValues:   map[string]string{"Color": "Red"},
				PriceSurcharge: 100,
				Quantity:       20,
				Active:         true,
			},
		},
	}

	_, err := db.AddProductWithVariants(ctx, product)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Now retrieve it
	retrieved, err := db.GetProductWithVariants(ctx, product.ID)
	if err != nil {
		t.Fatalf("GetProductWithVariants() error = %v", err)
	}

	if retrieved.ID != product.ID {
		t.Errorf("Expected ID %s, got %s", product.ID, retrieved.ID)
	}

	if len(retrieved.Options) != 1 {
		t.Errorf("Expected 1 option, got %d", len(retrieved.Options))
	}

	if len(retrieved.Variants) != 1 {
		t.Errorf("Expected 1 variant, got %d", len(retrieved.Variants))
	}

	if retrieved.Variants[0].Quantity != 20 {
		t.Errorf("Expected variant quantity 20, got %d", retrieved.Variants[0].Quantity)
	}
}

func TestGenerateUniqueSlug(t *testing.T) {
	db, ctx := bootstrap(t)

	// Create a product with slug "test-product"
	_, err := db.ProductQueries.ExecContext(ctx, `
		INSERT INTO product (id, name, slug, desc, amount, digital, active, deleted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, security.RandomString(), "Test", "test-product", "desc", 1000, "file", true, false)
	if err != nil {
		t.Fatalf("Failed to create test product: %v", err)
	}

	tests := []struct {
		name      string
		input     string
		excludeID string
		want      string
	}{
		{
			name:  "new unique slug",
			input: "New Product",
			want:  "new-product",
		},
		{
			name:  "duplicate slug",
			input: "Test Product",
			want:  "test-product-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := db.GenerateUniqueSlug(ctx, tt.input, tt.excludeID)
			if err != nil {
				t.Errorf("GenerateUniqueSlug() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("GenerateUniqueSlug() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListProducts_PaginationConsistency(t *testing.T) {
	db, ctx := bootstrap(t)

	// Create 15 products to test pagination
	productIDs := make([]string, 15)
	for i := 0; i < 15; i++ {
		input := validProductInput()
		input.Name = strings.Repeat("Product ", i+1)
		input.Slug = strings.ToLower(strings.ReplaceAll(input.Name, " ", "-"))

		p, err := db.AddProduct(ctx, input)
		if err != nil {
			t.Fatalf("AddProduct %d: %v", i, err)
		}
		productIDs[i] = p.ID

		// Activate for private listing
		if err := db.UpdateActive(ctx, p.ID); err != nil {
			t.Fatalf("UpdateActive %d: %v", i, err)
		}
	}

	pageSize := 5

	// Test: Multiple calls to same page should return identical results
	t.Run("ConsistentResults", func(t *testing.T) {
		page1_call1, err := db.ListProducts(ctx, true, pageSize, 0, "")
		if err != nil {
			t.Fatalf("ListProducts call 1: %v", err)
		}

		page1_call2, err := db.ListProducts(ctx, true, pageSize, 0, "")
		if err != nil {
			t.Fatalf("ListProducts call 2: %v", err)
		}

		if len(page1_call1.Products) != len(page1_call2.Products) {
			t.Errorf("Page 1 length mismatch: call1=%d, call2=%d",
				len(page1_call1.Products), len(page1_call2.Products))
		}

		for i := range page1_call1.Products {
			if page1_call1.Products[i].ID != page1_call2.Products[i].ID {
				t.Errorf("Page 1 position %d mismatch: call1=%s, call2=%s",
					i, page1_call1.Products[i].ID, page1_call2.Products[i].ID)
			}
		}
	})

	// Test: Products are ordered by created DESC, id ASC
	t.Run("OrderedByCreatedDesc", func(t *testing.T) {
		allProducts, err := db.ListProducts(ctx, true, 100, 0, "")
		if err != nil {
			t.Fatalf("ListProducts: %v", err)
		}

		for i := 1; i < len(allProducts.Products); i++ {
			prev := allProducts.Products[i-1]
			curr := allProducts.Products[i]

			// Should be ordered by created DESC
			if prev.Created < curr.Created {
				t.Errorf("Products not ordered by created DESC at position %d: prev.created=%d > curr.created=%d",
					i, prev.Created, curr.Created)
			}

			// If same created timestamp, should be ordered by ID ASC
			if prev.Created == curr.Created && prev.ID > curr.ID {
				t.Errorf("Products with same timestamp not ordered by ID ASC at position %d: prev.id=%s > curr.id=%s",
					i, prev.ID, curr.ID)
			}
		}
	})

	// Test: No duplicate products across pages
	t.Run("NoDuplicatesAcrossPages", func(t *testing.T) {
		seenIDs := make(map[string]bool)
		totalPages := 3

		for page := 0; page < totalPages; page++ {
			offset := page * pageSize
			products, err := db.ListProducts(ctx, true, pageSize, offset, "")
			if err != nil {
				t.Fatalf("ListProducts page %d: %v", page, err)
			}

			for _, p := range products.Products {
				if seenIDs[p.ID] {
					t.Errorf("Duplicate product ID %s found on page %d", p.ID, page)
				}
				seenIDs[p.ID] = true
			}
		}
	})

	// Test: Same page returns same products across multiple calls
	t.Run("Page5Consistency", func(t *testing.T) {
		offset := 4 * pageSize // Page 5 (0-indexed: page 0,1,2,3,4)

		call1, err := db.ListProducts(ctx, true, pageSize, offset, "")
		if err != nil {
			t.Fatalf("ListProducts page 5 call 1: %v", err)
		}

		call2, err := db.ListProducts(ctx, true, pageSize, offset, "")
		if err != nil {
			t.Fatalf("ListProducts page 5 call 2: %v", err)
		}

		call3, err := db.ListProducts(ctx, true, pageSize, offset, "")
		if err != nil {
			t.Fatalf("ListProducts page 5 call 3: %v", err)
		}

		if len(call1.Products) != len(call2.Products) || len(call1.Products) != len(call3.Products) {
			t.Errorf("Page 5 length inconsistent: call1=%d, call2=%d, call3=%d",
				len(call1.Products), len(call2.Products), len(call3.Products))
		}

		for i := range call1.Products {
			if call1.Products[i].ID != call2.Products[i].ID || call1.Products[i].ID != call3.Products[i].ID {
				t.Errorf("Page 5 position %d inconsistent across calls: call1=%s, call2=%s, call3=%s",
					i, call1.Products[i].ID, call2.Products[i].ID, call3.Products[i].ID)
			}
		}
	})
}
