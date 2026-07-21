# Product Variants UI Fixes

## Goal
Fix critical UI bugs and improve user experience in the product variants form components.

## Problem Statement

The product variants feature has four main issues:

1. **Toggle State Bug**: When a product with existing variants loads, the toggle shows OFF but variant data is visible. Toggling inverts the expected behavior - ON hides data, OFF shows empty form. Saved variant data disappears.

2. **Generic Labels**: Option name shows generic "Option Name" instead of numbered "Option 1 Name", "Option 2 Name". Value inputs share a single "Option Values" label instead of individual placeholders.

3. **Missing Unique IDs**: Variant table inputs lack unique `id` attributes, causing accessibility and form management issues.

4. **Prop Mismatch**: VariantManager passes `checked` prop to Toggle component, but Toggle expects `value` prop.

## Root Causes

**Toggle Bug**: Two technical issues -
- VariantManager passes `checked={localHasVariants}` but Toggle component's bindable prop is named `value`
- The `$effect` in VariantManager constantly resyncs `localHasVariants` with the `hasVariants` prop, creating a reactive loop that reverts user actions

**Labels**: Hardcoded translation keys without dynamic numbering

**IDs**: FormInput components instantiated without `id` prop

## Architecture

### Component Hierarchy
```
VariantManager
├── FormToggle (enable/disable variants)
├── OptionEditor (for each option)
│   ├── FormInput (option name)
│   └── FormInput[] (option values)
└── VariantTable
    └── FormInput[] (SKU, price, quantity per variant)
```

### Data Flow
1. Parent component passes `hasVariants`, `options`, `variants` props to VariantManager
2. VariantManager maintains local state for user edits
3. On changes, VariantManager calls `onUpdate()` callback to notify parent
4. Props flow back down, triggering re-sync via `$effect`

**Problem**: The re-sync includes `localHasVariants`, which reverts user toggle actions.

## Solution Design

### 1. Toggle Component Enhancement

**File**: `web/admin/src/lib/components/form/Toggle.svelte`

Add backward-compatible props:
```typescript
interface Props {
  id?: string
  value?: boolean      // Existing bindable
  checked?: boolean    // New: alias for value
  label?: string       // New: optional label text
  disabled?: boolean
  onchange?: () => void
}

// Derive actual value
let computedValue = $derived(checked ?? value)
```

Render label if provided:
```svelte
{#if label}
  <span class="ml-3 text-sm font-medium text-gray-700">{label}</span>
{/if}
```

### 2. VariantManager State Management

**File**: `web/admin/src/lib/components/product/VariantManager.svelte`

**Change 1**: Remove `localHasVariants` from `$effect`
```typescript
// Before:
$effect(() => {
  localHasVariants = hasVariants  // REMOVE THIS
  localOptions = [...options]
  localVariants = [...variants]
})

// After:
$effect(() => {
  localOptions = [...options]
  localVariants = [...variants]
})
```

**Rationale**: `localHasVariants` should only be set on mount and by user toggle. Parent component should not control it after mount.

**Change 2**: Fix Toggle prop name
```svelte
<!-- Before -->
<FormToggle
  checked={localHasVariants}
  onchange={toggleHasVariants}
  label={t('products.enableVariants')}
  {disabled}
/>

<!-- After -->
<FormToggle
  value={localHasVariants}
  onchange={toggleHasVariants}
  label={t('products.enableVariants')}
  {disabled}
/>
```

### 3. OptionEditor Label Updates

**File**: `web/admin/src/lib/components/product/OptionEditor.svelte`

**Change 1**: Numbered option name label
```svelte
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

**Change 2**: Remove generic "Option Values" label
```svelte
<!-- DELETE THIS -->
<label class="mb-2 block text-sm font-medium text-gray-700">
  {t('products.optionValues')}
