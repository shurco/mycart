<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import FormInput from '../form/Input.svelte'
  import FormTextarea from '../form/Textarea.svelte'
  import FormToggle from '../form/Toggle.svelte'
  import { loadPaymentSettings, savePaymentSettings, togglePaymentActive } from '$lib/composables/usePaymentSettings'
  import { systemStore } from '$lib/stores/system'
  import {
    MIN_MERCHANT_ID_LENGTH,
    MIN_PROJECT_ID_LENGTH,
    MIN_PRIVATE_KEY_LENGTH,
    ERROR_MESSAGES
  } from '$lib/constants/validation'
  import type { SpectrocoinSettings } from '$lib/types/models'
  import { translate } from '$lib/i18n'
  import { DrawerFooter, DrawerHeader } from '$lib/components'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    onclose?: () => void
  }

  let { onclose }: Props = $props()

  let settings = $state<SpectrocoinSettings>({
    active: false,
    merchant_id: '',
    project_id: '',
    callback_merchant_id: 0,
    callback_api_id: 0,
    private_key: ''
  })
  // The callback identity is numeric but the form takes it as typed text: an
  // input bound to a number turns an empty box into 0, which is
  // indistinguishable from an id that was never filled in. The strings are
  // converted on submit.
  let callbackMerchantID = $state('')
  let callbackApiID = $state('')
  let formErrors = $state<Record<string, string>>({})
  let unsubscribe: (() => void) | null = null

  // Zero means "not configured": the server accepts it and the callback handler
  // refuses to match it, so an empty box must stay savable — a shop that does
  // not take SpectroCoin callbacks still has to be able to save its settings.
  // Anything typed has to be a positive integer instead: sent as NaN it would
  // reach the server as zero, and past Number.MAX_SAFE_INTEGER it would reach it
  // as a different number than SpectroCoin reports, rejecting every callback
  // with nothing to point at the cause.
  function parseMerchantNumber(value: string): number | null {
    const trimmed = value.trim()
    if (trimmed === '') return 0

    const parsed = Number(trimmed)
    return /^\d+$/.test(trimmed) && Number.isSafeInteger(parsed) && parsed > 0 ? parsed : null
  }

  onMount(async () => {
    settings = await loadPaymentSettings<SpectrocoinSettings>('spectrocoin', settings)
    callbackMerchantID = settings.callback_merchant_id ? String(settings.callback_merchant_id) : ''
    callbackApiID = settings.callback_api_id ? String(settings.callback_api_id) : ''

    unsubscribe = systemStore.subscribe((store) => {
      if (store.payments?.spectrocoin !== undefined) {
        settings.active = store.payments.spectrocoin
      }
    })
  })

  onDestroy(() => {
    unsubscribe?.()
  })

  async function handleSubmit() {
    formErrors = {}

    if (!settings.merchant_id || settings.merchant_id.length < MIN_MERCHANT_ID_LENGTH) {
      formErrors.merchant_id = ERROR_MESSAGES.MERCHANT_ID_TOO_SHORT
      return
    }
    if (!settings.project_id || settings.project_id.length < MIN_PROJECT_ID_LENGTH) {
      formErrors.project_id = ERROR_MESSAGES.PROJECT_ID_TOO_SHORT
      return
    }
    const merchantNumber = parseMerchantNumber(callbackMerchantID)
    const apiNumber = parseMerchantNumber(callbackApiID)
    if (merchantNumber === null) {
      formErrors.callback_merchant_id = t('payment.callbackMerchantIdInvalid')
      return
    }
    if (apiNumber === null) {
      formErrors.callback_api_id = t('payment.callbackApiIdInvalid')
      return
    }
    // A half-filled pair can never match a callback, and would leave the
    // operator with nothing on screen to explain why every one is rejected.
    if ((merchantNumber === 0) !== (apiNumber === 0)) {
      if (merchantNumber === 0) {
        formErrors.callback_merchant_id = t('payment.callbackMerchantIdInvalid')
      } else {
        formErrors.callback_api_id = t('payment.callbackApiIdInvalid')
      }
      return
    }
    if (!settings.private_key || settings.private_key.length < MIN_PRIVATE_KEY_LENGTH) {
      formErrors.private_key = ERROR_MESSAGES.PRIVATE_KEY_TOO_SHORT
      return
    }

    await savePaymentSettings('spectrocoin', {
      ...settings,
      callback_merchant_id: merchantNumber,
      callback_api_id: apiNumber
    })
  }

  async function handleToggleActive() {
    const previousValue = settings.active
    const success = await togglePaymentActive('spectrocoin', settings.active)

    if (!success) {
      settings.active = previousValue
    }
  }

  function close() {
    onclose?.()
  }
</script>

<div>
  <DrawerHeader title="Spectrocoin">
    {#snippet actions()}
      <FormToggle
        id="spectrocoin-active"
        bind:value={settings.active}
        disabled={Object.keys(formErrors).length > 0}
        onchange={handleToggleActive}
      />
    {/snippet}
  </DrawerHeader>

  <form onsubmit={(e) => { e.preventDefault(); handleSubmit() }}>
    <div class="flow-root">
      <dl class="mx-auto -my-3 mt-2 mb-0 space-y-4 text-sm">
        <FormInput
          id="merchant_id"
          type="text"
          title={t('payment.merchantId')}
          bind:value={settings.merchant_id}
          error={formErrors.merchant_id}
          ico="key"
        />
        <div class="mt-5">
          <FormInput
            id="project_id"
            type="text"
            title={t('payment.projectId')}
            bind:value={settings.project_id}
            error={formErrors.project_id}
            ico="key"
          />
        </div>
        <div class="mt-5">
          <FormInput
            id="callback_merchant_id"
            type="text"
            title={t('payment.callbackMerchantId')}
            bind:value={callbackMerchantID}
            error={formErrors.callback_merchant_id}
            ico="key"
          />
        </div>
        <div class="mt-5">
          <FormInput
            id="callback_api_id"
            type="text"
            title={t('payment.callbackApiId')}
            bind:value={callbackApiID}
            error={formErrors.callback_api_id}
            ico="key"
          />
        </div>
        <p class="text-xs text-gray-500 mt-1">{t('payment.callbackIdentityHint')}</p>
        <div class="mt-5">
          <FormTextarea id="private_key" title={t('payment.privateKey')} bind:value={settings.private_key} rows={15} />
          {#if formErrors.private_key}
            <span class="error text-red-500">{formErrors.private_key}</span>
          {/if}
        </div>
      </dl>
    </div>

    <DrawerFooter onclose={close} submitLabel={t('common.save')} />
  </form>
</div>
