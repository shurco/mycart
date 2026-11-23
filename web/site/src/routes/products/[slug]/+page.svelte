<script lang="ts">
  import { page } from '$app/state'
  import { apiGet } from '$lib/utils/api'
  import type { Product } from '$lib/types/models'
  import { cartStore } from '$lib/stores/cart'
  import { formatCurrencyWithTruncation } from '$lib/utils/currency'
  import { settingsStore } from '$lib/stores/settings'
  import { toggleCartItem } from '$lib/utils/cart'
  import { chargesStock } from '$lib/utils/digital'
  import { updateSEOTags } from '$lib/utils/seo'
  import { isBrowser } from '$lib/utils/browser'
  import NotFoundPage from '$lib/components/NotFoundPage.svelte'
  import VariantSelector from '$lib/components/VariantSelector.svelte'
  import QuantityInput from '$lib/components/QuantityInput.svelte'
  import ProductGallery from '$lib/components/ProductGallery.svelte'
  import ProductDownloadNotice from '$lib/components/ProductDownloadNotice.svelte'
  import ProductCartButton from '$lib/components/ProductCartButton.svelte'
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
  let selectedVariant = $state<ProductVariant | null>(null)
  let selectedQuantity = $state(1)
  // Guards against a stale response from a previous slug overwriting the
  // currently displayed product.
  let latestSlugRequest = ''

  let currency = $derived($settingsStore?.main.currency || '')
  let truncationSettings = $derived($settingsStore?.payment?.truncation)
  let numberFormat = $derived($settingsStore?.payment?.number_format)
  let symbolMode = $derived($settingsStore?.payment?.symbol_display?.storefront)
  let cart = $derived($cartStore)
  let accountEnabled = $derived($settingsStore?.account?.enabled === true)

  let cartItem = $derived.by(() => {
    const p = product
    if (!p) return null
    if (p.has_variants) {
      const v = selectedVariant
      return v ? cart.find((item) => item.id === p.id && item.variant_id === v.id) : null
    }
    return cart.find((item) => item.id === p.id && !item.variant_id)
  })

  let inCart = $derived(!!cartItem)
  let currentQuantity = $derived(cartItem?.quantity || 1)

  let productCartItems = $derived.by(() => {
    const p = product
    return p ? cart.filter((item) => item.id === p.id) : []
  })

  let canAddToCart = $derived.by(() => {
    if (!product) return false
    if (product.has_variants) {
      if (!selectedVariant) return false
      // The count on a download is not stock, so a zero there is not a sold-out
      // variant — the panel's form offers that zero by default. Refusing it
      // would put the shop's whole catalogue behind a choice nobody can make.
      return !chargesStock(product) || selectedVariant.quantity > 0
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
      selectedVariant = null
      loading = true
      latestSlugRequest = slug
      loadProduct(slug)
    }
  })

  async function loadProduct(slug: string) {
    const res = await apiGet<Product>(`/api/products/${slug}`)

    // A newer navigation happened while this request was in flight.
    if (latestSlugRequest !== slug) return

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
        <ProductGallery images={product.images} name={product.name} />

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

            {#if chargesStock(product)}
              <div class="mb-6">
                <label class="mb-2 block text-sm font-black uppercase tracking-wider text-black">
                  {t('product.quantity')}
                </label>
                <div class="flex justify-center">
                  <QuantityInput
                    quantity={inCart ? currentQuantity : selectedQuantity}
                    onIncrement={handleQuantityIncrement}
                    onDecrement={handleQuantityDecrement}
                    onChange={handleQuantityChange}
                  />
                </div>
              </div>
            {:else}
              <ProductDownloadNotice {accountEnabled} />
            {/if}

            <ProductCartButton {inCart} disabled={!canAddToCart} onToggle={handleToggleCart} />
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
