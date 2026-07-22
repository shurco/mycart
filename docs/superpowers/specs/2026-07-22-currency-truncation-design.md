# Currency Unit Truncation Design

**Date:** 2026-07-22  
**Status:** Approved  
**Author:** Claude (with user approval)

## Overview

Add currency unit truncation to simplify price displays across admin and storefront. Support three modes: None (full precision), Fixed (admin-selected unit), and Flexible (auto-selected unit). Settings are per-currency with separate configuration for admin panel vs storefront.

## Requirements

### Functional Requirements

1. **Three truncation modes:**
   - **None:** Display full precision (e.g., $1,352.00, ₩1,352)
   - **Fixed:** Admin chooses specific unit (e.g., 1K, 만)
   - **Flexible:** Automatic unit selection based on amount

2. **Per-currency configuration:**
   - Each currency has its own truncation settings
   - Settings independent for admin panel vs storefront
   - Supported abbreviations differ by currency

3. **Currency patterns:**
   - **USD pattern** (K/M with 1000 threshold): USD, GBP, EUR, CAD
   - **KRW pattern** (천, 만, 십만, 백만, 천만, 억, 십억, 백억): KRW, JPY

4. **Formatting rules:**
   - Always show 2 decimal places for truncated amounts (e.g., 1.35K, 13.52만)
   - Show full amount if below selected unit threshold (e.g., $50 with 1K → $50.00)
   - Localize units based on language (만 in Korean, 10K in English)

5. **Flexible mode thresholds:**
   - USD pattern: 999 → full, 1000 → 1.00K, 999,999 → 999.99K, 1,000,000 → 1.00M
   - KRW pattern: Progressive thresholds at 천(1K), 만(10K), 십만(100K), etc.

6. **Default behavior:**
   - New installations default to "None" mode
   - Existing installations preserve current display (backward compatible)

### Non-Functional Requirements

- Fast performance (client-side formatting)
- Testable with TDD approach
- Backward compatible with existing price displays
- Localizable unit labels
- Mobile-responsive settings UI

## Architecture

### Three-Layer Design

**Layer 1: Backend (Settings Storage)**
- Extend `PaymentSettings` to store truncation config
- Store as JSON in existing settings system
- Provide via existing `/api/settings/payment` endpoint
- Minimal backend logic (validation only)

**Layer 2: Frontend Configuration (Static Definitions)**
- Unit definitions: maps units to numeric values (K→1000, 만→10000)
- Currency pattern templates (USD pattern, KRW pattern)
- Localization files with translated unit labels

**Layer 3: Frontend Logic (Formatting)**
- Enhanced `formatCurrency` utility
- Reads truncation settings from store
- Applies logic based on mode/unit/locale
- Separate contexts for admin vs storefront

**Data Flow:**
```
Backend Settings → Frontend Config + Unit Definitions + i18n → formatCurrency → Display
```

## Data Models

### Backend: PaymentSettings

```typescript
interface CurrencyTruncationSettings {
  mode: 'none' | 'fixed' | 'flexible'
  fixed_unit?: string  // e.g., 'K', 'M', '만', '천' (only when mode='fixed')
}

interface PaymentSettings {
  currency: string
  truncation: {
    admin: Record<string, CurrencyTruncationSettings>      // keyed by currency code
    storefront: Record<string, CurrencyTruncationSettings> // keyed by currency code
  }
}
```

**Example JSON:**
```json
{
  "currency": "USD",
  "truncation": {
    "admin": {
      "USD": { "mode": "none" },
      "KRW": { "mode": "flexible" }
    },
    "storefront": {
      "USD": { "mode": "flexible" },
      "KRW": { "mode": "fixed", "fixed_unit": "만" }
    }
  }
}
```

### Frontend: Unit Definitions

```typescript
interface UnitDefinition {
  value: number                      // numeric value (1000, 10000, etc.)
  keys: Record<string, string>       // localized keys: { en: 'K', ko: '천' }
}

interface CurrencyPattern {
  type: 'usd' | 'krw'
  units: UnitDefinition[]            // sorted largest to smallest
}
```

**Configuration:** `web/*/src/lib/config/currencyUnits.ts`

