# Product Variants & Enhanced UX Design Specification

**Date:** 2026-07-21  
**Status:** Approved  
**Authors:** Development Team

## Executive Summary

This specification covers four major enhancements to the mycart e-commerce platform:

1. **Product Variants** - Support for products with multiple options (Size, Color, Material) with up to 3 option types and auto-generated variant combinations
2. **Inventory Management** - Dual-level quantity tracking (product-level for simple products, variant-level for products with variants)
3. **Smart Slug Generation** - Auto-generate URL-friendly slugs from product names with conflict resolution
4. **CSV Import/Export** - Bulk product management with validation preview and flexible import modes

## Requirements Summary

### Product Variants
- Support up to 3 option types per product (e.g., Size, Color, Material)
- Support up to 10 values per option type
- Auto-generate all variant combinations (cartesian product)
- Maximum 100 variants per product
- Each variant has:
  - Optional SKU (unique if provided)
  - Price surcharge (added/subtracted from base product price)
  - Quantity (required, minimum 0)
  - Individual activation status
  - Optional variant-specific images
- Product-level images (shared) + optional variant-level images

### Inventory Management
- Products without variants: track quantity at product level
- Products with variants: track quantity at variant level
- Product-level quantity ignored when `has_variants=true`
- Inventory decrements on successful purchase
- Prevent overselling (reject checkout if stock insufficient)
- Cart warnings when product/variant becomes unavailable or price changes

### Slug Generation
- Auto-generate from product name using `github.com/gosimple/slug`
- Handle duplicates by appending numbers: `yoga-strap`, `yoga-strap-2`, etc.
- Admin can manually override auto-generated slug
- Warn admin when changing slug on existing product (breaks URLs)
- Validate slug format and uniqueness

### CSV Import/Export
- Single CSV row per product
- Variants encoded in delimited format within the row
- Fields: `option1_name`, `option1_values` (semicolon-separated), `option2_name`, etc.
- Image handling via URLs (system downloads on import, exports URLs)
- Import modes:
  - **Upsert**: Update existing (by slug) + add new
  - **Preview first**: Show summary of changes before import
  - **Validation**: Display errors with line numbers
- Mandatory CSV fields: name, slug, amount, digital type
- Export all products to same format

---

## 1. Database Schema

### New Tables

```sql
-- Product options (e.g., "Size", "Color", "Material")
CREATE TABLE product_option (
    id          TEXT PRIMARY KEY NOT NULL,
    product_id  TEXT NOT NULL,
    name        TEXT NOT NULL,  -- "Size", "Color", etc.
    position    INTEGER DEFAULT 0,
    created     TIMESTAMP DEFAULT (datetime('now')),
    FOREIGN KEY (product_id) REFERENCES product(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_option_product_id ON product_option (product_id);

-- Option values (e.g., "Medium", "Large", "Black", "Orange")
CREATE TABLE product_option_value (
    id          TEXT PRIMARY KEY NOT NULL,
    option_id   TEXT NOT NULL,
    value       TEXT NOT NULL,  -- "Medium", "Black", etc.
    position    INTEGER DEFAULT 0,
    FOREIGN KEY (option_id) REFERENCES product_option(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_option_value_option_id ON product_option_value (option_id);

-- Product variants (combinations of option values)
CREATE TABLE product_variant (
    id              TEXT PRIMARY KEY NOT NULL,
    product_id      TEXT NOT NULL,
    sku             TEXT UNIQUE,  -- Optional, but unique if provided
    price_surcharge NUMERIC DEFAULT 0,  -- Added/subtracted from product base price
    quantity        INTEGER DEFAULT 0,
    active          BOOLEAN DEFAULT TRUE,
    created         TIMESTAMP DEFAULT (datetime('now')),
    updated         TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES product(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_variant_product_id ON product_variant (product_id);
CREATE INDEX idx_product_variant_sku ON product_variant (sku);

-- Junction table: links variants to their option value combinations
CREATE TABLE product_variant_option (
    variant_id      TEXT NOT NULL,
    option_value_id TEXT NOT NULL,
    PRIMARY KEY (variant_id, option_value_id),
    FOREIGN KEY (variant_id) REFERENCES product_variant(id) ON DELETE CASCADE,
    FOREIGN KEY (option_value_id) REFERENCES product_option_value(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_variant_option_variant ON product_variant_option (variant_id);
CREATE INDEX idx_product_variant_option_value ON product_variant_option (option_value_id);

-- Variant-specific images
CREATE TABLE product_variant_image (
    id          TEXT PRIMARY KEY NOT NULL,
    variant_id  TEXT NOT NULL,
    name        TEXT NOT NULL,
    ext         TEXT NOT NULL,
    orig_name   TEXT NOT NULL,
    position    INTEGER DEFAULT 0,
    FOREIGN KEY (variant_id) REFERENCES product_variant(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_variant_image_variant_id ON product_variant_image (variant_id);
```

### Modified Tables

```sql
-- Add columns to existing product table
ALTER TABLE product ADD COLUMN quantity INTEGER DEFAULT 0;
ALTER TABLE product ADD COLUMN sku TEXT UNIQUE;
ALTER TABLE product ADD COLUMN has_variants BOOLEAN DEFAULT FALSE;

-- Add position to existing product_image for ordering
ALTER TABLE product_image ADD COLUMN position INTEGER DEFAULT 0;
```

### Data Model Relationships

```
product (1) ──→ (N) product_option
                      └─→ (N) product_option_value
                      
product (1) ──→ (N) product_variant
                      └─→ (N) product_variant_option ──→ product_option_value
                      └─→ (N) product_variant_image
```

### Business Rules

1. Products without variants use `product.quantity` and `product.sku`
2. Products with variants (`has_variants=true`) ignore `product.quantity`, use variant-level quantities
3. Maximum 3 options per product (enforced in application layer)
4. Maximum 10 values per option (enforced in application layer)
5. Maximum 100 variants per product (enforced in application layer)
6. Variant SKU is optional but must be globally unique if provided
7. Product SKU is optional but must be globally unique if provided
8. Variants must have at least one option value combination
9. All variants must have quantity set (default 0)
10. Soft delete: `product.deleted=true` keeps data for historical orders

---

## 2. Backend Architecture (Go)

### New Models (`internal/models/products.go`)

```go
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
        validation.Field(&v.Quantity, validation.Required, validation.Min(0)),
        validation.Field(&v.OptionValues, validation.Required, validation.Length(1, 3)),
    )
}
```

### Modified Product Model

