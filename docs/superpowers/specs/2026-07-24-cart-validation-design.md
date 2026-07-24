# Cart Validation System Design

**Date:** 2026-07-24  
**Status:** Approved  
**Author:** Claude Sonnet 4.5

## Overview

This design adds comprehensive validation to the cart checkout flow, ensuring quantity availability and price accuracy at the moment of cart creation. When mismatches are detected, the system returns detailed error information and corrected values, allowing the frontend to highlight changes and guide users to resolve issues before payment.

## Requirements

### Functional Requirements

1. **Quantity Validation**
   - Validate requested quantity against current product/variant stock
   - Reject entire item if requested quantity exceeds available (no partial fulfillment)
   - Check variant-specific inventory when variant is selected

2. **Price Validation**
   - Validate unit price (base price + variant surcharge if applicable)
   - Detect price changes between cart build time and checkout time
   - Verify total price calculation accuracy

3. **Error Response**
   - Return HTTP 409 Conflict when validation fails
   - Include detailed error information per item (current vs requested values)
   - Provide corrected cart data for frontend to update display

4. **Frontend Handling**
   - Show modal overlay with validation errors
   - Update cart display with yellow highlights on changed prices/quantities
   - Disable quantity controls for out-of-stock items
   - Require user to remove unavailable items before proceeding

5. **Scope**
   - Apply validation to both `/api/cart/create` (PortOne) and `/cart/payment` (traditional providers)
   - Consistent behavior across all payment flows

### Non-Functional Requirements

- Minimal performance impact (one additional database query)
- Testable validation logic in isolation
- Maintain existing error handling patterns
- No breaking changes to existing API contracts for successful flows

## Architecture Overview

### System Components

**Backend (Go):**

1. **Validation Function** (`internal/queries/cart.go`)
   - `ValidateCartItems(ctx, db, products, currency) -> ValidationResult`
   - Fetches current product/variant data
   - Compares requested vs actual: quantity, unit price, total
   - Returns structured validation errors

2. **Handler Updates** (`internal/handlers/public/cart.go`)
   - `CreateCart` - call validation before `db.AddCart()`
   - `Payment` - call validation before `db.AddCart()`
   - On validation failure: return 409 Conflict with error details
   - On success: proceed with existing cart creation flow

3. **Response Models** (`internal/models/cart.go`)
   - New structs: `CartValidationResult`, `CartValidationError`, `CorrectedCartItem`
   - Fields: item index, product/variant IDs, error type, requested/current values

**Frontend (Svelte):**

1. **Cart Page** (`web/site/src/routes/cart/+page.svelte`)
   - Catch 409 response from cart creation
   - Show modal overlay with validation errors
   - Update cart store with corrected prices/quantities
   - Apply yellow background to changed items
   - Disable quantity controls for out-of-stock items

2. **Cart Store** (`$lib/stores/cart`)
   - Add validation state tracking
   - Method to update items with server-validated data
   - Flag items as "needs_deletion" when quantity unavailable

### Data Flow

```
User clicks checkout
  ↓
Frontend: POST /api/cart/create or /cart/payment
  ↓
Backend: Parse request → ValidateCartItems()
  ↓
Validation checks each item:
  - Variant-specific quantity available?
  - Unit price (base + surcharge) matches current?
  - Total price correct?
  ↓
If validation fails:
  → Return 409 with ValidationErrors + corrected cart data
  ↓
  Frontend: Show modal → Update cart display → Highlight changes
  ↓
If validation passes:
  → Continue with cart creation (existing flow)
```

## Validation Function Design

### Function Signature

```go
// ValidateCartItems validates cart items against current product data
// Returns validation errors if any item has quantity/price mismatch
func ValidateCartItems(
    ctx context.Context,
    db *CartQueries,
    requestedProducts []models.CartProduct,
    requestedCurrency string,
) (*CartValidationResult, error)
```

### Validation Logic

For each cart item, validate in this order:

1. **Product Existence** - Does the product still exist and is active?
2. **Variant Existence** - If variant_id provided, does it exist and is active?
3. **Quantity Available** - For variant: check `variant.Quantity >= requested`. For non-variant: check `product.Quantity >= requested`
4. **Unit Price Match** - Calculate current unit price (base + variant surcharge if applicable), compare to what frontend calculated
5. **Total Price** - Verify `sum(unit_price * quantity)` matches expected total

### Validation Result Structure

