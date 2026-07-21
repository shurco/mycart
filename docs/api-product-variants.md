# Product Variants API Documentation

This document describes the API endpoints for managing product variants in myCart.

## Table of Contents

- [Overview](#overview)
- [Data Models](#data-models)
- [Admin API Endpoints](#admin-api-endpoints)
- [Public API Endpoints](#public-api-endpoints)
- [CSV Import/Export Format](#csv-importexport-format)

## Overview

The Product Variants API allows you to:
- Create products with multiple options (size, color, etc.)
- Generate variant combinations automatically
- Manage individual variant pricing, inventory, and SKUs
- Import/export products with variants via CSV
- Query products with variant information

## Data Models

### ProductOption

Represents a product option (e.g., "Size", "Color").

```json
{
  "id": "01234567-89ab-cdef-0123-456789abcdef",
  "product_id": "01234567-89ab-cdef-0123-456789abcdef",
  "name": "Size",
  "values": [
    {
      "id": "01234567-89ab-cdef-0123-456789abcdef",
      "option_id": "01234567-89ab-cdef-0123-456789abcdef",
      "value": "Small"
    },
    {
      "id": "01234567-89ab-cdef-0123-456789abcdef",
      "option_id": "01234567-89ab-cdef-0123-456789abcdef",
      "value": "Medium"
    }
  ]
}
```

### ProductVariant

Represents a specific variant combination (e.g., "Small + Red").

```json
{
  "id": "01234567-89ab-cdef-0123-456789abcdef",
  "product_id": "01234567-89ab-cdef-0123-456789abcdef",
  "sku": "TSHIRT-S-RED",
  "price_surcharge": 0,
  "quantity": 100,
  "option_values": {
    "Size": "Small",
    "Color": "Red"
  },
  "active": true
}
```

### Product (with variants)

Complete product object including variant information.

```json
{
  "id": "01234567-89ab-cdef-0123-456789abcdef",
  "name": "Premium T-Shirt",
  "slug": "premium-t-shirt",
  "amount": 1999,
  "has_variants": true,
  "quantity": 0,
  "sku": "",
  "options": [
    {
      "id": "01234567-89ab-cdef-0123-456789abcdef",
      "product_id": "01234567-89ab-cdef-0123-456789abcdef",
      "name": "Size",
      "values": [
        {
          "id": "01234567-89ab-cdef-0123-456789abcdef",
          "option_id": "01234567-89ab-cdef-0123-456789abcdef",
          "value": "Small"
        },
        {
          "id": "01234567-89ab-cdef-0123-456789abcdef",
          "option_id": "01234567-89ab-cdef-0123-456789abcdef",
          "value": "Medium"
        }
      ]
    },
    {
      "id": "01234567-89ab-cdef-0123-456789abcdef",
      "product_id": "01234567-89ab-cdef-0123-456789abcdef",
      "name": "Color",
      "values": [
        {
          "id": "01234567-89ab-cdef-0123-456789abcdef",
          "option_id": "01234567-89ab-cdef-0123-456789abcdef",
          "value": "Red"
        },
        {
          "id": "01234567-89ab-cdef-0123-456789abcdef",
          "option_id": "01234567-89ab-cdef-0123-456789abcdef",
          "value": "Blue"
        }
      ]
    }
  ],
  "variants": [
    {
      "id": "01234567-89ab-cdef-0123-456789abcdef",
      "product_id": "01234567-89ab-cdef-0123-456789abcdef",
      "sku": "TSHIRT-S-RED",
      "price_surcharge": 0,
      "quantity": 100,
      "option_values": {
        "Size": "Small",
        "Color": "Red"
      },
      "active": true
    },
    {
      "id": "01234567-89ab-cdef-0123-456789abcdef",
      "product_id": "01234567-89ab-cdef-0123-456789abcdef",
      "sku": "TSHIRT-S-BLUE",
      "price_surcharge": 200,
      "quantity": 50,
      "option_values": {
        "Size": "Small",
        "Color": "Blue"
      },
      "active": true
    }
  ]
}
```

## Admin API Endpoints

### Create/Update Product with Variants

**Endpoint**: `POST /_/api/products` (create) or `PATCH /_/api/products/:id` (update)

**Authentication**: Required (admin session)

**Request Body** (multipart/form-data):

```json
{
  "name": "Premium T-Shirt",
  "slug": "premium-t-shirt",
  "amount": 1999,
  "has_variants": true,
  "options": [
    {
      "name": "Size",
      "values": [
        { "value": "Small" },
        { "value": "Medium" },
        { "value": "Large" }
      ]
    },
    {
      "name": "Color",
      "values": [
        { "value": "Red" },
        { "value": "Blue" }
      ]
    }
  ],
  "variants": [
    {
      "sku": "TSHIRT-S-RED",
      "price_surcharge": 0,
      "quantity": 100,
      "option_values": {
        "Size": "Small",
        "Color": "Red"
      },
      "active": true
    },
    {
      "sku": "TSHIRT-M-RED",
      "price_surcharge": 0,
      "quantity": 80,
      "option_values": {
        "Size": "Medium",
        "Color": "Red"
      },
      "active": true
    }
  ]
}
```

**Response**: `200 OK`

```json
{
  "id": "01234567-89ab-cdef-0123-456789abcdef",
  "name": "Premium T-Shirt",
  "slug": "premium-t-shirt",
  "amount": 1999,
  "has_variants": true,
  "options": [...],
  "variants": [...]
}
```

**Validation Rules**:
- Maximum 3 options per product
- Maximum 10 values per option
- All option values in variants must match defined options
- `price_surcharge` is in cents (can be negative)
- `quantity` must be >= 0

### Get Product by ID

**Endpoint**: `GET /_/api/products/:id`

**Authentication**: Required (admin session)

**Response**: `200 OK`

Returns complete product object with options and variants (if `has_variants: true`).

### Get Product by Slug

**Endpoint**: `GET /_/api/products/:slug`

**Authentication**: Required (admin session)

**Response**: `200 OK`

Returns complete product object with options and variants (if `has_variants: true`).

### List Products

**Endpoint**: `GET /_/api/products?page=1&limit=20`

**Authentication**: Required (admin session)

**Query Parameters**:
- `page` (integer, default: 1): Page number
- `limit` (integer, default: 20): Results per page

**Response**: `200 OK`

```json
{
  "data": [
    {
      "id": "01234567-89ab-cdef-0123-456789abcdef",
      "name": "Premium T-Shirt",
      "slug": "premium-t-shirt",
      "amount": 1999,
      "has_variants": true,
      "options": [...],
      "variants": [...]
    }
  ],
  "total": 50,
  "page": 1,
  "limit": 20
}
```

### CSV Import

**Endpoint**: `POST /_/api/products/csv/import`

**Authentication**: Required (admin session)

**Request Body** (multipart/form-data):

```
file: <CSV file>
```

**Response**: `200 OK`

```json
{
  "imported": 10,
  "skipped": 2,
  "errors": [
    {
      "line": 5,
      "error": "Invalid variant format"
    }
  ]
}
```

**CSV Format**: See [CSV Import/Export Format](#csv-importexport-format) section below.

### CSV Export

**Endpoint**: `GET /_/api/products/csv/export`

**Authentication**: Required (admin session)

**Response**: `200 OK` (Content-Type: text/csv)

Returns CSV file with all products including variants.

## Public API Endpoints

### Get Product by Slug (Public)

**Endpoint**: `GET /api/products/:slug`

**Authentication**: Not required

**Response**: `200 OK`

Returns complete product object with options and variants for active products.

### List Products (Public)

**Endpoint**: `GET /api/products?page=1&limit=20`

**Authentication**: Not required

**Query Parameters**:
- `page` (integer, default: 1): Page number
- `limit` (integer, default: 20): Results per page

**Response**: `200 OK`

Returns list of active products with variant information.

## CSV Import/Export Format

### Basic Format

CSV files must use UTF-8 encoding. Complex fields (options, variants) use semicolon (`;`) and pipe (`|`) delimiters.

### CSV Headers

```csv
name,slug,brief,description,amount,quantity,sku,has_variants,options,variants,attributes,active,images
```

### Field Formats

#### Simple Product (No Variants)

```csv
name,slug,brief,description,amount,quantity,sku,has_variants,options,variants,attributes,active,images
"Basic T-Shirt","basic-tshirt","Simple cotton shirt","High quality cotton t-shirt",1999,100,"BASIC-TSHIRT",false,"","","Casual;Cotton",true,""
```

#### Product with Variants

```csv
name,slug,brief,description,amount,quantity,sku,has_variants,options,variants,attributes,active,images
"Premium T-Shirt","premium-tshirt","Premium cotton shirt","Premium quality t-shirt with multiple options",1999,0,"",true,"Size:S;M;L|Color:Red;Blue;Black","S;Red;TSHIRT-S-RED;0;100|S;Blue;TSHIRT-S-BLUE;200;50|M;Red;TSHIRT-M-RED;0;80","Premium;Cotton",true,""
```

### Options Format

**Pattern**: `OptionName:Value1;Value2;Value3|NextOption:Value1;Value2`

**Example**: `Size:S;M;L|Color:Red;Blue;Black`

This creates:
- Option "Size" with values: S, M, L
- Option "Color" with values: Red, Blue, Black

**Rules**:
- Up to 3 options separated by `|`
- Each option format: `Name:Value1;Value2;...`
- Up to 10 values per option separated by `;`

### Variants Format

**Pattern**: `Value1;Value2;SKU;PriceSurcharge;Quantity|NextVariant...`

**Example**: `S;Red;TSHIRT-S-RED;0;100|S;Blue;TSHIRT-S-BLUE;200;50`

This creates:
- Variant 1: Size=S, Color=Red, SKU=TSHIRT-S-RED, price_surcharge=0 cents, quantity=100
- Variant 2: Size=S, Color=Blue, SKU=TSHIRT-S-BLUE, price_surcharge=200 cents, quantity=50

**Rules**:
- Multiple variants separated by `|`
- Each variant format: `OptionValue1;OptionValue2;...;SKU;PriceSurcharge;Quantity`
- Option values must appear in the same order as options
- Price surcharge in cents (can be negative: `-500` = -$5.00)
- Quantity must be >= 0

### Complete Example

```csv
name,slug,brief,description,amount,quantity,sku,has_variants,options,variants,attributes,active,images
"Premium T-Shirt","premium-tshirt","Premium cotton shirt","High quality cotton t-shirt",1999,0,"",true,"Size:S;M;L|Color:Red;Blue","S;Red;TSHIRT-S-RED;0;100|S;Blue;TSHIRT-S-BLUE;200;50|M;Red;TSHIRT-M-RED;0;80|M;Blue;TSHIRT-M-BLUE;200;60|L;Red;TSHIRT-L-RED;300;40|L;Blue;TSHIRT-L-BLUE;500;30","Premium;Cotton",true,""
"Basic Mug",,"Simple ceramic mug","Classic white ceramic mug",999,200,"MUG-WHITE",false,"","","Kitchen;Ceramic",true,""
```

### Import Validation

The CSV importer validates:

1. **Required fields**: name, amount
2. **Numeric fields**: amount, quantity, price_surcharge must be valid integers
3. **Slug uniqueness**: Auto-generates unique slugs with incremental suffixes if conflicts
4. **Option structure**: 1-3 options, 1-10 values per option
5. **Variant completeness**: Number of option value combinations must match number of variants
6. **Variant format**: Each variant must have correct number of option values + SKU + price + quantity

### Import Behavior

- **Slug conflicts**: Automatically appends `-2`, `-3`, etc. to ensure uniqueness
- **Missing slug**: Auto-generates from product name
- **Validation errors**: Returns detailed error messages with line numbers
- **Transaction safety**: All products in CSV imported atomically or none at all
- **Existing products**: Import creates new products; does not update existing ones

### Export Behavior

- **All products**: Exports all active and inactive products
- **Complete data**: Includes all options and variants
- **Format consistency**: Uses same semicolon/pipe format for re-import compatibility
- **UTF-8 encoding**: Ensures international character support

## Error Responses

All endpoints return standard error responses:

```json
{
  "error": "Error message description",
  "code": "ERROR_CODE"
}
```

**Common Error Codes**:
- `400 Bad Request`: Invalid request data or validation failure
- `401 Unauthorized`: Authentication required
- `404 Not Found`: Product not found
- `500 Internal Server Error`: Server error

## Price Calculation

Final price for a variant:

```
final_price = product.amount + variant.price_surcharge
```

Example:
- Product base price: 1999 cents ($19.99)
- Variant price surcharge: 200 cents ($2.00)
- Final price: 2199 cents ($21.99)

Negative surcharges are supported for discounts:
- Product base price: 1999 cents ($19.99)
- Variant price surcharge: -500 cents (-$5.00)
- Final price: 1499 cents ($14.99)

## Inventory Management

- **Products without variants**: `product.quantity` tracks inventory
- **Products with variants**: Each `variant.quantity` tracks inventory independently
- **Stock status**: quantity = 0 shows as "OUT OF STOCK"
- **Active status**: `variant.active = false` hides variant from storefront regardless of quantity

## Best Practices

1. **SKU Naming**: Use consistent, descriptive SKU patterns (e.g., `PRODUCT-OPTION1-OPTION2`)
2. **Price Surcharges**: Use surcharges for size/quality tiers rather than absolute prices
3. **Inventory**: Set variant quantities accurately; 0 = out of stock
4. **Option Order**: Define options in logical order (Size before Color, License Type before Duration)
5. **CSV Import**: Test with small batches first; validate CSV format before large imports
6. **Validation**: Always validate response data; API returns detailed validation errors