```go
type Product struct {
    Core
    Name        string          `json:"name"`
    Brief       string          `json:"brief,omitempty"`
    Description string          `json:"description,omitempty"`
    Images      []File          `json:"images,omitempty"`
    Slug        string          `json:"slug"`
    Amount      int             `json:"amount"`
    Quantity    int             `json:"quantity"`           // NEW
    SKU         string          `json:"sku,omitempty"`      // NEW
    HasVariants bool            `json:"has_variants"`       // NEW
    Options     []ProductOption `json:"options,omitempty"`  // NEW
    Variants    []ProductVariant `json:"variants,omitempty"` // NEW
    Metadata    []Metadata      `json:"metadata,omitempty"`
    Attributes  []string        `json:"attributes,omitempty"`
    Digital     Digital         `json:"digital,omitempty"`
    Active      bool            `json:"active"`
    Seo         *Seo            `json:"seo,omitempty"`
}

// Validate validates Product (enhanced)
func (v Product) Validate() error {
    return validation.ValidateStruct(&v,
        validation.Field(&v.ID, validation.Length(15, 15)),
        validation.Field(&v.Name, validation.Required, validation.Length(3, 100)),
        validation.Field(&v.Description, validation.NotNil),
        validation.Field(&v.Images),
        validation.Field(&v.Slug, validation.Required, validation.Length(3, 100)),
        validation.Field(&v.Amount, validation.Min(0)),
        validation.Field(&v.Quantity, validation.Min(0)),
        validation.Field(&v.SKU, validation.Length(0, 50)),
        validation.Field(&v.Options, validation.Length(0, 3)), // Max 3 options
        validation.Field(&v.Variants, validation.Length(0, 100)), // Max 100 variants
        validation.Field(&v.Metadata),
        validation.Field(&v.Attributes, validation.Each(validation.Length(3, 254))),
        validation.Field(&v.Digital),
        validation.Field(&v.Seo),
    )
}
```

### New Queries (`internal/queries/products.go`)

```go
// Variant management
func (q *ProductQueries) AddProductWithVariants(ctx context.Context, product *Product) (*Product, error)
func (q *ProductQueries) UpdateProductVariants(ctx context.Context, productID string, options []ProductOption, variants []ProductVariant) error
func (q *ProductQueries) GetProductWithVariants(ctx context.Context, productID string) (*Product, error)
func (q *ProductQueries) DeleteVariant(ctx context.Context, variantID string) error
func (q *ProductQueries) UpdateVariantQuantity(ctx context.Context, variantID string, quantity int) error
func (q *ProductQueries) GetVariantBySKU(ctx context.Context, sku string) (*ProductVariant, error)
func (q *ProductQueries) GetProductVariants(ctx context.Context, productID string) ([]ProductVariant, error)

// Slug management
func (q *ProductQueries) GenerateUniqueSlug(ctx context.Context, name string, excludeProductID string) (string, error)
func (q *ProductQueries) IsSlugAvailable(ctx context.Context, slug string, excludeProductID string) (bool, error)

// CSV support
func (q *ProductQueries) BulkImportProducts(ctx context.Context, products []Product) (*ImportResult, error)
func (q *ProductQueries) ExportProductsToCSV(ctx context.Context) ([]Product, error)

// Inventory queries
func (q *ProductQueries) GetLowStockProducts(ctx context.Context, threshold int) ([]Product, error)
func (q *ProductQueries) GetProductQuantity(ctx context.Context, productID string, variantID string) (int, error)
```

### New Handlers (`internal/handlers/private/product.go`)

```go
// Variant endpoints
POST   /api/_/products/:product_id/variants       // Add variants to product
PUT    /api/_/products/:product_id/variants/:id   // Update specific variant
DELETE /api/_/products/:product_id/variants/:id   // Delete variant
GET    /api/_/products/:product_id/variants       // List product variants

// Slug generation
POST   /api/_/products/slug/generate              // Generate slug from name
// Request: {"name": "Product Name", "exclude_id": "existing_product_id"}
// Response: {"slug": "product-name-2"}

// CSV import/export
POST   /api/_/products/import/preview             // Upload CSV, get preview (no import)
POST   /api/_/products/import                     // Execute import (after preview)
GET    /api/_/products/export                     // Download CSV
```

### Slug Service (`pkg/slugify/slug.go`)

```go
package slugify

import (
    "context"
    "database/sql"
    "fmt"
    
    "github.com/gosimple/slug"
)

type SlugService struct {
    db *sql.DB
}

func NewSlugService(db *sql.DB) *SlugService {
    return &SlugService{db: db}
}

// Generate creates URL-friendly slug from name, ensures uniqueness
func (s *SlugService) Generate(ctx context.Context, name string, excludeID string) (string, error) {
    base := slug.Make(name) // using github.com/gosimple/slug
    
    // Check if slug exists, append number if needed
    final := base
    counter := 2
    for {
        exists, err := s.exists(ctx, final, excludeID)
        if err != nil {
            return "", err
        }
        if !exists {
            return final, nil
        }
        final = fmt.Sprintf("%s-%d", base, counter)
        counter++
    }
}

func (s *SlugService) exists(ctx context.Context, slug string, excludeID string) (bool, error) {
    query := "SELECT COUNT(*) FROM product WHERE slug = ? AND id != ?"
    var count int
    err := s.db.QueryRowContext(ctx, query, slug, excludeID).Scan(&count)
    return count > 0, err
}
```

### CSV Service (`pkg/csvimport/importer.go`)

