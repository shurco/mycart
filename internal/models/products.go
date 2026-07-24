package models

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

// Products is ...
type Products struct {
	Total    int       `json:"total"`
	Currency string    `json:"currency"`
	Products []Product `json:"products"`
}

// Product is ...
type Product struct {
	Core
	Name        string           `json:"name"`
	Brief       string           `json:"brief,omitempty"`
	Description string           `json:"description,omitempty"`
	Images      []File           `json:"images,omitempty"`
	Slug        string           `json:"slug"`
	Amount      int              `json:"amount"`
	Quantity    int              `json:"quantity"`           // NEW
	SKU         string           `json:"sku,omitempty"`      // NEW
	HasVariants bool             `json:"has_variants"`       // NEW
	Options     []ProductOption  `json:"options,omitempty"`  // NEW
	Variants    []ProductVariant `json:"variants,omitempty"` // NEW
	Metadata    []Metadata       `json:"metadata,omitempty"`
	Attributes  []string         `json:"attributes,omitempty"`
	Digital     Digital          `json:"digital,omitempty"`
	Active      bool             `json:"active"`
	Seo         *Seo             `json:"seo,omitempty"`
}

// Validate is ...
func (v Product) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.ID, validation.Length(15, 15)),
		validation.Field(&v.Name, validation.Length(3, 100)),
		validation.Field(&v.Description, validation.NotNil),
		validation.Field(&v.Images),
		validation.Field(&v.Slug, validation.Required, validation.Length(3, 100)),
		validation.Field(&v.Amount, validation.Min(0)),
		validation.Field(&v.Quantity, validation.Min(0)),          // NEW
		validation.Field(&v.SKU, validation.Length(0, 50)),        // NEW
		validation.Field(&v.Options, validation.Length(0, 3)),     // NEW - Max 3 options
		validation.Field(&v.Variants, validation.Length(0, 100)),  // NEW - Max 100 variants
		validation.Field(&v.Metadata),
		validation.Field(&v.Attributes, validation.Each(validation.Length(3, 254))),
		validation.Field(&v.Digital),
		validation.Field(&v.Seo),
	)
}

// Metadata is ...
type Metadata struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Validate is ...
func (v Metadata) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Key, validation.Required, validation.Length(1, 20)),
		validation.Field(&v.Value, validation.Required, validation.Min(0)),
	)
}

// Digital is ...
type Digital struct {
	Type   string `json:"type"`
	Filled bool   `json:"filled,omitempty"`
	Files  []File `json:"files,omitempty"`
	Data   []Data `json:"data,omitempty"`
}

// Validate is ...
func (v Digital) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Type, validation.Required, validation.In("file", "data", "api")),
		validation.Field(&v.Files),
		validation.Field(&v.Data, validation.Each(validation.Length(1, 254))),
	)
}

// File is ...
type File struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Ext      string `json:"ext"`
	OrigName string `json:"orig_name,omitempty"`
}

// Validate is ...
func (v File) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.ID, validation.Length(15, 15)),
		validation.Field(&v.Name, is.UUIDv4),
		// validation.Field(&v.Ext, validation.In("jpeg", "png")),
	)
}

// Data is ...
type Data struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	CartID  string `json:"cart_id"`
}

// Validate is ...
func (v Data) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.ID, validation.Length(15, 15)),
		validation.Field(&v.Content, validation.Length(1, 254)),
		// validation.Field(&v.Ext, validation.In("jpeg", "png")),
	)
}

// ProductOption represents an option type (Size, Color, etc.)
type ProductOption struct {
	ID        string              `json:"id"`
	ProductID string              `json:"product_id"`
	Name      string              `json:"name"`
	Values    []ProductOptionValue `json:"values"`
	Position  int                 `json:"position"`
	Created   int64               `json:"created"`
}

// Validate validates ProductOption
func (v ProductOption) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Name, validation.Required, validation.Length(1, 50)),
		validation.Field(&v.Values, validation.Required, validation.Length(1, 10)), // Max 10 values
	)
}

// ProductOptionValue represents a specific value (Medium, Black, etc.)
type ProductOptionValue struct {
	ID       string `json:"id"`
	OptionID string `json:"option_id,omitempty"`
	Value    string `json:"value"`
	Position int    `json:"position"`
}

// Validate validates ProductOptionValue
func (v ProductOptionValue) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Value, validation.Required, validation.Length(1, 100)),
	)
}

// ProductVariant represents a specific combination of options
type ProductVariant struct {
	ID             string            `json:"id"`
	ProductID      string            `json:"product_id"`
	SKU            string            `json:"sku,omitempty"`
	OptionValues   map[string]string `json:"option_values"` // {"Size": "Medium", "Color": "Black"}
	PriceSurcharge int               `json:"price_surcharge"` // cents
	Quantity       int               `json:"quantity"`
	Images         []File            `json:"images,omitempty"`
	Active         bool              `json:"active"`
	Created        int64             `json:"created"`
	Updated        int64             `json:"updated,omitempty"`
}

// Validate validates ProductVariant
func (v ProductVariant) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.SKU, validation.Length(0, 50)),
		validation.Field(&v.Quantity, validation.Min(0)),
		validation.Field(&v.OptionValues, validation.Required, validation.Length(1, 3)),
	)
}
