# Currency Symbol Display Mode - Design Spec

**Date:** 2026-01-23  
**Feature:** Toggle between currency symbols and language names for price display

## Overview

Add a setting to switch between two display modes for currency:
- **Currency Symbol mode**: `$130`, `¥130`, `₩130`
- **Language Symbol mode**: `130 Dollar`, `130 Yen`, `130 Won`

The setting is separate for admin panel and storefront, allowing different display preferences for each context.

## Requirements

### Functional Requirements

1. **Two Display Modes**
   - Currency Symbol: Use currency symbol (e.g., $, ¥, ₩) before the number
   - Language Symbol: Use currency name after the number with space (e.g., "130 Dollar")

2. **Separate Admin/Storefront Settings**
   - Admin panel can use Currency Symbol mode
   - Storefront can use Language Symbol mode
   - Settings are independent

3. **Multi-language Support**
   - Currency names in English, Korean, and Chinese
   - Uses UI locale to determine which translation to show
   - Format: `[number] [currency_name_in_locale]`

4. **Truncation Integration**
   - Currency Symbol + truncation: `$1.5K`
   - Language Symbol + truncation: `1.5K Dollar`
   - Format: `[number][unit] [currency_name]`

5. **Backward Compatibility**
   - Default: Currency Symbol mode (current behavior)
   - Existing installations continue working without changes
   - Optional field with sensible defaults

### Non-Functional Requirements

1. **Performance**: No impact on formatting performance
2. **Validation**: Backend validates mode values are 'currency' or 'language'
3. **Fallback**: Missing translations fall back to English

## Data Model

### Backend (Go)

**New Type:**
```go
// SymbolDisplaySettings defines currency display mode per context
type SymbolDisplaySettings struct {
    Admin      string `json:"admin"`      // "currency" or "language"
    Storefront string `json:"storefront"` // "currency" or "language"
}
```

**Extended Payment Struct:**
```go
type Payment struct {
    Currency      string                  `json:"currency"`
    Truncation    *TruncationSettings     `json:"truncation,omitempty"`
    NumberFormat  *NumberFormatSettings   `json:"number_format,omitempty"`
    SymbolDisplay *SymbolDisplaySettings  `json:"symbol_display,omitempty"` // NEW
}
```

**Validation:**
```go
func validateSymbolDisplay(value interface{}) error {
    if value == nil {
        return nil // optional field
    }
    
    sd, ok := value.(*SymbolDisplaySettings)
    if !ok || sd == nil {
        return nil
    }
    
    validModes := map[string]bool{"currency": true, "language": true}
    
    if !validModes[sd.Admin] {
        return validation.NewError("invalid_symbol_display", 
            "admin symbol_display must be 'currency' or 'language'")
    }
    
    if !validModes[sd.Storefront] {
        return validation.NewError("invalid_symbol_display",
            "storefront symbol_display must be 'currency' or 'language'")
    }
    
    return nil
}
```

### Frontend (TypeScript)

**Type Definitions:**
```typescript
interface SymbolDisplaySettings {
  admin: 'currency' | 'language'
  storefront: 'currency' | 'language'
}

interface PaymentSettings {
  currency: string
  truncation?: TruncationSettings
  number_format?: NumberFormatSettings
  symbol_display?: SymbolDisplaySettings  // NEW
}

interface CurrencyConfig {
  code: string
  symbol: string
  decimals: number
  name: string  // Keep for backward compatibility
  names: {      // NEW - localized names
    en: string
    ko: string
    zh: string
  }
}
```

**Default Values:**
```typescript
const defaultSymbolDisplay: SymbolDisplaySettings = {
  admin: 'currency',
  storefront: 'currency'
}
```

## Translation Data

### Currency Names in 3 Languages

| Currency | English | Korean | Chinese |
|----------|---------|--------|---------|
| USD | Dollar | 달러 | 美元 |
| EUR | Euro | 유로 | 欧元 |
| GBP | Pound | 파운드 | 英镑 |
| JPY | Yen | 엔 | 日元 |
| KRW | Won | 원 | 韩元 |
| AUD | Dollar | 달러 | 澳元 |
| CAD | Dollar | 달러 | 加元 |
| CHF | Franc | 프랑 | 瑞郎 |
| CNY | Yuan | 위안 | 元 |
| SEK | Krona | 크로나 | 克朗 |

### Updated CURRENCIES Config

