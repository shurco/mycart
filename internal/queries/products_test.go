package queries

import (
	"strings"
	"testing"

	"github.com/shurco/mycart/internal/models"
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

	// Activate + add digital data so public queries can see it.
	if err := db.UpdateActive(ctx, p.ID); err != nil {
		t.Fatalf("UpdateActive: %v", err)
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
		t.Error("IsProduct must return true for active product with digital data")
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
