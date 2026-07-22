# Currency Symbol Display Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add toggle between currency symbols ($, ¥, ₩) and language names (Dollar, Yen, Won) for price display with separate admin/storefront settings.

**Architecture:** Extend Payment settings with SymbolDisplaySettings containing admin and storefront modes. Add currency name translations (en, ko, zh) to currencies config. Update formatCurrency functions to support both display modes.

**Tech Stack:** Go 1.26, TypeScript, Svelte 5, ozzo-validation

## Global Constraints

- Backend validation: symbol_display values must be 'currency' or 'language'
- Default mode: 'currency' (backward compatible)
- Supported locales: en, ko, zh
- Fallback: missing translations use English
- No breaking changes to existing APIs

---

## File Structure

### Backend
- `internal/models/setting.go` - SymbolDisplaySettings type, validation
- `internal/queries/setting.go` - JSON serialization support

### Frontend (Admin)
- `web/admin/src/lib/types/models.ts` - TypeScript interfaces
- `web/admin/src/lib/config/currencies.ts` - Currency translations
- `web/admin/src/lib/utils/currency.ts` - Formatting logic
- `web/admin/src/routes/settings/payment/+page.svelte` - UI controls

### Frontend (Storefront)
- `web/site/src/lib/types/models.ts` - TypeScript interfaces
- `web/site/src/lib/config/currencies.ts` - Currency translations
- `web/site/src/lib/utils/currency.ts` - Formatting logic
- `web/site/src/lib/components/ProductCard.svelte` - Product display
- `web/site/src/routes/products/[slug]/+page.svelte` - Product detail
- `web/site/src/routes/cart/+page.svelte` - Cart display

---

### Task 1: Backend - Add SymbolDisplaySettings Type

**Files:**
- Modify: `internal/models/setting.go`

**Interfaces:**
- Consumes: None
- Produces: `SymbolDisplaySettings` struct with `Admin` and `Storefront` string fields, `validateSymbolDisplay` function

- [ ] **Step 1: Add SymbolDisplaySettings type after NumberFormatSettings**

In `internal/models/setting.go`, add after line 136 (after NumberFormatSettings):

```go
// SymbolDisplaySettings defines currency display mode per context
type SymbolDisplaySettings struct {
	Admin      string `json:"admin"`      // "currency" or "language"
	Storefront string `json:"storefront"` // "currency" or "language"
}
```

- [ ] **Step 2: Add SymbolDisplay field to Payment struct**

Update the Payment struct (around line 50-55) to add the new field:

```go
// Payment is ...
type Payment struct {
	Currency      string                  `json:"currency"`
	Truncation    *TruncationSettings     `json:"truncation,omitempty"`
	NumberFormat  *NumberFormatSettings   `json:"number_format,omitempty"`
	SymbolDisplay *SymbolDisplaySettings  `json:"symbol_display,omitempty"`
}
```

- [ ] **Step 3: Add validation call in Payment.Validate()**

Update the Validate method (around line 58-64) to include symbol_display validation:

```go
// Validate is ...
func (v Payment) Validate() error {
	return validation.ValidateStruct(&v,
		validation.Field(&v.Currency, is.CurrencyCode),
		validation.Field(&v.Truncation, validation.By(validateTruncation)),
		validation.Field(&v.NumberFormat, validation.By(validateNumberFormat)),
		validation.Field(&v.SymbolDisplay, validation.By(validateSymbolDisplay)),
	)
}
```

- [ ] **Step 4: Add validateSymbolDisplay function**

Add after validateNumberFormat function (after line 124):

```go
// validateSymbolDisplay validates symbol display settings
func validateSymbolDisplay(value interface{}) error {
	if value == nil {
		return nil // symbol_display is optional
	}

	sd, ok := value.(*SymbolDisplaySettings)
	if !ok || sd == nil {
		return nil
	}

	validModes := map[string]bool{"currency": true, "language": true}

	if sd.Admin != "" && !validModes[sd.Admin] {
		return validation.NewError("symbol_display_invalid_mode",
			"admin symbol_display must be 'currency' or 'language'")
	}

	if sd.Storefront != "" && !validModes[sd.Storefront] {
		return validation.NewError("symbol_display_invalid_mode",
			"storefront symbol_display must be 'currency' or 'language'")
	}

	return nil
}
```

