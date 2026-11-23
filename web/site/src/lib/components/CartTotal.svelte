<script lang="ts">
  import { settingsStore } from '$lib/stores/settings'
  import { formatCurrency } from '$lib/utils/currency'
  import { translate, locale } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    // Total in cents (amount * quantity summed over the cart)
    cartTotal: number
    currency: string
  }

  let { cartTotal, currency }: Props = $props()

  let numberFormat = $derived($settingsStore?.payment?.number_format)
  let symbolMode = $derived($settingsStore?.payment?.symbol_display?.storefront)
  let currentLocale = $derived($locale)

  // Computed value instead of function - using formatCurrency (no truncation) for exact totals
  let totalCartAmount = $derived(
    cartTotal === 0
      ? t('product.free')
      : formatCurrency(cartTotal / 100, currency, numberFormat, symbolMode, currentLocale)
  )
</script>

<!-- Total -->
<div class="brutal-card mb-8 bg-yellow-300 p-8">
  <div class="flex items-center justify-between">
    <span class="text-3xl font-black tracking-tighter text-black uppercase"> {t('cart.total')} </span>
    <span class="text-4xl font-black {cartTotal === 0 ? 'text-green-500' : 'text-black'}" data-testid="cart-total">
      {totalCartAmount}
    </span>
  </div>
</div>
