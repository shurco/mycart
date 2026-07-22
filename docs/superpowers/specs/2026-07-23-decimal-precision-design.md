# Decimal Precision Settings Design Spec

**Date:** 2026-07-23  
**Feature:** Add decimal precision and trailing zeros controls to price formatting

## Overview

Add two global number formatting controls to Payment Settings:
1. **Decimal Precision** (dropdown: 0, 1, or 2) - how many decimal places to show
2. **Show Trailing Zeros** (checkbox) - whether to display trailing zeros (1.00 vs 1)

These settings apply globally to all prices across admin and storefront, for both truncated (K, M, 만) and non-truncated amounts.

## Requirements

### Functional Requirements

1. **Global Settings**: One set of formatting rules applies to all currencies
2. **Shared Context**: Admin and storefront use identical formatting (no separate configs)
3. **Universal Application**: Settings affect both regular prices ($13,520) and truncated prices ($13.52K)
4. **Currency Override**: Global precision overrides currency's natural decimal places (JPY/KRW can show .00 if configured)
5. **Precision-First**: Round to precision first, then apply trailing zeros rule

### Behavior Specification

| Value | Precision | Trailing Zeros | Output |
|-------|-----------|----------------|--------|
| 1.00 | 2 | true | 1.00 |
| 1.00 | 2 | false | 1 |
| 1.23 | 2 | true | 1.23 |
| 1.23 | 2 | false | 1.23 |
| 1.234 | 1 | true | 1.2 |
| 1.234 | 1 | false | 1.2 |
| 1.234 | 0 | either | 1 |
| 1.20 | 2 | true | 1.20 |
| 1.20 | 2 | false | 1.2 |

**Rules:**
- When `precision=0`: Always show integers (trailing zeros setting ignored)
- When `precision=1` or `2`: Apply precision, then trim trailing zeros if `show_trailing_zeros=false`
- Default values: `precision=2`, `show_trailing_zeros=true` (current behavior)

## Architecture

### Data Flow

```
Admin UI (Payment Settings)
  ↓
Save number_format to database (setting table, payment group)
  ↓
Load into payment settings store
  ↓
Pass to formatCurrency() / formatCurrencyWithTruncation()
  ↓
Apply decimal precision rules
  ↓
Display formatted prices across all pages
```

### Component Structure

**Backend:**
- `internal/models/setting.go` - Add `NumberFormatSettings` struct to `Payment` model
- `internal/queries/setting.go` - Handle JSON serialization (existing code reused)
- `internal/handlers/private/setting.go` - Validation for decimal_precision (0-2)

**Frontend (Admin):**
- `web/admin/src/lib/types/models.ts` - Add `NumberFormatSettings` interface
- `web/admin/src/routes/settings/payment/+page.svelte` - New "Number Formatting" section
- `web/admin/src/lib/utils/currency.ts` - Update formatting functions

**Frontend (Storefront):**
- `web/site/src/lib/types/models.ts` - Add `NumberFormatSettings` interface
- `web/site/src/lib/utils/currency.ts` - Update formatting functions
- Price display components automatically use updated formatters

## Data Models

### Backend (Go)

```go
// internal/models/setting.go

type Payment struct {
    Currency     string                 `json:"currency"`
    Truncation   *TruncationSettings    `json:"truncation,omitempty"`
    NumberFormat *NumberFormatSettings  `json:"number_format,omitempty"`  // NEW
}

type NumberFormatSettings struct {
    DecimalPrecision  int  `json:"decimal_precision"`    // 0, 1, or 2
    ShowTrailingZeros bool `json:"show_trailing_zeros"`  // true or false
}
```

**Validation Rules:**
```go
func (v Payment) Validate() error {
    return validation.ValidateStruct(&v,
        validation.Field(&v.Currency, is.CurrencyCode),
        validation.Field(&v.Truncation, validation.By(validateTruncation)),
        validation.Field(&v.NumberFormat, validation.By(validateNumberFormat)), // NEW
    )
}

func validateNumberFormat(value interface{}) error {
    if value == nil {
        return nil // optional
    }
    
    nf, ok := value.(*NumberFormatSettings)
    if !ok || nf == nil {
        return nil
    }
    
    if nf.DecimalPrecision < 0 || nf.DecimalPrecision > 2 {
        return validation.NewError("number_format_invalid_precision",
            "decimal_precision must be 0, 1, or 2")
    }
    
    return nil
}
```

### Frontend (TypeScript)

```typescript
// web/admin/src/lib/types/models.ts
// web/site/src/lib/types/models.ts

export interface NumberFormatSettings {
  decimal_precision: 0 | 1 | 2
  show_trailing_zeros: boolean
}

export interface PaymentSettings {
  currency: string
  truncation?: TruncationSettings
  number_format?: NumberFormatSettings  // NEW
}
```

**Default Values:**
```typescript
const DEFAULT_NUMBER_FORMAT: NumberFormatSettings = {
  decimal_precision: 2,
  show_trailing_zeros: true
}
```

## UI Design

