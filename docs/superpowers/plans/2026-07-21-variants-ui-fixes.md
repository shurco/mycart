# Product Variants UI Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix toggle state bug, improve label clarity, and add unique IDs to variant form inputs

**Architecture:** Four focused UI fixes to Svelte 5 components - Toggle, VariantManager, OptionEditor, and VariantTable. Changes are isolated to component files with no backend modifications.

**Tech Stack:** Svelte 5 (runes: $state, $derived, $effect), TypeScript

## Global Constraints

- Maintain backward compatibility with existing form components
- Follow Svelte 5 runes best practices ($state, $derived, $effect)
- Don't break existing product form functionality  
- Keep changes localized to variant-related components
- No backend API or database schema changes

---

### Task 1: Fix Toggle Component Props

**Files:**
- Modify: `web/admin/src/lib/components/form/Toggle.svelte`

**Interfaces:**
- Consumes: None
- Produces: Toggle component with `checked` prop (alias for `value`) and `label` prop for VariantManager to use

- [ ] **Step 1: Add `checked` and `label` props to Toggle interface**

```typescript
interface Props {
  id?: string
  value?: boolean
  checked?: boolean    // NEW: alias for value
  label?: string       // NEW: optional label text
  disabled?: boolean
  onchange?: () => void
}

let {
  id = 'name',
  value = $bindable(false),
  checked,              // NEW
  label,                // NEW
  disabled = false,
  onchange
}: Props = $props()
```

- [ ] **Step 2: Add derived value that uses checked if provided**

```typescript
// Add after the props destructuring
let computedValue = $derived(checked ?? value)
```

- [ ] **Step 3: Update checkbox binding to use computedValue**

```svelte
<!-- Change line 29 from: -->
bind:checked={value}

<!-- To: -->
bind:checked={computedValue}
```

- [ ] **Step 4: Add label rendering after the toggle element**

```svelte
<!-- Add after the closing </label> tag around line 62 -->
{#if label}
  <span class="ml-3 text-sm font-medium text-gray-700">{label}</span>
{/if}
```

- [ ] **Step 5: Test in browser**

Navigate to product edit page with variants, verify:
- Toggle still works with existing `value` prop usage
- Console shows no errors

Expected: No visual changes yet, but component accepts new props

- [ ] **Step 6: Commit**

```bash
git add web/admin/src/lib/components/form/Toggle.svelte
git commit -m "feat(admin): add checked and label props to Toggle component

- Add checked prop as alias for value (backward compatibility)
- Add label prop for optional label text
- Use derived value to support both prop names

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Fix VariantManager State Sync

**Files:**
- Modify: `web/admin/src/lib/components/product/VariantManager.svelte`

**Interfaces:**
- Consumes: Toggle component with `value` and `label` props (from Task 1)
- Produces: VariantManager with correct toggle state behavior

- [ ] **Step 1: Remove localHasVariants from $effect**

Find the `$effect` around line 29 and modify:

```typescript
// Change from:
$effect(() => {
  localHasVariants = hasVariants    // REMOVE THIS LINE
  localOptions = [...options]
  localVariants = [...variants]
})

// To:
$effect(() => {
  localOptions = [...options]
  localVariants = [...variants]
})
```

- [ ] **Step 2: Change Toggle prop from checked to value**

Find the FormToggle around line 157 and modify:

```svelte
<!-- Change from: -->
<FormToggle
  checked={localHasVariants}
  onchange={toggleHasVariants}
  label={t('products.enableVariants')}
  {disabled}
/>

<!-- To: -->
<FormToggle
  value={localHasVariants}
  onchange={toggleHasVariants}
  label={t('products.enableVariants')}
  {disabled}
/>
```

- [ ] **Step 3: Test toggle behavior in browser**

Test steps:
1. Navigate to product edit: http://localhost:5173/_/products (click edit on 짜장면)
2. Scroll to Product Variants section
3. Verify toggle shows ON (checked state matches has_variants=true)
4. Click toggle to OFF
5. Verify variant options disappear
6. Click toggle to ON
7. Verify empty option form appears
8. Toggle OFF again
9. Verify data clears and toggle shows OFF

Expected: Toggle state correctly reflects and controls variant visibility

- [ ] **Step 4: Commit**

```bash
git add web/admin/src/lib/components/product/VariantManager.svelte
git commit -m "fix(admin): fix variant toggle state sync bug

