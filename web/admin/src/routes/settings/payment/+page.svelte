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
  import TruncationSettings from '$lib/components/TruncationSettings.svelte'
  import { systemStore } from '$lib/stores/system'
  import { loadSettings as loadSettingsHelper, saveSettings } from '$lib/utils/settingsHelpers'
  import { loadData } from '$lib/utils/apiHelpers'
  import { translate } from '$lib/i18n'
  import { CURRENCIES } from '$lib/config/currencies'
  import { DRAWER_CLOSE_DELAY_MS } from '$lib/constants/ui'
  import type { PaymentSettings, TruncationSettings as TruncationSettingsType, CurrencyTruncationSettings } from '$lib/types/models'

  // Reactive translation function
  let t = $derived($translate)

  let drawerOpen = $state(false)
  let drawerMode = $state<'stripe' | 'paypal' | 'spectrocoin' | 'coinbase' | null>(null)
  let payments = $state<Record<string, boolean>>({})
  let payment = $state<PaymentSettings>({
    currency: ''
  })
  let formErrors = $state<Record<string, string>>({})

  const currencyOptions = CURRENCIES.map(c => c.code)

  // Initialize truncation settings with defaults
  const defaultTruncation = (): TruncationSettingsType => ({
    admin: {},
    storefront: {}
  })

  // Ensure all currencies have default 'none' mode
  function ensureDefaults(truncation: TruncationSettingsType | undefined): TruncationSettingsType {
    const result = truncation || defaultTruncation()

    CURRENCIES.forEach(curr => {
      if (!result.admin[curr.code]) {
        result.admin[curr.code] = { mode: 'none' }
      }
      if (!result.storefront[curr.code]) {
        result.storefront[curr.code] = { mode: 'none' }
      }
    })

    return result
  }

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
    payment.truncation = ensureDefaults(paymentSettings.truncation)
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

  function handleTruncationChange(
    context: 'admin' | 'storefront',
    currency: string,
    settings: CurrencyTruncationSettings
  ) {
    if (!payment.truncation) {
      payment.truncation = defaultTruncation()
    }
    payment.truncation[context][currency] = settings
  }

  async function handleTruncationSubmit() {
    await saveSettings('payment', payment, 'Truncation settings saved')
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
      <h2 class="mb-5">Price Display Settings</h2>

      <div class="max-w-4xl space-y-6">
        <!-- Admin Panel Settings -->
        <div>
          <h3 class="mb-3 text-lg font-semibold">Admin Panel</h3>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            {#each CURRENCIES as curr}
              <TruncationSettings
                currency={curr.code}
                context="admin"
                value={payment.truncation?.admin[curr.code] || { mode: 'none' }}
                onChange={(settings) => handleTruncationChange('admin', curr.code, settings)}
              />
            {/each}
          </div>
        </div>

        <!-- Storefront Settings -->
        <div>
          <h3 class="mb-3 text-lg font-semibold">Storefront</h3>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
            {#each CURRENCIES as curr}
              <TruncationSettings
                currency={curr.code}
                context="storefront"
                value={payment.truncation?.storefront[curr.code] || { mode: 'none' }}
                onChange={(settings) => handleTruncationChange('storefront', curr.code, settings)}
              />
            {/each}
          </div>
        </div>

        <div class="pt-4">
          <FormButton
            type="button"
            name={t('common.save')}
            color="green"
            onclick={handleTruncationSubmit}
          />
        </div>
      </div>
    </div>

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
