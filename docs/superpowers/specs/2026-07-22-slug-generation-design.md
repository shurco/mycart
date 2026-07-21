# Auto Slug Generation Feature Design

**Date:** 2026-07-22  
**Feature:** Automatic slug generation for products with fallback handling  
**Status:** Approved

## Overview

Add automatic slug generation to the admin product form that generates URL-friendly slugs from product names using the existing `/api/_/products/slug/generate` API endpoint, with client-side fallback on API failure.

## Requirements

1. Change slug field placeholder from default to "SLUG"
2. Auto-generate slug when product name loses focus (blur)
3. Only generate when slug field is empty
4. Works in both "add" and "edit" modes
5. Call `/api/_/products/slug/generate` API (no exclude_id parameter)
6. On API failure: show inline error and use client-side fallback
7. Add translation keys for all supported languages (en, ko, zh)

## Architecture

The feature consists of three main components:

1. **Slug Generator Utility** - Reusable utility module for slug generation logic
2. **Form Integration** - Blur event handler on product name field
3. **Translation Updates** - Placeholder and error message translations

### Architecture Diagram

```
User types name → blur event → check slug empty → generateSlug()
                                                        ↓
                                            ┌───────────┴───────────┐
                                            ↓                       ↓
                                    API call success        API call fails
                                            ↓                       ↓
                                    Return API slug         Generate fallback
                                            ↓                       ↓
                                    formData.slug = slug    formData.slug = fallback
                                    Clear errors            formErrors.slug = error message
```

## Components & Files

### New Files

**`web/admin/src/lib/utils/slugGenerator.ts`**

Utility module with three exported functions:

```typescript
/**
 * Generate slug from API
 * @param name - Product name
 * @returns Promise<string> - Generated slug from server
 * @throws Error on API failure
 */
export async function generateSlugFromAPI(name: string): Promise<string>

/**
 * Generate slug client-side as fallback
 * @param name - Product name  
 * @returns string - URL-friendly slug
 */
export function generateSlugFallback(name: string): string

/**
 * Generate slug with API first, fallback on error
 * @param name - Product name
 * @returns Promise<{slug: string, error?: string}>
 */
export async function generateSlug(name: string): Promise<{slug: string, error?: string}>
```

### Modified Files

1. **`web/admin/src/routes/products/+page.svelte`**
   - Import `generateSlug` utility
   - Add `placeholder` prop to slug FormInput
   - Add `onfocusout` handler to name FormInput
   - Handle slug generation response

2. **`web/admin/src/lib/i18n/locales/en.json`**
   - Add `products.slugPlaceholder`
   - Add `products.slugGenerationError`

3. **`web/admin/src/lib/i18n/locales/ko.json`**
   - Add Korean translations

4. **`web/admin/src/lib/i18n/locales/zh.json`**
   - Add Chinese translations

## Data Flow

### Step-by-Step Flow

1. **User Action:** User types product name and clicks/tabs away
2. **Blur Event:** `onfocusout` handler fires on name FormInput
3. **Condition Check:** If `formData.slug` is not empty, exit early
4. **Slug Generation:** Call `generateSlug(formData.name)`
5. **API Attempt:** Utility makes POST request to `/api/_/products/slug/generate`
   - Request body: `{ name: formData.name }`
   - No `exclude_id` parameter (API handles uniqueness)
6. **Success Path:**
   - Extract slug from API response: `result.slug`
   - Set `formData.slug = result.slug`
   - Clear any existing error: `delete formErrors.slug`
7. **Failure Path:**
   - Generate client-side fallback slug
   - Set `formData.slug = fallbackSlug`
   - Set `formErrors.slug = t('products.slugGenerationError')`
8. **UI Update:** Slug field shows generated value, error message appears if API failed

### State Changes

| State Variable | On Success | On Failure |
|----------------|------------|------------|
| `formData.slug` | API slug | Fallback slug |
| `formErrors.slug` | Cleared | Error message |

## Error Handling

### API Failure Scenarios

1. Network error (timeout, unreachable)
2. Server error (500, 503)
3. Validation error (empty name, invalid format)
4. Malformed response

### Handling Strategy

All errors are caught and handled gracefully:
- No exceptions thrown to user code
- Always populate slug field (never leave empty)
- Show inline error message only on API failure
- User can manually edit slug at any time

### Client-Side Fallback Algorithm

Transform product name into URL-friendly slug:

```typescript
function generateSlugFallback(name: string): string {
  return name
    .toLowerCase()                          // "My Product" → "my product"
    .replace(/\s+/g, '-')                   // "my product" → "my-product"
    .replace(/[^a-z0-9-]/g, '')             // Remove special chars
    .replace(/-+/g, '-')                    // Remove consecutive hyphens
    .replace(/^-|-$/g, '')                  // Trim hyphens from edges
}
```

**Examples:**
- `"My Product!"` → `"my-product"`
- `"Product   Name"` → `"product-name"`
- `"Test@2024"` → `"test2024"`
- `"Hello - World"` → `"hello-world"`

### User Experience

- Slug field always populated after blur (if initially empty)
- Red error message appears below slug field on API failure
- User can manually override generated slug
- Error clears on manual edit or next successful generation

