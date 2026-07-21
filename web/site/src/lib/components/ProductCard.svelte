<script lang="ts">
  import type { Product } from '$lib/types/models'
  import { cartStore } from '$lib/stores/cart'
  import { costFormat } from '$lib/utils/costFormat'
  import { settingsStore } from '$lib/stores/settings'
  import { getFirstImageUrl } from '$lib/utils/imageUrl'
  import { toggleCartItem } from '$lib/utils/cart'
  import { handleNavigation } from '$lib/utils/navigation'
  import { translate } from '$lib/i18n'
  import QuantityInput from './QuantityInput.svelte'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    product: Product
    index?: number
  }

  let { product, index = 0 }: Props = $props()

  let currency = $derived($settingsStore?.main.currency || '')
  let cart = $derived($cartStore)

  let hasVariants = $derived(product.has_variants)
  let variantCount = $derived(product.variants?.length || 0)
  let cartItem = $derived(cart.find(item => item.id === product.id && !item.variant_id))
  let inCart = $derived(!!cartItem)
  let currentQuantity = $derived(cartItem?.quantity || 1)
  let selectedQuantity = $state(1)

  // Sync selectedQuantity with cart when item added/removed
  $effect(() => {
    if (cartItem) {
      selectedQuantity = cartItem.quantity
    } else {
      selectedQuantity = 1
    }
  })

  function handleQuantityIncrement() {
    if (inCart) {
      cartStore.incrementQuantity(product.id, undefined)
    } else {
      selectedQuantity++
    }
  }

  function handleQuantityDecrement() {
    if (inCart) {
      cartStore.decrementQuantity(product.id, undefined)
    } else {
      selectedQuantity = Math.max(1, selectedQuantity - 1)
    }
  }

  function handleQuantityChange(newQty: number) {
    if (inCart) {
      cartStore.updateQuantity(product.id, undefined, newQty)
    } else {
      selectedQuantity = newQty
    }
  }

  function handleToggleCart(e: MouseEvent) {
    e.stopPropagation()
    toggleCartItem(product, cart, null, selectedQuantity)
  }
</script>

<li class="flex h-full flex-col">
  <a
    href="/products/{product.slug}"
    onclick={(e) => handleNavigation(e, `/products/${product.slug}`)}
    class="block flex-shrink-0 cursor-pointer"
  >
    <div class="relative overflow-hidden border-4 border-black bg-white">
      <img
        src={getFirstImageUrl(product.images, 'md')}
        alt={product.name}
        class="h-64 w-full object-cover transition-transform duration-500 hover:scale-110"
        loading="lazy"
      />
      <div class="absolute top-4 right-4">
        <div class="border-4 border-black bg-yellow-300 px-3 py-1 text-xs font-black tracking-wider uppercase">{t('product.new')}</div>
      </div>
    </div>
  </a>

  <div class="flex flex-1 flex-col border-x-4 border-b-4 border-black bg-white p-6">
    <div class="mb-4 flex flex-1 items-start justify-between gap-4">
      <a
        href="/products/{product.slug}"
        onclick={(e) => handleNavigation(e, `/products/${product.slug}`)}
        class="flex-1 cursor-pointer"
      >
        <h3
          class="mb-2 text-xl font-black tracking-tight text-black uppercase decoration-yellow-300 decoration-4 underline-offset-4 hover:underline"
        >
          {product.name}
        </h3>
        {#if hasVariants}
          <p class="text-sm font-bold text-gray-600">
            {variantCount} {variantCount === 1 ? t('product.variant') : t('product.variants')}
          </p>
        {/if}
      </a>
    </div>

    <div class="mt-auto flex flex-col gap-4">
      <div class="flex items-baseline gap-2">
        <span class="text-3xl font-black tracking-tight text-black">
          {costFormat(product.amount) === 'free' ? t('product.free') : costFormat(product.amount)}
        </span>
        {#if product.amount !== 0 && product.amount}
          <span class="text-lg font-bold text-gray-600 uppercase">{currency}</span>
        {/if}
      </div>

      <div class="flex flex-col items-end gap-3">
        {#if hasVariants}
          <a
            href="/products/{product.slug}"
            onclick={(e) => handleNavigation(e, `/products/${product.slug}`)}
            class="relative z-10 cursor-pointer border-4 border-black bg-blue-500 px-6 py-3 text-sm font-black tracking-wider uppercase text-white transition-all duration-200 hover:-translate-x-1 hover:-translate-y-1 hover:shadow-[8px_8px_0px_0px_rgba(0,0,0,1)] whitespace-nowrap"
          >
            {t('product.details')}
          </a>
        {:else}
          {#if !inCart}
            <QuantityInput
              quantity={selectedQuantity}
              onIncrement={handleQuantityIncrement}
              onDecrement={handleQuantityDecrement}
              onChange={handleQuantityChange}
            />
          {/if}

          <button
            onclick={handleToggleCart}
            class="relative z-10 cursor-pointer border-4 border-black px-6 py-3 text-sm font-black tracking-wider uppercase transition-all duration-200 hover:-translate-x-1 hover:-translate-y-1 hover:shadow-[8px_8px_0px_0px_rgba(0,0,0,1)] whitespace-nowrap {inCart
              ? 'bg-red-500 text-white'
              : 'bg-green-500 text-white'}"
          >
            {#if !inCart}
              <span class="flex items-center gap-2">
                <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <use href="/assets/img/sprite.svg#plus" />
                </svg>
                <span>{t('product.add')}</span>
              </span>
            {:else}
              <span class="flex items-center gap-2">
                <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <use href="/assets/img/sprite.svg#minus" />
                </svg>
                <span>{t('product.remove')}</span>
              </span>
            {/if}
          </button>
        {/if}
      </div>
    </div>
  </div>
</li>