```typescript
export const CURRENCY_PATTERNS: Record<string, CurrencyPattern> = {
  'USD': {
    type: 'usd',
    units: [
      { value: 1000000, keys: { en: 'M' } },
      { value: 1000, keys: { en: 'K' } }
    ]
  },
  'GBP': { type: 'usd', /* same as USD */ },
  'EUR': { type: 'usd', /* same as USD */ },
  'CAD': { type: 'usd', /* same as USD */ },
  
  'KRW': {
    type: 'krw',
    units: [
      { value: 10000000000, keys: { en: '10B', ko: '백억' } },
      { value: 1000000000, keys: { en: '1B', ko: '십억' } },
      { value: 100000000, keys: { en: '100M', ko: '억' } },
      { value: 10000000, keys: { en: '10M', ko: '천만' } },
      { value: 1000000, keys: { en: '1M', ko: '백만' } },
      { value: 100000, keys: { en: '100K', ko: '십만' } },
      { value: 10000, keys: { en: '10K', ko: '만' } },
      { value: 1000, keys: { en: '1K', ko: '천' } }
    ]
  },
  'JPY': { type: 'krw', /* same as KRW */ }
}
```

## Implementation Details

### Backend Changes

**Minimal backend work - mainly storage and validation:**

1. **Settings Model** (`internal/models/settings.go`):
   - Add `Truncation` field to `PaymentSettings` struct
   - Nested structure for admin/storefront contexts

2. **API Endpoints** (existing):
   - `GET /api/settings/payment` - returns settings with truncation
   - `POST /api/settings/payment` - saves truncation settings
   - No new endpoints needed

3. **Validation:**
   - Mode must be 'none', 'fixed', or 'flexible'
   - Fixed mode requires `fixed_unit`
   - Unit must exist in currency's pattern

4. **Migration:**
   - Add default truncation structure to existing PaymentSettings
   - Default all currencies to mode='none'
   - Run on application startup if truncation field missing

### Frontend Configuration

**Helper Functions:**

```typescript
// Get pattern for a currency (falls back to USD pattern)
function getCurrencyPattern(currencyCode: string): CurrencyPattern

// Get localized unit label
function getUnitLabel(unitValue: number, currencyCode: string, locale: string): string

// Auto-select best unit for flexible mode
function selectFlexibleUnit(amount: number, pattern: CurrencyPattern): UnitDefinition | null
```

### Core Formatting Logic

**Enhanced formatCurrency utility:**

**File:** `web/*/src/lib/utils/currency.ts`

**Function Signature:**
```typescript
function formatCurrencyWithTruncation(
  amount: number,                   // raw amount in cents
  currencyCode: string,             // e.g., 'USD', 'KRW'
  context: 'admin' | 'storefront',
  truncationSettings?: PaymentSettings['truncation'],
  locale?: string                   // e.g., 'en', 'ko'
): string
```

**Logic Flow:**

1. **Get truncation config** for currency + context
2. **Mode: None**
   - Use existing formatCurrency logic
   - Return full formatted amount
3. **Mode: Fixed**
   - Get unit value from pattern
   - If `amount >= unit value`:
     - Divide: `divided = amount / unit value`
     - Format: `divided.toFixed(2)`
     - Get localized unit label
     - Return: `${symbol}${divided}${unitLabel}`
   - Else: return full amount (bypass truncation)
4. **Mode: Flexible**
   - Get currency pattern
   - Find largest unit where `amount >= unit value`
   - Apply truncation as in Fixed mode
   - If no unit found: return full amount

**Example Transformations:**

| Amount | Currency | Mode | Unit/Locale | Output |
|--------|----------|------|-------------|--------|
| 135200 | USD | flexible | en | $1,352.00 |
| 1352000 | USD | flexible | en | $13.52K |
| 156200000 | USD | flexible | en | $1.56M |
| 1000000 | KRW | fixed | 만, ko | ₩100.00만 |
| 1000000 | KRW | fixed | 만, en | ₩100.0010K |
| 50000 | KRW | fixed | 만, ko | ₩50,000 |
| 15000000 | KRW | flexible | ko | ₩1.50천만 |
| 15000000 | KRW | flexible | en | ₩15.00M |