- Remove localHasVariants from \$effect to prevent reactive loop
- Change Toggle prop from checked to value (correct prop name)
- Toggle now correctly shows/hides variant form

Fixes issue where toggle showed inverted state on page load
and toggling would lose saved variant data.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Update OptionEditor Labels

**Files:**
- Modify: `web/admin/src/lib/components/product/OptionEditor.svelte`

**Interfaces:**
- Consumes: FormInput with `label` and `placeholder` props
- Produces: OptionEditor with numbered labels "Option N Name" and placeholders "Option N Value M"

- [ ] **Step 1: Update option name label to include number**

Find the FormInput for option name around line 82 and modify:

```svelte
<!-- Change from: -->
<FormInput
  label={t('products.optionName')}
  type="text"
  value={localOption.name}
  oninput={updateOptionName}
  placeholder={t('products.optionNamePlaceholder')}
  {disabled}
  required
/>

<!-- To: -->
<FormInput
  label="Option {optionIndex + 1} Name"
  type="text"
  value={localOption.name}
  oninput={updateOptionName}
  placeholder={t('products.optionNamePlaceholder')}
  {disabled}
  required
/>
```

- [ ] **Step 2: Remove generic "Option Values" label**

Find and delete the label element around lines 94-96:

```svelte
<!-- DELETE THESE LINES: -->
<label class="mb-2 block text-sm font-medium text-gray-700">
  {t('products.optionValues')}
</label>
```

- [ ] **Step 3: Add specific placeholders to value inputs**

Find the FormInput for option values around line 101 and modify:

```svelte
<!-- Change from: -->
<FormInput
  type="text"
  value={value.value}
  oninput={(e) => updateValueName(index, e)}
  placeholder={t('products.optionValuePlaceholder')}
  {disabled}
  required
/>

<!-- To: -->
<FormInput
  type="text"
  value={value.value}
  oninput={(e) => updateValueName(index, e)}
  placeholder="Option {optionIndex + 1} Value {index + 1}"
  {disabled}
  required
/>
```

- [ ] **Step 4: Test labels in browser**

Test steps:
1. Navigate to 짜장면 product edit
2. Toggle variants ON
3. Verify first option shows "Option 1 Name" label
4. Verify first value input shows placeholder "Option 1 Value 1"
5. Click "Add Value" button
6. Verify second value input shows placeholder "Option 1 Value 2"
7. Click "Add Option" button to add second option
8. Verify second option shows "Option 2 Name" label
9. Verify its value shows placeholder "Option 2 Value 1"

Expected: All labels and placeholders show correct numbers

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/lib/components/product/OptionEditor.svelte
git commit -m "feat(admin): add numbered labels and placeholders to option editor

- Change option name label from 'Option Name' to 'Option N Name'
- Remove generic 'Option Values' label
- Add specific placeholders 'Option N Value M' to each value input

Improves clarity by showing which option and value number is being edited.

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Add Unique IDs to VariantTable Inputs

**Files:**
- Modify: `web/admin/src/lib/components/product/VariantTable.svelte`

**Interfaces:**
- Consumes: FormInput with `id` prop
- Produces: VariantTable with unique IDs for each input (format: `variant-{index}-{field}`)

- [ ] **Step 1: Add id to SKU input**

Find the SKU FormInput around line 97 and modify:

```svelte
<!-- Change from: -->
<FormInput
  type="text"
  value={variant.sku || ''}
  oninput={(e) => updateVariantSKU(index, e)}
  placeholder="SKU"
  {disabled}
/>

<!-- To: -->
<FormInput
  id="variant-{index}-sku"
  type="text"
  value={variant.sku || ''}
  oninput={(e) => updateVariantSKU(index, e)}
  placeholder="SKU"
  {disabled}
/>
```

- [ ] **Step 2: Add id to price surcharge input**

Find the price surcharge FormInput around line 106 and modify:

```svelte
<!-- Change from: -->
<FormInput
  type="number"
  value={String(variant.price_surcharge)}
  oninput={(e) => updateVariantPrice(index, e)}
  placeholder="0"
  {disabled}
  min="0"
/>

<!-- To: -->
<FormInput
  id="variant-{index}-price"
  type="number"
  value={String(variant.price_surcharge)}
  oninput={(e) => updateVariantPrice(index, e)}
  placeholder="0"
  {disabled}
  min="0"
/>
```

