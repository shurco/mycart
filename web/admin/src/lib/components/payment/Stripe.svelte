<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import FormButton from '../form/Button.svelte'
  import FormInput from '../form/Input.svelte'
  import FormToggle from '../form/Toggle.svelte'
  import { loadPaymentSettings, savePaymentSettings, togglePaymentActive } from '$lib/composables/usePaymentSettings'
  import { systemStore } from '$lib/stores/system'
  import { MIN_SECRET_KEY_LENGTH, ERROR_MESSAGES } from '$lib/constants/validation'
  import type { StripeSettings } from '$lib/types/models'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    onclose?: () => void
  }

  let { onclose }: Props = $props()

  let settings = $state<StripeSettings>({
    active: false,
    secret_key: ''
  })
  let formErrors = $state<Record<string, string>>({})
  let unsubscribe: (() => void) | null = null

  onMount(async () => {
    settings = await loadPaymentSettings<StripeSettings>('stripe', settings)

    unsubscribe = systemStore.subscribe((store) => {
      if (store.payments?.stripe !== undefined) {
        settings.active = store.payments.stripe
      }
    })
  })

  onDestroy(() => {
    unsubscribe?.()
  })

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault()
    formErrors = {}

    if (!settings.secret_key || settings.secret_key.length < MIN_SECRET_KEY_LENGTH) {
      formErrors.secret_key = ERROR_MESSAGES.SECRET_KEY_TOO_SHORT
      return
    }

    await savePaymentSettings('stripe', settings)
  }

  async function handleToggleActive() {
    const previousValue = settings.active
    const success = await togglePaymentActive('stripe', settings.active)

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
        <h1>Stripe</h1>
      </div>
      <div class="pt-1">
        <FormToggle
          id="stripe-active"
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
          id="secret_key"
          type="text"
          title={t('payment.secretKey')}
          bind:value={settings.secret_key}
          error={formErrors.secret_key}
          ico="key"
        />
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
