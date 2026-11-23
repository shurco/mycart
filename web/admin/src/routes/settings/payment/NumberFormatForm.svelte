<script lang="ts">
  import { DrawerFooter, DrawerHeader, FormSelect, FormToggle } from '$lib/components'
  import { formatCurrency } from '$lib/utils/currency'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    decimalPrecision: string
    showTrailingZeros: boolean
    currency: string
    onsubmit: () => void
    onclose: () => void
  }

  let {
    decimalPrecision = $bindable(),
    showTrailingZeros = $bindable(),
    currency,
    onsubmit,
    onclose
  }: Props = $props()

  function formatPreview(value: number): string {
    const nf = {
      decimal_precision: parseInt(decimalPrecision) as 0 | 1 | 2,
      show_trailing_zeros: showTrailingZeros
    }
    return formatCurrency(value, currency || 'USD', nf)
  }
</script>

<DrawerHeader title={t('settings.numberFormatting')} />

<form onsubmit={(e) => { e.preventDefault(); onsubmit() }}>
  <div class="flow-root">
    <dl class="mx-auto -my-3 mt-2 mb-0 space-y-4 text-sm">
      <FormSelect
        id="decimal-precision"
        title={t('settings.decimalPrecision')}
        options={['0', '1', '2']}
        bind:value={decimalPrecision}
        ico="hash"
      />
    </dl>

    <div class="mt-2 mb-4 flex items-center justify-between gap-4">
      <div>
        <h3>{t('settings.showTrailingZeros')}</h3>
        <p class="text-sm text-gray-500">{t('settings.showTrailingZerosDesc')}</p>
      </div>
      <FormToggle id="trailing-zeros" bind:value={showTrailingZeros} />
    </div>

    <div class="mt-3 text-sm text-gray-600">
      <div>{t('settings.preview')}: 1.00 → {formatPreview(1.0)}</div>
      <div>{t('settings.preview')}: 1.23 → {formatPreview(1.23)}</div>
    </div>
  </div>

  <DrawerFooter {onclose} submitLabel={t('common.save')} />
</form>