```go
type CartValidationResult struct {
    Valid  bool                    // false if any errors
    Errors []CartValidationError   // detailed errors per item
    CorrectedItems []CorrectedCartItem // server-side corrected values
}

type CartValidationError struct {
    ItemIndex      int     // position in cart array
    ProductID      string
    VariantID      *string
    ErrorType      string  // "quantity_unavailable", "price_changed", "product_inactive"
    
    // Quantity fields
    RequestedQty   int
    AvailableQty   int
    
    // Price fields (in cents)
    RequestedUnitPrice  int
    CurrentUnitPrice    int
    RequestedTotal      int
    CurrentTotal        int
}

type CorrectedCartItem struct {
    ProductID       string
    VariantID       *string
    Quantity        int // same as requested, or 0 if unavailable
    UnitPrice       int // current price
    Available       bool // false if should be deleted
}
```

### Error Types

- `"quantity_unavailable"` - Requested quantity exceeds available stock (treat as out of stock)
- `"price_changed"` - Unit price has changed since cart was built
- `"product_inactive"` - Product or variant is no longer active
- `"product_not_found"` - Product doesn't exist in database

## API Response Structure

### Success Response (200 OK)

No changes to existing success responses - validation passed, cart created normally.

### Validation Failure Response (409 Conflict)

**HTTP Status:** 409 Conflict (indicates client data conflicts with server state)

**Response Body:**
```json
{
  "success": false,
  "message": "Cart validation failed",
  "result": {
    "validation_errors": [
      {
        "item_index": 0,
        "product_id": "prod_abc123",
        "variant_id": "var_xyz456",
        "error_type": "quantity_unavailable",
        "requested_qty": 5,
        "available_qty": 0,
        "requested_unit_price": 2999,
        "current_unit_price": 2999,
        "requested_total": 14995,
        "current_total": 0
      },
      {
        "item_index": 1,
        "product_id": "prod_def789",
        "variant_id": null,
        "error_type": "price_changed",
        "requested_qty": 2,
        "available_qty": 10,
        "requested_unit_price": 1999,
        "current_unit_price": 2499,
        "requested_total": 3998,
        "current_total": 4998
      }
    ],
    "corrected_cart": [
      {
        "product_id": "prod_abc123",
        "variant_id": "var_xyz456",
        "quantity": 0,
        "unit_price": 2999,
        "available": false
      },
      {
        "product_id": "prod_def789",
        "variant_id": null,
        "quantity": 2,
        "unit_price": 2499,
        "available": true
      }
    ]
  }
}
```

### Response Fields Explained

**validation_errors[]:**
- Detailed list of what went wrong with each item
- Frontend uses this to show modal/overlay message
- Contains both requested and current values for comparison

**corrected_cart[]:**
- Server's corrected version of the cart
- Frontend uses this to update the cart store
- `available: false` means item should be marked for deletion
- `available: true` with different price means update price and highlight

### Other Error Responses (unchanged)

- **400 Bad Request** - Malformed request, missing required fields
- **500 Internal Server Error** - Database errors, unexpected failures

## Handler Integration

### Integration Points

Both `CreateCart` and `Payment` handlers will integrate validation at the same point in their flow - after binding the request but before creating the cart record.

### CreateCart Handler Changes

**Location:** `internal/handlers/public/cart.go` - `CreateCart()` function (line ~103)

**Current flow:**
1. Bind request → Get currency → Fetch products → Calculate total → Create cart

**New flow:**
1. Bind request → Get currency → Fetch products
2. **→ Validate cart items** ← NEW
3. **→ If validation fails, return 409** ← NEW
4. → Calculate total → Create cart (existing)

**Code insertion point:** After line 124 (`db.ListProducts`) and before line 127 (`var amountTotal`)

### Payment Handler Changes

**Location:** `internal/handlers/public/cart.go` - `Payment()` function (line ~246)

**Current flow:**
1. Bind request → Get settings → Fetch products → Build items → Calculate total → Create cart

**New flow:**
1. Bind request → Get settings → Fetch products
2. **→ Validate cart items** ← NEW
3. **→ If validation fails, return 409** ← NEW
4. → Build items → Calculate total → Create cart (existing)

**Code insertion point:** After line 268 (`db.ListProducts`) and before line 273 (`items := make`)

### Validation Call Pattern

Both handlers will use identical validation logic:

