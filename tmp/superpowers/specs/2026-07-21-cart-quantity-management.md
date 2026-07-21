# Cart Quantity Management Design Spec

**Date:** 2026-07-21  
**Goal:** Add quantity management to cart system and improve variant product UX on list and detail pages

## Overview

Redesign product list and detail pages to support cart quantity management, especially for products with variants. Current system treats cart as binary (in/out) with no quantity support. New system allows users to add multiple quantities of products/variants, with smart UI that adapts based on variant availability.

## Architecture

**Approach:** Incremental enhancement of existing cart system

- Extend CartItem type with quantity field
- Add quantity management methods to cart store
- Build reusable QuantityInput component
- Update ProductCard to detect variants and show appropriate UI
- Update product detail page with quantity input and cart items display

**Tech Stack:**
- Svelte 5 (existing)
- TypeScript
- LocalStorage for cart persistence (existing)
- Existing cart store pattern (writable store)

## Global Constraints

- Minimum quantity: 1 (cannot decrement below 1, separate remove button required)
- Quantity starts at 1 when adding to cart
- If item already in cart, show existing quantity
- Cart persists to localStorage on every change
- Maintain brutalist UI design (bold borders, high contrast, uppercase text)
- Backward compatible: existing carts migrate silently (quantity defaults to 1)

---

## Component 1: Data Model Changes

### CartItem Type Extension

**File:** `web/site/src/lib/types/models.ts`

```typescript
export interface CartItem {
  id: string              // product id
  name: string           // product name
  slug: string           // product slug
  amount: number         // final price (base + variant surcharge)
  quantity: number       // NEW - number of this item in cart (min: 1)
  image?: { name: string; ext: string } | null
  variant_id?: string    // optional variant identifier
  variant_name?: string  // optional variant display name (e.g., "Size: S, Color: Red")
}
```

**Why quantity field:**
- Supports multiple of same item/variant
- Required for e-commerce checkout flow
- Enables bulk operations (increase/decrease)

**Migration Strategy:**
- On cart load from localStorage, check if quantity field exists
- If missing, set quantity = 1 (default)
- Happens in `loadFromStorage()` function
- Silent migration, preserves user's cart

---

## Component 2: Cart Store Methods

### New Methods

**File:** `web/site/src/lib/stores/cart.ts`

#### updateQuantity
```typescript
updateQuantity: (productId: string, variantId: string | undefined, quantity: number) => void
```
- Updates quantity for specific cart item (product + optional variant)
- Clamps quantity to minimum of 1 (enforces business rule)
- Saves updated cart to localStorage
- Triggers reactivity for all subscribers

**Implementation:**
```typescript
updateQuantity: (productId: string, variantId: string | undefined, quantity: number) => {
  update((items) => {
    const newItems = items.map(item => {
      const matches = variantId 
        ? (item.id === productId && item.variant_id === variantId)
        : (item.id === productId && !item.variant_id)
      
      if (matches) {
        return { ...item, quantity: Math.max(1, quantity) }
      }
      return item
    })
    saveToStorage(newItems)
    return newItems
  })
}
```

#### incrementQuantity
```typescript
incrementQuantity: (productId: string, variantId: string | undefined) => void
```
- Helper method: increases quantity by 1
- Calls updateQuantity with current + 1

#### decrementQuantity
```typescript
decrementQuantity: (productId: string, variantId: string | undefined) => void
```
- Helper method: decreases quantity by 1
- Stops at minimum (1), does not remove item
- Calls updateQuantity with max(current - 1, 1)

### Updated add Method

**Change:** When adding item already in cart, increase quantity instead of ignoring

```typescript
add: (item: CartItem) => {
  update((items) => {
    const existing = items.find((i) => {
      if (item.variant_id) {
        return i.id === item.id && i.variant_id === item.variant_id
      }
      return i.id === item.id && !i.variant_id
    })

    if (existing) {
      // NEW: Update quantity instead of returning unchanged
      return items.map(i => 
        i === existing 
          ? { ...i, quantity: i.quantity + item.quantity }
          : i
      )
    }

    const newItems = [...items, { ...item, quantity: item.quantity || 1 }]
    saveToStorage(newItems)
    return newItems
  })
}
```

**Why:** Adding same item should increase quantity, not be ignored (better UX)