### UI Components

**Location:** `web/admin/src/routes/settings/payment/+page.svelte`

**New Section: Price Display Settings**

Added below existing currency selection:

```
─────────────────────────────────
Price Display Settings

┌─ Admin Panel ──────────────────┐
│ USD                            │
│   Mode: [Dropdown: None ▼]    │
│                                │
│ KRW                            │
│   Mode: [Dropdown: Fixed ▼]   │
│   Unit: [Dropdown: 만 ▼]      │
│   Preview: ₩1,352 → ₩13.52만   │
└────────────────────────────────┘

┌─ Storefront ───────────────────┐
│ USD                            │
│   Mode: [Dropdown: Flexible ▼]│
│   Preview: $13,520 → $13.52K   │
│                                │
│ KRW                            │
│   Mode: [Dropdown: None ▼]    │
└────────────────────────────────┘

[Save Settings]
─────────────────────────────────
```

**Component Structure:**

- **TruncationSettings.svelte** (new component)
  - Props: `currency`, `context`, `settings`, `onChange`
  - Displays mode dropdown
  - Conditionally shows unit dropdown (only for fixed mode)
  - Shows live preview of truncation
  - Reuses `FormSelect` component

- **Payment settings page** (updated)
  - Loops through all currencies
  - Renders two sections: Admin and Storefront
  - Each section contains TruncationSettings for each currency
  - Save button persists all settings

**Unit Dropdown Options:**

Dynamically populated based on currency:
- USD: K, M
- KRW (Korean locale): 천, 만, 십만, 백만, 천만, 억, 십억, 백억
- KRW (English locale): 1K, 10K, 100K, 1M, 10M, 100M, 1B, 10B

### Integration Points

**Files to Update:**

**Admin Panel:**
- `web/admin/src/routes/products/+page.svelte` - product list
- `web/admin/src/routes/carts/+page.svelte` - cart list
- `web/admin/src/lib/components/cart/View.svelte` - cart detail
- Product edit forms
- Dashboard/reports

**Storefront:**
- `web/site/src/lib/components/ProductCard.svelte` - product cards
- `web/site/src/routes/products/[slug]/+page.svelte` - product detail
- `web/site/src/routes/cart/+page.svelte` - cart page
- `web/site/src/routes/cart/payment/success/+page.svelte` - confirmation

**Update Pattern:**

```typescript
// Old
costFormat(product.amount)

// New
import { formatCurrencyWithTruncation } from '$lib/utils/currency'
import { settingsStore } from '$lib/stores/settings'
import { locale } from '$lib/i18n'

formatCurrencyWithTruncation(
  product.amount,
  $settingsStore?.payment?.currency || 'USD',
  'storefront',  // or 'admin'
  $settingsStore?.payment?.truncation,
  $locale
)
```

**Settings Store Enhancement:**

Ensure `settingsStore` loads truncation settings:
- Load on app initialization
- Available globally via Svelte store
- SSR-compatible

## Testing Strategy

### Unit Tests

**File:** `web/*/src/lib/utils/currency.test.ts`

**Test Suites:**

1. **Unit value calculations**
   ```typescript
   expect(CURRENCY_PATTERNS['USD'].units[1].value).toBe(1000)
   expect(CURRENCY_PATTERNS['KRW'].units[6].value).toBe(10000)
   ```

2. **Flexible mode auto-selection**
   ```typescript
   // USD
   expect(selectFlexibleUnit(999, 'USD')).toBe(null)
   expect(selectFlexibleUnit(1000, 'USD').value).toBe(1000) // K
   expect(selectFlexibleUnit(999999, 'USD').value).toBe(1000) // K
   expect(selectFlexibleUnit(1000000, 'USD').value).toBe(1000000) // M
   
   // KRW
   expect(selectFlexibleUnit(999, 'KRW')).toBe(null)
   expect(selectFlexibleUnit(1000, 'KRW').value).toBe(1000) // 천
   expect(selectFlexibleUnit(10000, 'KRW').value).toBe(10000) // 만
   ```

