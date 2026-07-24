<script lang="ts">
  import { onMount } from 'svelte'
  import { page } from '$app/state'
  import { cartStore } from '$lib/stores/cart'
  import { settingsStore } from '$lib/stores/settings'
  import { apiGet } from '$lib/utils/api'
  import { formatCurrency } from '$lib/utils/currency'
  import { costFormat } from '$lib/utils/costFormat'
  import { getProductImageUrl } from '$lib/utils/imageUrl'
  import { goto } from '$app/navigation'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface CartItem {
    id: string
    name: string
    slug: string
    amount: number
    quantity: number
    image?: { name: string; ext: string } | null
  }

  interface CartData {
    id: string
    email: string
    amount_total: number
    currency: string
    payment_status: string
    payment_system: string
    items: CartItem[]
  }

  let cart = $state<CartData | null>(null)
  let loading = $state(true)
  let error = $state<string | undefined>(undefined)
  let currency = $derived($settingsStore?.main.currency || '')

  // Get cart_id from query parameters reactively
  let cartId = $derived(page.url.searchParams.get('cart_id'))

  onMount(async () => {
    // Clear local cart
    cartStore.clear()

    if (!cartId) {
      error = t('payment.success.cartIdMissing')
      loading = false
      return
    }

    // Load cart information from API
    loading = true
    const res = await apiGet<CartData>(`/api/cart/${cartId}`)
    if (res.success && res.result) {
      cart = res.result
    } else {
      error = res.message || t('payment.success.loadFailed')
    }
    loading = false
  })

  function totalAmount(): string {
    if (!cart) return formatCurrency(0, currency)
    return formatCurrency(cart.amount_total / 100, cart.currency)
  }
</script>

<div class="min-h-screen bg-white px-4 py-12 sm:px-6 lg:px-8">
  <div class="mx-auto max-w-screen-xl">
    <div class="mx-auto max-w-3xl">
      {#if loading}
        <div class="brutal-card p-12 text-center">
          <div class="inline-block border-4 border-black bg-yellow-300 px-8 py-6">
            <p class="text-2xl font-black tracking-wider text-black uppercase">{t('common.loading')}</p>
          </div>
        </div>
      {:else if error}
        <div class="brutal-card bg-red-300 p-12">
          <h1 class="mb-4 text-4xl font-black tracking-tighter text-black uppercase">{t('error.errorTitle')}</h1>
          <p class="text-lg text-black">{error}</p>
        </div>
      {:else if cart}
        <div class="brutal-card mb-8 bg-green-300 p-8 sm:p-12">
          <header class="text-center">
            <h1 class="mb-4 text-4xl font-black tracking-tighter text-black uppercase sm:text-5xl">
              {t('payment.success.title')}
            </h1>
            <p class="text-lg tracking-wide text-black">
              {t('payment.success.message')}
            </p>
          </header>
        </div>

        <div class="brutal-card p-8 sm:p-12">
          <h2 class="mb-6 border-b-4 border-black pb-4 text-3xl font-black tracking-tighter text-black uppercase">
            {t('payment.success.orderDetails')}
          </h2>
          <ul class="mb-8 space-y-4">
            {#each cart.items as item (`${item.id}-${item.variant_id || 'no-variant'}-${item.quantity}`)}
              <li class="border-4 border-black bg-white p-4">
                <div class="flex items-center gap-4">
                  <div class="overflow-hidden border-4 border-black">
                    <img src={getProductImageUrl(item.image, 'sm')} alt={item.name} class="h-20 w-20 object-cover" />
                  </div>
                  <div class="flex-1">
                    <a
                      href="/products/{item.slug}"
                      class="cursor-pointer text-xl font-black tracking-tight text-black uppercase decoration-yellow-300 decoration-4 underline-offset-4 hover:underline"
                    >
                      {item.name}
                    </a>
                    {#if item.variant_options && Object.keys(item.variant_options).length > 0}
                      <p class="mt-1 text-base text-gray-700">
                        {#each Object.entries(item.variant_options) as [key, value], i}
                          {#if i > 0}<span class="mx-1">•</span>{/if}<span class="font-medium">{key}:</span> {value}
                        {/each}
                      </p>
                    {/if}
                    {#if item.variant_sku}
                      <p class="mt-1 text-sm text-gray-600">SKU: {item.variant_sku}</p>
                    {/if}
                    <p class="mt-1 text-lg text-gray-700">{t('payment.success.quantity')} {item.quantity}</p>
                  </div>
                  <div class="text-right">
                    <p class="text-2xl font-black text-black">
                      {item.amount === 0 ? t('product.free') : formatCurrency((item.amount * item.quantity) / 100, cart.currency)}
                    </p>
                    <p class="text-lg text-gray-600">
                      {item.amount === 0 ? t('product.free') : formatCurrency(item.amount / 100, cart.currency) + ' ' + t('payment.success.each')}
                    </p>
                  </div>
                </div>
              </li>
            {/each}
          </ul>

          <div class="border-t-4 border-black pt-6">
            <div class="flex items-center justify-between">
              <span class="text-3xl font-black tracking-tighter text-black uppercase"> {t('cart.total')} </span>
              <span class="text-4xl font-black text-black">
                {totalAmount()}
              </span>
            </div>
          </div>

          <div class="mt-8 text-center">
            <a
              href="/"
              class="inline-block border-4 border-black bg-yellow-300 px-6 py-4 transition-colors hover:bg-yellow-400"
            >
              <p class="text-lg font-black tracking-wider text-black uppercase">
                {t('common.backToHome') || 'Back to Home'}
              </p>
            </a>
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>
