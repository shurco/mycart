<script lang="ts">
  import { DrawerFooter, DrawerHeader } from '$lib/components'
  import TruncationSettings from '$lib/components/TruncationSettings.svelte'
  import type { PaymentSettings, CurrencyTruncationSettings } from '$lib/types/models'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    payment: PaymentSettings
    onchange: (context: 'admin' | 'storefront', currency: string, settings: CurrencyTruncationSettings) => void
    onsubmit: () => void
    onclose: () => void
  }

  let { payment, onchange, onsubmit, onclose }: Props = $props()

  const sections = [
    { context: 'admin', title: 'settings.adminPanel' },
    { context: 'storefront', title: 'settings.storefront' }
  ] as const
</script>

<DrawerHeader title={t('settings.priceDisplaySettings')} />

<form onsubmit={(e) => { e.preventDefault(); onsubmit() }}>
  <div class="space-y-6">
    {#each sections as section (section.context)}
      <div class="space-y-3">
        <h3>{t(section.title)}</h3>
        <TruncationSettings
          currency={payment.currency}
          context={section.context}
          value={payment.truncation?.[section.context][payment.currency] || { mode: 'none' }}
          onChange={(settings) => onchange(section.context, payment.currency, settings)}
          numberFormat={payment.number_format}
        />
      </div>
    {/each}
  </div>

  <DrawerFooter {onclose} submitLabel={t('common.save')} />
</form>
