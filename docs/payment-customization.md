# Payment Interface Customization

This document describes how to customize the appearance and behavior of the payment interface for customers in myCart.

## Overview

This document describes how to customize the appearance and behavior of the payment interface for customers in myCart.

## Main Files

### Cart and Payment Selection Page
**Path**: `web/site/src/routes/cart/+page.svelte`

This is the main file displayed to customers during checkout. Here you can customize:
- Appearance of payment provider cards
- Order of payment methods display
- Additional information (fees, logos, badges)
- Layout (vertical list or grid)

### Payment Result Pages
- **Successful Payment**: `web/site/src/routes/cart/payment/success/+page.svelte`
- **Canceled Payment**: `web/site/src/routes/cart/payment/cancel/+page.svelte`

### Utilities and Translations
- **Payment Utilities**: `web/site/src/lib/utils/payment.ts`
- **Cart Store**: `web/site/src/lib/stores/cart.ts`
- **Translations**: `web/site/src/lib/i18n/locales/en.json` (and other languages)

## Quick Start: Simple Changes

### 1. Change Provider Card Styles

Find in `web/site/src/routes/cart/+page.svelte` (around lines ~280-290):

```svelte
<label
  for="stripe"
  class="block cursor-pointer border-4 border-black bg-white p-6 peer-checked:border-yellow-300 peer-checked:bg-yellow-300"
>
```

Change Tailwind classes to your desired style:

```svelte
<!-- Example: softer design -->
<label
  for="stripe"
  class="block cursor-pointer rounded-lg border-2 border-gray-300 bg-white p-6 shadow-md hover:shadow-xl peer-checked:border-blue-500 peer-checked:bg-blue-50"
>
```

### 2. Add Provider Logos

Add SVG logos to `web/site/static/assets/img/payments/`:
- `stripe.svg`
- `paypal.svg`
- `spectrocoin.svg`
- `coinbase.svg`

Then in the component:

```svelte
<label for="stripe" class="...">
  <div class="flex items-center gap-4">
    <img src="/assets/img/payments/stripe.svg" alt="Stripe" class="h-10" />
    <div>
      <p class="text-xl font-bold">{t('cart.stripe')}</p>
      <p class="text-sm">{t('cart.stripeDescription')}</p>
    </div>
  </div>
</label>
```

### 3. Change Layout to Grid

Replace `class="space-y-4"` with grid:

```svelte
<fieldset class="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
  <!-- provider cards -->
</fieldset>
```

### 4. Add Badges and Recommendations

```svelte
<label for="stripe" class="relative ...">
  <!-- Card content -->
  
  <!-- Recommendation badge -->
  <span class="absolute -top-2 left-1/2 -translate-x-1/2 rounded bg-green-500 px-3 py-1 text-xs font-bold text-white uppercase">
    Recommended
  </span>
</label>
```

## Advanced Settings

### Creating Provider Configuration

Create `web/site/src/lib/config/payment.ts`:

```typescript
export const PAYMENT_PROVIDER_ORDER = ['stripe', 'paypal', 'spectrocoin', 'coinbase'] as const;

export const PAYMENT_PROVIDER_INFO = {
  stripe: {
    name: 'Stripe',
    description: 'Credit/Debit Cards',
    icon: '/assets/img/payments/stripe.svg',
    badge: 'Recommended',
    fee: '2.9% + $0.30'
  },
  paypal: {
    name: 'PayPal',
    description: 'PayPal balance or cards',
    icon: '/assets/img/payments/paypal.svg',
    badge: 'Fast',
    fee: '3.4% + fixed'
  },
  spectrocoin: {
    name: 'SpectroCoin',
    description: 'Cryptocurrencies',
    icon: '/assets/img/payments/spectrocoin.svg',
    fee: 'From 1%'
  },
  coinbase: {
    name: 'Coinbase Commerce',
    description: 'Cryptocurrencies via Coinbase',
    icon: '/assets/img/payments/coinbase.svg',
    fee: '1%'
  }
};
```

### Creating Reusable Component

Create `web/site/src/lib/components/PaymentProviderCard.svelte`:

```svelte
<script lang="ts">
  interface Props {
    id: string
    name: string
    description: string
    icon?: string
    badge?: string
    fee?: string
    selected: boolean
    onSelect: (id: string) => void
  }
  
  let { id, name, description, icon, badge, fee, selected, onSelect }: Props = $props()
</script>

<div class="relative">
  <input 
    type="radio" 
    {id} 
    checked={selected} 
    onchange={() => onSelect(id)}
    class="peer hidden" 
  />
  <label
    for={id}
    class="block cursor-pointer border-4 border-black bg-white p-6 transition-all peer-checked:border-yellow-300 peer-checked:bg-yellow-300"
  >
    <div class="flex items-center gap-4">
      {#if icon}
        <img src={icon} alt={name} class="h-12 w-12" />
      {/if}
      <div class="flex-1">
        <p class="mb-1 text-xl font-black uppercase">{name}</p>
        <p class="text-lg">{description}</p>
        {#if fee}
          <p class="mt-1 text-sm text-gray-600">Fee: {fee}</p>
        {/if}
      </div>
    </div>
  </label>
  
  {#if badge}
    <span class="absolute -top-2 right-4 rounded bg-green-500 px-3 py-1 text-xs font-bold text-white uppercase">
      {badge}
    </span>
  {/if}
</div>
```