```go
// Validate cart items before processing
validationResult, err := queries.ValidateCartItems(
    c.Context(),
    db,
    payment.Products,
    currency,
)
if err != nil {
    log.ErrorStack(err)
    return webutil.StatusInternalServerError(c)
}

if !validationResult.Valid {
    return webutil.Response(c, fiber.StatusConflict, "Cart validation failed", map[string]any{
        "validation_errors": validationResult.Errors,
        "corrected_cart":    validationResult.CorrectedItems,
    })
}

// Continue with existing flow...
```

### Why This Approach

- **Minimal disruption** - Validation happens right after we have product data, before any calculations
- **Fail fast** - Don't calculate totals or create payment sessions if data is stale
- **Consistent placement** - Same integration pattern in both handlers makes maintenance easier
- **Existing error handling** - Uses established `webutil.Response` pattern

## Frontend Integration

### Cart Page Updates

**Location:** `web/site/src/routes/cart/+page.svelte`

### State Management

Add new reactive state variables:

```typescript
let validationErrors = $state<ValidationError[]>([])
let showValidationModal = $state(false)
let highlightedItems = $state<Set<string>>(new Set())
```

### API Response Handling

Update both `createCartRecord()` and the traditional payment flow to catch 409 responses:

```typescript
async function createCartRecord(email: string, cart: any[]): Promise<string> {
  const cartCreateRes = await apiPost<CartCreateResponse>('/api/cart/create', {
    email: email,
    provider: 'portone',
    products: cart.map((item) => ({
      id: item.id,
      variant_id: item.variant_id || undefined,
      quantity: item.quantity
    }))
  })

  // Handle validation errors (409 Conflict)
  if (cartCreateRes.status === 409) {
    handleValidationErrors(
      cartCreateRes.result.validation_errors,
      cartCreateRes.result.corrected_cart
    )
    throw new Error('Cart validation failed')
  }

  if (!cartCreateRes.success || !cartCreateRes.result?.cart_id) {
    throw new Error('Failed to create cart: ' + (cartCreateRes.message || 'Unknown error'))
  }

  return cartCreateRes.result.cart_id
}
```

### Validation Error Handling

```typescript
function handleValidationErrors(errors: ValidationError[], correctedCart: CorrectedCartItem[]) {
  // Store errors for modal display
  validationErrors = errors
  showValidationModal = true

  // Update cart store with corrected values
  const updatedCart = $cartStore.map((item, index) => {
    const corrected = correctedCart[index]
    const hasError = errors.some(e => e.item_index === index)
    
    if (!corrected.available) {
      // Mark for deletion - disable quantity control
      return { ...item, needsDeletion: true, disabled: true }
    }
    
    if (hasError) {
      // Price changed - update and mark for highlight
      highlightedItems.add(getItemKey(item))
      return { ...item, amount: corrected.unit_price }
    }
    
    return item
  })

  cartStore.set(updatedCart)
}

function getItemKey(item: any): string {
  return item.variant_id ? `${item.id}_${item.variant_id}` : item.id
}
```

### UI Components

**Validation Modal:**
```svelte
{#if showValidationModal}
  <Overlay onclose={() => showValidationModal = false}>
    <div class="validation-modal">
      <h2>{t('cart.validation_errors_title')}</h2>
      <ul>
        {#each validationErrors as error}
          <li>
            {#if error.error_type === 'quantity_unavailable'}
              <p>{t('cart.out_of_stock')} - {t('cart.please_remove')}</p>
            {:else if error.error_type === 'price_changed'}
              <p>{t('cart.price_updated')}: 
                <span class="old-price">{formatCurrency(error.requested_unit_price, currency)}</span>
                → <span class="new-price">{formatCurrency(error.current_unit_price, currency)}</span>
              </p>
            {/if}
          </li>
        {/each}
      </ul>
      <button onclick={() => showValidationModal = false}>
        {t('cart.review_cart')}
      </button>
    </div>
  </Overlay>
{/if}
```

**CartItemCard Highlighting:**

Update `CartItemCard.svelte` to accept `highlighted` and `needsDeletion` props:

```svelte
<script lang="ts">
  let { item, highlighted = false, needsDeletion = false } = $props()
</script>

<div class="cart-item" class:highlighted={highlighted} class:needs-deletion={needsDeletion}>
  <div class="price" class:price-changed={highlighted}>
    {formatCurrency(item.amount, currency)}
  </div>
  
  <input 
    type="number" 
    bind:value={item.quantity}
    disabled={needsDeletion}
  />
</div>

<style>
  .highlighted {
    background-color: #fff9c4; /* Yellow highlight */
    transition: background-color 0.3s ease;
  }
  
  .price-changed {
    background-color: #ffeb3b;
    padding: 2px 4px;
    border-radius: 3px;
  }
  
  .needs-deletion {
    opacity: 0.6;
  }
  
  .needs-deletion input {
    cursor: not-allowed;
  }
</style>
```