3. **Fixed mode truncation**
   ```typescript
   // Amount >= unit
   expect(formatCurrency(135200, 'USD', 'storefront', {mode:'fixed', unit:'K'}, 'en'))
     .toBe('$1.35K')
   
   // Amount < unit
   expect(formatCurrency(50000, 'USD', 'storefront', {mode:'fixed', unit:'K'}, 'en'))
     .toBe('$500.00')
   ```

4. **None mode**
   ```typescript
   expect(formatCurrency(135200, 'USD', 'storefront', {mode:'none'}, 'en'))
     .toBe('$1,352.00')
   ```

5. **Localization**
   ```typescript
   expect(formatCurrency(1000000, 'KRW', 'storefront', {mode:'fixed', unit:'만'}, 'ko'))
     .toBe('₩100.00만')
   expect(formatCurrency(1000000, 'KRW', 'storefront', {mode:'fixed', unit:'만'}, 'en'))
     .toBe('₩100.0010K')
   ```

6. **Edge cases**
   ```typescript
   // Zero amount
   expect(formatCurrency(0, 'USD', 'storefront', {mode:'flexible'}, 'en'))
     .toBe('$0.00')
   
   // Very large amount
   expect(formatCurrency(999999999900, 'USD', 'storefront', {mode:'flexible'}, 'en'))
     .toBe('$9,999.99M')
   
   // Missing settings (fallback to none)
   expect(formatCurrency(135200, 'USD', 'storefront', undefined, 'en'))
     .toBe('$1,352.00')
   ```

### Integration Tests

1. **Settings persistence**
   - POST truncation settings → GET → verify correct data returned
   - Invalid mode → 400 error
   - Fixed without unit → 400 error

2. **Component tests** (Svelte Testing Library)
   - TruncationSettings component renders correctly
   - Unit dropdown hidden when mode != 'fixed'
   - Preview updates when settings change

3. **E2E tests**
   - Admin sets truncation to flexible
   - View product on storefront
   - Verify truncated price displayed

### Test Data Fixtures

```typescript
const TEST_CASES = [
  { amount: 135200, currency: 'USD', mode: 'none', locale: 'en', expected: '$1,352.00' },
  { amount: 135200, currency: 'USD', mode: 'flexible', locale: 'en', expected: '$1.35K' },
  { amount: 1352000, currency: 'USD', mode: 'flexible', locale: 'en', expected: '$13.52K' },
  { amount: 156200000, currency: 'USD', mode: 'flexible', locale: 'en', expected: '$1.56M' },
  { amount: 50000, currency: 'USD', mode: 'fixed', unit: 'K', locale: 'en', expected: '$500.00' },
  { amount: 1000000, currency: 'KRW', mode: 'fixed', unit: '만', locale: 'ko', expected: '₩100.00만' },
  { amount: 1000000, currency: 'KRW', mode: 'fixed', unit: '만', locale: 'en', expected: '₩100.0010K' },
  { amount: 50000, currency: 'KRW', mode: 'fixed', unit: '만', locale: 'ko', expected: '₩50,000' },
  { amount: 15000000, currency: 'KRW', mode: 'flexible', locale: 'ko', expected: '₩1.50천만' },
  { amount: 15000000, currency: 'KRW', mode: 'flexible', locale: 'en', expected: '₩15.00M' },
]
```

## Migration & Rollout

### Database Migration

**On application startup:**

```go
// Check if truncation field exists in payment settings
settings := loadPaymentSettings()
if settings.Truncation == nil {
    // Initialize with defaults
    settings.Truncation = &TruncationSettings{
        Admin: make(map[string]CurrencyTruncationSettings),
        Storefront: make(map[string]CurrencyTruncationSettings),
    }
    
    // Set all currencies to 'none' mode
    for _, currency := range CURRENCIES {
        settings.Truncation.Admin[currency.Code] = CurrencyTruncationSettings{Mode: "none"}
        settings.Truncation.Storefront[currency.Code] = CurrencyTruncationSettings{Mode: "none"}
    }
    
    savePaymentSettings(settings)
}
```

### Backward Compatibility

**Frontend graceful degradation:**

```typescript
// Default to 'none' if settings missing
const truncationSettings = settings?.payment?.truncation || {
  admin: {},
  storefront: {}
}

// Default to 'none' if currency missing
const currencySettings = truncationSettings[context][currencyCode] || { mode: 'none' }
```