```go
package csvimport

import (
    "context"
    "database/sql"
    "encoding/csv"
    "io"
    "strings"
)

type CSVImporter struct {
    db *sql.DB
}

type ImportResult struct {
    TotalRows int      `json:"total_rows"`
    Imported  int      `json:"imported"`
    Updated   int      `json:"updated"`
    Skipped   int      `json:"skipped"`
    Errors    []Error  `json:"errors"`
}

type Error struct {
    Line    int    `json:"line"`
    Message string `json:"message"`
}

type ImportMode string

const (
    ModeUpsert ImportMode = "upsert"
)

// ValidateAndPreview parses CSV, returns preview without importing
func (c *CSVImporter) ValidateAndPreview(file io.Reader) (*ImportResult, []Product, error) {
    reader := csv.NewReader(file)
    
    // Read header
    header, err := reader.Read()
    if err != nil {
        return nil, nil, err
    }
    
    // Validate required columns
    requiredCols := []string{"name", "slug", "amount"}
    // ... validation logic
    
    result := &ImportResult{}
    products := []Product{}
    
    lineNum := 2 // Start from 2 (1 is header)
    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            result.Errors = append(result.Errors, Error{
                Line: lineNum,
                Message: err.Error(),
            })
            lineNum++
            continue
        }
        
        // Parse product from CSV row
        product, err := c.parseProduct(record, header)
        if err != nil {
            result.Errors = append(result.Errors, Error{
                Line: lineNum,
                Message: err.Error(),
            })
            result.Skipped++
        } else {
            products = append(products, product)
        }
        
        lineNum++
    }
    
    result.TotalRows = lineNum - 2
    return result, products, nil
}

// Import executes the import based on user's choice (add/update/skip)
func (c *CSVImporter) Import(ctx context.Context, products []Product, mode ImportMode) (*ImportResult, error) {
    result := &ImportResult{}
    
    for _, product := range products {
        // Check if product exists by slug
        exists, err := c.productExists(ctx, product.Slug)
        if err != nil {
            return nil, err
        }
        
        if exists && mode == ModeUpsert {
            // Update existing product
            if err := c.updateProduct(ctx, product); err != nil {
                result.Errors = append(result.Errors, Error{
                    Message: fmt.Sprintf("Failed to update %s: %v", product.Slug, err),
                })
            } else {
                result.Updated++
            }
        } else if !exists {
            // Insert new product
            if err := c.insertProduct(ctx, product); err != nil {
                result.Errors = append(result.Errors, Error{
                    Message: fmt.Sprintf("Failed to insert %s: %v", product.Slug, err),
                })
            } else {
                result.Imported++
            }
        } else {
            result.Skipped++
        }
    }
    
    return result, nil
}

func (c *CSVImporter) parseProduct(record []string, header []string) (Product, error) {
    // Parse basic fields
    product := Product{}
    
    for i, col := range header {
        switch col {
        case "name":
            product.Name = record[i]
        case "slug":
            product.Slug = record[i]
        case "amount":
            // Parse amount
        case "option1_name":
            // Parse delimited options
            // "Size" -> option name
        case "option1_values":
            // "M;L;XL" -> split by semicolon
        // ... more fields
        }
    }
    
    // Generate variants from options
    if len(product.Options) > 0 {
        product.HasVariants = true
        product.Variants = generateVariants(product.Options)
    }
    
    return product, nil
}
```

### CSV Format Specification

```csv
name,slug,sku,brief,description,amount,quantity,digital,active,option1_name,option1_values,option2_name,option2_values,variant_prices,variant_quantities,variant_skus,image_urls
T-Shirt,t-shirt,TSHIRT-BASE,Comfy tee,100% cotton shirt,2500,0,file,true,Size,M;L;XL,Color,Red;Blue,0;0;500;500;1000;1000,10;5;8;3;12;6,TSHIRT-M-RED;TSHIRT-M-BLUE;TSHIRT-L-RED;TSHIRT-L-BLUE;TSHIRT-XL-RED;TSHIRT-XL-BLUE,https://example.com/tshirt1.jpg;https://example.com/tshirt2.jpg
```

**Field explanations:**
- `option1_name`, `option2_name`, `option3_name` - option type names
- `option1_values`, `option2_values`, `option3_values` - semicolon-separated values
- `variant_prices` - semicolon-separated surcharges (in order of cartesian product)
- `variant_quantities` - semicolon-separated quantities
- `variant_skus` - semicolon-separated SKUs (optional)
- `image_urls` - semicolon-separated image URLs

**Variant ordering:** Cartesian product in order: option1 × option2 × option3
- Example: Size (M, L) × Color (Red, Blue) = M-Red, M-Blue, L-Red, L-Blue

---

## 3. Frontend Components (Svelte)

### Admin Interface (`web/admin/src/`)

#### Modified Product Form (`routes/products/+page.svelte`)

Enhanced sections:

1. **Basic Info** (existing + new fields):
   - Name → triggers slug auto-generation
   - Slug → auto-filled, editable, warning if changed
   - SKU → optional, unique
   - Brief, Description
   - Amount → base price
   - **Quantity** → only shown if `has_variants=false`
   - Active toggle

2. **Variants Section** (new):
   - Toggle: "This product has variants"
   - If enabled → `<VariantManager>` component
   - Auto-generates variants from options

3. **Images** (modified):
   - Product-level images (always)
   - Variant-level images (in variant table)

#### New Component: `lib/components/product/VariantManager.svelte`

```svelte
<script lang="ts">
  import OptionEditor from './OptionEditor.svelte'
  import VariantTable from './VariantTable.svelte'
  import { generateAllCombinations } from '$lib/utils/variantGenerator'
  
  export let hasVariants = false
  export let options: ProductOption[] = []
  export let variants: ProductVariant[] = []
  export let basePrice = 0
  export let currency = 'USD'
  
  // Auto-generate variants when options change
  function handleOptionsChange() {
    if (options.length === 0) {
      variants = []
      return
    }
    
    // Validate limits
    if (options.length > 3) {
      showError('Maximum 3 option types allowed')
      return
    }
    
    const totalValues = options.reduce((sum, opt) => sum * opt.values.length, 1)
    if (totalValues > 100) {
      showError('Maximum 100 variants allowed')
      return
    }
    
    // Generate variants
    variants = generateAllCombinations(options)
  }
</script>

<div class="variant-manager">
  <label class="checkbox-label">
    <input type="checkbox" bind:checked={hasVariants} />
    <span>This product has variants (size, color, material, etc.)</span>
  </label>
  
  {#if hasVariants}
    <div class="variant-content">
      <OptionEditor 
        bind:options 
        on:change={handleOptionsChange}
      />
      
      {#if variants.length > 0}
        <VariantTable 
          bind:variants 
          {basePrice} 
          {currency} 
        />
      {/if}
      
      <div class="variant-summary">
        <p>{variants.length} variant{variants.length !== 1 ? 's' : ''} will be created</p>
      </div>
    </div>
  {/if}
</div>

<style>
  .variant-manager {
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 1.5rem;
    margin: 1rem 0;
  }
  
  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-weight: 500;
  }
  
  .variant-content {
    margin-top: 1rem;
  }
  
  .variant-summary {
    margin-top: 1rem;
    padding: 0.75rem;
    background: var(--info-bg);
    border-radius: 4px;
  }
</style>
```

#### New Component: `lib/components/product/OptionEditor.svelte`

