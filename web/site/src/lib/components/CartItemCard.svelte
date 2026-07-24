<script lang="ts">
  import type { CartItem } from '$lib/types/models'
  import { cartStore } from '$lib/stores/cart'
  import { formatCurrencyWithTruncation } from '$lib/utils/currency'
  import { settingsStore } from '$lib/stores/settings'
  import { getProductImageUrl } from '$lib/utils/imageUrl'
  import { translate, locale } from '$lib/i18n'
  import QuantityInput from './QuantityInput.svelte'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    item: CartItem
    highlighted?: boolean
    needsDeletion?: boolean
  }

  let { item, highlighted = false, needsDeletion = false }: Props = $props()

  let currency = $derived($settingsStore?.main.currency || '')
  let truncationSettings = $derived($settingsStore?.payment?.truncation)
  let numberFormat = $derived($settingsStore?.payment?.number_format)
  let symbolMode = $derived($settingsStore?.payment?.symbol_display?.storefront)
  let currentLocale = $derived($locale)

  function handleIncrement() {
    cartStore.incrementQuantity(item.id, item.variant_id)
  }

  function handleDecrement() {
    cartStore.decrementQuantity(item.id, item.variant_id)
  }

  function handleChange(newQty: number) {
    cartStore.updateQuantity(item.id, item.variant_id, newQty)
  }

  function handleRemove() {
    if (item.variant_id) {
      cartStore.removeVariant(item.id, item.variant_id)
    } else {
      cartStore.remove(item.id)
    }
  }
</script>

<div class="flex items-center gap-4 border-4 border-black bg-white p-4" class:highlighted={highlighted} class:needs-deletion={needsDeletion}>
  <!-- Product Image -->
  <div class="h-20 w-20 flex-shrink-0 border-4 border-black bg-gray-100">
    {#if item.image}
      <img
        src={getProductImageUrl(item.image, 'sm')}
        alt={item.name}
        class="h-full w-full object-cover"
      />
    {:else}
      <img
        src="/assets/img/noimage.png"
        alt=""
        class="h-full w-full object-cover"
      />
    {/if}
  </div>

  <!-- Product Info -->
  <div class="flex-1">
    <h3 class="mb-1 text-lg font-black uppercase text-black">
      {item.name}
    </h3>
    {#if item.variant_name}
      <p class="mb-2 text-sm font-bold text-gray-600">
        {item.variant_name}
      </p>
    {/if}
    <div class="flex items-baseline gap-2">
      <span class="text-xl font-black {item.amount === 0 ? 'text-green-500' : 'text-black'}" class:price-changed={highlighted}>
        {item.amount === 0
          ? t('product.free')
          : formatCurrencyWithTruncation(item.amount, currency, 'storefront', truncationSettings, currentLocale, numberFormat, symbolMode)}
      </span>
    </div>
  </div>

  <!-- Quantity Controls -->
  <div class="flex flex-shrink-0 items-center gap-3">
    <QuantityInput
      quantity={item.quantity}
      onIncrement={handleIncrement}
      onDecrement={handleDecrement}
      onChange={handleChange}
      disabled={needsDeletion}
    />

    <button
      onclick={handleRemove}
      class="border-4 border-black bg-red-500 px-4 py-2 text-sm font-black uppercase text-white transition-all hover:-translate-x-0.5 hover:-translate-y-0.5 hover:shadow-[4px_4px_0px_0px_rgba(0,0,0,1)]"
      aria-label={t('cart.remove')}
    >
      {t('cart.remove')}
    </button>
  </div>
</div>

<style>
  .highlighted {
    background-color: #fff9c4;
    transition: background-color 0.3s ease;
    border-color: #fdd835;
  }

  .price-changed {
    background-color: #ffeb3b;
    padding: 2px 4px;
    border-radius: 3px;
    font-weight: bold;
  }

  .needs-deletion {
    opacity: 0.6;
    position: relative;
  }

  .needs-deletion::after {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: repeating-linear-gradient(
      45deg,
      transparent,
      transparent 10px,
      rgba(255, 0, 0, 0.05) 10px,
      rgba(255, 0, 0, 0.05) 20px
    );
    pointer-events: none;
  }
</style>