- [ ] **Step 5: Commit backend types**

```bash
git add internal/models/setting.go
git commit -m "feat(backend): add SymbolDisplaySettings type and validation

Add symbol_display setting to Payment with admin/storefront contexts.
Validates modes are 'currency' or 'language'.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Backend - Add Symbol Display to Query Serialization

**Files:**
- Modify: `internal/queries/setting.go`

**Interfaces:**
- Consumes: `SymbolDisplaySettings` from Task 1
- Produces: JSON serialization support for symbol_display field

- [ ] **Step 1: Add symbol_display to Payment GroupFieldMap**

In `internal/queries/setting.go`, find the GroupFieldMap for "payment" and add symbol_display:

```go
"payment": {
	"currency":      "text",
	"truncation":    "json",
	"number_format": "json",
	"symbol_display": "json", // NEW
},
```

- [ ] **Step 2: Add JSON marshal/unmarshal cases for SymbolDisplaySettings**

In the SetGroupField function, add case for symbol_display (similar to truncation):

```go
case "symbol_display":
	if value == "" {
		payment.SymbolDisplay = nil
	} else {
		var sd models.SymbolDisplaySettings
		if err := json.Unmarshal([]byte(value), &sd); err != nil {
			return fmt.Errorf("invalid symbol_display JSON: %w", err)
		}
		payment.SymbolDisplay = &sd
	}
```

In the GetGroupField function, add case for symbol_display:

```go
case "symbol_display":
	if payment.SymbolDisplay != nil {
		data, err := json.Marshal(payment.SymbolDisplay)
		if err != nil {
			return "", fmt.Errorf("marshal symbol_display: %w", err)
		}
		return string(data), nil
	}
	return "", nil
```

- [ ] **Step 3: Commit query changes**

```bash
git add internal/queries/setting.go
git commit -m "feat(backend): add symbol_display JSON serialization

Enable loading and saving symbol_display settings from database.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Admin - Add TypeScript Types

**Files:**
- Modify: `web/admin/src/lib/types/models.ts`

**Interfaces:**
- Consumes: None
- Produces: `SymbolDisplaySettings` interface, extended `PaymentSettings`

- [ ] **Step 1: Add SymbolDisplaySettings interface**

Add after NumberFormatSettings interface:

```typescript
export interface SymbolDisplaySettings {
  admin: 'currency' | 'language'
  storefront: 'currency' | 'language'
}
```

- [ ] **Step 2: Add symbol_display to PaymentSettings**

Update PaymentSettings interface:

```typescript
export interface PaymentSettings {
  currency: string
  truncation?: TruncationSettings
  number_format?: NumberFormatSettings
  symbol_display?: SymbolDisplaySettings
}
```

- [ ] **Step 3: Commit TypeScript types**

```bash
git add web/admin/src/lib/types/models.ts
git commit -m "feat(admin): add SymbolDisplaySettings TypeScript interface

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Admin - Add Currency Name Translations

**Files:**
- Modify: `web/admin/src/lib/config/currencies.ts`

**Interfaces:**
- Consumes: None
- Produces: Extended `CurrencyConfig` with `names` field containing en/ko/zh translations

- [ ] **Step 1: Update CurrencyConfig interface**

```typescript
export interface CurrencyConfig {
  code: string
  symbol: string
  decimals: number
  name: string
  names: {
    en: string
    ko: string
    zh: string
  }
}
```

- [ ] **Step 2: Add translations to all currencies**

Update CURRENCIES array:

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

- [ ] **Step 3: Commit currency translations**

```bash
git add web/admin/src/lib/config/currencies.ts
git commit -m "feat(admin): add currency name translations (en/ko/zh)

Add localized currency names for English, Korean, and Chinese.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 5: Admin - Update formatCurrency Function

**Files:**
- Modify: `web/admin/src/lib/utils/currency.ts`

**Interfaces:**
- Consumes: `CurrencyConfig.names` from Task 4
- Produces: Updated `formatCurrency` and `formatCurrencyWithTruncation` with symbol mode support

- [ ] **Step 1: Update formatCurrency signature and implementation**