```svelte
<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import SvgIcon from '../SvgIcon.svelte'
  
  export let options: ProductOption[] = []
  
  const dispatch = createEventDispatcher()
  
  function addOption() {
    if (options.length >= 3) {
      alert('Maximum 3 option types allowed')
      return
    }
    
    options = [...options, {
      id: crypto.randomUUID(),
      name: '',
      values: [],
      position: options.length
    }]
  }
  
  function addValue(optionIndex: number) {
    const option = options[optionIndex]
    if (option.values.length >= 10) {
      alert('Maximum 10 values per option')
      return
    }
    
    option.values = [...option.values, {
      id: crypto.randomUUID(),
      value: '',
      position: option.values.length
    }]
    
    options = options
    dispatch('change')
  }
  
  function removeOption(index: number) {
    options = options.filter((_, i) => i !== index)
    dispatch('change')
  }
  
  function removeValue(optionIndex: number, valueIndex: number) {
    options[optionIndex].values = options[optionIndex].values.filter((_, i) => i !== valueIndex)
    options = options
    dispatch('change')
  }
</script>

<div class="option-editor">
  <h3>Product Options</h3>
  
  {#each options as option, optionIdx}
    <div class="option-group">
      <div class="option-header">
        <input 
          type="text" 
          placeholder="Option name (e.g., Size, Color)" 
          bind:value={option.name}
          on:blur={() => dispatch('change')}
        />
        <button 
          type="button" 
          class="btn-icon" 
          on:click={() => removeOption(optionIdx)}
          aria-label="Delete option"
        >
          <SvgIcon name="trash" />
        </button>
      </div>
      
      <div class="option-values">
        {#each option.values as value, valueIdx}
          <div class="value-row">
            <input 
              type="text" 
              placeholder="Value (e.g., Medium, Red)" 
              bind:value={value.value}
              on:blur={() => dispatch('change')}
            />
            <button 
              type="button" 
              class="btn-icon-small" 
              on:click={() => removeValue(optionIdx, valueIdx)}
            >
              ×
            </button>
          </div>
        {/each}
        
        <button 
          type="button" 
          class="btn-add-value" 
          on:click={() => addValue(optionIdx)}
        >
          + Add value
        </button>
      </div>
    </div>
  {/each}
  
  {#if options.length < 3}
    <button type="button" class="btn-add-option" on:click={addOption}>
      + Add option
    </button>
  {/if}
</div>
```

#### New Component: `lib/components/product/VariantTable.svelte`

Displays variants grouped by first option (Shopify-style):

```svelte
<script lang="ts">
  import { formatPrice } from '$lib/utils'
  import Upload from '../form/Upload.svelte'
  
  export let variants: ProductVariant[] = []
  export let basePrice = 0
  export let currency = 'USD'
  
  // Group variants by first option
  $: groupedVariants = groupByFirstOption(variants)
  
  function groupByFirstOption(variants: ProductVariant[]) {
    const groups = new Map()
    
    variants.forEach(variant => {
      const firstOptionKey = Object.keys(variant.option_values)[0]
      const firstOptionValue = variant.option_values[firstOptionKey]
      
      if (!groups.has(firstOptionValue)) {
        groups.set(firstOptionValue, [])
      }
      
      groups.get(firstOptionValue).push(variant)
    })
    
    return groups
  }
</script>

<div class="variant-table">
  <table>
    <thead>
      <tr>
        <th>Variant</th>
        <th>SKU</th>
        <th>Price</th>
        <th>Quantity</th>
        <th>Active</th>
      </tr>
    </thead>
    <tbody>
      {#each [...groupedVariants.entries()] as [groupName, groupVariants]}
        <tr class="group-header">
          <td colspan="5">
            <strong>{groupName}</strong>
            <span class="group-count">({groupVariants.length} variants)</span>
          </td>
        </tr>
        
        {#each groupVariants as variant}
          <tr class="variant-row">
            <td class="variant-name">
              <div class="variant-details">
                <Upload 
                  bind:files={variant.images} 
                  accept="image/*"
                  maxSize={5242880}
                  class="variant-image-upload"
                />
                <span>{Object.values(variant.option_values).join(' / ')}</span>
              </div>
            </td>
            <td>
              <input 
                type="text" 
                bind:value={variant.sku} 
                placeholder="Optional"
                class="input-sku"
              />
            </td>
            <td>
              <div class="price-input">
                <input 
                  type="number" 
                  bind:value={variant.price_surcharge}
                  step="1"
                  class="input-surcharge"
                />
                <span class="price-display">
                  = {formatPrice(basePrice + variant.price_surcharge, currency)}
                </span>
              </div>
            </td>
            <td>
              <input 
                type="number" 
                bind:value={variant.quantity} 
                min="0"
                required
                class="input-quantity"
              />
            </td>
            <td>
              <input 
                type="checkbox" 
                bind:checked={variant.active}
              />
            </td>
          </tr>
        {/each}
      {/each}
    </tbody>
  </table>
</div>
```

#### New Page: `routes/products/import/+page.svelte`