</label>
```

**Change 3**: Add specific placeholders to value inputs
```svelte
{#each localOption.values as value, index}
  <div class="mb-2 flex items-center gap-2">
    <div class="flex-1">
      <FormInput
        type="text"
        value={value.value}
        oninput={(e) => updateValueName(index, e)}
        placeholder="Option {optionIndex + 1} Value {index + 1}"
        {disabled}
        required
      />
    </div>
    <!-- delete button -->
  </div>
{/each}
```

### 4. VariantTable Unique IDs

**File**: `web/admin/src/lib/components/product/VariantTable.svelte`

Add unique `id` to each FormInput:

```svelte
{#each localVariants as variant, index}
  <tr>
    <td>
      <FormInput
        id="variant-{index}-sku"
        type="text"
        value={variant.sku || ''}
        oninput={(e) => updateVariantSKU(index, e)}
        placeholder="SKU"
        {disabled}
      />
    </td>
    <td>
      <FormInput
        id="variant-{index}-price"
        type="number"
        value={String(variant.price_surcharge)}
        oninput={(e) => updateVariantPrice(index, e)}
        placeholder="0"
        {disabled}
        min="0"
      />
    </td>
    <td>
      <FormInput
        id="variant-{index}-quantity"
        type="number"
        value={String(variant.quantity)}
        oninput={(e) => updateVariantQuantity(index, e)}
        placeholder="0"
        {disabled}
        min="0"
      />
    </td>
  </tr>
{/each}
```

**ID Format**: `variant-{index}-{field}` where:
- `index`: 0-based variant position in array
- `field`: `sku`, `price`, or `quantity`

## Files Changed

1. `web/admin/src/lib/components/form/Toggle.svelte` - Add `checked` and `label` props
2. `web/admin/src/lib/components/product/VariantManager.svelte` - Fix state sync and prop name
3. `web/admin/src/lib/components/product/OptionEditor.svelte` - Update labels and placeholders
4. `web/admin/src/lib/components/product/VariantTable.svelte` - Add unique IDs

## Testing

### Toggle Behavior
1. Load product with `has_variants=true` and existing options/variants
   - **Expected**: Toggle shows ON, variant data visible
2. Click toggle to OFF
   - **Expected**: Variant data disappears, options/variants cleared
3. Click toggle to ON
   - **Expected**: Empty option form appears with "Add Option" button
4. Toggle OFF → ON → OFF again
   - **Expected**: Behavior consistent each time, no data loss

### Labels
1. Add first option
   - **Expected**: Label shows "Option 1 Name"
2. Add second option
   - **Expected**: Label shows "Option 2 Name"
3. Add values to Option 1
   - **Expected**: Placeholders show "Option 1 Value 1", "Option 1 Value 2", etc.
4. Add values to Option 2
   - **Expected**: Placeholders show "Option 2 Value 1", "Option 2 Value 2", etc.

### Unique IDs
1. Generate variants
2. Inspect DOM
   - **Expected**: Each input has unique ID like `variant-0-sku`, `variant-1-price`, etc.
3. Click input, check `document.activeElement.id`
   - **Expected**: Correct unique ID

## Success Criteria

- [ ] Toggle shows correct state on initial load with existing variants
- [ ] Toggling ON shows variant form
- [ ] Toggling OFF hides variant form and clears data
- [ ] Variant data persists across page refreshes when toggle is ON
- [ ] Option labels show "Option 1 Name", "Option 2 Name", etc.
- [ ] Value inputs show placeholders "Option N Value M"
- [ ] All variant table inputs have unique IDs
- [ ] Form remains accessible and keyboard-navigable

## Edge Cases

- Product with no variants (hasVariants=false) → Toggle OFF, no data shown
- Product with 3 options and 100 variants → All IDs still unique
- Deleting an option → Remaining options renumber correctly (Option 1, Option 2)
- Deleting a value → Remaining values renumber correctly (Value 1, Value 2)

## Non-Goals

- Changing the overall variant generation logic
- Modifying backend API
- Updating database schema
- Changing translation keys (except using them dynamically)
- Refactoring to use two-way binding throughout

## Constraints

- Maintain backward compatibility with existing form components
- Follow Svelte 5 runes best practices ($state, $derived, $effect)
- Don't break existing product form functionality
- Keep changes localized to variant-related components