Replace the existing formatCurrency function:

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

  // Language symbol mode
  if (symbolMode === 'language') {
    const currentLocale = locale || 'en'
    const currencyName = currency.names?.[currentLocale] || currency.names?.en || currency.name
    return `${formatted} ${currencyName}`
  }

  // Default: currency symbol mode
  return `${currency.symbol}${formatted}`
}
```

- [ ] **Step 2: Update formatCurrencyWithTruncation for symbol mode**

Update the function to pass symbolMode through and handle language mode with truncation:

```typescript
export function formatCurrencyWithTruncation(
  amount: number,
  currencyCode: string,
  context: 'admin' | 'storefront',
  truncationSettings?: TruncationSettings,
  locale?: string,
  numberFormat?: NumberFormatSettings,
  symbolMode?: 'currency' | 'language'
): string {
  // Default to 'none' mode if settings missing
  if (!truncationSettings || !truncationSettings[context]) {
    return formatCurrency(amount / 100, currencyCode, numberFormat, symbolMode, locale)
  }

  const settings = truncationSettings[context][currencyCode]
  if (!settings || settings.mode === 'none') {
    return formatCurrency(amount / 100, currencyCode, numberFormat, symbolMode, locale)
  }

  const currency = CURRENCIES.find(c => c.code === currencyCode)
  if (!currency) {
    return `${amount / 100} ${currencyCode}`
  }

  const pattern = getCurrencyPattern(currencyCode)
  const currentLocale = locale || 'en'

  // Get currency name for language mode
  const getCurrencyName = () => {
    if (symbolMode === 'language') {
      return currency.names?.[currentLocale] || currency.names?.en || currency.name
    }
    return null
  }

  // Fixed mode: use specified unit
  if (settings.mode === 'fixed' && settings.fixed_unit) {
    const unit = pattern.units.find(u => {
      const label = getUnitLabel(u.value, currencyCode, currentLocale)
      return label === settings.fixed_unit ||
             Object.values(u.keys).includes(settings.fixed_unit)
    })

    if (unit && amount >= unit.value * 100) {
      const divided = (amount / 100) / unit.value
      const formatted = applyDecimalPrecision(divided, numberFormat)
      const unitLabel = getUnitLabel(unit.value, currencyCode, currentLocale)
      
      const currencyName = getCurrencyName()
      if (currencyName) {
        return `${formatted}${unitLabel} ${currencyName}`
      }
      return `${currency.symbol}${formatted}${unitLabel}`
    }

    return formatCurrency(amount / 100, currencyCode, numberFormat, symbolMode, locale)
  }

  // Flexible mode: auto-select best unit
  if (settings.mode === 'flexible') {
    const unit = selectFlexibleUnit(amount / 100, currencyCode)

    if (unit) {
      const divided = (amount / 100) / unit.value
      const formatted = applyDecimalPrecision(divided, numberFormat)
      const unitLabel = getUnitLabel(unit.value, currencyCode, currentLocale)
      
      const currencyName = getCurrencyName()
      if (currencyName) {
        return `${formatted}${unitLabel} ${currencyName}`
      }
      return `${currency.symbol}${formatted}${unitLabel}`
    }

    return formatCurrency(amount / 100, currencyCode, numberFormat, symbolMode, locale)
  }

  return formatCurrency(amount / 100, currencyCode, numberFormat, symbolMode, locale)
}
```

- [ ] **Step 3: Commit formatting updates**

```bash
git add web/admin/src/lib/utils/currency.ts
git commit -m "feat(admin): add symbol mode to formatCurrency functions

Support currency symbol ($) and language name (Dollar) modes.
Language mode shows: '130 Dollar' or '1.5K Dollar' with truncation.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 6: Admin - Add UI Controls

**Files:**
- Modify: `web/admin/src/routes/settings/payment/+page.svelte`

**Interfaces:**
- Consumes: `SymbolDisplaySettings` type from Task 3, updated `formatCurrency` from Task 5
- Produces: UI toggles for admin and storefront symbol display modes

- [ ] **Step 1: Add state variables for symbol display**

After the `showTrailingZeros` state declaration (around line 33), add:

```typescript
let symbolDisplay = $state<SymbolDisplaySettings>({
  admin: 'currency',
  storefront: 'currency'
})
```

