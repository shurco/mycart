<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import Main from '$lib/layouts/Main.svelte'
  import Drawer from '$lib/components/Drawer.svelte'
  import Stripe from '$lib/components/payment/Stripe.svelte'
  import Paypal from '$lib/components/payment/Paypal.svelte'
  import Spectrocoin from '$lib/components/payment/Spectrocoin.svelte'
  import Coinbase from '$lib/components/payment/Coinbase.svelte'
  import FormButton from '$lib/components/form/Button.svelte'
  import FormSelect from '$lib/components/form/Select.svelte'
  import { systemStore } from '$lib/stores/system'
  import { loadSettings as loadSettingsHelper, saveSettings } from '$lib/utils/settingsHelpers'
  import { loadData } from '$lib/utils/apiHelpers'
  import { translate } from '$lib/i18n'
  import { DRAWER_CLOSE_DELAY_MS } from '$lib/constants/ui'
  import type { PaymentSettings } from '$lib/types/models'

  // Reactive translation function
  let t = $derived($translate)

  let drawerOpen = $state(false)
  let drawerMode = $state<'stripe' | 'paypal' | 'spectrocoin' | 'coinbase' | null>(null)
  let payments = $state<Record<string, boolean>>({})
  let payment = $state<PaymentSettings>({
    currency: ''
  })
  let formErrors = $state<Record<string, string>>({})

  const currencyOptions = ['EUR', 'USD', 'JPY', 'GBP', 'AUD', 'CAD', 'CHF', 'CNY', 'SEK']

  let unsubscribe: (() => void) | null = null

  onMount(async () => {
    await loadPaymentSettings()

    // Subscribe to store updates only on client side
    unsubscribe = systemStore.subscribe((store) => {
      payments = store.payments || {}
    })
  })

  onDestroy(() => {
    if (unsubscribe) {
      unsubscribe()
    }
  })

  async function loadPaymentSettings() {
    const paymentProviders = await loadData<Record<string, boolean>>(
      '/api/cart/payment',
      'Failed to load payment settings'
    )
    if (paymentProviders) {
      payments = paymentProviders
      systemStore.update((store) => ({
        ...store,
        payments: payments
      }))
    }

    const paymentSettings = await loadSettingsHelper<PaymentSettings>('payment', payment)
    payment.currency = paymentSettings.currency
  }

  async function handleCurrencySubmit() {
    formErrors = {}

    if (!payment.currency) {
      formErrors.currency = 'Currency is required'
      return
    }

    if (!currencyOptions.includes(payment.currency)) {
      formErrors.currency = 'Currency must be one of: ' + currencyOptions.join(', ')
      return
    }

    await saveSettings('payment', payment, 'Currency saved')
  }

  function openDrawer(mode: 'stripe' | 'paypal' | 'spectrocoin' | 'coinbase') {
    drawerMode = mode
    drawerOpen = true
  }

  function closeDrawer() {
    drawerOpen = false
    setTimeout(() => {
      drawerMode = null
    }, DRAWER_CLOSE_DELAY_MS)
  }
</script>

<Main>
  <div class="pb-10">
    <header class="mb-4">
      <h1>Payment</h1>
    </header>

    <form onsubmit={(e) => { e.preventDefault(); handleCurrencySubmit(); }} class="max-w-2xl">
      <FormSelect
        id="currency"
        title={t('settings.currency')}
        options={currencyOptions}
        bind:value={payment.currency}
        error={formErrors.currency}
        ico="money"
      />
      <div class="pt-5">
        <FormButton type="submit" name={t('common.save')} color="green" />
      </div>
    </form>
    <hr class="mt-5" />

    <div class="mt-5">
      <h2 class="mb-5">Payment providers</h2>
      <div class="flex">
        <div
          class="cursor-pointer rounded p-2 {payments.stripe ? 'bg-green-200' : 'bg-gray-200'}"
          onclick={() => openDrawer('stripe')}
          role="button"
          tabindex="0"
          onkeydown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              openDrawer('stripe')
            }
          }}
        >
          Stripe
        </div>
        <div
          class="ml-5 cursor-pointer rounded p-2 {payments.paypal ? 'bg-green-200' : 'bg-gray-200'}"
          onclick={() => openDrawer('paypal')}
          role="button"
          tabindex="0"
          onkeydown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              openDrawer('paypal')
            }
          }}
        >
          Paypal
        </div>
        <div
          class="ml-5 cursor-pointer rounded p-2 {payments.spectrocoin ? 'bg-green-200' : 'bg-gray-200'}"
          onclick={() => openDrawer('spectrocoin')}
          role="button"
          tabindex="0"
          onkeydown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              openDrawer('spectrocoin')
            }
          }}
        >
          Spectrocoin
        </div>
        <div
          class="ml-5 cursor-pointer rounded p-2 {payments.coinbase ? 'bg-green-200' : 'bg-gray-200'}"
          onclick={() => openDrawer('coinbase')}
          role="button"
          tabindex="0"
          onkeydown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              openDrawer('coinbase')
            }
          }}
        >
          Coinbase
        </div>
      </div>
    </div>
  </div>
</Main>

{#if drawerOpen}
  <Drawer isOpen={drawerOpen} onclose={closeDrawer} maxWidth="725px">
    {#if drawerMode === 'stripe'}
      <Stripe onclose={closeDrawer} />
    {:else if drawerMode === 'paypal'}
      <Paypal onclose={closeDrawer} />
    {:else if drawerMode === 'spectrocoin'}
      <Spectrocoin onclose={closeDrawer} />
    {:else if drawerMode === 'coinbase'}
      <Coinbase onclose={closeDrawer} />
    {/if}
  </Drawer>
{/if}
