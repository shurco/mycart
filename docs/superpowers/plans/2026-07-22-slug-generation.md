# Auto Slug Generation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add automatic slug generation to admin product forms that generates URL-friendly slugs from product names with client-side fallback on API failure.

**Architecture:** Three-layer approach - (1) reusable slug generator utility module, (2) blur event handler integration in product form, (3) translation keys for placeholder and error messages across all supported languages.

**Tech Stack:** Svelte 5, TypeScript, existing API endpoint `/api/_/products/slug/generate`

## Global Constraints

- Svelte 5 runes syntax (`$state`, `$derived`) required
- TypeScript strict mode enabled
- Must support en, ko, zh languages
- Maintain existing form validation behavior
- No external dependencies (use native fetch API)

---

## File Structure

**New Files:**
- `web/admin/src/lib/utils/slugGenerator.ts` - Slug generation utility with API call and fallback

**Modified Files:**
- `web/admin/src/lib/i18n/locales/en.json` - English translations
- `web/admin/src/lib/i18n/locales/ko.json` - Korean translations  
- `web/admin/src/lib/i18n/locales/zh.json` - Chinese translations
- `web/admin/src/routes/products/+page.svelte` - Product form integration

---

### Task 1: Add Translation Keys

**Files:**
- Modify: `web/admin/src/lib/i18n/locales/en.json`
- Modify: `web/admin/src/lib/i18n/locales/ko.json`
- Modify: `web/admin/src/lib/i18n/locales/zh.json`

**Interfaces:**
- Consumes: Existing translation structure
- Produces: `products.slugPlaceholder: string`, `products.slugGenerationError: string`

- [ ] **Step 1: Add English translations**

Open `web/admin/src/lib/i18n/locales/en.json` and locate the `"products"` section. Add two new keys after the existing `"slug": "URL"` entry:

```json
"products": {
  "title": "Products",
  "noProducts": "No products found",
  "name": "Name",
  "slug": "URL",
  "slugPlaceholder": "SLUG",
  "slugGenerationError": "Failed to generate slug from server. Using fallback.",
  "brief": "Brief",
  ...
}
```

- [ ] **Step 2: Add Korean translations**

Open `web/admin/src/lib/i18n/locales/ko.json` and locate the `"products"` section. Add the same two keys:

```json
"products": {
  "title": "제품",
  "noProducts": "제품을 찾을 수 없습니다",
  "name": "이름",
  "slug": "URL",
  "slugPlaceholder": "SLUG",
  "slugGenerationError": "서버에서 슬러그 생성 실패. 대체 값을 사용합니다.",
  "brief": "요약",
  ...
}
```

- [ ] **Step 3: Add Chinese translations**

Open `web/admin/src/lib/i18n/locales/zh.json` and locate the `"products"` section. Add the same two keys:

```json
"products": {
  "title": "产品",
  "noProducts": "未找到产品",
  "name": "名称",
  "slug": "URL",
  "slugPlaceholder": "SLUG",
  "slugGenerationError": "无法从服务器生成slug。使用备用值。",
  "brief": "简介",
  ...
}
```

- [ ] **Step 4: Verify JSON syntax**

Run: `cd web/admin && npm run build`

Expected: Build succeeds with no JSON parsing errors

- [ ] **Step 5: Commit translation keys**

```bash
git add web/admin/src/lib/i18n/locales/*.json
git commit -m "feat(i18n): add slug placeholder and error translations

Add slugPlaceholder and slugGenerationError keys for en, ko, zh languages
to support auto-generated slug feature.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Create Slug Generator Utility

**Files:**
- Create: `web/admin/src/lib/utils/slugGenerator.ts`

**Interfaces:**
- Consumes: None (self-contained utility)
- Produces:
  - `generateSlugFallback(name: string): string`
  - `generateSlugFromAPI(name: string): Promise<string>`
  - `generateSlug(name: string): Promise<{slug: string, error?: string}>`

- [ ] **Step 1: Create utility file with fallback function**

Create `web/admin/src/lib/utils/slugGenerator.ts`:

```typescript
/**
 * Generate URL-friendly slug from product name (client-side fallback)
 * Transforms: "My Product!" → "my-product"
 * 
 * @param name - Product name to convert
 * @returns URL-friendly slug string
 */