- [ ] **Step 2: Load symbol_display in loadPaymentSettings**

In the `loadPaymentSettings` function, after loading number_format settings (around line 103):

```typescript
// Load symbol display settings
const sd = paymentSettings.symbol_display || {
  admin: 'currency',
  storefront: 'currency'
}
symbolDisplay.admin = sd.admin
symbolDisplay.storefront = sd.storefront
payment.symbol_display = sd
```

- [ ] **Step 3: Add handleSymbolDisplaySubmit function**

After `handleNumberFormatSubmit` (around line 164):

```typescript
async function handleSymbolDisplaySubmit() {
  payment.symbol_display = {
    admin: symbolDisplay.admin,
    storefront: symbolDisplay.storefront
  }
  await saveSettings('payment', payment, 'Symbol display saved')
  paymentSettingsStore.set(payment)
  incrementSettingsVersion()
}
```

- [ ] **Step 4: Add UI section for symbol display**

After the Number Formatting section (before the truncation section, around line 245), add:

```svelte
<hr class="mt-5" />

<div class="mt-5 max-w-2xl">
  <h2 class="mb-5">Currency Display</h2>

  <div class="mb-4">
    <h3 class="mb-2 text-sm font-medium text-gray-700">Admin Panel Display</h3>
    <div class="flex gap-2">
      <button
        type="button"
        class="flex-1 rounded border px-4 py-2 text-sm font-medium transition-colors {symbolDisplay.admin === 'currency' ? 'border-green-500 bg-green-50 text-green-700' : 'border-gray-300 bg-white text-gray-700 hover:bg-gray-50'}"
        onclick={() => symbolDisplay.admin = 'currency'}
      >
        Currency Symbol
        <span class="block text-xs text-gray-500 mt-1">$130</span>
      </button>
      <button
        type="button"
        class="flex-1 rounded border px-4 py-2 text-sm font-medium transition-colors {symbolDisplay.admin === 'language' ? 'border-green-500 bg-green-50 text-green-700' : 'border-gray-300 bg-white text-gray-700 hover:bg-gray-50'}"
        onclick={() => symbolDisplay.admin = 'language'}
      >
        Language Symbol
        <span class="block text-xs text-gray-500 mt-1">130 Dollar</span>
      </button>
    </div>
  </div>

  <div class="mb-4">
    <h3 class="mb-2 text-sm font-medium text-gray-700">Storefront Display</h3>
    <div class="flex gap-2">
      <button
        type="button"
        class="flex-1 rounded border px-4 py-2 text-sm font-medium transition-colors {symbolDisplay.storefront === 'currency' ? 'border-green-500 bg-green-50 text-green-700' : 'border-gray-300 bg-white text-gray-700 hover:bg-gray-50'}"
        onclick={() => symbolDisplay.storefront = 'currency'}
      >
        Currency Symbol
        <span class="block text-xs text-gray-500 mt-1">$130</span>
      </button>
      <button
        type="button"
        class="flex-1 rounded border px-4 py-2 text-sm font-medium transition-colors {symbolDisplay.storefront === 'language' ? 'border-green-500 bg-green-50 text-green-700' : 'border-gray-300 bg-white text-gray-700 hover:bg-gray-50'}"
        onclick={() => symbolDisplay.storefront = 'language'}
      >
        Language Symbol
        <span class="block text-xs text-gray-500 mt-1">130 Dollar</span>
      </button>
    </div>
  </div>

  <div class="pt-5">
    <FormButton onclick={handleSymbolDisplaySubmit} name="Save" color="green" />
  </div>
</div>
```

- [ ] **Step 5: Commit admin UI**

