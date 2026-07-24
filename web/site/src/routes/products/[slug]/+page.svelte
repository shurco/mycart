<script lang="ts">
  import { page } from '$app/state'
  import { apiGet } from '$lib/utils/api'
  import type { Product } from '$lib/types/models'
  import { cartStore } from '$lib/stores/cart'
  import { costFormat } from '$lib/utils/costFormat'
  import { formatCurrencyWithTruncation } from '$lib/utils/currency'
  import { settingsStore } from '$lib/stores/settings'
  import { getProductImageUrl } from '$lib/utils/imageUrl'
  import { toggleCartItem } from '$lib/utils/cart'
  import { updateSEOTags } from '$lib/utils/seo'
  import { isBrowser } from '$lib/utils/browser'
  import NotFoundPage from '$lib/components/NotFoundPage.svelte'
  import VariantSelector from '$lib/components/VariantSelector.svelte'
  import QuantityInput from '$lib/components/QuantityInput.svelte'
  import CartItemCard from '$lib/components/CartItemCard.svelte'
  import { translate, locale } from '$lib/i18n'
  import { sanitizeHTML } from '$lib/utils/sanitize'
  import type { ProductVariant } from '$lib/types/models'

  // Reactive translation function
  let t = $derived($translate)
  let currentLocale = $derived($locale)

  let product = $state<Product | null>(null)
  let load = $state(false)
  let notFound = $state(false)
  let loading = $state(true)
  let currentSlide = $state(0)
  let selectedVariant = $state<ProductVariant | null>(null)
  let selectedQuantity = $state(1)

  let currency = $derived($settingsStore?.main.currency || '')
  let truncationSettings = $derived($settingsStore?.payment?.truncation)
  let numberFormat = $derived($settingsStore?.payment?.number_format)
  let symbolMode = $derived($settingsStore?.payment?.symbol_display?.storefront)
  let cart = $derived($cartStore)

  let cartItem = $derived(
    !product ? null :
    product.has_variants ? (
      !selectedVariant ? null :
      cart.find((item) => item.id === product.id && item.variant_id === selectedVariant.id)
    ) :
    cart.find((item) => item.id === product.id && !item.variant_id)
  )

  let inCart = $derived(!!cartItem)
  let currentQuantity = $derived(cartItem?.quantity || 1)

  let productCartItems = $derived(
    !product ? [] :
    cart.filter((item) => item.id === product.id)
  )

  let displayPrice = $derived(() => {
    if (!product) return 0
    if (product.has_variants && selectedVariant) {
      return product.amount + selectedVariant.price_surcharge
    }
    return product.amount
  })

  let canAddToCart = $derived(() => {
    if (!product) return false
    if (product.has_variants) {
      return selectedVariant !== null && selectedVariant.quantity > 0
    }
    return true
  })

  $effect(() => {
    const slug = page.params.slug
    if (slug) {
      // Reset state when slug changes
      product = null
      load = false
      notFound = false
      currentSlide = 0
      selectedVariant = null
      loadProduct(slug)
    }
  })

  async function loadProduct(slug: string) {
    const res = await apiGet<Product>(`/api/products/${slug}`)
    loading = false

    if (res.success && res.result) {
      product = res.result
      load = true

      if (isBrowser() && product.seo) {
        updateSEOTags(product.seo)
      }
    } else {
      // Product not found
      notFound = true
    }
  }

  function handleVariantChange(variant: ProductVariant | null) {
    selectedVariant = variant
  }

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
      const variantId = selectedVariant?.id
      cartStore.incrementQuantity(product!.id, variantId)
    } else {
      selectedQuantity++
    }
  }

  function handleQuantityDecrement() {
    if (inCart) {
      const variantId = selectedVariant?.id
      cartStore.decrementQuantity(product!.id, variantId)
    } else {
      selectedQuantity = Math.max(1, selectedQuantity - 1)
    }
  }

  function handleQuantityChange(newQty: number) {
    if (inCart) {
      const variantId = selectedVariant?.id
      cartStore.updateQuantity(product!.id, variantId, newQty)
    } else {
      selectedQuantity = newQty
    }
  }

  function handleToggleCart() {
    if (!product) return
    if (product.has_variants && !selectedVariant) return
    toggleCartItem(product, cart, selectedVariant, selectedQuantity)
  }

  function nextSlide(length: number) {
    currentSlide = (currentSlide + 1) % length
  }

  function prevSlide(length: number) {
    currentSlide = (currentSlide + length - 1) % length
  }
</script>