- [ ] **Step 3: Add id to quantity input**

Find the quantity FormInput around line 119 and modify:

```svelte
<!-- Change from: -->
<FormInput
  type="number"
  value={String(variant.quantity)}
  oninput={(e) => updateVariantQuantity(index, e)}
  placeholder="0"
  {disabled}
  min="0"
/>

<!-- To: -->
<FormInput
  id="variant-{index}-quantity"
  type="number"
  value={String(variant.quantity)}
  oninput={(e) => updateVariantQuantity(index, e)}
  placeholder="0"
  {disabled}
  min="0"
/>
```

- [ ] **Step 4: Test unique IDs in browser**

Test steps:
1. Navigate to 짜장면 product edit
2. Scroll to "Generated Variants" section (should show 6 variants)
3. Open browser DevTools (F12)
4. Click on first variant's SKU input
5. In console, run: `document.activeElement.id`
6. Verify output is "variant-0-sku"
7. Click on first variant's price input
8. Run same console command
9. Verify output is "variant-0-price"
10. Click on second variant's SKU input
11. Verify output is "variant-1-sku"
12. Inspect DOM and verify all 18 inputs (6 variants × 3 fields) have unique IDs

Expected: All inputs have unique IDs following pattern `variant-{index}-{field}`

- [ ] **Step 5: Commit**

```bash
git add web/admin/src/lib/components/product/VariantTable.svelte
git commit -m "feat(admin): add unique IDs to variant table inputs

- Add id prop to SKU, price, and quantity inputs
- Format: variant-{index}-{field} (e.g., variant-0-sku)
- Improves accessibility and form management

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 5: Final Verification

**Files:**
- Test: Manual testing in browser

**Interfaces:**
- Consumes: All previous tasks
- Produces: Verified working implementation

- [ ] **Step 1: Test toggle behavior with existing product**

Test steps:
1. Navigate to 짜장면 product edit page
2. Verify toggle shows ON (matches has_variants=true in database)
3. Verify variant options (Size, Spice Level) are visible
4. Verify variant table shows 6 variants
5. Click toggle OFF
6. Verify all variant UI disappears
7. Click toggle ON
8. Verify empty state appears with "Add Option" button
9. Refresh page
10. Verify toggle is still ON and empty option form shows

Expected: Toggle correctly controls variant visibility, state persists

- [ ] **Step 2: Test option labels with new options**

Test steps:
1. From empty option state, click "Add Option"
2. Verify label shows "Option 1 Name"
3. Verify first value placeholder shows "Option 1 Value 1"
4. Enter "Test Size" for option name
5. Enter "Small" for first value
6. Click "Add Value" button
7. Verify second value placeholder shows "Option 1 Value 2"
8. Enter "Large" for second value
9. Click "Add Option" to add second option
10. Verify label shows "Option 2 Name"
11. Verify its first value placeholder shows "Option 2 Value 1"

Expected: All labels and placeholders show correct numbers

- [ ] **Step 3: Test variant table unique IDs**

Test steps:
1. Complete Option 1 with values "Small", "Large"
2. Complete Option 2 with values "Red", "Blue"
3. Verify 4 variants generated in table
4. Open DevTools, click each input field
5. Run `document.activeElement.id` in console
6. Verify IDs: variant-0-sku, variant-0-price, variant-0-quantity, variant-1-sku, etc.
7. Enter test data in SKU fields
8. Tab through all inputs
9. Verify keyboard navigation works correctly

Expected: All inputs have unique IDs, form is keyboard-accessible

- [ ] **Step 4: Test edge cases**

Test steps:
1. Toggle OFF to clear all data
2. Toggle ON
3. Add 3 options with 2 values each
4. Verify all labels numbered correctly (Option 1, 2, 3)
5. Delete Option 2
6. Verify remaining options still show correct numbers
7. Verify variants re-generate correctly
8. Save product
9. Refresh page
10. Verify all data persists correctly

Expected: Labels renumber dynamically, data persists

- [ ] **Step 5: Document completion**

All success criteria met:
- [x] Toggle shows correct state on initial load
- [x] Toggling ON shows variant form
- [x] Toggling OFF hides variant form
- [x] Option labels show "Option N Name"
- [x] Value inputs show placeholders "Option N Value M"
- [x] All variant table inputs have unique IDs
- [x] Form remains accessible and keyboard-navigable

No commit needed - this is verification only.