```bash
git add web/admin/src/routes/settings/payment/+page.svelte
git commit -m "feat(admin): add symbol display UI controls

Add toggles for admin and storefront currency display modes.
Users can independently choose currency symbols or language names.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 7: Storefront - Add TypeScript Types

**Files:**
- Modify: `web/site/src/lib/types/models.ts`

**Interfaces:**
- Consumes: None
- Produces: `SymbolDisplaySettings` interface, extended `PaymentSettings`

- [ ] **Step 1: Add SymbolDisplaySettings interface**

Add after NumberFormatSettings interface (same as admin):

```typescript
export interface SymbolDisplaySettings {
  admin: 'currency' | 'language'
  storefront: 'currency' | 'language'
}
```

- [ ] **Step 2: Add symbol_display to PaymentSettings**

Update PaymentSettings interface:

```typescript
export interface PaymentSettings {
  currency: string
  truncation?: TruncationSettings
  number_format?: NumberFormatSettings
  symbol_display?: SymbolDisplaySettings
}
```

- [ ] **Step 3: Commit storefront types**

```bash
git add web/site/src/lib/types/models.ts
git commit -m "feat(storefront): add SymbolDisplaySettings TypeScript interface

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 8: Storefront - Add Currency Name Translations

**Files:**
- Modify: `web/site/src/lib/config/currencies.ts`

**Interfaces:**
- Consumes: None
- Produces: Extended `CurrencyConfig` with `names` field

- [ ] **Step 1: Update CurrencyConfig and CURRENCIES array**

Same changes as Task 4 - update interface and add translations to all currencies:

```typescript
export interface CurrencyConfig {
  code: string
  symbol: string
  decimals: number
  name: string
  names: {
    en: string
    ko: string
    zh: string
  }
}

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

- [ ] **Step 2: Commit storefront currency translations**

```bash
git add web/site/src/lib/config/currencies.ts
git commit -m "feat(storefront): add currency name translations (en/ko/zh)

Add localized currency names for English, Korean, and Chinese.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 9: Storefront - Update formatCurrency Function

**Files:**
- Modify: `web/site/src/lib/utils/currency.ts`

**Interfaces:**
- Consumes: `CurrencyConfig.names` from Task 8
- Produces: Updated `formatCurrency` and `formatCurrencyWithTruncation` with symbol mode support

- [ ] **Step 1: Update formatCurrency and formatCurrencyWithTruncation**

Same updates as Task 5 - replace the functions with symbol mode support:

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

  if (symbolMode === 'language') {
    const currentLocale = locale || 'en'
    const currencyName = currency.names?.[currentLocale] || currency.names?.en || currency.name
    return `${formatted} ${currencyName}`
  }

  return `${currency.symbol}${formatted}`
}
```

And update `formatCurrencyWithTruncation` with the same logic as Task 5.

- [ ] **Step 2: Commit storefront formatting**

```bash
git add web/site/src/lib/utils/currency.ts
git commit -m "feat(storefront): add symbol mode to formatCurrency functions

Support currency symbol ($) and language name (Dollar) modes.
Language mode shows: '130 Dollar' or '1.5K Dollar' with truncation.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 10: Storefront - Update ProductCard Component

**Files:**
- Modify: `web/site/src/lib/components/ProductCard.svelte`

**Interfaces:**
- Consumes: Updated `formatCurrencyWithTruncation` from Task 9, `settingsStore` with symbol_display
- Produces: ProductCard displaying prices with correct symbol mode

- [ ] **Step 1: Add derived symbol mode from settings**

After the existing derived values, add:

```typescript
let symbolMode = $derived(
  $settingsStore?.payment?.symbol_display?.storefront || 'currency'
)
```

- [ ] **Step 2: Pass symbolMode to formatCurrencyWithTruncation**

Find all calls to `formatCurrencyWithTruncation` and add symbolMode parameter:

```typescript
formatCurrencyWithTruncation(
  product.price,
  currency,
  'storefront',
  truncationSettings,
  currentLocale,
  numberFormat,
  symbolMode  // ADD THIS
)
```

- [ ] **Step 3: Commit ProductCard update**

```bash
git add web/site/src/lib/components/ProductCard.svelte
git commit -m "feat(storefront): apply symbol display mode to ProductCard

Read storefront symbol_display setting and pass to formatting functions.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 11: Storefront - Update Product Detail Page

**Files:**
- Modify: `web/site/src/routes/products/[slug]/+page.svelte`

**Interfaces:**
- Consumes: Updated `formatCurrencyWithTruncation` from Task 9
- Produces: Product detail page with correct symbol mode

- [ ] **Step 1: Add derived symbol mode**

Add after existing derived values:

```typescript
let symbolMode = $derived(
  $settingsStore?.payment?.symbol_display?.storefront || 'currency'
)
```

- [ ] **Step 2: Pass symbolMode to formatCurrencyWithTruncation calls**

Update all formatting calls to include symbolMode parameter.

- [ ] **Step 3: Commit product detail update**

```bash
git add web/site/src/routes/products/[slug]/+page.svelte
git commit -m "feat(storefront): apply symbol display mode to product detail

