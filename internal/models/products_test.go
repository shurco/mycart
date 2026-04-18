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
		{"missing amount", func(p *Product) { p.Amount = 0 }, true},
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