---

## Component 3: QuantityInput Component

### Purpose
Reusable component for quantity selection with increment/decrement buttons and manual input.

**File:** `web/site/src/lib/components/QuantityInput.svelte`

### Props
```typescript
interface Props {
  quantity: number           // Current quantity value
  min?: number              // Minimum allowed (default: 1)
  max?: number              // Maximum allowed (optional)
  disabled?: boolean        // Disable all controls (default: false)
  onIncrement: () => void   // Called when + button clicked
  onDecrement: () => void   // Called when - button clicked
  onChange: (value: number) => void  // Called when manual input changes
}
```

### Layout
```
┌────────────────────────┐
│  [-]  [  2  ]  [+]    │
└────────────────────────┘
```

### Behavior
- **Minus button:** Calls onDecrement, disabled when quantity = min
- **Number input:** Manual entry, validates on blur (clamps to min/max)
- **Plus button:** Calls onIncrement, disabled if max set and quantity = max
- **Validation:** Non-numeric input reverts to previous value

### Styling
- Brutalist design: thick black borders, high contrast
- Buttons: square, bold borders, uppercase
- Input: center-aligned text, monospace font
- Disabled state: reduced opacity (0.5)

---

## Component 4: ProductCard Updates

### Purpose
Display product in list view, adapt UI based on variant availability.

**File:** `web/site/src/lib/components/ProductCard.svelte`

### New Logic: Variant Detection

```typescript
let hasVariants = $derived(product.has_variants && product.options?.length > 0)
let cartItem = $derived(cart.find(item => item.id === product.id && !item.variant_id))
let inCart = $derived(!!cartItem)
let currentQuantity = $derived(cartItem?.quantity || 1)
```

### Layout: Product WITHOUT Variants

```
┌─────────────────────────────┐
│      Product Image          │
├─────────────────────────────┤
│ Product Name                │
│ $25.00 USD                  │
│                             │
│ [- Qty: 2 +]  [Add Cart]    │
└─────────────────────────────┘
```

**Components:**
- QuantityInput (if in cart, shows current quantity; else shows 1)
- Add/Remove button (toggles cart with current quantity)

**Behavior:**
- Not in cart: QuantityInput defaults to 1, button says "Add to Cart"
- In cart: QuantityInput shows cart quantity, button says "Remove"
- Increment/decrement: updates cart quantity immediately
- Add: calls `cartStore.add()` with selected quantity
- Remove: calls `cartStore.remove()`

### Layout: Product WITH Variants

```
┌─────────────────────────────┐
│      Product Image          │
├─────────────────────────────┤
│ Product Name                │
│ $25.00 USD                  │
│                             │
│ "3 variants available"      │
│ [Select Options →]          │
└─────────────────────────────┘
```

**Components:**
- Variant summary text (e.g., "3 variants" or "Multiple options")
- "Select Options" button → navigates to product detail page

**Behavior:**
- No quantity input (must select variant first)
- No add/remove button (requires variant selection)
- Button navigates to `/products/{slug}`

---

## Component 5: Product Detail Page Updates

### Purpose
Allow variant selection, quantity input, cart management for current product.

**File:** `web/site/src/routes/products/[slug]/+page.svelte`

### Changes

#### 1. Remove "In Stock" Display
- Delete stock status indicator
- Replace with quantity input

#### 2. Add Quantity State
```typescript
let selectedQuantity = $state(1)
```

#### 3. Update Layout (above fold)

**Before:**
```
[Variant Selector]
In Stock
[Add to Cart / Remove from Cart]
```

**After:**
```
[Variant Selector]
[- Qty: 1 +]
[Add to Cart / Remove from Cart]
```

#### 4. Smart Add/Remove Button Logic

```typescript
let cartItem = $derived(() => {
  if (!product) return null
  if (product.has_variants && selectedVariant) {
    return cart.find(item => item.id === product.id && item.variant_id === selectedVariant.id)
  }
  return cart.find(item => item.id === product.id && !item.variant_id)
})

let inCart = $derived(!!cartItem())
```

**Button shows:**
- "Add to Cart" if !inCart
- "Remove from Cart" if inCart

