<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import FormButton from '../form/Button.svelte'
  import FormInput from '../form/Input.svelte'
  import FormToggle from '../form/Toggle.svelte'
  import { loadPaymentSettings, savePaymentSettings, togglePaymentActive } from '$lib/composables/usePaymentSettings'
  import { systemStore } from '$lib/stores/system'
  import {
    MIN_PORTONE_STORE_ID_LENGTH,
    MIN_PORTONE_CHANNEL_KEY_LENGTH,
    MIN_PORTONE_API_SECRET_LENGTH,
    ERROR_MESSAGES
  } from '$lib/constants/validation'
  import type { PortoneSettings } from '$lib/types/models'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    onclose?: () => void
  }

  let { onclose }: Props = $props()

  const AVAILABLE_CURRENCIES = ['KRW', 'USD', 'JPY', 'EUR', 'GBP']

  let settings = $state<PortoneSettings>({
    active: false,
    store_id: '',
    channel_key: '',
    api_secret: '',
    debug_enabled: false,
    supported_currencies: ['KRW']
  })
  let formErrors = $state<Record<string, string>>({})
  let unsubscribe: (() => void) | null = null

  onMount(async () => {
    const loaded = await loadPaymentSettings<PortoneSettings>('portone', settings)
    settings = {
      ...loaded,
      debug_enabled: loaded.debug_enabled ?? false,
      supported_currencies: loaded.supported_currencies ?? ['KRW']
    }

    unsubscribe = systemStore.subscribe((store) => {
      if (store.payments?.portone !== undefined) {
        settings.active = store.payments.portone
      }
    })
  })

  onDestroy(() => {
    unsubscribe?.()
  })

  function toggleCurrency(currency: string) {
    const currencies = settings.supported_currencies ?? []
    const index = currencies.indexOf(currency)
    if (index > -1) {
      settings.supported_currencies = currencies.filter(c => c !== currency)
    } else {
      settings.supported_currencies = [...currencies, currency]
    }
  }

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault()
    formErrors = {}

    if (!settings.store_id || settings.store_id.length < MIN_PORTONE_STORE_ID_LENGTH) {
      formErrors.store_id = ERROR_MESSAGES.PORTONE_STORE_ID_TOO_SHORT
      return
    }
    if (!settings.channel_key || settings.channel_key.length < MIN_PORTONE_CHANNEL_KEY_LENGTH) {
      formErrors.channel_key = ERROR_MESSAGES.PORTONE_CHANNEL_KEY_TOO_SHORT
      return
    }
    if (!settings.api_secret || settings.api_secret.length < MIN_PORTONE_API_SECRET_LENGTH) {
      formErrors.api_secret = ERROR_MESSAGES.PORTONE_API_SECRET_TOO_SHORT
      return
    }

    await savePaymentSettings('portone', settings)
  }

  async function handleToggleActive() {
    const previousValue = settings.active
    const success = await togglePaymentActive('portone', settings.active)

    if (!success) {
      settings.active = previousValue
    }
  }

  function close() {
    onclose?.()
  }
</script>

<div>
  <div class="pb-8">
    <div class="flex items-center">
      <div class="pr-3">
        <h1>PortOne</h1>
      </div>
      <div class="pt-1">
        <FormToggle
          id="portone-active"
          bind:value={settings.active}
          disabled={Object.keys(formErrors).length > 0}
          onchange={handleToggleActive}
        />
      </div>
    </div>
  </div>

  <form onsubmit={handleSubmit}>
    <div class="flow-root">
      <dl class="mx-auto -my-3 mt-2 mb-0 space-y-4 text-sm">
        <FormInput
          id="store_id"
          type="text"
          title={t('payment.storeId')}
          bind:value={settings.store_id}
          error={formErrors.store_id}
          ico="key"
        />
      </dl>

      <dl class="mx-auto -my-3 mt-5 mb-0 space-y-4 text-sm">
        <FormInput
          id="channel_key"
          type="text"
          title={t('payment.channelKey')}
          bind:value={settings.channel_key}
          error={formErrors.channel_key}
          ico="key"
        />
      </dl>

      <dl class="mx-auto -my-3 mt-5 mb-0 space-y-4 text-sm">
        <FormInput
          id="api_secret"
          type="password"
          title={t('payment.secretKey')}
          bind:value={settings.api_secret}
          error={formErrors.api_secret}
          ico="key"
        />
      </dl>

      <dl class="mx-auto -my-3 mt-5 mb-0 space-y-4 text-sm">
        <div class="flex items-center gap-4">
          <label for="debug-enabled" class="font-medium">{t('payment.portone.debugEnabled') || 'Debug Mode'}</label>
          <FormToggle
            id="debug-enabled"
            bind:value={settings.debug_enabled}
          />
        </div>
        <p class="text-xs text-gray-500 mt-1">{t('payment.portone.debugEnabledDesc') || 'Show debug logs in browser console'}</p>
      </dl>

      <dl class="mx-auto -my-3 mt-5 mb-0 space-y-4 text-sm">
        <div>
          <label class="font-medium block mb-2">{t('payment.portone.supportedCurrencies') || 'Supported Currencies'}</label>
          <p class="text-xs text-gray-500 mb-3">{t('payment.portone.supportedCurrenciesDesc') || 'Select currencies this provider accepts'}</p>
          <div class="flex flex-wrap gap-3">
            {#each AVAILABLE_CURRENCIES as currency}
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={settings.supported_currencies?.includes(currency)}
                  onchange={() => toggleCurrency(currency)}
                  class="w-4 h-4"
                />
                <span>{currency}</span>
              </label>
            {/each}
          </div>
        </div>
      </dl>
    </div>

    <div class="pt-8">
      <div class="flex">
        <div class="flex-none">
          <FormButton type="submit" name={t('common.save')} color="green" />
        </div>
        <div class="grow"></div>
        <div class="flex-none">
          <FormButton type="button" name={t('common.close')} color="gray" onclick={close} />
        </div>
      </div>
    </div>
  </form>
</div>