Read storefront symbol_display setting and pass to formatting functions.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 12: Storefront - Update Cart Page

**Files:**
- Modify: `web/site/src/routes/cart/+page.svelte`

**Interfaces:**
- Consumes: Updated `formatCurrencyWithTruncation` from Task 9
- Produces: Cart page with correct symbol mode

- [ ] **Step 1: Add derived symbol mode**

Add after existing derived values:

```typescript
let symbolMode = $derived(
  $settingsStore?.payment?.symbol_display?.storefront || 'currency'
)
```

- [ ] **Step 2: Pass symbolMode to all formatCurrencyWithTruncation calls**

Update all formatting calls to include symbolMode parameter.

- [ ] **Step 3: Commit cart page update**

```bash
git add web/site/src/routes/cart/+page.svelte
git commit -m "feat(storefront): apply symbol display mode to cart page

Read storefront symbol_display setting and pass to formatting functions.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 13: Build and Test

**Files:**
- None (testing task)

**Interfaces:**
- Consumes: All previous tasks
- Produces: Working builds and verified functionality

- [ ] **Step 1: Build admin frontend**

```bash
cd web/admin && npx vite build
```

Expected: Build succeeds with no TypeScript errors

- [ ] **Step 2: Build storefront frontend**

```bash
cd ../site && npx vite build
```

Expected: Build succeeds with no TypeScript errors

- [ ] **Step 3: Start server and test**

```bash
cd ../.. && go run ./cmd serve
```

Expected: Server starts on port 8080

- [ ] **Step 4: Manual testing checklist**

1. Open admin panel: http://localhost:8080/_/settings/payment
2. Verify Currency Display section appears
3. Test Admin Panel toggle:
   - Select "Language Symbol"
   - Click Save
   - Navigate to products or carts page in admin
   - Verify prices show "130 Dollar" format
4. Test Storefront toggle:
   - Select "Language Symbol" 
   - Click Save
   - Open storefront: http://localhost:8080/
   - Verify prices show "130 Dollar" format
5. Test with different currencies (KRW, JPY)
6. Test with truncation enabled
7. Verify cache invalidation works (changes apply immediately)

- [ ] **Step 5: Verify backward compatibility**

1. Clear symbol_display from database
2. Reload pages
3. Verify defaults to currency symbol mode

---

### Task 14: Final Commit and Documentation

**Files:**
- None (final commit task)

**Interfaces:**
- Consumes: All completed tasks
- Produces: Complete feature commit

- [ ] **Step 1: Review all changes**

```bash
git status
git diff
```

Expected: All changes related to symbol display feature

- [ ] **Step 2: Final commit if any uncommitted changes**

```bash
git add -A
git commit -m "feat: currency symbol display mode complete

Complete implementation of currency vs language symbol toggle.

Features:
- Separate admin/storefront symbol display settings
- Currency symbol mode: $130, ¥130, ₩130
- Language symbol mode: 130 Dollar, 130 Yen, 130 Won
- Translations: English, Korean, Chinese
- Works with truncation: 1.5K Dollar
- Backward compatible defaults

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

- [ ] **Step 3: Verify git log**

```bash
git log --oneline -15
```

Expected: All feature commits present

---

## Success Criteria Checklist

- [ ] Backend validates symbol_display modes correctly
- [ ] Admin UI shows separate toggles for admin and storefront
- [ ] Currency symbol mode displays: $130, ¥130, ₩130
- [ ] Language symbol mode displays: 130 Dollar (en), 130 달러 (ko), 130 美元 (zh)
- [ ] Truncation works: 1.5K Dollar in language mode
- [ ] Settings persist and cache invalidates immediately
- [ ] Backward compatible - undefined symbol_display defaults to currency
- [ ] All 10 currencies have complete translations for 3 languages
- [ ] No TypeScript errors in builds
- [ ] Manual testing passes all scenarios