```svelte
<script lang="ts">
  import Main from '$lib/layouts/Main.svelte'
  import FormButton from '$lib/components/form/Button.svelte'
  import { showMessage } from '$lib/utils'
  import type { ImportResult } from '$lib/types/models'
  
  let csvFile: File | null = null
  let previewData: ImportResult | null = null
  let importing = false
  
  async function handleFileUpload() {
    if (!csvFile) return
    
    const formData = new FormData()
    formData.append('file', csvFile)
    
    try {
      const response = await fetch('/api/_/products/import/preview', {
        method: 'POST',
        body: formData
      })
      
      if (!response.ok) throw new Error('Preview failed')
      
      previewData = await response.json()
    } catch (err) {
      showMessage('Failed to preview CSV', 'error')
    }
  }
  
  async function confirmImport() {
    if (!csvFile || !previewData) return
    
    importing = true
    const formData = new FormData()
    formData.append('file', csvFile)
    
    try {
      const response = await fetch('/api/_/products/import', {
        method: 'POST',
        body: formData,
        headers: {
          'X-Import-Mode': 'upsert'
        }
      })
      
      if (!response.ok) throw new Error('Import failed')
      
      const result = await response.json()
      showMessage(
        `Imported ${result.imported} products, updated ${result.updated}`, 
        'success'
      )
      
      previewData = null
      csvFile = null
    } catch (err) {
      showMessage('Import failed', 'error')
    } finally {
      importing = false
    }
  }
  
  async function downloadCSV() {
    try {
      const response = await fetch('/api/_/products/export')
      if (!response.ok) throw new Error('Export failed')
      
      const blob = await response.blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `products-${new Date().toISOString().split('T')[0]}.csv`
      a.click()
      URL.revokeObjectURL(url)
    } catch (err) {
      showMessage('Export failed', 'error')
    }
  }
</script>

<Main title="Import/Export Products">
  <div class="import-export">
    <section class="import-section">
      <h2>Import Products</h2>
      <p class="help-text">
        Upload a CSV file to bulk import products. 
        <a href="/docs/csv-format.md" target="_blank">View CSV format documentation</a>
      </p>
      
      <div class="file-upload">
        <input 
          type="file" 
          accept=".csv" 
          on:change={(e) => csvFile = e.target.files[0]}
          id="csv-file"
        />
        <label for="csv-file" class="file-label">
          {csvFile ? csvFile.name : 'Choose CSV file...'}
        </label>
      </div>
      
      <FormButton 
        on:click={handleFileUpload} 
        disabled={!csvFile}
      >
        Preview Import
      </FormButton>
      
      {#if previewData}
        <div class="preview">
          <h3>Import Preview</h3>
          
          <div class="preview-stats">
            <div class="stat">
              <span class="stat-label">Total rows:</span>
              <span class="stat-value">{previewData.total_rows}</span>
            </div>
            <div class="stat success">
              <span class="stat-label">To add:</span>
              <span class="stat-value">{previewData.to_add || 0}</span>
            </div>
            <div class="stat info">
              <span class="stat-label">To update:</span>
              <span class="stat-value">{previewData.to_update || 0}</span>
            </div>
            {#if previewData.errors?.length > 0}
              <div class="stat error">
                <span class="stat-label">Errors:</span>
                <span class="stat-value">{previewData.errors.length}</span>
              </div>
            {/if}
          </div>
          
          {#if previewData.errors?.length > 0}
            <div class="errors">
              <h4>Errors</h4>
              <ul>
                {#each previewData.errors as error}
                  <li>
                    <strong>Line {error.line}:</strong> {error.message}
                  </li>
                {/each}
              </ul>
            </div>
          {/if}
          
          <div class="preview-actions">
            <FormButton 
              on:click={confirmImport} 
              disabled={importing || previewData.errors?.length > 0}
              loading={importing}
            >
              Import (Update existing + Add new)
            </FormButton>
            <button 
              type="button" 
              class="btn-cancel" 
              on:click={() => previewData = null}
            >
              Cancel
            </button>
          </div>
        </div>
      {/if}
    </section>
    
    <section class="export-section">
      <h2>Export Products</h2>
      <p class="help-text">
        Download all products as a CSV file
      </p>
      
      <FormButton on:click={downloadCSV}>
        Download CSV
      </FormButton>
    </section>
  </div>
</Main>
```

### Storefront (`web/site/src/`)

#### Modified Product Page (`routes/products/[slug]/+page.svelte`)

```svelte
<script lang="ts">
  import { page } from '$app/stores'
  import ProductGallery from '$lib/components/ProductGallery.svelte'
  import VariantSelector from '$lib/components/VariantSelector.svelte'
  import { formatPrice } from '$lib/utils'
  import type { Product, ProductVariant } from '$lib/types/models'
  
  export let data: { product: Product }
  
  const { product } = data
  
  let selectedOptions: Record<string, string> = {}
  let selectedVariant: ProductVariant | null = null
  
  // Find matching variant when options change
  $: {
    if (product.has_variants && Object.keys(selectedOptions).length > 0) {
      selectedVariant = findVariant(selectedOptions)
    }
  }
  
  $: displayPrice = selectedVariant 
    ? product.amount + selectedVariant.price_surcharge
    : product.amount
    
  $: inStock = product.has_variants
    ? (selectedVariant?.quantity ?? 0) > 0
    : product.quantity > 0
    
  $: displayImages = selectedVariant?.images?.length > 0
    ? selectedVariant.images
    : product.images
  
  function findVariant(options: Record<string, string>): ProductVariant | null {
    return product.variants?.find(v => {
      return Object.entries(options).every(([key, value]) => {
        return v.option_values[key] === value
      })
    }) || null
  }
  
  function addToCart() {
    // Add to cart logic
    const cartItem = {
      product_id: product.id,
      variant_id: selectedVariant?.id,
      quantity: 1,
      snapshot: {
        name: product.name,
        price: displayPrice,
        variant_options: selectedOptions
      }
    }
    
    // ... add to cart
  }
</script>

<div class="product-detail">
  <div class="images">
    <ProductGallery images={displayImages} />
  </div>
  
  <div class="info">
    <h1>{product.name}</h1>
    
    {#if product.brief}
      <p class="brief">{product.brief}</p>
    {/if}
    
    <p class="price">{formatPrice(displayPrice, $page.data.currency)}</p>
    
    {#if product.has_variants}
      <VariantSelector
        options={product.options}
        variants={product.variants}
        bind:selected={selectedOptions}
      />
    {/if}
    
    <div class="stock-info">
      {#if product.has_variants && selectedVariant}
        <p class="stock">
          {selectedVariant.quantity > 0 
            ? `${selectedVariant.quantity} in stock`
            : 'Out of stock'
          }
        </p>
      {:else if !product.has_variants}
        <p class="stock">
          {product.quantity > 0 
            ? `${product.quantity} in stock`
            : 'Out of stock'
          }
        </p>
      {/if}
    </div>
    
    <button 
      class="btn-add-cart" 
      disabled={!inStock || (product.has_variants && !selectedVariant)}
      on:click={addToCart}
    >
      {inStock ? 'Add to Cart' : 'Out of Stock'}
    </button>
    
    <div class="description">
      {@html product.description}
    </div>
  </div>
</div>
```

#### New Component: `lib/components/VariantSelector.svelte`

