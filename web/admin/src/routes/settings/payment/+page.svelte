<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import Main from '$lib/layouts/Main.svelte'
  import Stripe from '$lib/components/payment/Stripe.svelte'
  import Paypal from '$lib/components/payment/Paypal.svelte'
  import Portone from '$lib/components/payment/Portone.svelte'
  import Spectrocoin from '$lib/components/payment/Spectrocoin.svelte'
  import Coinbase from '$lib/components/payment/Coinbase.svelte'
  import {
    PageHeader,
    PageState,
    Section,
    Chip,
    ChipGroup,
    Drawer,
    FormButton,
    FormSelect
  } from '$lib/components'
  import NumberFormatForm from './NumberFormatForm.svelte'
  import SymbolDisplayForm from './SymbolDisplayForm.svelte'
  import PriceDisplayForm from './PriceDisplayForm.svelte'
  import { incrementSettingsVersion, defaultTruncation, ensureDefaults } from './paymentDisplay'
  import { systemStore } from '$lib/stores/system'
  import { paymentSettingsStore } from '$lib/stores/payment'
  import { loadSettings as loadSettingsHelper, saveSettings } from '$lib/utils/settingsHelpers'
  import { loadData } from '$lib/utils/apiHelpers'
  import { translate } from '$lib/i18n'
  import { CURRENCIES } from '$lib/config/currencies'
  import { DRAWER_CLOSE_DELAY_MS } from '$lib/constants/ui'
  import { createDelayedReset } from '$lib/utils/delayedReset'
  import type { PaymentSettings, CurrencyTruncationSettings, SymbolDisplaySettings } from '$lib/types/models'

  // Reactive translation function
  let t = $derived($translate)

  let loading = $state(true)
  /** The drawer the page currently has open. */
  type DrawerMode =
    | 'number-format'
    | 'currency-display'
    | 'price-display'
    | 'stripe'
    | 'paypal'
    | 'portone'
    | 'spectrocoin'
    | 'coinbase'

  let drawerOpen = $state(false)
  let drawerMode = $state<DrawerMode | null>(null)

  // The three settings editors are form drawers, so 710px; provider drawers 725px.
  let drawerWidth = $derived(
    drawerMode === 'number-format' ||
      drawerMode === 'currency-display' ||
      drawerMode === 'price-display'
      ? '710px'
      : '725px'
  )
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
      t('payment.failedToLoadSettings')
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
    loading = false
  }

  async function handleCurrencySubmit() {
    formErrors = {}

    if (!payment.currency) {
      formErrors.currency = t('payment.currencyRequired')
      return
    }

    if (!currencyOptions.includes(payment.currency)) {
      formErrors.currency = t('payment.currencyMustBeOneOf', { list: currencyOptions.join(', ') })
      return
    }

    await saveSettings('payment', payment, t('payment.currencySaved'))
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
    await saveSettings('payment', payment, t('payment.truncationSaved'))
    incrementSettingsVersion()
  }

  async function handleNumberFormatSubmit() {
    payment.number_format = {
      decimal_precision: parseInt(decimalPrecision) as 0 | 1 | 2,
      show_trailing_zeros: showTrailingZeros
    }
    await saveSettings('payment', payment, t('payment.numberFormatSaved'))
    paymentSettingsStore.set(payment)
    incrementSettingsVersion()
  }

  async function handleSymbolDisplaySubmit() {
    payment.symbol_display = {
      admin: symbolDisplay.admin,
      storefront: symbolDisplay.storefront
    }
    await saveSettings('payment', payment, t('payment.symbolDisplaySaved'))
    paymentSettingsStore.set(payment)
    incrementSettingsVersion()
  }

  function openDrawer(mode: DrawerMode) {
    drawerMode = mode
    drawerOpen = true
  }

  const resetDrawer = createDelayedReset(DRAWER_CLOSE_DELAY_MS)

  function closeDrawer() {
    drawerOpen = false
    resetDrawer(() => {
      drawerMode = null
    })
  }
</script>

<Main>
  <PageHeader title={t('settings.payment')} />

  {#if loading}
    <PageState kind="loading" />
  {:else}
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
        <FormButton type="submit" name={t('common.save')} variant="primary" />
      </div>
    </form>

    <Section title={t('settings.displaySettings')}>
      <ChipGroup>
        <Chip onclick={() => openDrawer('number-format')}>{t('settings.numberFormatting')}</Chip>
        <Chip onclick={() => openDrawer('currency-display')}>{t('settings.currencyDisplay')}</Chip>
        {#if payment.currency}
          <Chip onclick={() => openDrawer('price-display')}>
            {t('settings.priceDisplaySettings')}
          </Chip>
        {/if}
      </ChipGroup>
    </Section>

    <Section title={t('settings.paymentProviders')}>
      <ChipGroup>
        <Chip active={payments.stripe} onclick={() => openDrawer('stripe')}>Stripe</Chip>
        <Chip active={payments.paypal} onclick={() => openDrawer('paypal')}>Paypal</Chip>
        <Chip active={payments.portone} onclick={() => openDrawer('portone')}>PortOne</Chip>
        <Chip active={payments.spectrocoin} onclick={() => openDrawer('spectrocoin')}>Spectrocoin</Chip>
        <Chip active={payments.coinbase} onclick={() => openDrawer('coinbase')}>Coinbase</Chip>
      </ChipGroup>
    </Section>
  {/if}
</Main>

{#if drawerOpen}
  <Drawer isOpen={drawerOpen} onclose={closeDrawer} maxWidth={drawerWidth}>
    {#if drawerMode === 'number-format'}
      <NumberFormatForm
        bind:decimalPrecision
        bind:showTrailingZeros
        currency={payment.currency}
        onsubmit={handleNumberFormatSubmit}
        onclose={closeDrawer}
      />
    {:else if drawerMode === 'currency-display'}
      <SymbolDisplayForm bind:symbolDisplay onsubmit={handleSymbolDisplaySubmit} onclose={closeDrawer} />
    {:else if drawerMode === 'price-display' && payment.currency}
      <PriceDisplayForm
        {payment}
        onchange={handleTruncationChange}
        onsubmit={handleTruncationSubmit}
        onclose={closeDrawer}
      />
    {:else if drawerMode === 'stripe'}
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