## Translation Keys

### English (`en.json`)

```json
"products": {
  "slugPlaceholder": "SLUG",
  "slugGenerationError": "Failed to generate slug from server. Using fallback."
}
```

### Korean (`ko.json`)

```json
"products": {
  "slugPlaceholder": "SLUG",
  "slugGenerationError": "서버에서 슬러그 생성 실패. 대체 값을 사용합니다."
}
```

### Chinese (`zh.json`)

```json
"products": {
  "slugPlaceholder": "SLUG",
  "slugGenerationError": "无法从服务器生成slug。使用备用值。"
}
```

## Implementation Details

### Slug FormInput Changes

**Before:**
```svelte
<FormInput 
  id="slug" 
  title={t('products.slug')} 
  bind:value={formData.slug} 
  error={formErrors.slug} 
  ico="glob-alt" 
/>
```

**After:**
```svelte
<FormInput 
  id="slug" 
  title={t('products.slug')} 
  bind:value={formData.slug} 
  error={formErrors.slug} 
  ico="glob-alt"
  placeholder={t('products.slugPlaceholder')}
/>
```

### Name FormInput Changes

**Add blur handler:**
```svelte
<FormInput 
  id="name" 
  title={t('products.name')} 
  bind:value={formData.name} 
  error={formErrors.name} 
  ico="at-symbol"
  onfocusout={handleNameBlur}
/>
```

**Handler implementation:**
```typescript
async function handleNameBlur() {
  // Only generate if slug is empty
  if (formData.slug) return
  
  // Don't generate for empty name
  if (!formData.name.trim()) return
  
  const result = await generateSlug(formData.name)
  formData.slug = result.slug
  
  if (result.error) {
    formErrors.slug = t('products.slugGenerationError')
  } else {
    delete formErrors.slug
  }
}
```

## Testing Plan

### Manual Testing Scenarios

1. **Happy Path - Add Mode**
   - Create new product
   - Type name: "Test Product"
   - Blur name field
   - Verify slug auto-fills with API-generated value

2. **Happy Path - Edit Mode**
   - Edit existing product
   - Clear slug field
   - Blur name field
   - Verify slug regenerates

3. **API Failure**
   - Disconnect network or block API
   - Add new product with name
   - Blur name field
   - Verify fallback slug appears
   - Verify error message shows in red below slug field

4. **Slug Not Empty**
   - Type product name
   - Manually enter slug
   - Blur name field
   - Verify slug doesn't change

5. **Special Characters**
   - Name: "My Product! @#$%"
   - Expected slug: "my-product"

6. **Multiple Spaces**
   - Name: "Product   Name   Test"
   - Expected slug: "product-name-test"

7. **Empty Name**
   - Leave name empty
   - Blur name field
   - Verify no slug generation

8. **Non-English Characters**
   - Korean name: "제품 이름"
   - API should handle properly
   - Fallback may strip non-ASCII

### Edge Cases

- Name with only special chars: "!@#$%"
- Very long product names (100+ chars)
- Name starting/ending with spaces
- Unicode emoji in name
- Existing slug same as would-be generated slug

### Unit Test Targets (Future)

If unit tests are added later:

```typescript
// slugGenerator.test.ts
describe('generateSlugFallback', () => {
  it('converts to lowercase', () => {
    expect(generateSlugFallback('TEST')).toBe('test')
  })
  
  it('replaces spaces with hyphens', () => {
    expect(generateSlugFallback('hello world')).toBe('hello-world')
  })
  
  it('removes special characters', () => {
    expect(generateSlugFallback('test@#$')).toBe('test')
  })
  
  it('removes consecutive hyphens', () => {
    expect(generateSlugFallback('test---name')).toBe('test-name')
  })
})
```

## API Contract

### Endpoint

`POST /api/_/products/slug/generate`

### Request

```json
{
  "name": "Product Name"
}
```

Note: Not including `exclude_id` parameter. API handles uniqueness internally.

### Response (Success)

```json
{
  "status": 200,
  "message": "Slug generated",
  "result": {
    "slug": "product-name"
  }
}
```

### Response (Error)

```json
{
  "status": 400,
  "message": "name is required"
}
```

## Success Criteria

- [ ] Slug placeholder displays "SLUG" in all languages
- [ ] Slug auto-generates on name blur when slug is empty
- [ ] Works in both add and edit modes
- [ ] API success: slug field populates, no error
- [ ] API failure: fallback slug populates, error shows
- [ ] User can manually override slug at any time
- [ ] No console errors or warnings
- [ ] All translations present in en, ko, zh
- [ ] Existing product editing doesn't break

## Future Enhancements

Potential improvements not in this scope:

1. Real-time slug preview as user types
2. "Regenerate" button next to slug field
3. Show slug uniqueness indicator (green checkmark)
4. Debounced auto-generation while typing
5. Slug history/suggestions dropdown

## Notes

- The API endpoint already exists and is tested
- FormInput component already supports placeholder prop
- Existing validation for slug minimum length still applies
- This feature enhances UX but doesn't change core product creation flow