```svelte
<script lang="ts">
  import type { ProductOption, ProductVariant } from '$lib/types/models'
  
  export let options: ProductOption[] = []
  export let variants: ProductVariant[] = []
  export let selected: Record<string, string> = {}
  
  function selectOption(optionName: string, value: string) {
    selected = { ...selected, [optionName]: value }
  }
  
  function isOptionAvailable(optionName: string, value: string): boolean {
    // Check if this option value is available given current selections
    const testSelection = { ...selected, [optionName]: value }
    
    return variants.some(v => {
      return Object.entries(testSelection).every(([key, val]) => {
        return v.option_values[key] === val
      }) && v.active && v.quantity > 0
    })
  }
</script>

<div class="variant-selector">
  {#each options as option}
    <div class="option">
      <label class="option-label">{option.name}</label>
      <div class="option-values">
        {#each option.values as value}
          {@const isAvailable = isOptionAvailable(option.name, value.value)}
          {@const isSelected = selected[option.name] === value.value}
          
          <button
            type="button"
            class="option-btn"
            class:selected={isSelected}
            class:unavailable={!isAvailable}
            disabled={!isAvailable}
            on:click={() => selectOption(option.name, value.value)}
          >
            {value.value}
          </button>
        {/each}
      </div>
    </div>
  {/each}
</div>

<style>
  .variant-selector {
    margin: 1.5rem 0;
  }
  
  .option {
    margin-bottom: 1rem;
  }
  
  .option-label {
    display: block;
    font-weight: 600;
    margin-bottom: 0.5rem;
  }
  
  .option-values {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  
  .option-btn {
    padding: 0.5rem 1rem;
    border: 2px solid var(--border-color);
    border-radius: 4px;
    background: white;
    cursor: pointer;
    transition: all 0.2s;
  }
  
  .option-btn:hover:not(:disabled) {
    border-color: var(--primary-color);
  }
  
  .option-btn.selected {
    border-color: var(--primary-color);
    background: var(--primary-light);
    font-weight: 600;
  }
  
  .option-btn.unavailable {
    opacity: 0.4;
    text-decoration: line-through;
    cursor: not-allowed;
  }
</style>
```

---

## 4. Data Flow

### Slug Generation Flow

```
User types product name → debounced onChange (300ms)
  ↓
Frontend: POST /api/_/products/slug/generate
  Body: {"name": "Yoga Strap 6ft", "exclude_id": "product123"}
  ↓
Backend: SlugService.Generate()
  1. Convert: "Yoga Strap 6ft" → "yoga-strap-6ft"
  2. Check database for conflicts
  3. If exists, append: "yoga-strap-6ft-2"
  4. Return unique slug
  ↓
Frontend: Update slug field (user can edit)
  ↓
On save: If slug changed for existing product
  → Show warning: "⚠️ Changing slug will break existing links"
```

### Variant Generation Flow

```
Admin toggles "has_variants" ON
  ↓
Admin adds options via OptionEditor:
  - Option 1: Size → [M, L, XL]
  - Option 2: Color → [Red, Blue]
  ↓
Frontend: generateAllCombinations()
  Cartesian product: M×Red, M×Blue, L×Red, L×Blue, XL×Red, XL×Blue (6)
  ↓
Display in VariantTable:
  - Auto-suggest SKU (editable)
  - Price surcharge input (default 0)
  - Quantity input (required, default 0)
  - Active toggle (default true)
  - Image upload (optional)
  ↓
User clicks Save → POST /api/_/products
  Body: {product data + options + variants}
  ↓
Backend validation:
  ✓ Max 3 options
  ✓ Max 10 values per option
  ✓ Max 100 variants
  ✓ All variants have quantity
  ✓ SKUs unique
  ↓
Database transaction:
  1. INSERT/UPDATE product (has_variants=true)
  2. INSERT product_option rows
  3. INSERT product_option_value rows
  4. INSERT product_variant rows
  5. INSERT product_variant_option junction rows
  6. INSERT product_variant_image rows
  ↓
COMMIT or ROLLBACK on error
```

### Storefront Variant Selection Flow

```
User visits /products/yoga-strap
  ↓
GET /api/products/{slug}
  Returns: product + options + active variants
  ↓
User selects: Size=M, Color=Red
  ↓
Frontend: Filter variants
  variant.option_values.Size === "M" && 
  variant.option_values.Color === "Red"
  ↓
Match found → Update UI:
  - Price: base + surcharge
  - Stock: variant.quantity
  - Images: variant.images || product.images
  ↓
User clicks "Add to Cart"
  ↓
Validate: variant.quantity > 0
  ↓
Add to cart:
  {
    product_id: "abc123",
    variant_id: "var456",
    snapshot: {
      name: "Yoga Strap",
      price: 2500,
      variant_options: {"Size": "M", "Color": "Red"}
    }
  }
```

### CSV Import Flow

```
Admin uploads CSV → POST /api/_/products/import/preview
  ↓
Backend: CSVImporter.ValidateAndPreview()
  1. Read all rows
  2. Parse delimited format:
     option1_name: Size | option1_values: M;L
  3. Validate each row:
     - Required fields present
     - Valid data types
     - Slug/SKU uniqueness
     - Image URLs accessible (HEAD request)
  4. Match existing by slug
  ↓
Return preview:
  {
    total_rows: 100,
    to_add: 30,
    to_update: 65,
    errors: [
      {line: 15, message: "Missing field: name"},
      {line: 23, message: "Duplicate SKU"}
    ]
  }
  ↓
Frontend: Show modal
  "Import will:
   ✓ Add 30 new products
   ✓ Update 65 existing
   ✗ 5 rows have errors
   
   [View Errors] [Cancel] [Import]"
  ↓
User clicks Import → POST /api/_/products/import
  Header: X-Import-Mode: upsert
  ↓
Backend: CSVImporter.Import()
  For each valid row in transaction:
    1. Check slug exists
    2. If yes: UPDATE product + variants
    3. If no: INSERT product + variants
    4. Download images from URLs
    5. Generate variants from options
  ↓
Return result:
  {
    imported: 30,
    updated: 65,
    skipped: 5,
    errors: [...]
  }
```

### CSV Export Flow

```
Admin clicks "Export" → GET /api/_/products/export
  ↓
Backend: Query all products with variants
  For each product:
    1. Flatten variant options → delimited format
    2. Convert image file IDs → URLs
    3. Generate CSV row
  ↓
CSV format:
  name,slug,sku,...,option1_name,option1_values,...,variant_prices,variant_quantities
  T-Shirt,tshirt,,...,Size,M;L,Color,Red;Blue,0;0;500;500,10;5;8;3
  ↓
Return: Content-Type: text/csv
  Content-Disposition: attachment; filename="products-2026-07-21.csv"
```

### Inventory Update Flow (Purchase)

```
Checkout → Create order
  ↓
For each cart item:
  BEGIN TRANSACTION
    IF variant_id EXISTS:
      UPDATE product_variant
      SET quantity = quantity - purchased_qty
      WHERE id = variant_id AND quantity >= purchased_qty
    ELSE:
      UPDATE product
      SET quantity = quantity - purchased_qty
      WHERE id = product_id AND quantity >= purchased_qty
    
    IF affected_rows = 0:
      ROLLBACK (oversold)
      RETURN error: "Insufficient stock"
    
  COMMIT
```

