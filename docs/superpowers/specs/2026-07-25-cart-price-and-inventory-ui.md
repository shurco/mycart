# Cart Unit Price Validation and Inventory UI Improvements

**Date:** 2026-07-25  
**Status:** Approved  
**Implementation:** Pending

## Overview

Two independent frontend improvements to enhance cart validation and admin UX:

1. **Storefront**: Send `unit_price` in cart creation requests to enable server-side price validation
2. **Admin**: Reposition quantity field for better discoverability

## Background

### Current State

**Cart Creation (Storefront)**
- Request payload sends: `id`, `variant_id`, `quantity`
- Missing: `unit_price` field
- Server cannot validate price changes between cart add and checkout

**Admin Product Form**
- Quantity field exists (lines 754-765 in `products/+page.svelte`)
- Location: Between amount field and slug field
- Conditionally hidden when `has_variants=true`
- Users have difficulty finding the field

### Problem

**Cart Validation**
- Backend `ValidateCartItems` expects `unit_price` to detect price changes
- Frontend doesn't send it, so price validation is incomplete
- Price change detection fails silently

**Admin UX**
- Quantity field is conditionally hidden, making it hard to discover
- Field appears before variants section, not logically grouped
- Users don't know where to set inventory for non-variant products

## Goals

1. Enable complete cart validation by sending unit prices
2. Improve inventory field discoverability in admin UI
3. Maintain existing behavior and validation logic
4. No breaking changes to APIs or data structures

## Non-Goals

- Changing cart data structure or storage
- Modifying validation logic (already implemented)
- Adding new validation types
- Changing variant inventory management

## Architecture

### System Components

**Storefront (`web/site`)**
- SvelteKit 5 application
- Cart stored in localStorage via `cartStore`
- Checkout flow in `routes/cart/+page.svelte`

**Admin (`web/admin`)**
- SvelteKit 5 application
- Product management in `routes/products/+page.svelte`
- Form with conditional field visibility

### Data Flow

**Cart Creation with Unit Price**

```
User clicks checkout
    ↓
createCartRecord() maps cart items
    ↓
Include unit_price from CartItem.amount
    ↓
POST /api/cart/create
    ↓
Server ValidateCartItems()
    ├─ Match: Cart created ✓
    └─ Mismatch: 409 with price_changed error
           ↓
       handleValidationErrors() updates cart
```

**Quantity Field Positioning**

```
Admin opens product edit
    ↓
Form renders with fields in order
    ↓
VariantManager component (toggle + editor)
    ↓
Quantity field (if has_variants=false)
    ↓
User toggles variants → field shows/hides
```

## Detailed Design

### Feature 1: Send Unit Price in Cart Creation

**File:** `web/site/src/routes/cart/+page.svelte`

**Change:** Modify `createCartRecord` function (lines 93-101)

**Current implementation:**
```typescript
const cartCreateRes = await apiPost<{ cart_id: string; amount_total: number; currency: string }>('/api/cart/create', {
  email: email,
  provider: 'portone',
  products: cart.map((item) => ({
    id: item.id,
    variant_id: item.variant_id || undefined,
    quantity: item.quantity
  }))
})
```

**Updated implementation:**
```typescript
const cartCreateRes = await apiPost<{ cart_id: string; amount_total: number; currency: string }>('/api/cart/create', {
  email: email,
  provider: 'portone',
  products: cart.map((item) => ({
    id: item.id,
    variant_id: item.variant_id || undefined,
    quantity: item.quantity,
    unit_price: item.amount  // ← Add this line
  }))
})
```

**Rationale:**
- CartItem already has `amount` field (unit price)
- No additional API calls needed
- Server already validates unit_price (backend implemented in previous commit)
- Existing error handling (`handleValidationErrors`) already handles price_changed errors

**Validation Flow:**
1. Client sends expected price from cart
2. Server compares against current product price
3. If mismatch: Returns 409 with `validation_errors` containing `price_changed` error
4. Client shows validation modal with corrected prices
5. User sees highlighted items with updated prices

### Feature 2: Reposition Quantity Field

**File:** `web/admin/src/routes/products/+page.svelte`

**Change:** Move quantity field to appear after VariantManager component

**Current location:** Lines 754-765 (between amount and slug fields)

**Target location:** Immediately after VariantManager component

**Current code:**
```svelte
{#if !formData.has_variants}
  <FormInput
    id="quantity"
    type="number"
    title={t('products.quantity')}
    bind:value={formData.quantity}
    error={formErrors.quantity}
    ico="cube"
    min="0"
    placeholder="0"
  />
{/if}
```

**Action:**
1. Locate VariantManager component in form
2. Cut quantity field block from current location
3. Paste immediately after VariantManager closing tag
4. Maintain exact conditional logic: `{#if !formData.has_variants}`

**Rationale:**
- Logical grouping: inventory settings near variants toggle
- Better discoverability: users see where inventory is managed
- Maintains existing behavior: still hidden when variants enabled
- No data structure changes

**Visual structure:**
```
Before:
  Amount
  Quantity (if !has_variants)
  Slug
  ...
  VariantManager

After:
  Amount
  Slug
  ...
  VariantManager
  Quantity (if !has_variants)
```

## Error Handling

### Cart Price Validation Errors

Existing implementation already handles validation errors:

**File:** `web/site/src/routes/cart/+page.svelte` (lines 59-88)

```typescript
function handleValidationErrors(errors: any[], correctedCart: any[]) {
  validationErrors = errors
  showValidationModal = true
  
  // Updates cart with corrected prices
  // Highlights items with errors
  // Marks unavailable items for deletion
}
```

