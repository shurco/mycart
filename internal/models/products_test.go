package models

import "testing"

func validProduct() Product {
	return Product{
		Core:        Core{ID: "prod12345678901"},
		Name:        "T-shirt",
		Description: "Cotton T-shirt",
		Slug:        "t-shirt",
		Amount:      100,
		Digital:     Digital{Type: "file"},
	}
}

func TestProduct_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(p *Product)
		wantErr bool
	}{
		{"baseline", func(p *Product) {}, false},
		{"bad ID length", func(p *Product) { p.ID = "short" }, true},
		{"missing slug", func(p *Product) { p.Slug = "" }, true},
		{"zero amount (free product)", func(p *Product) { p.Amount = 0 }, false},
		{"negative amount", func(p *Product) { p.Amount = -1 }, true},
		{"short name", func(p *Product) { p.Name = "x" }, true},
		{"attribute too short", func(p *Product) { p.Attributes = []string{"a"} }, true},
		{"metadata required", func(p *Product) {
			p.Metadata = []Metadata{{Key: "", Value: "v"}}
		}, true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := validProduct()
			tc.mutate(&p)
			err := p.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v wantErr=%v (%+v)", err, tc.wantErr, p)
			}
		})
	}
}

// Metadata.Validate applies validation.Min (an int rule) to Value (a
// string). The current behaviour is therefore "always errors with
// 'cannot convert string to int64'" — almost certainly a typo, but we
// lock the behaviour here so callers see a consistent failure mode until
// the model is intentionally fixed.
func TestMetadata_Validate_CurrentQuirk(t *testing.T) {
	t.Parallel()

	err := (Metadata{Key: "k", Value: "v"}).Validate()
	if err == nil {
		t.Fatal("Metadata.Validate stopped returning the Min-on-string quirk: please update this test")
	}
	// Missing key still fails first.
	if err := (Metadata{}).Validate(); err == nil {
		t.Error("missing key must fail")
	}
}

func TestDigital_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      Digital
		wantErr bool
	}{
		{"type file", Digital{Type: "file"}, false},
		{"type data", Digital{Type: "data"}, false},
		{"type api", Digital{Type: "api"}, false},
		{"empty type", Digital{}, true},
		{"bad type", Digital{Type: "bogus"}, true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.in.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestFile_Validate(t *testing.T) {
	t.Parallel()

	ok := File{ID: "file12345678901", Name: "00000000-0000-4000-8000-000000000000"}
	if err := ok.Validate(); err != nil {
		t.Errorf("valid file rejected: %v", err)
	}
	bad := File{ID: "bad", Name: "not-a-uuid"}
	if err := bad.Validate(); err == nil {
		t.Error("invalid file accepted")
	}
}

func TestData_Validate(t *testing.T) {
	t.Parallel()

	if err := (Data{ID: "data12345678901", Content: "hello"}).Validate(); err != nil {
		t.Errorf("valid data rejected: %v", err)
	}
	if err := (Data{ID: "short", Content: "hello"}).Validate(); err == nil {
		t.Error("bad ID length must fail")
	}
}

func TestProductOptionValidation(t *testing.T) {
	tests := []struct {
		name    string
		option  ProductOption
		wantErr bool
	}{
		{
			name: "valid option",
			option: ProductOption{
				Name: "Size",
				Values: []ProductOptionValue{
					{Value: "Small"},
					{Value: "Medium"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			option: ProductOption{
				Name: "",
				Values: []ProductOptionValue{
					{Value: "Small"},
				},
			},
			wantErr: true,
		},
		{
			name: "too many values",
			option: ProductOption{
				Name: "Size",
				Values: make([]ProductOptionValue, 11), // Max is 10
			},
			wantErr: true,
		},
		{
			name: "no values",
			option: ProductOption{
				Name:   "Size",
				Values: []ProductOptionValue{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.option.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ProductOption.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProductVariantValidation(t *testing.T) {
	tests := []struct {
		name    string
		variant ProductVariant
		wantErr bool
	}{
		{
			name: "valid variant",
			variant: ProductVariant{
				OptionValues:   map[string]string{"Size": "Medium"},
				Quantity:       10,
				PriceSurcharge: 0,
			},
			wantErr: false,
		},
		{
			name: "negative quantity",
			variant: ProductVariant{
				OptionValues:   map[string]string{"Size": "Medium"},
				Quantity:       -1,
				PriceSurcharge: 0,
			},
			wantErr: true,
		},
		{
			name: "too many option values",
			variant: ProductVariant{
				OptionValues: map[string]string{
					"Size":     "Medium",
					"Color":    "Red",
					"Material": "Cotton",
					"Style":    "Casual", // Max is 3
				},
				Quantity: 10,
			},
			wantErr: true,
		},
		{
			name: "no option values",
			variant: ProductVariant{
				OptionValues: map[string]string{},
				Quantity:     10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.variant.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ProductVariant.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProductValidationWithVariants(t *testing.T) {
	tests := []struct {
		name    string
		product Product
		wantErr bool
	}{
		{
			name: "valid product with variants",
			product: Product{
				Core:        Core{ID: "123456789012345"},
				Name:        "T-Shirt",
				Description: "A nice shirt",
				Slug:        "t-shirt",
				Amount:      2500,
				Quantity:    0,
				HasVariants: true,
				Options: []ProductOption{
					{Name: "Size", Values: []ProductOptionValue{{Value: "M"}}},
				},
				Variants: []ProductVariant{
					{OptionValues: map[string]string{"Size": "M"}, Quantity: 10},
				},
				Digital: Digital{Type: "file"},
			},
			wantErr: false,
		},
		{
			name: "too many options",
			product: Product{
				Core:        Core{ID: "123456789012345"},
				Name:        "T-Shirt",
				Description: "A nice shirt",
				Slug:        "t-shirt",
				Amount:      2500,
				HasVariants: true,
				Options: []ProductOption{
					{Name: "Size", Values: []ProductOptionValue{{Value: "M"}}},
					{Name: "Color", Values: []ProductOptionValue{{Value: "Red"}}},
					{Name: "Material", Values: []ProductOptionValue{{Value: "Cotton"}}},
					{Name: "Style", Values: []ProductOptionValue{{Value: "Casual"}}}, // Max is 3
				},
				Digital: Digital{Type: "file"},
			},
			wantErr: true,
		},
		{
			name: "too many variants",
			product: Product{
				Core:        Core{ID: "123456789012345"},
				Name:        "T-Shirt",
				Description: "A nice shirt",
				Slug:        "t-shirt",
				Amount:      2500,
				HasVariants: true,
				Variants:    make([]ProductVariant, 101), // Max is 100
				Digital:     Digital{Type: "file"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.product.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Product.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