### User Flow

1. User clicks checkout
2. API returns 409 with validation errors
3. Modal appears: "Some items need attention"
4. User clicks "Review Cart"
5. Cart page shows with yellow highlights on changed prices
6. Out-of-stock items have disabled quantity controls
7. User removes unavailable items or accepts new prices
8. User clicks checkout again (validation should pass now)

## Error Handling

### Backend Error Scenarios

**1. Database Errors During Validation**
- **Scenario:** `db.ListProducts()` fails during validation
- **Handling:** Log error, return 500 Internal Server Error
- **User Impact:** Generic error message, retry prompt

**2. Partial Product Data**
- **Scenario:** Some products in cart no longer exist in database
- **Handling:** Treat as validation error with type `"product_not_found"`
- **Response:** 409 with error details
- **User Impact:** Item marked for deletion, shown in validation modal

**3. Invalid Request Data**
- **Scenario:** Malformed cart products, negative quantities, missing fields
- **Handling:** Early validation in handler, return 400 Bad Request before calling validation function
- **User Impact:** Should not happen (frontend validation prevents this)

**4. Currency Mismatch**
- **Scenario:** Request currency doesn't match store currency setting
- **Handling:** Not a validation error - existing flow already handles this
- **Note:** Cart uses server's current currency setting

### Frontend Error Scenarios

**1. Network Failure During Checkout**
- **Scenario:** Request to `/api/cart/create` times out or fails
- **Handling:** Catch in try/catch, show error overlay
- **User Action:** Retry checkout button
- **Code:**
```typescript
try {
  const cartId = await createCartRecord(email, cart)
} catch (err) {
  if (err.message === 'Cart validation failed') {
    // Already handled by handleValidationErrors
    return
  }
  error = 'Network error. Please try again.'
  showOverlay = true
}
```

**2. API Returns Unexpected Format**
- **Scenario:** 409 response but missing validation_errors or corrected_cart
- **Handling:** Graceful degradation - show generic error, don't crash
- **Code:**
```typescript
if (cartCreateRes.status === 409) {
  if (!cartCreateRes.result?.validation_errors || !cartCreateRes.result?.corrected_cart) {
    error = 'Validation error occurred. Please refresh and try again.'
    showOverlay = true
    return
  }
  handleValidationErrors(...)
}
```

**3. User Closes Modal Without Fixing Issues**
- **Scenario:** User dismisses validation modal, tries to checkout again without changes
- **Handling:** Same validation errors will occur, modal shows again
- **Note:** This is expected behavior - user must fix issues to proceed

**4. Cart Store Update Fails**
- **Scenario:** `cartStore.set()` throws error (localStorage quota exceeded, etc.)
- **Handling:** Catch and show error, suggest clearing cart
- **Code:**
```typescript
try {
  cartStore.set(updatedCart)
} catch (err) {
  error = 'Failed to update cart. Please clear your cart and try again.'
  showOverlay = true
}
```

### Logging Strategy

**Backend:**
- Log all validation failures at INFO level (expected business logic)
- Log database errors at ERROR level
- Include cart contents hash in logs (for debugging without PII)

**Frontend:**
- Console.log validation errors in development
- Send validation failure events to analytics (if configured)
- Don't log user email or specific product details (privacy)

### Fallback Behavior

**If validation function crashes unexpectedly:**
- Catch error in handler
- Log with stack trace
- Return 500 to frontend
- Frontend shows: "Something went wrong. Please try again or contact support."

**If validation takes too long:**
- Database query timeout (existing SQLite timeout applies)
- Returns error, frontend shows network error message

## Testing Strategy

### Backend Tests

**1. Validation Function Unit Tests**

**Location:** `internal/queries/cart_test.go`

**Test cases:**
- `TestValidateCartItems_Success` - All items valid
- `TestValidateCartItems_QuantityUnavailable` - Requested qty > available
- `TestValidateCartItems_PriceChanged` - Unit price doesn't match
- `TestValidateCartItems_VariantQuantityUnavailable` - Variant-specific stock check
- `TestValidateCartItems_ProductInactive` - Product.Active = false
- `TestValidateCartItems_VariantInactive` - Variant.Active = false
- `TestValidateCartItems_ProductNotFound` - Product doesn't exist
- `TestValidateCartItems_VariantNotFound` - Variant doesn't exist
- `TestValidateCartItems_MultipleErrors` - Multiple items with different errors
- `TestValidateCartItems_PriceSurchargeCorrect` - Variant surcharge included correctly
- `TestValidateCartItems_EmptyCart` - Edge case: no products