```typescript
export const CURRENCIES: CurrencyConfig[] = [
  { 
    code: 'USD', symbol: '$', decimals: 2, name: 'US Dollar',
    names: { en: 'Dollar', ko: '달러', zh: '美元' }
  },
  { 
    code: 'EUR', symbol: '€', decimals: 2, name: 'Euro',
    names: { en: 'Euro', ko: '유로', zh: '欧元' }
  },
  { 
    code: 'GBP', symbol: '£', decimals: 2, name: 'British Pound',
    names: { en: 'Pound', ko: '파운드', zh: '英镑' }
  },
  { 
    code: 'JPY', symbol: '¥', decimals: 0, name: 'Japanese Yen',
    names: { en: 'Yen', ko: '엔', zh: '日元' }
  },
  { 
    code: 'KRW', symbol: '₩', decimals: 0, name: 'Korean Won',
    names: { en: 'Won', ko: '원', zh: '韩元' }
  },
  { 
    code: 'AUD', symbol: 'A$', decimals: 2, name: 'Australian Dollar',
    names: { en: 'Dollar', ko: '달러', zh: '澳元' }
  },
  { 
    code: 'CAD', symbol: 'C$', decimals: 2, name: 'Canadian Dollar',
    names: { en: 'Dollar', ko: '달러', zh: '加元' }
  },
  { 
    code: 'CHF', symbol: 'CHF', decimals: 2, name: 'Swiss Franc',
    names: { en: 'Franc', ko: '프랑', zh: '瑞郎' }
  },
  { 
    code: 'CNY', symbol: '¥', decimals: 2, name: 'Chinese Yuan',
    names: { en: 'Yuan', ko: '위안', zh: '元' }
  },
  { 
    code: 'SEK', symbol: 'kr', decimals: 2, name: 'Swedish Krona',
    names: { en: 'Krona', ko: '크로나', zh: '克朗' }
  }
]
```

## Formatting Logic

### Updated formatCurrency Function

```typescript
export function formatCurrency(
  amount: number, 
  currencyCode: string, 
  numberFormat?: NumberFormatSettings,
  symbolMode?: 'currency' | 'language',
  locale?: string
): string {
  const currency = CURRENCIES.find(c => c.code === currencyCode)
  if (!currency) {
    return `${amount} ${currencyCode}`
  }

  const precision = numberFormat?.decimal_precision ?? 2
  const showTrailing = numberFormat?.show_trailing_zeros ?? true

  const formatted = amount.toLocaleString('en-US', {
    minimumFractionDigits: showTrailing ? precision : 0,
    maximumFractionDigits: precision
  })

  // Determine which symbol to use
  if (symbolMode === 'language') {
    const currentLocale = locale || 'en'
    const currencyName = currency.names?.[currentLocale] || currency.names?.en || currency.name
    return `${formatted} ${currencyName}`
  }

  // Default: currency symbol
  return `${currency.symbol}${formatted}`
}
```

### Truncation Integration

**formatCurrencyWithTruncation updates:**
- When `symbolMode === 'language'`: `${formatted}${unitLabel} ${currencyName}`
- When `symbolMode === 'currency'`: `${currency.symbol}${formatted}${unitLabel}`

**Examples:**
- Currency mode: `$1.5K`, `₩1.5만`
- Language mode: `1.5K Dollar`, `1.5만 원`

## User Interface

### Admin Panel UI

**Location:** Settings → Payment → Number Formatting section

**Layout:**
```
Number Formatting
├─ Decimal Precision: [dropdown: 0, 1, 2]
├─ Show Trailing Zeros: [toggle switch]
├─ Admin Panel Currency Display:
│  └─ [Toggle: Currency Symbol $ | Language Symbol Dollar]
└─ Storefront Currency Display:
   └─ [Toggle: Currency Symbol $ | Language Symbol Dollar]
```

**Toggle Button Design:**
- Two-state toggle (similar to trailing zeros toggle)
- Left: "Currency Symbol" with icon/example: `$`
- Right: "Language Symbol" with example: `Dollar`
- Updates preview dynamically

**Preview Section:**
```
Preview (Admin): $130 → 130 Dollar
Preview (Storefront): $130 → 130 Dollar
```

### State Management

**Admin:**
```typescript
let symbolDisplay = $state<SymbolDisplaySettings>({
  admin: 'currency',
  storefront: 'currency'
})

function handleSymbolDisplaySubmit() {
  payment.symbol_display = symbolDisplay
  await saveSettings('payment', payment, 'Symbol display saved')
  incrementSettingsVersion()
}
```

