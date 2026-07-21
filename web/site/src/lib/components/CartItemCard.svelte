<script lang="ts">
  import type { CartItem } from '$lib/types/models'
  import { cartStore } from '$lib/stores/cart'
  import { costFormat } from '$lib/utils/costFormat'
  import { settingsStore } from '$lib/stores/settings'
  import { getProductImageUrl } from '$lib/utils/imageUrl'
  import { translate } from '$lib/i18n'
  import QuantityInput from './QuantityInput.svelte'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    item: CartItem
  }

  let { item }: Props = $props()

  let currency = $derived($settingsStore?.main.currency || '')

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

<div class="flex items-center gap-4 border-4 border-black bg-white p-4">
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
      <span class="text-xl font-black text-black">
        {costFormat(item.amount) === 'free' ? t('product.free') : costFormat(item.amount)}
      </span>
      {#if item.amount !== 0 && item.amount}
        <span class="text-sm font-bold uppercase text-gray-600">{currency}</span>
      {/if}
    </div>
  </div>

  <!-- Quantity Controls -->
  <div class="flex flex-shrink-0 items-center gap-3">
    <QuantityInput
      quantity={item.quantity}
      onIncrement={handleIncrement}
      onDecrement={handleDecrement}
      onChange={handleChange}
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