export function generateSlugFallback(name: string): string {
  return name
    .toLowerCase()                 // "My Product" → "my product"
    .replace(/\s+/g, '-')          // "my product" → "my-product"
    .replace(/[^a-z0-9-]/g, '')    // Remove special chars
    .replace(/-+/g, '-')           // Remove consecutive hyphens
    .replace(/^-|-$/g, '')         // Trim hyphens from edges
}
```

- [ ] **Step 2: Add API call function**

Add to `web/admin/src/lib/utils/slugGenerator.ts`:

```typescript
/**
 * Generate slug from API endpoint
 * 
 * @param name - Product name
 * @returns Promise<string> - Generated slug from server
 * @throws Error if API call fails or returns invalid response
 */
export async function generateSlugFromAPI(name: string): Promise<string> {
  const response = await fetch('/api/_/products/slug/generate', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ name })
  })

  if (!response.ok) {
    throw new Error(`API returned ${response.status}`)
  }

  const data = await response.json()
  
  if (!data.result?.slug) {
    throw new Error('Invalid API response format')
  }

  return data.result.slug
}
```

- [ ] **Step 3: Add orchestrator function**

Add to `web/admin/src/lib/utils/slugGenerator.ts`:

```typescript
/**
 * Generate slug with API-first approach and client-side fallback
 * 
 * @param name - Product name
 * @returns Promise<{slug: string, error?: string}> - Generated slug and optional error message
 */
export async function generateSlug(name: string): Promise<{slug: string, error?: string}> {
  try {
    const slug = await generateSlugFromAPI(name)
    return { slug }
  } catch (error) {
    // API failed - use client-side fallback
    const slug = generateSlugFallback(name)
    return { 
      slug, 
      error: 'api_failed'
    }
  }
}
```

- [ ] **Step 4: Test fallback function in browser console**

Run: `cd web/admin && npm run dev`

Open browser console and test:

```javascript
// In browser dev tools console (after importing the module)
generateSlugFallback('My Product!')      // Expected: 'my-product'
generateSlugFallback('Test   Name')      // Expected: 'test-name'
generateSlugFallback('Test@2024')        // Expected: 'test2024'
generateSlugFallback('!@#$')             // Expected: ''
```

Expected: All transformations work correctly

- [ ] **Step 5: Commit slug generator utility**

```bash
git add web/admin/src/lib/utils/slugGenerator.ts
git commit -m "feat(utils): add slug generator with API and fallback

Add slugGenerator utility with three functions:
- generateSlugFallback: client-side slug transformation
- generateSlugFromAPI: API call to /api/_/products/slug/generate
- generateSlug: orchestrator with API-first, fallback on error

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Integrate with Product Form

**Files:**
- Modify: `web/admin/src/routes/products/+page.svelte:651-682` (slug input in add mode)
- Modify: `web/admin/src/routes/products/+page.svelte:696` (slug input in edit mode)
- Modify: `web/admin/src/routes/products/+page.svelte:1-30` (imports and state)

**Interfaces:**
- Consumes: 
  - `generateSlug(name: string)` from `$lib/utils/slugGenerator`
  - `t(key: string)` from existing i18n context
  - `formData.name: string`, `formData.slug: string` from existing form state
- Produces: Auto-populated `formData.slug` on name blur, `formErrors.slug` on API failure

- [ ] **Step 1: Add import for slug generator**

At the top of `web/admin/src/routes/products/+page.svelte`, after existing imports (around line 27), add:

```typescript
import { generateSlug } from '$lib/utils/slugGenerator'
```

- [ ] **Step 2: Add blur handler function**

After the `handleAmountInput` function (around line 120), add:

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

- [ ] **Step 3: Add placeholder to slug input (add mode)**

Locate the slug FormInput in add mode (around line 676-682) and add the placeholder prop:

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

- [ ] **Step 4: Add placeholder to slug input (edit mode)**

Locate the slug FormInput in edit mode (around line 696) and add the placeholder prop:

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

- [ ] **Step 5: Add onfocusout handler to name input**

Locate the name FormInput (around line 651) and add the onfocusout prop:

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

- [ ] **Step 6: Verify TypeScript compilation**

Run: `cd web/admin && npm run build`

Expected: Build succeeds with no TypeScript errors

- [ ] **Step 7: Commit form integration**

