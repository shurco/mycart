<script lang="ts">
  import { onMount } from 'svelte'
  import { cartStore } from '$lib/stores/cart'
  import { settingsStore } from '$lib/stores/settings'
  import { apiGet, apiPost } from '$lib/utils/api'
  import { costFormat } from '$lib/utils/costFormat'
  import { getProductImageUrl } from '$lib/utils/imageUrl'
  import { hasPaymentProviders } from '$lib/utils/payment'
  import { getLocalStorage, setLocalStorage, removeLocalStorage } from '$lib/utils/browser'
  import type { PaymentMethods } from '$lib/types/models'
  import { goto } from '$app/navigation'
  import Overlay from '$lib/components/Overlay.svelte'
  import { handleNavigation } from '$lib/utils/navigation'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  let email = $state('')
  let provider = $state('')
  let payments = $state<PaymentMethods>({})
  let showOverlay = $state(false)
  let error = $state<string | undefined>(undefined)
  let isLoadingPaymentMethods = $state(false)

  let cart = $derived($cartStore)
  let currency = $derived($settingsStore?.main.currency || '')

  // Calculate total cart amount in cents
  let cartTotal = $derived(cart.reduce((sum, item) => sum + item.amount, 0))
  let isFree = $derived(cartTotal === 0)

  // Handle payment provider based on cart state
  $effect(() => {
    if (isFree) {
      // For free carts, don't auto-set provider to prevent accidental checkout
      // Provider will be set only when user explicitly clicks checkout button
      // Clear any existing provider selection when cart becomes free
      if (provider && provider !== 'dummy') {
        provider = ''
        removeLocalStorage('provider')
      }
    } else if (!isFree) {
      // If cart is no longer free, reset provider and load payment methods
      if (provider === 'dummy') {
        provider = ''
        removeLocalStorage('provider')
      }
      // Load payment methods if not already loaded and not currently loading
      if (!hasPaymentProviders(payments) && !isLoadingPaymentMethods) {
        loadPaymentMethods().catch((err) => {
          console.error('Failed to load payment methods:', err)
          error = 'Failed to load payment methods. Please refresh the page.'
          showOverlay = true
        })
      }
    }
  })

  async function loadPaymentMethods() {
    // Prevent multiple simultaneous calls
    if (isLoadingPaymentMethods) {
      return
    }

    isLoadingPaymentMethods = true
    try {
      const res = await apiGet<PaymentMethods>('/api/cart/payment')
      if (res.success && res.result) {
        payments = res.result
        provider = ''
        removeLocalStorage('provider')
      } else {
        throw new Error(res.message || 'Failed to load payment methods')
      }
    } finally {
      isLoadingPaymentMethods = false
    }
  }

  onMount(async () => {
    email = getLocalStorage('email')

    // If cart is not free, load payment methods
    // $effect will also handle this, but we load here on initial mount to avoid delay
    if (!isFree && !hasPaymentProviders(payments)) {
      await loadPaymentMethods().catch((err) => {
        console.error('Failed to load payment methods on mount:', err)
        error = 'Failed to load payment methods. Please refresh the page.'
        showOverlay = true
      })
    }
    // Don't auto-set provider for free carts on mount to prevent accidental checkout
  })

  let showPayments = $derived(!isFree && hasPaymentProviders(payments))

  // Computed value instead of function
  let totalCartAmount = $derived(costFormat(cartTotal) === 'free' ? t('product.free') : costFormat(cartTotal))

  async function checkOut(e: Event) {
    e.preventDefault()

    setLocalStorage('email', email)
    error = undefined

    const finalProvider = isFree ? 'dummy' : provider

    if (!isFree && !finalProvider) {
      error = t('cart.selectPaymentError')
      showOverlay = true
      return
    }

    setLocalStorage('provider', finalProvider)

    const cartData = {
      email,
      provider: finalProvider,
      products: cart.map((item) => ({ id: item.id, quantity: 1 }))
    }

    const res = await apiPost<{ url?: string }>('/cart/payment', cartData)
    if (res.success && res.result?.url) {
      window.location.href = res.result.url
    } else {
      error = res.message || t('payment.failed')
      showOverlay = true
    }
  }

  function closeOverlay() {
    showOverlay = false
    error = undefined
  }
</script>