### Cart Warning Flow

```
User has items in cart
  ↓
On cart view: Check each item
  1. Product exists? deleted=false
  2. Product active? active=true
  3. Variant exists? (if variant_id)
  4. Variant active? (if variant_id)
  5. Price changed? snapshot.price vs current price
  ↓
IF changed:
  Display warning:
  "⚠️ Cart items changed:
   - T-Shirt (M, Red): Price $25 → $30
   - Yoga Strap: No longer available
   
   [Review Cart]"
```

---

## 5. Error Handling

### Validation Rules

| Entity | Field | Rules |
|--------|-------|-------|
| Product | name | Required, 3-100 chars |
| Product | slug | Required, 3-100 chars, unique, URL-safe |
| Product | sku | Optional, 0-50 chars, unique if provided |
| Product | amount | Required, >= 0 |
| Product | quantity | Required, >= 0 |
| Product | options | Optional, max 3 items |
| Product | variants | Optional, max 100 items |
| ProductOption | name | Required, 1-50 chars |
| ProductOption | values | Required, 1-10 items |
| ProductOptionValue | value | Required, 1-100 chars |
| ProductVariant | sku | Optional, 0-50 chars, unique if provided |
| ProductVariant | quantity | Required, >= 0 |
| ProductVariant | option_values | Required, 1-3 items |

### Error Scenarios

| Scenario | Detection | HTTP Code | Response |
|----------|-----------|-----------|----------|
| Duplicate slug | On save, query database | 400 | `{"error": "Slug 'yoga-strap' already exists"}` |
| Duplicate SKU | On save, query database | 400 | `{"error": "SKU 'TSHIRT-001' already in use"}` |
| Too many variants | Count before save | 400 | `{"error": "Maximum 100 variants allowed (you have 120)"}` |
| Too many options | Count before save | 400 | `{"error": "Maximum 3 option types allowed"}` |
| Too many option values | Count before save | 400 | `{"error": "Maximum 10 values per option"}` |
| Missing variant quantity | Validate variants | 400 | `{"error": "Variant 'M-Red' missing quantity"}` |
| Invalid option combo | Validate junction table | 400 | `{"error": "Invalid option combination"}` |
| Oversold inventory | Check before checkout | 400 | `{"error": "Only 5 items in stock"}` |
| CSV parse error | Try parse, catch | 400 | Preview with errors array |
| CSV invalid URL | HEAD request to URL | N/A | Preview: "Line 15: Image URL not found" |
| CSV missing field | Check required cols | N/A | Preview: "Line 8: Missing 'name'" |
| Image too large | Check file size | 413 | `{"error": "Image exceeds 5MB limit"}` |
| Product deleted | Check on cart view | N/A | UI warning: "Item unavailable" |
| Price changed | Compare snapshot | N/A | UI warning: "Price changed" |

### Transaction Rollback

All multi-table operations wrapped in transactions:

```go
func (q *ProductQueries) AddProductWithVariants(ctx context.Context, product *Product) error {
    tx, err := q.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback()
    
    // 1. Insert product
    if err := insertProduct(tx, product); err != nil {
        return fmt.Errorf("insert product: %w", err)
    }
    
    // 2. Insert options
    for _, opt := range product.Options {
        if err := insertOption(tx, opt); err != nil {
            return fmt.Errorf("insert option: %w", err)
        }
    }
    
    // 3. Insert variants
    for _, variant := range product.Variants {
        if err := insertVariant(tx, variant); err != nil {
            return fmt.Errorf("insert variant: %w", err)
        }
    }
    
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit transaction: %w", err)
    }
    
    return nil
}
```

---

## 6. Testing Strategy

### Unit Tests

**Models** (`internal/models/products_test.go`):
```go
func TestProductValidation(t *testing.T)
func TestProductOptionValidation(t *testing.T)
func TestProductVariantValidation(t *testing.T)
func TestTooManyOptions(t *testing.T)
func TestTooManyVariants(t *testing.T)
```

**Slug Service** (`pkg/slugify/slug_test.go`):
```go
func TestSlugGeneration(t *testing.T)
func TestSlugUniqueness(t *testing.T)
func TestSlugSpecialCharacters(t *testing.T)
```

**CSV Importer** (`pkg/csvimport/import_test.go`):
```go
func TestCSVParsing(t *testing.T)
func TestCSVValidation(t *testing.T)
func TestVariantGeneration(t *testing.T)
```

### Integration Tests

**Queries** (`internal/queries/products_test.go`):
```go
func TestAddProductWithVariants(t *testing.T)
func TestUpdateProductVariants(t *testing.T)
func TestDeleteVariant(t *testing.T)
func TestDuplicateSKUPrevention(t *testing.T)
func TestInventoryDecrement(t *testing.T)
```

### API Tests

**Handlers** (`internal/handlers/private/product_test.go`):
```go
func TestCreateProductWithVariants(t *testing.T)
func TestUpdateVariant(t *testing.T)
func TestSlugGeneration(t *testing.T)
func TestCSVImport(t *testing.T)
func TestCSVExport(t *testing.T)
```

### Frontend Tests

**Components** (`web/admin/src/lib/components/product/VariantManager.test.ts`):
```typescript
describe('VariantManager', () => {
  test('generates all variant combinations')
  test('enforces max 100 variants')
  test('enforces max 3 options')
  test('enforces max 10 values per option')
})

describe('OptionEditor', () => {
  test('adds option')
  test('removes option')
  test('adds value')
  test('removes value')
})

describe('VariantSelector', () => {
  test('filters available options')
  test('updates price on selection')
  test('shows out of stock')
})
```

### Manual Testing Checklist

**Product Creation:**
- [ ] Create simple product (no variants)
- [ ] Create product with 1 option (3 values) → 3 variants
- [ ] Create product with 2 options → N×M variants
- [ ] Create product with 3 options → N×M×P variants
- [ ] Try 4 options → verify error
- [ ] Try 11 values in option → verify error
- [ ] Try combo generating >100 variants → verify error

**Slug Generation:**
- [ ] Type product name → slug auto-fills
- [ ] Create duplicate name → slug gets "-2"
- [ ] Manually edit slug → saves custom slug
- [ ] Edit slug on existing product → see warning

**Variants:**
- [ ] Upload variant-specific image
- [ ] Set variant price surcharge
- [ ] Set variant quantity
- [ ] Deactivate individual variant
- [ ] Delete variant

