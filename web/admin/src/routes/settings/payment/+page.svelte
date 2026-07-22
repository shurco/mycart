<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import Main from '$lib/layouts/Main.svelte'
  import Drawer from '$lib/components/Drawer.svelte'
  import Stripe from '$lib/components/payment/Stripe.svelte'
  import Paypal from '$lib/components/payment/Paypal.svelte'
  import Portone from '$lib/components/payment/Portone.svelte'
  import Spectrocoin from '$lib/components/payment/Spectrocoin.svelte'
  import Coinbase from '$lib/components/payment/Coinbase.svelte'
  import FormButton from '$lib/components/form/Button.svelte'
  import FormSelect from '$lib/components/form/Select.svelte'
  import TruncationSettings from '$lib/components/TruncationSettings.svelte'
  import { systemStore } from '$lib/stores/system'
  import { paymentSettingsStore } from '$lib/stores/payment'
  import { loadSettings as loadSettingsHelper, saveSettings } from '$lib/utils/settingsHelpers'
  import { loadData } from '$lib/utils/apiHelpers'
  import { formatCurrency } from '$lib/utils/currency'
  import { translate } from '$lib/i18n'
  import { CURRENCIES } from '$lib/config/currencies'
  import { DRAWER_CLOSE_DELAY_MS } from '$lib/constants/ui'
  import type { PaymentSettings, TruncationSettings as TruncationSettingsType, CurrencyTruncationSettings, SymbolDisplaySettings } from '$lib/types/models'

  // Settings version key for cache invalidation (localStorage is shared across tabs)
  const SETTINGS_VERSION_KEY = 'settings_version'

  // Increment settings version to invalidate storefront cache
  function incrementSettingsVersion() {
    if (typeof localStorage === 'undefined') return
    const currentVersion = parseInt(localStorage.getItem(SETTINGS_VERSION_KEY) || '1', 10)
    localStorage.setItem(SETTINGS_VERSION_KEY, (currentVersion + 1).toString())
  }

  // Reactive translation function
  let t = $derived($translate)

  let drawerOpen = $state(false)
  let drawerMode = $state<'stripe' | 'paypal' | 'portone' | 'spectrocoin' | 'coinbase' | null>(null)
  let payments = $state<Record<string, boolean>>({})
  let payment = $state<PaymentSettings>({
    currency: ''
  })
  let formErrors = $state<Record<string, string>>({})
  let decimalPrecision = $state('2')
  let showTrailingZeros = $state(true)
  let symbolDisplay = $state<SymbolDisplaySettings>({
    admin: 'currency',
    storefront: 'currency'
  })

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

    // Load number format settings
    const nf = paymentSettings.number_format || {
      decimal_precision: 2,
      show_trailing_zeros: true
    }
    decimalPrecision = String(nf.decimal_precision)
    showTrailingZeros = nf.show_trailing_zeros
    payment.number_format = nf

    // Load symbol display settings
    const sd = paymentSettings.symbol_display || {
      admin: 'currency',
      storefront: 'currency'
    }
    symbolDisplay.admin = sd.admin
    symbolDisplay.storefront = sd.storefront
    payment.symbol_display = sd

    // Update global store for other admin pages to access
    paymentSettingsStore.set(payment)
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
    incrementSettingsVersion()
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
    // Force reactivity by reassigning the object
    payment = { ...payment, truncation: { ...payment.truncation } }
  }

  async function handleTruncationSubmit() {
    await saveSettings('payment', payment, 'Truncation settings saved')
    incrementSettingsVersion()
  }

  function formatPreview(value: number): string {
    const nf = {
      decimal_precision: parseInt(decimalPrecision) as 0 | 1 | 2,
      show_trailing_zeros: showTrailingZeros
    }
    return formatCurrency(value, payment.currency || 'USD', nf)
  }

  async function handleNumberFormatSubmit() {
    payment.number_format = {
      decimal_precision: parseInt(decimalPrecision) as 0 | 1 | 2,
      show_trailing_zeros: showTrailingZeros
    }
    await saveSettings('payment', payment, 'Number formatting saved')
    paymentSettingsStore.set(payment)
    incrementSettingsVersion()
  }

  async function handleSymbolDisplaySubmit() {
    payment.symbol_display = {
      admin: symbolDisplay.admin,
      storefront: symbolDisplay.storefront
    }
    await saveSettings('payment', payment, 'Symbol display saved')
    paymentSettingsStore.set(payment)
    incrementSettingsVersion()
  }

  function openDrawer(mode: 'stripe' | 'paypal' | 'portone' | 'spectrocoin' | 'coinbase') {
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

    <div class="mt-5 max-w-2xl">
      <h2 class="mb-5">Number Formatting</h2>

      <FormSelect
        id="decimal-precision"
        title="Decimal Precision"
        options={['0', '1', '2']}
        bind:value={decimalPrecision}
        ico="hash"
      />

      <div class="mb-4 flex items-center">
        <div class="pr-3">
          <h3 class="text-sm font-medium text-gray-700">Show Trailing Zeros</h3>
          <p class="text-sm text-gray-500">Display 1.00 instead of 1</p>
        </div>
        <div class="pt-1">
          <label for="toggle_trailing-zeros" class="none relative h-6 w-10 cursor-pointer [-webkit-tap-highlight-color:_transparent]">
            <input
              type="checkbox"
              class="peer sr-only [&:checked_+_span_svg[data-checked-icon]]:block [&:checked_+_span_svg[data-unchecked-icon]]:hidden"
              id="toggle_trailing-zeros"
              bind:checked={showTrailingZeros}
            />
            <span class="absolute inset-y-0 start-0 z-10 m-1 inline-flex h-4 w-4 items-center justify-center rounded-full bg-white text-gray-400 transition-all peer-checked:start-4 peer-checked:text-green-600">
              <svg data-unchecked-icon xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd"></path>
              </svg>
              <svg data-checked-icon xmlns="http://www.w3.org/2000/svg" class="hidden h-3 w-3" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"></path>
              </svg>
            </span>
            <span class="absolute inset-0 rounded-full bg-gray-300 transition peer-checked:bg-green-500"></span>
          </label>
        </div>
      </div>

      <div class="mt-3 text-sm text-gray-600">
        <div>Preview: 1.00 → {formatPreview(1.00)}</div>
        <div>Preview: 1.23 → {formatPreview(1.23)}</div>
      </div>

      <div class="pt-5">
        <FormButton onclick={handleNumberFormatSubmit} name="Save" color="green" />
      </div>
    </div>

    <hr class="mt-5" />

    <div class="mt-5 max-w-2xl">
      <h2 class="mb-5">Currency Display</h2>

      <div class="mb-4">
        <h3 class="mb-2 text-sm font-medium text-gray-700">Admin Panel Display</h3>
        <div class="flex gap-2">
          <button
            type="button"
            class="flex-1 rounded border px-4 py-2 text-sm font-medium transition-colors {symbolDisplay.admin === 'currency' ? 'border-green-500 bg-green-50 text-green-700' : 'border-gray-300 bg-white text-gray-700 hover:bg-gray-50'}"
            onclick={() => symbolDisplay.admin = 'currency'}
          >
            Currency Symbol
            <span class="block text-xs text-gray-500 mt-1">$130</span>
          </button>
          <button
            type="button"
            class="flex-1 rounded border px-4 py-2 text-sm font-medium transition-colors {symbolDisplay.admin === 'language' ? 'border-green-500 bg-green-50 text-green-700' : 'border-gray-300 bg-white text-gray-700 hover:bg-gray-50'}"
            onclick={() => symbolDisplay.admin = 'language'}
          >
            Language Symbol
            <span class="block text-xs text-gray-500 mt-1">130 Dollar</span>
          </button>
        </div>
      </div>

      <div class="mb-4">
        <h3 class="mb-2 text-sm font-medium text-gray-700">Storefront Display</h3>
        <div class="flex gap-2">
          <button
            type="button"
            class="flex-1 rounded border px-4 py-2 text-sm font-medium transition-colors {symbolDisplay.storefront === 'currency' ? 'border-green-500 bg-green-50 text-green-700' : 'border-gray-300 bg-white text-gray-700 hover:bg-gray-50'}"
            onclick={() => symbolDisplay.storefront = 'currency'}
          >
            Currency Symbol
            <span class="block text-xs text-gray-500 mt-1">$130</span>
          </button>
          <button
            type="button"
            class="flex-1 rounded border px-4 py-2 text-sm font-medium transition-colors {symbolDisplay.storefront === 'language' ? 'border-green-500 bg-green-50 text-green-700' : 'border-gray-300 bg-white text-gray-700 hover:bg-gray-50'}"
            onclick={() => symbolDisplay.storefront = 'language'}
          >
            Language Symbol
            <span class="block text-xs text-gray-500 mt-1">130 Dollar</span>
          </button>
        </div>
      </div>

      <div class="pt-5">
        <FormButton onclick={handleSymbolDisplaySubmit} name="Save" color="green" />
      </div>
    </div>

    <hr class="mt-5" />

    {#if payment.currency}
      <div class="mt-5">
        <h2 class="mb-5">Price Display Settings</h2>

        <div class="max-w-4xl space-y-6">
          <!-- Admin Panel Settings -->
          <div>
            <h3 class="mb-3 text-lg font-semibold">Admin Panel</h3>
            <TruncationSettings
              currency={payment.currency}
              context="admin"
              value={payment.truncation?.admin[payment.currency] || { mode: 'none' }}
              onChange={(settings) => handleTruncationChange('admin', payment.currency, settings)}
              numberFormat={payment.number_format}
            />
          </div>

          <!-- Storefront Settings -->
          <div>
            <h3 class="mb-3 text-lg font-semibold">Storefront</h3>
            <TruncationSettings
              currency={payment.currency}
              context="storefront"
              value={payment.truncation?.storefront[payment.currency] || { mode: 'none' }}
              onChange={(settings) => handleTruncationChange('storefront', payment.currency, settings)}
              numberFormat={payment.number_format}
            />
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
    {/if}

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
          class="ml-5 cursor-pointer rounded p-2 {payments.portone ? 'bg-green-200' : 'bg-gray-200'}"
          onclick={() => openDrawer('portone')}
          role="button"
          tabindex="0"
          onkeydown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault()
              openDrawer('portone')
            }
          }}
        >
          PortOne
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
    {:else if drawerMode === 'portone'}
      <Portone onclose={closeDrawer} />
    {:else if drawerMode === 'spectrocoin'}
      <Spectrocoin onclose={closeDrawer} />
    {:else if drawerMode === 'coinbase'}
      <Coinbase onclose={closeDrawer} />
    {/if}
  </Drawer>
{/if}
