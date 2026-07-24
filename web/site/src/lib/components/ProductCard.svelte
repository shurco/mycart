<script lang="ts">
  import type { Product } from '$lib/types/models'
  import { cartStore } from '$lib/stores/cart'
  import { costFormat } from '$lib/utils/costFormat'
  import { formatCurrencyWithTruncation } from '$lib/utils/currency'
  import { settingsStore } from '$lib/stores/settings'
  import { getFirstImageUrl } from '$lib/utils/imageUrl'
  import { toggleCartItem } from '$lib/utils/cart'
  import { handleNavigation } from '$lib/utils/navigation'
  import { translate, locale } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)
  let currentLocale = $derived($locale)

  interface Props {
    product: Product
    index?: number
  }

  let { product, index = 0 }: Props = $props()

  let currency = $derived($settingsStore?.main.currency || '')
  let truncationSettings = $derived($settingsStore?.payment?.truncation)
  let numberFormat = $derived($settingsStore?.payment?.number_format)
  let symbolMode = $derived($settingsStore?.payment?.symbol_display?.storefront)
  let cart = $derived($cartStore)
  let inCart = $derived(cart.some((item) => item.id === product.id))

  function handleToggleCart(e: MouseEvent) {
    e.stopPropagation()
    toggleCartItem(product, cart)
  }
</script>

<li class="flex h-full flex-col" data-testid="product-card">
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
      </a>
    </div>

    <div class="mt-auto flex items-center justify-between gap-4">
      <div class="flex items-baseline gap-2">
        <span class="text-3xl font-black tracking-tight text-black">
          {costFormat(product.amount) === 'free'
            ? t('product.free')
            : formatCurrencyWithTruncation(
                product.amount,
                currency || 'USD',
                'storefront',
                truncationSettings,
                currentLocale,
                numberFormat,
                symbolMode
              )}
        </span>
      </div>

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
            <span>{t('product.addToCartShort')}</span>
          </span>
        {:else}
          <span class="flex items-center gap-2">
            <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <use href="/assets/img/sprite.svg#minus" />
            </svg>
            <span>{t('product.removeFromCartShort')}</span>
          </span>
        {/if}
      </button>
    </div>
  </div>
</li>