<section class="min-h-screen bg-white px-4 py-12 sm:px-6 lg:px-8">
  <div class="mx-auto max-w-screen-xl">
    <div class="mx-auto max-w-4xl">
      <!-- Header -->
      <header class="mb-12 text-center">
        <h1 class="mb-4 text-4xl font-black tracking-tighter text-black uppercase sm:text-5xl">
          {cart.length > 0 ? t('cart.yourCart') : t('cart.cartIsEmpty')}
        </h1>
        <div class="mx-auto h-1 w-32 bg-black"></div>
      </header>

      {#if cart.length === 0}
        <div class="brutal-card mb-8 p-8 text-center">
          <p class="mb-8 text-lg tracking-wide text-black">
            {t('cart.emptyMessage')}
          </p>

          <div class="flex justify-center">
            <a
              href="/"
              onclick={(e) => handleNavigation(e, '/')}
              class="inline-block cursor-pointer border-4 border-black bg-yellow-300 px-8 py-4 text-lg font-black tracking-wider text-black uppercase transition-all duration-200 hover:-translate-x-1 hover:-translate-y-1 hover:shadow-[12px_12px_0px_0px_rgba(0,0,0,1)]"
            >
              {t('cart.goToHome')}
            </a>
          </div>
        </div>
      {/if}

      <form onsubmit={checkOut}>
        {#if cart.length > 0}
          <!-- Cart Items -->
          <div class="mb-8">
            <h2 class="mb-6 text-3xl font-black tracking-tighter text-black uppercase">
              {t('cart.itemsCount', { count: cart.length })}
            </h2>
            <ul class="list-none space-y-4">
              {#each cart as item (item.id)}
                <li class="border-4 border-black bg-white p-4">
                  <div class="flex items-center gap-4">
                    <div class="overflow-hidden border-4 border-black">
                      <img src={getProductImageUrl(item.image, 'sm')} alt={item.name} class="h-20 w-20 object-cover" />
                    </div>
                    <div class="flex-1">
                      <a
                        href="/products/{item.slug}"
                        target="_blank"
                        class="cursor-pointer text-xl font-black tracking-tight text-black uppercase decoration-yellow-300 decoration-4 underline-offset-4 hover:underline"
                      >
                        {item.name}
                      </a>
                    </div>
                    <div class="flex items-center gap-4">
                      <span
                        class="text-2xl font-black {costFormat(item.amount) === 'free'
                          ? 'text-green-500'
                          : 'text-black'}"
                      >
                        {costFormat(item.amount) === 'free' ? t('product.free') : costFormat(item.amount)}
                        {#if item.amount !== 0 && item.amount}
                          {currency}
                        {/if}
                      </span>
                      <button
                        type="button"
                        class="cursor-pointer border-4 border-black bg-red-500 p-2 text-sm font-black text-white uppercase transition-all duration-200 hover:-translate-x-1 hover:-translate-y-1 hover:shadow-[6px_6px_0px_0px_rgba(0,0,0,1)]"
                        onclick={() => cartStore.remove(item.id)}
                        aria-label={t('cart.removeItem')}
                      >
                        <svg class="h-5 w-5">
                          <use href="/assets/img/sprite.svg#trash" />
                        </svg>
                      </button>
                    </div>
                  </div>
                </li>
              {/each}
            </ul>
          </div>

          <!-- Total -->
          <div class="brutal-card mb-8 bg-yellow-300 p-8">
            <div class="flex items-center justify-between">
              <span class="text-3xl font-black tracking-tighter text-black uppercase"> {t('cart.total')} </span>
              <span class="text-4xl font-black {cartTotal === 0 ? 'text-green-500' : 'text-black'}">
                {totalCartAmount}
                {#if cart.length > 0 && cartTotal !== 0}
                  {currency}
                {/if}
              </span>
            </div>
          </div>

          {#if isFree || showPayments}
            <!-- Email Input -->
            <div class="mt-16 mb-8">
              <h2 class="mb-6 text-3xl font-black tracking-tighter text-black uppercase">{t('cart.enterEmail')}</h2>
              <p class="mb-4 text-lg tracking-wide text-black">
                {#if isFree}
                  {t('cart.emailFreeDescription')}
                {:else}
                  {t('cart.emailPaidDescription')}
                {/if}
              </p>
              <label for="email" class="block">
                <input
                  type="email"
                  bind:value={email}
                  id="email"
                  required
                  class="w-full border-4 border-black bg-white px-6 py-4 text-lg font-black tracking-wider text-black uppercase focus:ring-4 focus:ring-yellow-300 focus:outline-none"
                  placeholder={t('cart.emailPlaceholder')}
                />
              </label>
            </div>

            <!-- Payment Provider Selection -->
            {#if showPayments}
              <div class="mt-16 mb-8">
                <h2 class="mb-6 text-3xl font-black tracking-tighter text-black uppercase">{t('cart.selectPaymentSystem')}</h2>
                <fieldset class="space-y-4">
                  {#if payments.stripe}
                    <div>
                      <input type="radio" bind:group={provider} value="stripe" id="stripe" class="peer hidden" />
                      <label
                        for="stripe"
                        class="block cursor-pointer border-4 border-black bg-white p-6 peer-checked:border-yellow-300 peer-checked:bg-yellow-300"
                      >
                        <p class="mb-2 text-xl font-black tracking-tight text-black uppercase">{t('cart.stripe')}</p>
                        <p class="text-lg text-black">{t('cart.stripeDescription')}</p>
                      </label>
                    </div>
                  {/if}

                  {#if payments.paypal}
                    <div>
                      <input type="radio" bind:group={provider} value="paypal" id="paypal" class="peer hidden" />
                      <label
                        for="paypal"
                        class="block cursor-pointer border-4 border-black bg-white p-6 peer-checked:border-yellow-300 peer-checked:bg-yellow-300"
                      >
                        <p class="mb-2 text-xl font-black tracking-tight text-black uppercase">{t('cart.paypal')}</p>
                        <p class="text-lg text-black">{t('cart.paypalDescription')}</p>
                      </label>
                    </div>
                  {/if}

                  {#if payments.spectrocoin}
                    <div>
                      <input
                        type="radio"
                        bind:group={provider}
                        value="spectrocoin"
                        id="spectrocoin"
                        class="peer hidden"
                      />
                      <label
                        for="spectrocoin"
                        class="block cursor-pointer border-4 border-black bg-white p-6 peer-checked:border-yellow-300 peer-checked:bg-yellow-300"
                      >
                        <p class="mb-2 text-xl font-black tracking-tight text-black uppercase">{t('cart.spectrocoin')}</p>
                        <p class="text-lg text-black">{t('cart.spectrocoinDescription')}</p>
                      </label>
                    </div>
                  {/if}

                  {#if payments.coinbase}
                    <div>
                      <input
                        type="radio"
                        bind:group={provider}
                        value="coinbase"
                        id="coinbase"
                        class="peer hidden"
                      />
                      <label
                        for="coinbase"
                        class="block cursor-pointer border-4 border-black bg-white p-6 peer-checked:border-yellow-300 peer-checked:bg-yellow-300"
                      >
                        <p class="mb-2 text-xl font-black tracking-tight text-black uppercase">{t('cart.coinbase')}</p>
                        <p class="text-lg text-black">{t('cart.coinbaseDescription')}</p>
                      </label>
                    </div>
                  {/if}
                </fieldset>
              </div>
            {/if}

            <!-- Checkout Button -->
            <div class="flex justify-end">
              <button
                type="submit"
                disabled={!email || (!isFree && !provider)}
                class="cursor-pointer border-4 border-black bg-green-500 px-12 py-4 text-xl font-black tracking-wider text-white uppercase transition-all duration-200 enabled:hover:-translate-x-1 enabled:hover:-translate-y-1 enabled:hover:shadow-[14px_14px_0px_0px_rgba(0,0,0,1)] disabled:cursor-not-allowed disabled:opacity-50"
              >
                {#if isFree}
                  {t('cart.getForFree')}
                {:else}
                  {t('cart.checkout')}
                {/if}
              </button>
            </div>
          {:else}
            <div class="brutal-card bg-red-300 p-8">
              <p class="text-center text-xl font-black tracking-wider text-black uppercase">
                {t('cart.noPaymentSystems')}
              </p>
            </div>
          {/if}
        {/if}
      </form>
    </div>
  </div>
  <Overlay show={showOverlay} {error} onClose={closeOverlay} />
</section>