**No changes needed** - price_changed errors already supported:
- Modal displays validation errors
- Items with price changes are highlighted
- Corrected prices shown to user
- User can review and proceed or cancel

### Admin Form Validation

Quantity field validation already exists:

**File:** `web/admin/src/routes/products/+page.svelte` (line 760)

```svelte
error={formErrors.quantity}
```

**No changes needed** - field position doesn't affect validation

## Testing Plan

### Feature 1: Unit Price Validation

**Test Case 1: Price Match**
1. Add product to cart (price: $50)
2. Proceed to checkout
3. Verify: Cart created successfully
4. Verify: Request includes `unit_price: 5000`

**Test Case 2: Price Change Detection**
1. Add product to cart (price: $50)
2. Admin changes product price to $60
3. User proceeds to checkout
4. Verify: Validation modal appears
5. Verify: Item shows new price ($60)
6. Verify: Item is highlighted
7. Verify: User can see difference (requested: $50, current: $60)

**Test Case 3: Variant Product Price**
1. Add variant product (base: $100, surcharge: $20)
2. Verify cart stores amount: 12000
3. Proceed to checkout
4. Verify: Request includes `unit_price: 12000`
5. Verify: Cart created successfully

**Test Case 4: Mixed Cart**
1. Add regular product (price: $30)
2. Add variant product (price: $120)
3. Proceed to checkout
4. Verify: Both unit_price values sent correctly
5. Verify: Cart created successfully

**Test Case 5: Multiple Price Changes**
1. Add 3 products to cart
2. Admin changes 2 product prices
3. User proceeds to checkout
4. Verify: Validation modal shows 2 price_changed errors
5. Verify: Both items highlighted
6. Verify: Correct prices shown

### Feature 2: Quantity Field Position

**Test Case 1: Non-Variant Product Edit**
1. Open product without variants
2. Verify: Quantity field appears after VariantManager section
3. Verify: Field is visible and editable
4. Edit quantity to 100
5. Save product
6. Verify: Quantity saved correctly

**Test Case 2: Enable Variants**
1. Open product without variants
2. Verify: Quantity field is visible
3. Toggle has_variants to ON
4. Verify: Quantity field disappears
5. Verify: VariantManager opens

**Test Case 3: Disable Variants**
1. Open product with variants
2. Verify: Quantity field not visible
3. Toggle has_variants to OFF
4. Verify: Quantity field appears after VariantManager
5. Verify: Field is editable

**Test Case 4: Create New Product**
1. Click "Add Product"
2. Verify: Quantity field appears after variants section
3. Enter quantity: 50
4. Save
5. Verify: Product created with quantity: 50

**Test Case 5: Product with Existing Variants**
1. Open product that has variants
2. Verify: Quantity field not shown
3. Verify: VariantManager shows variant inventory
4. No changes to variant behavior

### Browser Compatibility

Test in Firefox (user's primary browser based on request headers)

## Implementation Notes

### Cart Unit Price

**Affected files:**
- `web/site/src/routes/cart/+page.svelte` (1 line change)

**Dependencies:**
- Backend validation already implemented (commit: ca34f79)
- CartItem.amount field already populated
- Error handling already exists

**Risk:** Low - additive change only

### Quantity Field Repositioning

**Affected files:**
- `web/admin/src/routes/products/+page.svelte` (move existing code block)

**Dependencies:**
- VariantManager component must be located
- Form structure must be preserved

**Risk:** Very low - no logic changes, only position

## Migration

**No migration needed** - frontend-only changes:
- Cart unit_price is optional field (backend validates when present)
- Existing carts work unchanged
- Admin form just reorders existing field

## Security Considerations

**Cart Unit Price:**
- Client price is for validation only
- Server always uses authoritative database prices
- No trust placed in client-sent price
- Prevents TOCTOU issues (price changes between add and checkout)

**Admin Quantity:**
- No security impact - UI reordering only
- Existing authorization unchanged

## Performance

**Cart Unit Price:**
- No additional API calls
- No additional storage
- Negligible payload size increase (~20 bytes per item)

**Quantity Field:**
- No performance impact
- Same DOM elements, different order

## Future Enhancements

Potential follow-ups (not in scope):

1. Show price change diffs in validation modal (e.g., "$50 → $60 (+$10)")
2. Quantity field with variant summary (total inventory across all variants)
3. Bulk inventory updates in admin
4. Price history tracking

## Success Metrics

**Cart Validation:**
- Price change detection works in 100% of cases
- Validation errors displayed correctly
- No false positives/negatives

**Admin UX:**
- Quantity field discoverable without instructions
- No confusion about where to set inventory
- Maintains existing workflow

## Appendix

### Related Files

**Storefront:**
- `web/site/src/routes/cart/+page.svelte` - Cart checkout page
- `web/site/src/lib/stores/cart.ts` - Cart state management
- `web/site/src/lib/types/models.ts` - CartItem interface

**Admin:**
- `web/admin/src/routes/products/+page.svelte` - Product management page
- `web/admin/src/lib/components/product/VariantManager.svelte` - Variants component

**Backend:**
- `internal/models/cart.go` - CartProduct with UnitPrice field
- `internal/queries/cart.go` - ValidateCartItems with price validation
- `internal/handlers/public/cart.go` - CreateCart handler

### Backend Context

Recent backend implementation (commit ca34f79):
- Added `UnitPrice` field to `CartProduct` model
- Implemented `price_changed` validation in `validateCartItem`
- Implemented `variant_required` validation
- Fixed amount calculation to use validated prices

This frontend implementation completes the price validation feature.