**Storefront:**
- [ ] View product with variants
- [ ] Select variant → price updates
- [ ] Select variant → image updates
- [ ] Select out-of-stock variant → button disabled
- [ ] Add variant to cart → snapshot saved

**Inventory:**
- [ ] Purchase product → quantity decrements
- [ ] Purchase variant → variant quantity decrements
- [ ] Try oversell → checkout fails

**CSV:**
- [ ] Import valid CSV → preview shows summary
- [ ] Import with errors → preview shows line numbers
- [ ] Import duplicates → updates existing
- [ ] Import new products → adds new
- [ ] Export products → download CSV
- [ ] Re-import exported CSV → no errors

**Cart Warnings:**
- [ ] Add to cart, delete product → warning shown
- [ ] Add to cart, change price → warning shown
- [ ] Add to cart, deactivate variant → warning shown

---

## 7. Migration Strategy

### Database Migration

Create new migration file: `migrations/YYYYMMDDHHMMSS_product_variants.sql`

```sql
-- +goose Up
-- +goose StatementBegin

-- Add new columns to product table
ALTER TABLE product ADD COLUMN quantity INTEGER DEFAULT 0;
ALTER TABLE product ADD COLUMN sku TEXT;
ALTER TABLE product ADD COLUMN has_variants BOOLEAN DEFAULT FALSE;

-- Create unique index on sku (only if not null)
CREATE UNIQUE INDEX idx_product_sku ON product (sku) WHERE sku IS NOT NULL;

-- Add position to product_image
ALTER TABLE product_image ADD COLUMN position INTEGER DEFAULT 0;

-- Create new tables
CREATE TABLE product_option (
    id          TEXT PRIMARY KEY NOT NULL,
    product_id  TEXT NOT NULL,
    name        TEXT NOT NULL,
    position    INTEGER DEFAULT 0,
    created     TIMESTAMP DEFAULT (datetime('now')),
    FOREIGN KEY (product_id) REFERENCES product(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_option_product_id ON product_option (product_id);

CREATE TABLE product_option_value (
    id          TEXT PRIMARY KEY NOT NULL,
    option_id   TEXT NOT NULL,
    value       TEXT NOT NULL,
    position    INTEGER DEFAULT 0,
    FOREIGN KEY (option_id) REFERENCES product_option(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_option_value_option_id ON product_option_value (option_id);

CREATE TABLE product_variant (
    id              TEXT PRIMARY KEY NOT NULL,
    product_id      TEXT NOT NULL,
    sku             TEXT,
    price_surcharge NUMERIC DEFAULT 0,
    quantity        INTEGER DEFAULT 0,
    active          BOOLEAN DEFAULT TRUE,
    created         TIMESTAMP DEFAULT (datetime('now')),
    updated         TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES product(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_variant_product_id ON product_variant (product_id);
CREATE UNIQUE INDEX idx_product_variant_sku ON product_variant (sku) WHERE sku IS NOT NULL;

CREATE TABLE product_variant_option (
    variant_id      TEXT NOT NULL,
    option_value_id TEXT NOT NULL,
    PRIMARY KEY (variant_id, option_value_id),
    FOREIGN KEY (variant_id) REFERENCES product_variant(id) ON DELETE CASCADE,
    FOREIGN KEY (option_value_id) REFERENCES product_option_value(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_variant_option_variant ON product_variant_option (variant_id);
CREATE INDEX idx_product_variant_option_value ON product_variant_option (option_value_id);

CREATE TABLE product_variant_image (
    id          TEXT PRIMARY KEY NOT NULL,
    variant_id  TEXT NOT NULL,
    name        TEXT NOT NULL,
    ext         TEXT NOT NULL,
    orig_name   TEXT NOT NULL,
    position    INTEGER DEFAULT 0,
    FOREIGN KEY (variant_id) REFERENCES product_variant(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_variant_image_variant_id ON product_variant_image (variant_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE product_variant_image;
DROP TABLE product_variant_option;
DROP TABLE product_variant;
DROP TABLE product_option_value;
DROP TABLE product_option;

ALTER TABLE product_image DROP COLUMN position;
ALTER TABLE product DROP COLUMN has_variants;
ALTER TABLE product DROP COLUMN sku;
ALTER TABLE product DROP COLUMN quantity;

-- +goose StatementEnd
```

### Backward Compatibility

**Existing cart data:**
- Carts created before variants will only have `product_id`, no `variant_id`
- Backend handles both cases:
  ```go
  if cartItem.VariantID != "" {
      // Use variant quantity
  } else {
      // Use product quantity
  }
  ```

**API compatibility:**
- GET `/api/products` returns products with new fields (backwards compatible)
- Clients ignoring new fields will still work
- Old product format still accepted (no variants)

### Rollout Plan

1. **Phase 1: Backend**
   - Deploy migration
   - Deploy new backend code
   - Test existing products still work
   
2. **Phase 2: Admin UI**
   - Deploy admin frontend
   - Test creating variants
   - Test CSV import/export
   
3. **Phase 3: Storefront**
   - Deploy site frontend
   - Test variant selection
   - Test cart/checkout with variants

4. **Phase 4: Data Migration** (if needed)
   - Migrate existing products to new structure
   - Add default quantities to existing products

---

## 8. Future Enhancements (Out of Scope)

These are NOT part of this implementation but could be added later:

- **Variant-specific metadata** - Custom fields per variant
- **Bulk variant editing** - Update multiple variants at once
- **Variant images bulk upload** - Upload zip of images, auto-match to variants
- **Low stock alerts** - Email when quantity < threshold
- **Inventory history** - Track quantity changes over time
- **Reserved inventory** - Hold stock for unpaid carts
- **Variant combinations preview** - Show visual grid before generating
- **Conditional options** - Size L only available in Red
- **Dynamic pricing** - Time-based or user-group-based pricing
- **Multi-warehouse inventory** - Track quantity per location

---

## Summary

This specification covers:

✅ **Product Variants** - Up to 3 options, auto-generated combinations, max 100 variants  
✅ **Inventory Management** - Dual-level quantity tracking (product/variant)  
✅ **Smart Slug Generation** - Auto-generated, conflict resolution, manual override  
✅ **CSV Import/Export** - Delimited format, preview, validation, upsert mode  

**Database:** 6 new tables, 3 modified columns  
**Backend:** New models, queries, handlers, services  
**Frontend:** Admin variant manager, CSV importer, storefront variant selector  
**Testing:** Unit, integration, API, frontend tests + manual checklist  
**Migration:** Backward compatible, phased rollout  

**Approval status:** ✅ All sections approved by stakeholder