**Existing code compatibility:**
- Keep old `costFormat` function (mark as deprecated)
- Existing `formatCurrency` calls continue to work
- New `formatCurrencyWithTruncation` is additive

### Rollout Phases

**Phase 1: Backend + Core Logic**
- Add backend settings fields and validation
- Implement `formatCurrencyWithTruncation` utility
- Write and pass all unit tests
- Deploy with all defaults = 'none'
- **Result:** No visible changes, feature dormant

**Phase 2: Admin UI**
- Add truncation settings to payment page
- Test UI in staging
- Deploy
- **Result:** Admins can configure, but prices still use old formatting

**Phase 3: Admin Panel Integration**
- Update admin price displays to use new formatting
- Test with various settings
- Deploy
- **Result:** Admin panel respects truncation settings

**Phase 4: Storefront Integration**
- Update storefront price displays
- Test across currencies and modes
- Deploy
- **Result:** Customers see truncated prices

**Phase 5: Cleanup (optional, later)**
- Remove deprecated `costFormat` function
- Remove feature flag if used

### Feature Flag (Optional)

Environment variable for quick rollback:

```bash
ENABLE_PRICE_TRUNCATION=true
```

Check in formatting function:
```typescript
if (!import.meta.env.ENABLE_PRICE_TRUNCATION) {
  return formatCurrency(amount, currencyCode) // old behavior
}
```

## Success Criteria

1. **Functionality:**
   - All three modes (none, fixed, flexible) work correctly
   - Settings persist across sessions
   - Localization works for all supported languages
   - Edge cases handled gracefully

2. **Performance:**
   - Formatting takes < 1ms per call
   - No visible lag in product listings
   - Settings load time < 100ms

3. **Testing:**
   - 100% unit test coverage for core logic
   - All integration tests pass
   - No regressions in existing price displays

4. **User Experience:**
   - Settings UI is intuitive and clear
   - Preview helps admins understand impact
   - Prices display correctly in all contexts

5. **Backward Compatibility:**
   - Existing installations upgrade smoothly
   - No breaking changes to API
   - Old code continues to work

## Open Questions

None - all design questions resolved during brainstorming phase.

## References

- Current implementation: `web/*/src/lib/utils/currency.ts`
- Currency config: `web/*/src/lib/config/currencies.ts`
- Payment settings: `web/admin/src/routes/settings/payment/+page.svelte`
- Product card: `web/site/src/lib/components/ProductCard.svelte`

## Appendix: Currency Unit Tables

### USD Pattern (K/M)

| Unit | Value | Threshold | Example Input | Example Output |
|------|-------|-----------|---------------|----------------|
| K | 1,000 | 1,000 | 1,352 | $1.35K |
| M | 1,000,000 | 1,000,000 | 1,562,000 | $1.56M |

### KRW Pattern (Korean Units)

| Unit (KO) | Unit (EN) | Value | Threshold | Example Input | Example Output (KO) | Example Output (EN) |
|-----------|-----------|-------|-----------|---------------|---------------------|---------------------|
| 천 | 1K | 1,000 | 1,000 | 1,000 | ₩1.00천 | ₩1.001K |
| 만 | 10K | 10,000 | 10,000 | 13,520 | ₩1.35만 | ₩1.3510K |
| 십만 | 100K | 100,000 | 100,000 | 135,200 | ₩1.35십만 | ₩1.35100K |
| 백만 | 1M | 1,000,000 | 1,000,000 | 1,352,000 | ₩1.35백만 | ₩1.351M |
| 천만 | 10M | 10,000,000 | 10,000,000 | 13,520,000 | ₩1.35천만 | ₩1.3510M |
| 억 | 100M | 100,000,000 | 100,000,000 | 135,200,000 | ₩1.35억 | ₩1.35100M |
| 십억 | 1B | 1,000,000,000 | 1,000,000,000 | 1,352,000,000 | ₩1.35십억 | ₩1.351B |
| 백억 | 10B | 10,000,000,000 | 10,000,000,000 | 13,520,000,000 | ₩1.35백억 | ₩1.3510B |