### Payment Settings Page Layout

```
Payment Settings
├─ Currency Selection (existing)
├─ Number Formatting (NEW SECTION)
│  ├─ Decimal Precision dropdown (0, 1, 2)
│  ├─ Show Trailing Zeros checkbox
│  ├─ Preview examples
│  └─ Save button
└─ Price Display Settings (existing truncation settings)
```

### Number Formatting Section (Svelte)

```svelte
<div class="mt-5 max-w-2xl">
  <h2 class="mb-5">Number Formatting</h2>
  
  <FormSelect
    id="decimal-precision"
    title="Decimal Precision"
    options={['0', '1', '2']}
    bind:value={decimalPrecision}
  />
  
  <FormCheckbox
    id="trailing-zeros"
    title="Show Trailing Zeros"
    description="Show 1.00 instead of 1"
    bind:value={showTrailingZeros}
  />
  
  <div class="text-sm text-gray-600 mt-3">
    <div>Preview: 1.00 → {formatPreview(1.00)}</div>
    <div>Preview: 1.23 → {formatPreview(1.23)}</div>
  </div>
  
  <div class="pt-5">
    <FormButton onclick={handleNumberFormatSubmit}>Save</FormButton>
  </div>
</div>
```

### State Management

```typescript
// Load settings
let decimalPrecision = $state('2')
let showTrailingZeros = $state(true)

onMount(() => {
  const nf = payment.number_format || { decimal_precision: 2, show_trailing_zeros: true }
  decimalPrecision = String(nf.decimal_precision)
  showTrailingZeros = nf.show_trailing_zeros
})

// Save settings
async function handleNumberFormatSubmit() {
  payment.number_format = {
    decimal_precision: parseInt(decimalPrecision),
    show_trailing_zeros: showTrailingZeros
  }
  await saveSettings('payment', payment, 'Number formatting saved')
}

// Preview
function formatPreview(value: number): string {
  const nf = {
    decimal_precision: parseInt(decimalPrecision),
    show_trailing_zeros: showTrailingZeros
  }
  return applyDecimalPrecision(value, nf)
}
```

## Formatting Logic

### Core Algorithm

```typescript
// web/admin/src/lib/utils/currency.ts
// web/site/src/lib/utils/currency.ts

function applyDecimalPrecision(
  value: number, 
  settings?: NumberFormatSettings
): string {
  // Default: precision=2, trailing_zeros=true (current behavior)
  const precision = settings?.decimal_precision ?? 2
  const showTrailing = settings?.show_trailing_zeros ?? true
  
  // Step 1: Round to desired precision
  const rounded = value.toFixed(precision)
  
  // Step 2: Remove trailing zeros if disabled
  if (!showTrailing) {
    // parseFloat removes trailing zeros: "1.00" → "1", "1.20" → "1.2"
    return parseFloat(rounded).toString()
  }
  
  // Keep trailing zeros: "1.00", "1.20", "1.23"
  return rounded
}
```

### Updated Function Signatures

```typescript
export function formatCurrency(
  amount: number, 
  currencyCode: string,
  numberFormat?: NumberFormatSettings  // NEW
): string

export function formatCurrencyWithTruncation(
  amount: number,
  currencyCode: string,
  context: 'admin' | 'storefront',
  truncationSettings?: TruncationSettings,
  locale?: string,
  numberFormat?: NumberFormatSettings  // NEW
): string
```

### formatCurrency Implementation

```typescript
export function formatCurrency(
  amount: number, 
  currencyCode: string,
  numberFormat?: NumberFormatSettings
): string {
  const currency = CURRENCIES.find(c => c.code === currencyCode)
  if (!currency) return `${amount} ${currencyCode}`
  
  // Apply decimal precision
  const precision = numberFormat?.decimal_precision ?? 2
  const showTrailing = numberFormat?.show_trailing_zeros ?? true
  
  const formatted = amount.toLocaleString('en-US', {
    minimumFractionDigits: showTrailing ? precision : 0,
    maximumFractionDigits: precision
  })
  
  return `${currency.symbol}${formatted}`
}
```

### formatCurrencyWithTruncation Updates

**Line 57 (fixed mode) and Line 72 (flexible mode):**

Replace:
```typescript
const formatted = divided.toFixed(2)
```

With:
```typescript
const formatted = applyDecimalPrecision(divided, numberFormat)
```

This ensures truncated prices (1.23K, 1.23만) respect the global precision settings.

## Implementation Details

### Database Storage

Settings stored in existing `setting` table under the `payment` key:

```json
{
  "currency": "USD",
  "truncation": { ... },
  "number_format": {
    "decimal_precision": 2,
    "show_trailing_zeros": true
  }
}
```

No schema changes required - JSON serialization handles the new field.

### Backward Compatibility

- Existing installations without `number_format` field default to `precision=2, trailing_zeros=true` (current behavior)
- No migration needed
- Settings are optional (`omitempty` in Go, `?` in TypeScript)

### Component Updates

All price-displaying components automatically benefit from the updated formatting functions. No per-component changes needed:

**Admin:**
- Product list (`/products`)
- Cart list (`/carts`)
- Cart detail view
- TruncationSettings preview

**Storefront:**
- Product cards (`/`)
- Product detail page (`/products/:slug`)
- Cart page (`/cart`)

## Testing

### Backend Tests

**Validation tests** (`internal/models/setting_test.go`):
```go
func TestNumberFormatValidation(t *testing.T) {
    // Valid: precision 0, 1, 2
    // Invalid: precision -1, 3, "invalid"
    // Optional: nil/missing is valid
}
```

**Serialization tests** (`internal/queries/setting_test.go`):
```go
func TestNumberFormatSaveLoad(t *testing.T) {
    // Save settings with number_format
    // Load and verify values match
}
```

### Frontend Tests

**Formatting tests** (`web/admin/src/lib/utils/currency.test.ts`):

```typescript
describe('applyDecimalPrecision', () => {
  test('precision=2, trailing=true', () => {
    expect(apply(1.00, {decimal_precision: 2, show_trailing_zeros: true})).toBe('1.00')
    expect(apply(1.23, {decimal_precision: 2, show_trailing_zeros: true})).toBe('1.23')
  })
  
  test('precision=2, trailing=false', () => {
    expect(apply(1.00, {decimal_precision: 2, show_trailing_zeros: false})).toBe('1')
    expect(apply(1.20, {decimal_precision: 2, show_trailing_zeros: false})).toBe('1.2')
    expect(apply(1.23, {decimal_precision: 2, show_trailing_zeros: false})).toBe('1.23')
  })
  
  test('precision=1', () => {
    expect(apply(1.234, {decimal_precision: 1, show_trailing_zeros: true})).toBe('1.2')
    expect(apply(1.234, {decimal_precision: 1, show_trailing_zeros: false})).toBe('1.2')
  })
  
  test('precision=0', () => {
    expect(apply(1.234, {decimal_precision: 0, show_trailing_zeros: true})).toBe('1')
    expect(apply(1.234, {decimal_precision: 0, show_trailing_zeros: false})).toBe('1')
  })
  
  test('defaults when settings undefined', () => {
    expect(apply(1.00, undefined)).toBe('1.00')  // Default: 2, true
  })
})
```

### Integration Testing Checklist

**Admin Panel:**
1. Navigate to `/settings/payment`
2. Set decimal precision to 0, trailing zeros to false → Save
3. Verify product list shows prices as integers (no decimals)
4. Set decimal precision to 2, trailing zeros to true → Save
5. Verify product list shows prices with .00
6. Refresh page → settings should persist
7. Check cart list and cart detail views → same formatting

**Storefront:**
1. Visit product list (`/`) → verify same formatting as admin
2. Visit product detail → verify same formatting
3. Add to cart and visit `/cart` → verify cart prices and total use same formatting
4. Test with truncated prices (fixed-K mode) → verify 13.52K respects precision

**Edge Cases:**
- JPY currency with precision=2, trailing=true → ¥100.00 (overrides currency default)
- KRW currency with precision=0 → ₩1,352 (integer)
- Very small amounts: $0.01 with precision=0 → $0
- Very large amounts: $1,234,567 with separators

## Migration Plan

**Phase 1: Backend**
1. Add `NumberFormatSettings` struct to `internal/models/setting.go`
2. Add validation function for `decimal_precision`
3. Test serialization/deserialization

**Phase 2: Frontend (Admin)**
1. Add TypeScript interface to `models.ts`
2. Update `currency.ts` formatting functions
3. Add Number Formatting section to Payment Settings page
4. Test save/load functionality

**Phase 3: Frontend (Storefront)**
1. Add TypeScript interface to `models.ts`
2. Update `currency.ts` formatting functions
3. Verify all price displays use updated formatters

**Phase 4: Testing**
1. Unit tests for formatting algorithm
2. Integration tests for admin settings
3. Visual regression tests for storefront
4. Manual testing across all pages

## Success Criteria

- [ ] Admin can set decimal precision (0, 1, or 2)
- [ ] Admin can toggle trailing zeros
- [ ] Settings save to database and persist after refresh
- [ ] All admin prices (products, carts) respect settings
- [ ] All storefront prices (products, cart) respect settings
- [ ] Truncated prices (K, M, 만) respect settings
- [ ] Non-truncated prices respect settings
- [ ] Global settings override currency defaults (JPY can show .00)
- [ ] Defaults maintain backward compatibility (precision=2, trailing=true)
- [ ] Preview in settings page shows accurate examples

## Open Questions

None - all design decisions confirmed during brainstorming.

## References

- Existing truncation feature: `docs/superpowers/specs/2026-07-22-currency-truncation-design.md` (if exists)
- Payment settings: `web/admin/src/routes/settings/payment/+page.svelte`
- Currency formatting: `web/admin/src/lib/utils/currency.ts`, `web/site/src/lib/utils/currency.ts`
- Backend models: `internal/models/setting.go`
- Backend queries: `internal/queries/setting.go`