```bash
git add web/admin/src/routes/products/+page.svelte
git commit -m "feat(products): add auto slug generation on name blur

- Import generateSlug utility
- Add handleNameBlur function to generate slug when name loses focus
- Add placeholder prop to slug inputs in both add and edit modes
- Only generate when slug field is empty
- Show error message on API failure, use client-side fallback

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Manual Testing and Verification

**Files:**
- Test: All modified files via browser

**Interfaces:**
- Consumes: Complete feature implementation
- Produces: Verified working feature

- [ ] **Step 1: Start dev server**

Run: `cd web/admin && npm run dev`

Expected: Dev server starts without errors

- [ ] **Step 2: Test happy path - add mode**

1. Navigate to products page
2. Click "Add Product" button
3. Type "Test Product" in name field
4. Click or tab to slug field
5. Verify slug auto-fills with "test-product"
6. Verify no error message appears
7. Verify placeholder shows "SLUG" before typing

Expected: Slug generates successfully from API

- [ ] **Step 3: Test happy path - edit mode**

1. Click edit on any existing product
2. Clear the slug field completely
3. Click back to name field and change it to "Updated Name"
4. Blur the name field (click elsewhere)
5. Verify slug generates "updated-name"

Expected: Slug regenerates in edit mode

- [ ] **Step 4: Test slug not empty condition**

1. Add new product
2. Type "My Product" in name
3. Manually type "custom-slug" in slug field
4. Click back to name field, change to "Different Name"
5. Blur name field
6. Verify slug remains "custom-slug" (unchanged)

Expected: Existing slug not overwritten

- [ ] **Step 5: Test special characters**

1. Add new product
2. Type "My Product! @#$%" in name
3. Blur name field
4. Verify slug is "my-product" (special chars removed)

Expected: Clean slug without special characters

- [ ] **Step 6: Test multiple spaces**

1. Add new product
2. Type "Product   Name   Test" in name (multiple spaces)
3. Blur name field
4. Verify slug is "product-name-test" (single hyphens)

Expected: No consecutive hyphens in slug

- [ ] **Step 7: Test empty name**

1. Add new product
2. Leave name field empty
3. Blur name field
4. Verify slug remains empty (no generation)

Expected: No slug generation for empty name

- [ ] **Step 8: Test API failure fallback**

1. Open browser DevTools Network tab
2. Add network throttling or use "Offline" mode
3. Add new product with name "Offline Test"
4. Blur name field
5. Verify slug populates with "offline-test" (fallback)
6. Verify red error message appears: "Failed to generate slug from server. Using fallback."

Expected: Fallback works, error message displays

- [ ] **Step 9: Test language switching**

1. Switch admin language to Korean (if available)
2. Add new product
3. Verify placeholder shows "SLUG" 
4. Test with offline mode to see Korean error message

Expected: Translations work in all languages

- [ ] **Step 10: Test form submission**

1. Add new product with auto-generated slug
2. Fill required fields
3. Submit form
4. Verify product saves successfully
5. Verify product appears in list with correct slug

Expected: Full workflow works end-to-end

- [ ] **Step 11: Verify no console errors**

Check browser console during all tests above.

Expected: No console errors or warnings

- [ ] **Step 12: Document test results**

Create a test summary in your terminal or notes:

```
✓ Auto-generation works in add mode
✓ Auto-generation works in edit mode  
✓ Slug not overwritten if already filled
✓ Special characters handled correctly
✓ Multiple spaces handled correctly
✓ Empty name doesn't trigger generation
✓ API failure shows fallback + error
✓ Translations work (en, ko, zh)
✓ Form submission works
✓ No console errors
```

Expected: All tests pass

---

## Self-Review Checklist

**Spec Coverage:**
- ✅ Requirement 1: Slug placeholder "SLUG" - Task 1 + Task 3 Step 3-4
- ✅ Requirement 2: Auto-generate on blur - Task 3 Step 2 + Step 5
- ✅ Requirement 3: Only when slug empty - Task 3 Step 2 (guard condition)
- ✅ Requirement 4: Add and edit modes - Task 3 Step 3-4 (both modes)
- ✅ Requirement 5: API call without exclude_id - Task 2 Step 2
- ✅ Requirement 6: API failure handling - Task 2 Step 3, Task 3 Step 2
- ✅ Requirement 7: All translations - Task 1 Step 1-3

**Placeholder Scan:**
- No TBD, TODO, or vague instructions
- All code blocks complete and specific
- All file paths absolute and exact
- All test scenarios have expected outcomes

**Type Consistency:**
- `generateSlugFallback(name: string): string` - consistent across all tasks
- `generateSlugFromAPI(name: string): Promise<string>` - consistent
- `generateSlug(name: string): Promise<{slug: string, error?: string}>` - consistent
- Translation keys match: `products.slugPlaceholder`, `products.slugGenerationError`

**Testing:**
- Manual testing covers all spec scenarios
- Edge cases tested (empty, special chars, multiple spaces)
- Both happy path and failure path tested
- All languages verified