Usage:

```svelte
<script lang="ts">
  import PaymentProviderCard from '$lib/components/PaymentProviderCard.svelte'
  import { PAYMENT_PROVIDER_INFO } from '$lib/config/payment'
</script>

<fieldset class="space-y-4">
  {#each Object.entries(payments).filter(([_, active]) => active) as [key, _]}
    {@const info = PAYMENT_PROVIDER_INFO[key]}
    <PaymentProviderCard
      id={key}
      {...info}
      selected={provider === key}
      onSelect={(id) => provider = id}
    />
  {/each}
</fieldset>
```

## Text Customization

All texts are stored in `web/site/src/lib/i18n/locales/`:

**en.json**:
```json
{
  "cart": {
    "stripe": "Credit/Debit Card",
    "stripeDescription": "Visa, Mastercard, Amex",
    "paypal": "PayPal",
    "paypalDescription": "PayPal account or card",
    "spectrocoin": "Cryptocurrency",
    "spectrocoinDescription": "Bitcoin, Ethereum, and more",
    "coinbase": "Coinbase Commerce",
    "coinbaseDescription": "Pay with Bitcoin, Ethereum, and other crypto",
    "paymentSecure": "🔒 All payments are secure and encrypted",
    "recommended": "Recommended"
  }
}
```

## Design Examples

### Minimalist Design

```svelte
<label
  for="stripe"
  class="flex items-center justify-between rounded-lg border border-gray-200 bg-white p-4 hover:bg-gray-50 peer-checked:border-blue-500 peer-checked:bg-blue-50"
>
  <div class="flex items-center gap-3">
    <img src="/assets/img/payments/stripe.svg" alt="Stripe" class="h-8" />
    <span class="font-medium">{t('cart.stripe')}</span>
  </div>
  <div class="h-5 w-5 rounded-full border-2 border-gray-300 peer-checked:border-blue-500 peer-checked:bg-blue-500"></div>
</label>
```

### Card Design with Grid

```svelte
<fieldset class="grid grid-cols-3 gap-4">
  <label
    for="stripe"
    class="flex cursor-pointer flex-col items-center rounded-xl border-2 border-gray-200 bg-white p-6 hover:border-blue-500 peer-checked:border-blue-500 peer-checked:ring-4 peer-checked:ring-blue-100"
  >
    <img src="/assets/img/payments/stripe.svg" alt="Stripe" class="mb-3 h-12" />
    <span class="text-center font-semibold">{t('cart.stripe')}</span>
  </label>
</fieldset>
```

### Detailed Design

```svelte
<label
  for="stripe"
  class="block cursor-pointer rounded-lg border-2 border-gray-200 bg-white p-6 shadow-sm hover:shadow-md peer-checked:border-blue-500 peer-checked:shadow-lg"
>
  <div class="flex items-start gap-4">
    <img src="/assets/img/payments/stripe.svg" alt="Stripe" class="h-12 w-12" />
    <div class="flex-1">
      <h3 class="mb-1 text-lg font-bold">{t('cart.stripe')}</h3>
      <p class="mb-2 text-sm text-gray-600">{t('cart.stripeDescription')}</p>
      <div class="flex items-center gap-2">
        <img src="/assets/img/cards/visa.svg" alt="Visa" class="h-6" />
        <img src="/assets/img/cards/mastercard.svg" alt="Mastercard" class="h-6" />
        <img src="/assets/img/cards/amex.svg" alt="Amex" class="h-6" />
      </div>
      <p class="mt-2 text-xs text-gray-500">Fee: 2.9% + $0.30</p>
    </div>
    <div class="text-green-500 peer-checked:block hidden">✓</div>
  </div>
</label>
```

## Recommendations

1. **Accessibility**: use proper ARIA attributes
2. **Mobile devices**: test on all screen sizes
3. **Performance**: optimize icons (use SVG)
4. **Branding**: follow provider guidelines when using their logos
5. **UX**: show loading states and handle errors

## Styling

The project uses Tailwind CSS. You can:
- Use built-in Tailwind utilities
- Add custom classes in `web/site/src/app.css`
- Configure in `web/site/tailwind.config.js`

## Additional Help

Or open an issue on GitHub: https://github.com/shurco/mycart/issues