{#if loading}
  <div class="flex min-h-screen items-center justify-center bg-white">
    <div class="inline-block border-4 border-black bg-yellow-300 px-8 py-6">
      <p class="text-xl font-black tracking-wider text-black uppercase">{t('common.loading')}</p>
    </div>
  </div>
{:else if notFound}
  <NotFoundPage />
{:else if load && product}
  <section class="min-h-screen bg-white px-4 py-12 sm:px-6 lg:px-8">
    <div class="mx-auto max-w-screen-xl">
      <h1 class="mb-8 text-4xl font-black tracking-tighter text-black uppercase sm:text-5xl">
        {product.name}
      </h1>
      <div class="grid grid-cols-1 gap-8 lg:grid-cols-2 lg:gap-12">
        <!-- Product Images -->
        <div>
          {#if !product.images || product.images.length === 0}
            <div class="relative h-[400px] border-4 border-black bg-white sm:h-[500px]">
              <img src="/assets/img/noimage.png" alt="" class="absolute inset-0 h-full w-full object-cover" />
            </div>
          {:else if product.images.length === 1}
            <div class="relative h-[400px] overflow-hidden border-4 border-black bg-white sm:h-[500px]">
              <img
                src={getProductImageUrl(product.images[0], 'md')}
                alt={product.name}
                class="absolute inset-0 h-full w-full object-cover"
              />
            </div>
          {:else}
            <div class="relative h-[400px] overflow-hidden border-4 border-black bg-white sm:h-[500px]">
              <div
                class="flex h-full w-full transition-transform duration-500 ease-in-out"
                style="transform: translateX(-{currentSlide * 100}%)"
              >
                {#each product.images as image (image.id || image.name)}
                  <div class="h-full w-full flex-shrink-0">
                    <img
                      src={getProductImageUrl(image, 'md')}
                      alt={product.name}
                      class="block h-full w-full object-cover"
                    />
                  </div>
                {/each}
              </div>
              <button
                onclick={() => prevSlide(product.images.length)}
                class="absolute top-1/2 left-4 cursor-pointer border-4 border-black bg-yellow-300 p-3 text-xl font-black text-black transition-all duration-200 hover:-translate-x-1 hover:-translate-y-1 hover:shadow-[6px_6px_0px_0px_rgba(0,0,0,1)]"
                aria-label={t('product.previousImage')}
              >
                ←
              </button>
              <button
                onclick={() => nextSlide(product.images.length)}
                class="absolute top-1/2 right-4 cursor-pointer border-4 border-black bg-yellow-300 p-3 text-xl font-black text-black transition-all duration-200 hover:-translate-x-1 hover:-translate-y-1 hover:shadow-[6px_6px_0px_0px_rgba(0,0,0,1)]"
                aria-label={t('product.nextImage')}
              >
                →
              </button>
            </div>
          {/if}
        </div>

        <!-- Product Info -->
        <div class="space-y-6">
          <div class="brutal-card p-8">
            {#if product.attributes && product.attributes.length > 0}
              <div class="mb-6 flex flex-wrap gap-2">
                {#each product.attributes as attr (attr)}
                  <span class="bg-blue-300 px-4 py-2 text-sm font-black tracking-wider text-black uppercase">
                    {attr}
                  </span>
                {/each}
              </div>
            {/if}

            {#if product.brief}
              <div class="mb-6">
                <p class="text-lg leading-relaxed text-black">
                  {product.brief}
                </p>
              </div>
            {/if}

            {#if !product.has_variants}
              <div class="mb-6 flex items-baseline gap-3">
                <span class="text-5xl font-black tracking-tight text-black">
                  {product.amount === 0
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
            {/if}

            {#if product.has_variants && product.options && product.variants}
              <div class="mb-6">
                <VariantSelector
                  options={product.options}
                  variants={product.variants}
                  basePrice={product.amount}
                  {currency}
                  onVariantChange={handleVariantChange}
                />
              </div>
            {/if}

            <div class="mb-6">
              <label class="mb-2 block text-sm font-black uppercase tracking-wider text-black">
                {t('product.quantity')}
              </label>
              <div class="flex justify-center">
                {#if inCart}
                  <QuantityInput
                    quantity={currentQuantity}
                    onIncrement={handleQuantityIncrement}
                    onDecrement={handleQuantityDecrement}
                    onChange={handleQuantityChange}
                  />
                {:else}
                  <QuantityInput
                    quantity={selectedQuantity}
                    onIncrement={handleQuantityIncrement}
                    onDecrement={handleQuantityDecrement}
                    onChange={handleQuantityChange}
                  />
                {/if}
              </div>
            </div>

            <button
              onclick={handleToggleCart}
              disabled={!canAddToCart()}
              class="w-full cursor-pointer border-4 border-black px-8 py-4 text-lg font-black tracking-wider uppercase transition-all duration-200 hover:-translate-x-1 hover:-translate-y-1 hover:shadow-[12px_12px_0px_0px_rgba(0,0,0,1)] disabled:cursor-not-allowed disabled:opacity-50 {inCart
                ? 'bg-red-500 text-white'
                : 'bg-green-500 text-white'}"
            >
              {#if !inCart}
                <span class="flex items-center justify-center gap-3">
                  <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <use href="/assets/img/sprite.svg#plus" />
                  </svg>
                  <span>{t('product.addToCart')}</span>
                </span>
              {:else}
                <span class="flex items-center justify-center gap-3">
                  <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <use href="/assets/img/sprite.svg#minus" />
                  </svg>
                  <span>{t('product.removeFromCart')}</span>
                </span>
              {/if}
            </button>
          </div>
        </div>
      </div>

      {#if productCartItems.length > 0}
        <div class="mt-12">
          <h2 class="mb-6 text-3xl font-black tracking-tighter uppercase text-black">
            {t('cart.inYourCart')}
          </h2>
          <div class="space-y-4">
            {#each productCartItems as item (item.variant_id || 'base')}
              <CartItemCard {item} />
            {/each}
          </div>
        </div>
      {/if}

      {#if product.description}
        <div class="mt-12">
          <h2 class="mb-6 text-3xl font-black tracking-tighter text-black uppercase">{t('product.description')}</h2>
          <div class="prod_desc text-lg leading-relaxed text-black">
            {@html sanitizeHTML(product.description)}
          </div>
        </div>
      {/if}
    </div>
  </section>
{/if}
