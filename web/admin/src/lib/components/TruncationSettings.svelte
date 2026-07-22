<script lang="ts">
  import FormSelect from './form/Select.svelte'
  import { getCurrencyPattern, getUnitLabel } from '$lib/config/currencyUnits'
  import { formatCurrencyWithTruncation } from '$lib/utils/currency'
  import { locale } from '$lib/i18n'
  import type { CurrencyTruncationSettings, TruncationSettings } from '$lib/types/models'

  interface Props {
    currency: string
    context: 'admin' | 'storefront'
    value: CurrencyTruncationSettings
    onChange: (settings: CurrencyTruncationSettings) => void
  }

  let { currency, context, value, onChange }: Props = $props()

  let currentLocale = $derived($locale)
  let pattern = $derived(getCurrencyPattern(currency))
  let showUnitDropdown = $derived(value.mode === 'fixed')

  // Generate unit options based on currency pattern
  let unitOptions = $derived(
    pattern.units.map(unit => getUnitLabel(unit.value, currency, currentLocale))
  )

  const modeOptions = ['none', 'fixed', 'flexible']

  function handleModeChange(newMode: string) {
    onChange({
      mode: newMode as 'none' | 'fixed' | 'flexible',
      fixed_unit: newMode === 'fixed' ? unitOptions[0] : undefined
    })
  }

  function handleUnitChange(newUnit: string) {
    onChange({
      ...value,
      fixed_unit: newUnit
    })
  }

  // Preview: show how 13520 would be formatted
  let previewAmount = $derived(() => {
    const tempSettings: TruncationSettings = {
      admin: context === 'admin' ? { [currency]: value } : {},
      storefront: context === 'storefront' ? { [currency]: value } : {}
    }
    return formatCurrencyWithTruncation(1352000, currency, context, tempSettings, currentLocale)
  })
</script>

<div class="mb-4 rounded border border-gray-300 p-4">
  <div class="mb-2 font-semibold text-gray-700">{currency}</div>

  <div class="space-y-3">
    <FormSelect
      id="{context}-{currency}-mode"
      title="Mode"
      options={modeOptions}
      value={value.mode}
      onchange={(e) => handleModeChange(e.currentTarget.value)}
    />

    {#if showUnitDropdown}
      <FormSelect
        id="{context}-{currency}-unit"
        title="Unit"
        options={unitOptions}
        value={value.fixed_unit || unitOptions[0]}
        onchange={(e) => handleUnitChange(e.currentTarget.value)}
      />
    {/if}

    {#if value.mode !== 'none'}
      <div class="text-sm text-gray-600">
        Preview: $13,520 → {previewAmount()}
      </div>
    {/if}
  </div>
</div>