**Storefront:**
- Reads `paymentSettings.symbol_display.storefront`
- Passes to formatCurrency as `symbolMode` parameter
- Defaults to 'currency' if undefined

## Error Handling

### Validation Errors

**Backend:**
- Invalid mode value → return validation error
- Missing field → treat as null (valid, optional)

**Frontend:**
- Missing locale translation → fallback to English
- Missing currency → fallback to currency.name or symbol
- Undefined symbol_display → default to 'currency'

### Fallback Chain

```typescript
// Get currency name with fallback
const currencyName = 
  currency.names?.[currentLocale] ||  // Preferred locale
  currency.names?.en ||                // English fallback
  currency.name ||                     // Legacy name field
  currencyCode                         // Last resort
```

## Migration & Compatibility

### Backward Compatibility

1. **Existing Installations:**
   - `symbol_display` field is optional
   - Undefined/null treated as 'currency' mode
   - No breaking changes to existing behavior

2. **API Compatibility:**
   - GET /api/_/settings/payment returns symbol_display if set
   - Missing symbol_display = defaults apply on frontend
   - POST validates if provided, accepts null

3. **Data Migration:**
   - No migration needed
   - New field added, existing data unchanged

### Default Behavior

| Scenario | Behavior |
|----------|----------|
| New installation | `{ admin: 'currency', storefront: 'currency' }` |
| Existing installation | Undefined → defaults to 'currency' mode |
| Missing locale | Fallback to English translation |
| Unknown currency | Use symbol or code as fallback |

## Testing Strategy

### Unit Tests

1. **Backend Validation:**
   - Valid modes ('currency', 'language') pass
   - Invalid modes rejected with error
   - Null/undefined allowed (optional field)

2. **Frontend Formatting:**
   - Currency mode: `formatCurrency(..., 'currency')` returns `$130`
   - Language mode: `formatCurrency(..., 'language', 'en')` returns `130 Dollar`
   - Language mode Korean: `formatCurrency(..., 'language', 'ko')` returns `130 달러`
   - Language mode Chinese: `formatCurrency(..., 'language', 'zh')` returns `130 美元`
   - Truncation + language: returns `1.5K Dollar`

3. **Fallback Logic:**
   - Unknown locale → uses English
   - Missing translation → uses currency.name
   - Undefined symbolMode → defaults to 'currency'

### Integration Tests

1. **Settings Save/Load:**
   - Save symbol_display settings
   - Load and verify correct values
   - Verify defaults when field missing

2. **Cache Invalidation:**
   - Saving symbol_display increments version
   - Storefront cache invalidated on change

3. **UI Rendering:**
   - Admin panel shows correct toggle state
   - Storefront displays prices in correct mode
   - Preview updates when toggling

## Files to Modify

### Backend
- `internal/models/setting.go` - Add SymbolDisplaySettings type and validation
- `internal/queries/setting.go` - Add symbol_display to GroupFieldMap

### Frontend (Admin)
- `web/admin/src/lib/types/models.ts` - Add SymbolDisplaySettings interface
- `web/admin/src/lib/config/currencies.ts` - Add names field with translations
- `web/admin/src/lib/utils/currency.ts` - Update formatCurrency signature
- `web/admin/src/routes/settings/payment/+page.svelte` - Add UI toggles

### Frontend (Storefront)
- `web/site/src/lib/types/models.ts` - Add SymbolDisplaySettings interface
- `web/site/src/lib/config/currencies.ts` - Add names field with translations
- `web/site/src/lib/utils/currency.ts` - Update formatCurrency signature
- `web/site/src/lib/components/ProductCard.svelte` - Pass symbolMode
- `web/site/src/routes/products/[slug]/+page.svelte` - Pass symbolMode
- `web/site/src/routes/cart/+page.svelte` - Pass symbolMode

## Success Criteria

1. Admin can toggle between currency and language modes independently for admin/storefront
2. Currency symbol mode shows: `$130`, `¥130`, `₩130`
3. Language symbol mode shows: `130 Dollar` (en), `130 달러` (ko), `130 美元` (zh)
4. Truncation works correctly: `1.5K Dollar` in language mode
5. Settings persist and sync via cache invalidation
6. Backward compatible - existing installations unaffected
7. All translations complete for 10 currencies in 3 languages
