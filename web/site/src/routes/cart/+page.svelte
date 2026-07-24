<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { cartStore } from '$lib/stores/cart'
  import { settingsStore } from '$lib/stores/settings'
  import { apiGet, apiPost } from '$lib/utils/api'
  import { formatCurrency } from '$lib/utils/currency'
  import { costFormat } from '$lib/utils/costFormat'
  import { getProductImageUrl } from '$lib/utils/imageUrl'
  import { hasPaymentProviders } from '$lib/utils/payment'
  import { getLocalStorage, setLocalStorage, removeLocalStorage } from '$lib/utils/browser'
  import type { PaymentMethods } from '$lib/types/models'
  import { goto } from '$app/navigation'
  import Overlay from '$lib/components/Overlay.svelte'
  import CartItemCard from '$lib/components/CartItemCard.svelte'
  import { handleNavigation } from '$lib/utils/navigation'
  import { translate, locale } from '$lib/i18n'
  import * as PortOne from '@portone/browser-sdk/v2'

  // UUID generator with fallback for non-secure contexts (HTTP)
  function generateUUID(): string {
    if (typeof crypto !== "undefined" && crypto.randomUUID) {
      return crypto.randomUUID()
    }
    // Fallback for HTTP or older browsers
    return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, (c) => {
      const r = Math.random() * 16 | 0, v = c === "x" ? r : (r & 0x3 | 0x8)
      return v.toString(16)
    })
  }

  // Reactive translation function
  let t = $derived($translate)

  let email = $state('')
  let provider = $state('')
  let payments = $state<PaymentMethods>({})
  let showOverlay = $state(false)
  let error = $state<string | undefined>(undefined)
  let isLoadingPaymentMethods = $state(false)
  let hasAttemptedLoadingPayments = $state(false)
  let portoneStoreId = $state('')
  let portoneChannelKey = $state('')
  let portoneDebugEnabled = $state(false)
  let validationErrors = $state<any[]>([])
  let showValidationModal = $state(false)
  let highlightedItems = $state<Set<string>>(new Set())

  // Helper function for conditional logging
  function debugLog(...args: any[]) {
    if (portoneDebugEnabled) {
      console.log(...args)
    }
  }

  function getItemKey(item: any): string {
    return item.variant_id ? `${item.id}_${item.variant_id}` : item.id
  }

  function handleValidationErrors(errors: any[], correctedCart: any[]) {
    validationErrors = errors
    showValidationModal = true

    const updatedCart = $cartStore.map((item, index) => {
      const corrected = correctedCart[index]
      if (!corrected) return item

      const hasError = errors.some(e => e.item_index === index)

      if (!corrected.available) {
        return { ...item, needsDeletion: true, disabled: true }
      }

      if (hasError) {
        highlightedItems.add(getItemKey(item))
        return { ...item, amount: corrected.unit_price }
      }

      return item
    })

    try {
      cartStore.set(updatedCart)
    } catch (err) {
      console.error('Failed to update cart store:', err)
      error = 'Failed to update cart. Please clear your cart and try again.'
      showOverlay = true
    }
  }

  // Create cart record and return cart_id
  async function createCartRecord(email: string, cart: any[]): Promise<string> {
    debugLog('Creating cart record...')
    const cartCreateRes = await apiPost<{ cart_id: string; amount_total: number; currency: string }>('/api/cart/create', {
      email: email,
      provider: 'portone',
      products: cart.map((item) => ({
        id: item.id,
        variant_id: item.variant_id || undefined,
        quantity: item.quantity,
        unit_price: item.amount
      }))
    })
    debugLog('Cart create response:', cartCreateRes)

    // Handle validation errors (409 Conflict)
    if (cartCreateRes.status === 409) {
      if (!cartCreateRes.result?.validation_errors || !cartCreateRes.result?.corrected_cart) {
        throw new Error('Validation error occurred. Please refresh and try again.')
      }
      handleValidationErrors(
        cartCreateRes.result.validation_errors,
        cartCreateRes.result.corrected_cart
      )
      throw new Error('Cart validation failed')
    }

    if (!cartCreateRes.success || !cartCreateRes.result?.cart_id) {
      throw new Error('Failed to create cart: ' + (cartCreateRes.message || 'Unknown error'))
    }

    return cartCreateRes.result.cart_id
  }

  // Verify payment with backend
  async function verifyPayment(paymentId: string, cartId: string): Promise<boolean> {
    console.log('Verifying payment with backend...')
    const verifyRes = await apiPost('/api/payment/portone/complete', {
      payment_id: paymentId,
      cart_id: cartId
    })
    console.log('Backend verification response:', verifyRes)

    if (!verifyRes.success) {
      throw new Error('Payment verification failed: ' + (verifyRes.message || 'Unknown error'))
    }

    return true
  }

  // Handle PortOne payment flow
  async function handlePortonePayment(email: string, cart: any[], cartTotal: number, currency: string) {
    debugLog('=== PORTONE PAYMENT FLOW ===')
    debugLog('PortOne SDK available?', typeof PortOne !== 'undefined')
    debugLog('PortOne.requestPayment available?', typeof PortOne?.requestPayment === 'function')

    // Validate PortOne configuration
    if (!portoneStoreId || !portoneChannelKey) {
      throw new Error('PortOne configuration not loaded. Please refresh the page.')
    }

    debugLog('PortOne payment request:', {
      storeId: portoneStoreId,
      channelKey: portoneChannelKey,
      cartTotal: cartTotal,
      currency: currency,
      protocol: window.location.protocol,
      hostname: window.location.hostname
    })

    // Generate unique payment ID
    const paymentId = `payment-${generateUUID()}`
    debugLog('Generated payment ID:', paymentId)

    // Create cart and get cart_id
    const cartId = await createCartRecord(email, cart)
    debugLog('Cart created with ID:', cartId)

    // Prepare and execute payment request
    const paymentRequest = {
      storeId: portoneStoreId,
      channelKey: portoneChannelKey,
      paymentId: paymentId,
      orderName: `Order ${cart.length} items`,
      totalAmount: cartTotal,
      currency: "KRW",
      payMethod: "EASY_PAY",
      customData: { cart_id: cartId }
    }
    console.log('Payment request object:', paymentRequest)

    // Call PortOne SDK
    console.log('Calling PortOne.requestPayment...')
    const response = await PortOne.requestPayment(paymentRequest)
    console.log('PortOne payment response received:', response)

    // Check for payment errors
    if (response.code != null) {
      console.error('PortOne returned error code:', response.code, 'message:', response.message)
      throw new Error(response.message)
    }

    // Verify payment with backend
    await verifyPayment(response.paymentId, cartId)
    console.log('Payment verified successfully!')

    // Clear cart and redirect to success
    cartStore.set([])
    removeLocalStorage('email')
    removeLocalStorage('provider')
    goto('/cart/payment/success')
  }

  let cart = $derived($cartStore)
  let currency = $derived($settingsStore?.main.currency || '')
  let truncationSettings = $derived($settingsStore?.payment?.truncation)
  let numberFormat = $derived($settingsStore?.payment?.number_format)
  let symbolMode = $derived($settingsStore?.payment?.symbol_display?.storefront)
  let currentLocale = $derived($locale)

  // Calculate total cart amount in cents (amount * quantity for each item)
  let cartTotal = $derived(cart.reduce((sum, item) => sum + (item.amount * item.quantity), 0))
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
      // Reset payment loading flag when cart becomes free
      // This allows reloading if user adds paid items again later
      hasAttemptedLoadingPayments = false
    } else if (!isFree) {
      // If cart is no longer free, reset provider and load payment methods
      if (provider === 'dummy') {
        provider = ''
        removeLocalStorage('provider')
      }
      // Load payment methods only once per cart state change - don't retry if already attempted
      if (!hasAttemptedLoadingPayments && !isLoadingPaymentMethods) {
        loadPaymentMethods().catch(() => {
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
    hasAttemptedLoadingPayments = true
    try {
      const res = await apiGet<PaymentMethods>('/api/cart/payment')
      if (res.success && res.result) {
        payments = res.result
        provider = ''
        removeLocalStorage('provider')

        // Load PortOne config if portone is available
        if (payments.portone) {
          const portoneRes = await apiGet<{ store_id: string; channel_key: string; debug_enabled: boolean }>('/api/cart/portone-config')
          if (portoneRes.success && portoneRes.result) {
            portoneStoreId = portoneRes.result.store_id
            portoneChannelKey = portoneRes.result.channel_key
            portoneDebugEnabled = portoneRes.result.debug_enabled
          }
        }
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
      await loadPaymentMethods().catch(() => {
        error = 'Failed to load payment methods. Please refresh the page.'
        showOverlay = true
      })
    }
    // Don't auto-set provider for free carts on mount to prevent accidental checkout

    // Handle browser back/forward cache (bfcache) - reload cart when page restored from cache
    const handlePageShow = (event: PageTransitionEvent) => {
      if (event.persisted) {
        // Page was restored from bfcache, reload cart from storage
        cartStore.reload()
      }
    }

    window.addEventListener('pageshow', handlePageShow)

    return () => {
      window.removeEventListener('pageshow', handlePageShow)
    }
  })

  let showPayments = $derived(!isFree && hasPaymentProviders(payments))

  // Computed value instead of function - using formatCurrency (no truncation) for exact totals
  let totalCartAmount = $derived(
    cartTotal === 0
      ? t('product.free')
      : formatCurrency(cartTotal / 100, currency, numberFormat, symbolMode, currentLocale)
  )

  async function handlePortoneCheckout() {
    showOverlay = true
    try {
      await handlePortonePayment(email, cart, cartTotal, currency)
    } catch (err) {
      console.error('PortOne payment error (caught exception):', err)
      console.error('Error type:', typeof err)
      console.error('Error details:', err)
      if (err instanceof Error) {
        console.error('Error message:', err.message)
        console.error('Error stack:', err.stack)
      }

      // Don't show error overlay if validation modal is already showing
      if (err instanceof Error && err.message === 'Cart validation failed') {
        showOverlay = false
        return
      }

      error = err instanceof Error ? `Payment error: ${err.message}` : 'Payment failed. Please try again.'
      showOverlay = true
    }
  }

  async function handleStandardCheckout(finalProvider: string) {
    const cartData = {
      email,
      provider: finalProvider,
      products: cart.map((item) => ({
        id: item.id,
        variant_id: item.variant_id || undefined,
        quantity: item.quantity
      }))
    }

    const res = await apiPost<{ url?: string; validation_errors?: any[]; corrected_cart?: any[] }>('/cart/payment', cartData)

    // Handle validation errors (409 Conflict)
    if (res.status === 409 && res.result?.validation_errors && res.result?.corrected_cart) {
      handleValidationErrors(res.result.validation_errors, res.result.corrected_cart)
      return
    }

    if (res.success && res.result?.url) {
      window.location.href = res.result.url
    } else {
      error = res.message || t('payment.failed')
      showOverlay = true
    }
  }

  async function checkOut(e: Event) {
    e.preventDefault()

    debugLog('=== CHECKOUT STARTED ===')
    debugLog('Email:', email)
    debugLog('Provider:', provider)
    debugLog('Cart:', cart)
    debugLog('Is Free:', isFree)

    setLocalStorage('email', email)
    error = undefined

    const finalProvider = isFree ? 'dummy' : provider

    if (!isFree && !finalProvider) {
      console.error('No payment provider selected')
      error = t('cart.selectPaymentError')
      showOverlay = true
      return
    }

    setLocalStorage('provider', finalProvider)

    if (provider === 'portone') {
      await handlePortoneCheckout()
      return
    }

    await handleStandardCheckout(finalProvider)
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
            <div class="space-y-4">
              {#each cart as item (`${item.id}-${item.variant_id || 'no-variant'}`)}
                <CartItemCard
                  {item}
                  highlighted={highlightedItems.has(getItemKey(item))}
                  needsDeletion={item.needsDeletion || false}
                />
              {/each}
            </div>
          </div>

          <!-- Total -->
          <div class="brutal-card mb-8 bg-yellow-300 p-8">
            <div class="flex items-center justify-between">
              <span class="text-3xl font-black tracking-tighter text-black uppercase"> {t('cart.total')} </span>
              <span class="text-4xl font-black {cartTotal === 0 ? 'text-green-500' : 'text-black'}" data-testid="cart-total">
                {totalCartAmount}
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

                  {#if payments.portone}
                    <div>
                      <input type="radio" bind:group={provider} value="portone" id="portone" class="peer hidden" />
                      <label
                        for="portone"
                        class="block cursor-pointer border-4 border-black bg-white p-6 peer-checked:border-yellow-300 peer-checked:bg-yellow-300"
                      >
                        <p class="mb-2 text-xl font-black tracking-tight text-black uppercase">{t('cart.portone')}</p>
                        <p class="text-lg text-black">{t('cart.portoneDescription')}</p>
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

  {#if showValidationModal}
    <Overlay show={true} onClose={() => showValidationModal = false}>
      <div class="validation-modal bg-white p-8 border-4 border-black max-w-lg mx-auto">
        <h2 class="text-2xl font-black tracking-tight text-red-600 uppercase mb-4">{t('cart.validation_errors_title')}</h2>
        <ul class="space-y-3 mb-6">
          {#each validationErrors as error}
            <li class="bg-yellow-100 p-4 border-2 border-black">
              {#if error.error_type === 'quantity_unavailable'}
                <p class="font-bold">{t('cart.out_of_stock')} - {t('cart.please_remove')}</p>
              {:else if error.error_type === 'price_changed'}
                <p>{t('cart.price_updated')}:
                  <span class="line-through text-gray-600">{formatCurrency(error.requested_unit_price, currency)}</span>
                  → <span class="font-bold text-green-700">{formatCurrency(error.current_unit_price, currency)}</span>
                </p>
              {:else if error.error_type === 'product_inactive' || error.error_type === 'product_not_found'}
                <p class="font-bold">{t('cart.out_of_stock')} - {t('cart.please_remove')}</p>
              {/if}
            </li>
          {/each}
        </ul>
        <button
          onclick={() => showValidationModal = false}
          class="w-full border-4 border-black bg-blue-500 px-6 py-3 text-lg font-black tracking-wider text-white uppercase hover:bg-blue-600"
        >
          {t('cart.review_cart')}
        </button>
      </div>
    </Overlay>
  {/if}
</section>