**2. Handler Integration Tests**

**Location:** `internal/handlers/public/cart_test.go`

**Test cases:**
- `TestCreateCart_ValidationSuccess` - 200 response, cart created
- `TestCreateCart_ValidationFailure_409` - 409 with validation_errors
- `TestCreateCart_ValidationError_500` - DB error during validation
- `TestPayment_ValidationSuccess` - 200 response, payment URL returned
- `TestPayment_ValidationFailure_409` - 409 with validation_errors
- `TestPayment_QuantityUnavailable` - Specific 409 scenario
- `TestPayment_PriceChanged` - Specific 409 scenario

### Frontend Tests

**1. Unit Tests**

**Location:** `web/site/src/routes/cart/+page.test.ts`

**Test cases:**
```typescript
test('handleValidationErrors updates cart store correctly')
test('handleValidationErrors shows modal')
test('handleValidationErrors highlights changed items')
test('handleValidationErrors marks unavailable items')
test('getItemKey generates correct keys for variants')
```

**2. Browser E2E Tests**

**Location:** `tests/cart-validation.browser.test.ts`

**Test scenarios:**
```typescript
test('checkout with valid cart succeeds')
test('checkout with out-of-stock item shows validation modal')
test('checkout with price change shows validation modal')
test('user can remove unavailable item and retry checkout')
test('validation errors are localized correctly')
```

### Test Data Setup

**Mock Product Data:**
```go
// In stock, normal price
productInStock := &models.Product{
    ID: "prod_test_001",
    Amount: 2999,
    Quantity: 10,
    Active: true,
}

// Out of stock
productOutOfStock := &models.Product{
    ID: "prod_test_002",
    Amount: 1999,
    Quantity: 0,
    Active: true,
}

// Price changed
productPriceChanged := &models.Product{
    ID: "prod_test_003",
    Amount: 3499, // was 2999
    Quantity: 5,
    Active: true,
}

// With variant
productWithVariant := &models.Product{
    ID: "prod_test_004",
    Amount: 2000,
    HasVariants: true,
    Variants: []models.ProductVariant{
        {
            ID: "var_test_001",
            PriceSurcharge: 500,
            Quantity: 3,
            Active: true,
        },
    },
}
```

### Testing Workflow

**Pre-commit:**
1. Run backend unit tests (`go test ./internal/queries/...`)
2. Run handler tests (`go test ./internal/handlers/...`)
3. Run frontend unit tests (`npm test`)

**CI Pipeline:**
1. All unit tests
2. Integration tests
3. E2E browser tests with validation scenarios
4. Coverage check (aim for 80%+ on validation logic)

### Manual Testing Checklist

Before deployment, manually verify:
- [ ] Out-of-stock item prevents checkout, shows modal
- [ ] Price change updates cart display with yellow highlight
- [ ] Multiple validation errors show all in modal
- [ ] Variant-specific quantity validation works
- [ ] Modal is localized in all supported languages (en, ko, zh)
- [ ] Disabled quantity controls for unavailable items
- [ ] Remove unavailable item, retry checkout succeeds
- [ ] Network error shows appropriate message
- [ ] Works in both PortOne (`/api/cart/create`) and traditional (`/cart/payment`) flows

## Implementation Files

### Backend Files to Modify/Create

1. **`internal/models/cart.go`**
   - Add: `CartValidationResult`, `CartValidationError`, `CorrectedCartItem` structs

2. **`internal/queries/cart.go`**
   - Add: `ValidateCartItems()` function

3. **`internal/handlers/public/cart.go`**
   - Modify: `CreateCart()` - add validation call
   - Modify: `Payment()` - add validation call

4. **`internal/queries/cart_test.go`**
   - Add: Unit tests for `ValidateCartItems()`

5. **`internal/handlers/public/cart_test.go`**
   - Add: Integration tests for validation in handlers

### Frontend Files to Modify/Create

1. **`web/site/src/routes/cart/+page.svelte`**
   - Add: Validation error handling
   - Add: Modal component
   - Modify: `createCartRecord()` to catch 409
   - Modify: Payment flow to catch 409