**On click:**
- Add: calls `toggleCartItem(product, cart, selectedVariant)` with selectedQuantity
- Remove: calls `cartStore.removeVariant()` or `cartStore.remove()`

**Note:** When switching variants, if new variant is in cart, show "Remove"; if not, show "Add"

#### 5. Cart Items Section (below button)

**New Section Title:** "In Your Cart (for this product):"

**Filter Logic:**
```typescript
let productCartItems = $derived(
  cart.filter(item => item.id === product?.id)
)
```

**Display:**
- If no items: hide section
- If items exist: show CartItemCard for each

**Layout:**
```
┌─────────────────────────────────────┐
│ In Your Cart:                       │
├─────────────────────────────────────┤
│ ┌─────────────────────────────────┐ │
│ │ Size: S, Color: Red             │ │
│ │ $25.00                          │ │
│ │ [- Qty: 2 +]  [Remove]          │ │
│ └─────────────────────────────────┘ │
│ ┌─────────────────────────────────┐ │
│ │ Size: L, Color: Blue            │ │
│ │ $28.00                          │ │
│ │ [- Qty: 1 +]  [Remove]          │ │
│ └─────────────────────────────────┘ │
└─────────────────────────────────────┘
```

---

## Component 6: CartItemCard Component

### Purpose
Display single cart item with quantity controls and remove button.

**File:** `web/site/src/lib/components/CartItemCard.svelte`

### Props
```typescript
interface Props {
  item: CartItem
  onQuantityChange: (newQuantity: number) => void
  onRemove: () => void
}
```

### Layout
```
┌─────────────────────────────────┐
│ Size: S, Color: Red             │
│ $25.00                          │
│ [- Qty: 2 +]  [Remove]          │
└─────────────────────────────────┘
```

### Components
- Variant name (or product name if no variant)
- Price display
- QuantityInput component
- Remove button

### Behavior
- QuantityInput calls onQuantityChange when +/- clicked
- Remove button calls onRemove
- Brutalist styling (borders, high contrast)

---

## Edge Cases & Error Handling

### Edge Case 1: Adding Same Variant Already in Cart
**Scenario:** Cart has 2x "Size: S", user adds 3 more "Size: S"  
**Behavior:** Quantity becomes 5 total (existing + new)  
**Implementation:** Updated `add()` method increases quantity

### Edge Case 2: Product Without Variants
**Scenario:** Product has no options/variants  
**Behavior:**  
- ProductCard: shows quantity input + add/remove button
- Detail page: works same as variant products
- CartItem has no variant_id

### Edge Case 3: Switching Variants on Detail Page
**Scenario:** User has "Size: S" in cart (qty: 2), selects "Size: M"  
**Behavior:**  
- Button shows "Add to Cart" (different variant, not in cart)
- Quantity input resets to 1
- Both "Size: S" and "Size: M" show in cart items section after adding

### Edge Case 4: Manual Quantity Input Validation
**Input: "0" or negative** → Validates to min (1)  
**Input: non-numeric** → Reverts to current value, shows validation message  
**Input: exceeds max (if set)** → Clamps to max value

### Edge Case 5: Variant Becomes Unavailable
**Scenario:** Cart item references variant that was deleted/disabled  
**Behavior:**  
- Display in cart with "(No longer available)" tag
- Allow removal
- Disable quantity changes
- Show warning on cart/checkout page

### Edge Case 6: Cart Migration
**Scenario:** User has old cart data (no quantity field)  
**Behavior:**  
- On load, detect missing quantity
- Set quantity = 1 for all items
- Save updated cart
- User sees no disruption

---

## Testing Strategy

### Unit Tests

**Cart Store (`cart.ts`):**
```
✓ add() sets quantity=1 for new items
✓ add() increases quantity for existing items
✓ updateQuantity() updates correct item
✓ updateQuantity() clamps to min=1
✓ incrementQuantity() increases by 1
✓ decrementQuantity() stops at 1
✓ remove() deletes item completely
✓ removeVariant() deletes specific variant
✓ Migration: old cart items get quantity=1
✓ localStorage sync on every change
```

**QuantityInput Component:**
```
✓ Renders with initial value
✓ Increment button increases value, calls onIncrement
✓ Decrement button decreases value, calls onDecrement
✓ Decrement disabled at min value
✓ Manual input validates to min/max
✓ Non-numeric input reverts to previous
✓ onChange callback fires with validated value
```

