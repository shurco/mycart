<script lang="ts">
  import { onMount } from 'svelte'
  import { cartStore } from '$lib/stores/cart'
  import { settingsStore } from '$lib/stores/settings'
    import { hasPaymentProviders } from '$lib/utils/payment'
  import { getLocalStorage, setLocalStorage, removeLocalStorage } from '$lib/utils/browser'
  import {
    applyCartCorrections,
    fetchPaymentMethods,
    getItemKey,
    handlePortonePayment,
    postStandardCheckout
  } from '$lib/utils/checkout'
  import type { PaymentMethods } from '$lib/types/models'
  import CartEmpty from '$lib/components/CartEmpty.svelte'
  import CartEmailField from '$lib/components/CartEmailField.svelte'
  import Overlay from '$lib/components/Overlay.svelte'
  import CartItemsList from '$lib/components/CartItemsList.svelte'
  import CartTotal from '$lib/components/CartTotal.svelte'
  import CartCheckoutButton from '$lib/components/CartCheckoutButton.svelte'
  import CartValidationModal from '$lib/components/CartValidationModal.svelte'
  import PaymentProviderOptions from '$lib/components/PaymentProviderOptions.svelte'
  import { translate } from '$lib/i18n'

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

  function handleValidationErrors(errors: any[], correctedCart: any[]) {
    validationErrors = errors
    showValidationModal = true

    const corrections = applyCartCorrections($cartStore, errors, correctedCart)
    corrections.highlighted.forEach((key) => highlightedItems.add(key))

    try {
      cartStore.set(corrections.cart)
    } catch {
      error = 'Failed to update cart. Please clear your cart and try again.'
      showOverlay = true
    }
  }

  let cart = $derived($cartStore)
  let currency = $derived($settingsStore?.main.currency || '')

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
      const loaded = await fetchPaymentMethods()
      payments = loaded.payments
      provider = ''
      removeLocalStorage('provider')

      if (loaded.portone) {
        portoneStoreId = loaded.portone.storeId
        portoneChannelKey = loaded.portone.channelKey
        portoneDebugEnabled = loaded.portone.debugEnabled
      }
    } finally {
      isLoadingPaymentMethods = false
    }
  }

  onMount(() => {
    // Always reload cart from localStorage on mount to ensure fresh data
    cartStore.reload()

    email = getLocalStorage('email')

    // If cart is not free, load payment methods
    // $effect will also handle this, but we load here on initial mount to avoid delay
    if (!isFree && !hasPaymentProviders(payments)) {
      loadPaymentMethods().catch(() => {
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

  async function handlePortoneCheckout() {
    showOverlay = true
    try {
      await handlePortonePayment(
        { storeId: portoneStoreId, channelKey: portoneChannelKey },
        email,
        cart,
        cartTotal,
        currency,
        handleValidationErrors,
        debugLog
      )
    } catch (err) {
      debugLog('PortOne payment error:', err)

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
    const res = await postStandardCheckout(email, finalProvider, cart)

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
        <CartEmpty />
      {/if}

      <form onsubmit={checkOut}>
        {#if cart.length > 0}
          <CartItemsList {cart} {highlightedItems} />

          <CartTotal {cartTotal} {currency} />

          {#if isFree || showPayments}
            <CartEmailField bind:email {isFree} />

            <!-- Payment Provider Selection -->
            {#if showPayments}
              <div class="mt-16 mb-8">
                <h2 class="mb-6 text-3xl font-black tracking-tighter text-black uppercase">{t('cart.selectPaymentSystem')}</h2>
                <PaymentProviderOptions {payments} bind:provider />
              </div>
            {/if}

            <CartCheckoutButton {isFree} disabled={!email || (!isFree && !provider)} />
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
    <CartValidationModal errors={validationErrors} {currency} onClose={() => showValidationModal = false} />
  {/if}
</section>