2. **`web/site/src/lib/components/CartItemCard.svelte`**
   - Add: `highlighted` and `needsDeletion` props
   - Add: Yellow highlight styles
   - Add: Disabled state for quantity controls

3. **`web/site/src/lib/i18n/locales/en.json`**
   - Add: Validation error messages

4. **`web/site/src/lib/i18n/locales/ko.json`**
   - Add: Validation error messages (Korean)

5. **`web/site/src/lib/i18n/locales/zh.json`**
   - Add: Validation error messages (Chinese)

6. **`web/site/src/routes/cart/+page.test.ts`**
   - Add: Unit tests for validation handling

7. **`tests/cart-validation.browser.test.ts`**
   - Add: E2E tests for validation scenarios

## Internationalization (i18n)

### Required Translation Keys

**English (`en.json`):**
```json
{
  "cart": {
    "validation_errors_title": "Some items need attention",
    "out_of_stock": "This item is no longer available",
    "please_remove": "Please remove it to continue",
    "price_updated": "Price has been updated",
    "review_cart": "Review Cart",
    "quantity_unavailable": "Requested quantity not available"
  }
}
```

**Korean (`ko.json`):**
```json
{
  "cart": {
    "validation_errors_title": "일부 항목에 주의가 필요합니다",
    "out_of_stock": "이 상품은 더 이상 사용할 수 없습니다",
    "please_remove": "계속하려면 제거하세요",
    "price_updated": "가격이 업데이트되었습니다",
    "review_cart": "장바구니 검토",
    "quantity_unavailable": "요청한 수량을 사용할 수 없습니다"
  }
}
```

**Chinese (`zh.json`):**
```json
{
  "cart": {
    "validation_errors_title": "某些商品需要注意",
    "out_of_stock": "此商品已不可用",
    "please_remove": "请将其删除以继续",
    "price_updated": "价格已更新",
    "review_cart": "查看购物车",
    "quantity_unavailable": "请求的数量不可用"
  }
}
```

## Security Considerations

1. **Price Tampering Prevention**
   - Never trust client-provided prices
   - Always recalculate from server-side product data
   - Validation ensures client can't manipulate totals

2. **Quantity Manipulation**
   - Validate against actual inventory
   - Prevent overselling through strict validation
   - No partial fulfillment reduces complexity

3. **Data Exposure**
   - Validation errors don't expose sensitive business data
   - Only return data client already had access to (prices, quantities)
   - No internal error details in production

4. **Rate Limiting**
   - Existing rate limiting applies to cart endpoints
   - Validation doesn't add new attack vectors

## Performance Considerations

1. **Query Optimization**
   - `ValidateCartItems()` reuses `db.ListProducts()` query pattern
   - Single query fetches all needed product/variant data
   - No N+1 query issues

2. **Response Size**
   - 409 response adds ~500 bytes per validation error
   - Typical case: 1-3 errors = <2KB additional data
   - Acceptable overhead for correctness

3. **Caching**
   - Product data is not cached (must be current)
   - Cart validation always queries fresh data
   - This is intentional for accuracy

4. **Database Load**
   - One additional query per checkout attempt
   - Validation failures prevent unnecessary cart creation
   - Net neutral or positive impact

## Future Enhancements

### Out of Scope for Initial Implementation

1. **Stock Reservation**
   - Reserve inventory during cart validation
   - Hold for N minutes to prevent race conditions
   - Requires inventory locking system

2. **Partial Fulfillment**
   - Allow reducing quantity to maximum available
   - Auto-adjust instead of rejecting
   - Adds UX complexity

3. **Price Change Notifications**
   - Email users when cart items change price
   - Proactive notification before checkout
   - Requires background job system

4. **Validation Caching**
   - Cache validation results for N seconds
   - Reduce duplicate validation on retry
   - Adds staleness risk

### Migration Path

These enhancements can be added incrementally:
- Stock reservation: Extend `ValidateCartItems()` to call reservation system
- Partial fulfillment: Change validation logic, update frontend handling
- Notifications: Add webhook/job after validation failure
- Caching: Add cache layer around validation function

## Conclusion

This design provides robust cart validation that:
- Ensures data accuracy at checkout time
- Provides clear user feedback when issues occur
- Maintains consistency across all payment flows
- Follows existing code patterns and conventions
- Is testable, maintainable, and extensible

The shared validation function approach balances simplicity with correctness, preventing overselling and price discrepancies without adding significant complexity to the system.