**ProductCard Component:**
```
✓ No variants: shows quantity + add button
✓ Has variants: shows "N variants" + "Select Options"
✓ In cart (no variants): shows current quantity
✓ Not in cart: defaults quantity to 1
✓ Increment/decrement updates cart immediately
✓ Add button adds with selected quantity
✓ Remove button removes item
```

**Product Detail Page:**
```
✓ Shows quantity input (default 1)
✓ Smart button: "Add" when variant not in cart
✓ Smart button: "Remove" when variant in cart
✓ Button switches on variant selection change
✓ Cart items section shows only current product
✓ Cart items section hidden when empty
✓ Quantity changes update cart store
✓ Remove button removes specific variant
```

**CartItemCard Component:**
```
✓ Displays variant name or product name
✓ Displays price
✓ QuantityInput shows current quantity
✓ Increment calls onQuantityChange
✓ Decrement calls onQuantityChange (min 1)
✓ Remove button calls onRemove
```

### Integration Tests

```
✓ Add product from list → appears in cart
✓ Add variant from detail → appears in cart items section
✓ Change quantity on detail → updates ProductCard state
✓ Remove from detail → ProductCard updates to "Add"
✓ Navigate between products → cart items filter correctly
✓ Add multiple variants of same product → all show in cart items
✓ Reload page → cart persists with quantities
```

### Manual Testing Checklist

```
□ Product list: variant vs non-variant display correct
□ Add to cart from list (non-variant product)
□ Navigate to detail page for variant product
□ Select variant, add to cart with quantity 3
□ Change variant selection, verify button changes
□ Add second variant to cart
□ Verify both variants show in cart items section
□ Increment quantity for one variant
□ Decrement quantity to 1 (verify can't go below)
□ Remove one variant from cart items
□ Verify other variant remains
□ Reload page, verify cart persisted
□ Test on mobile: layout responsive
□ Test brutalist styling: borders, contrast, uppercase
```

---

## Implementation Notes

### File Changes Summary

**New Files:**
- `web/site/src/lib/components/QuantityInput.svelte`
- `web/site/src/lib/components/CartItemCard.svelte`

**Modified Files:**
- `web/site/src/lib/types/models.ts` - Add quantity to CartItem
- `web/site/src/lib/stores/cart.ts` - Add updateQuantity, increment, decrement methods
- `web/site/src/lib/utils/cart.ts` - Update toggleCartItem to support quantity
- `web/site/src/lib/components/ProductCard.svelte` - Add variant detection, quantity input
- `web/site/src/routes/products/[slug]/+page.svelte` - Add quantity input, cart items section

### Data Flow

1. **User adds item:**
   - ProductCard or Detail page → toggleCartItem() → cartStore.add() → localStorage
   - Cart state updates → all components re-render

2. **User changes quantity:**
   - QuantityInput → onIncrement/onDecrement → cartStore.updateQuantity() → localStorage
   - Cart state updates → UI reflects new quantity

3. **User removes item:**
   - CartItemCard → onRemove → cartStore.remove/removeVariant() → localStorage
   - Cart state updates → item disappears from UI

### Styling Guidelines

- Follow existing brutalist design patterns
- Thick black borders (4px)
- High contrast colors (black text on white/yellow/green/red backgrounds)
- Uppercase text for buttons and labels
- Hover effects: translate + box shadow
- Disabled state: opacity 0.5, no hover effects

---

## Success Criteria

**User Experience:**
- ✓ Can add multiple quantities of products/variants
- ✓ Clear distinction between variant and non-variant products on list
- ✓ Easy quantity adjustment (increment/decrement buttons)
- ✓ Can see all cart items for current product on detail page
- ✓ Can manage (edit quantity, remove) cart items without leaving detail page

**Technical:**
- ✓ Cart persists to localStorage with quantities
- ✓ Existing carts migrate without data loss
- ✓ All tests pass
- ✓ No TypeScript errors
- ✓ Mobile responsive
- ✓ Brutalist design maintained

**Business:**
- ✓ Enables multi-quantity purchases
- ✓ Reduces friction for variant selection
- ✓ Clear cart management UX
- ✓ Foundation for future features (inventory limits, bulk discounts)
