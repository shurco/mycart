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

  // Generate combined options: none, fixed-K, fixed-M, fixed-B, flexible
  let combinedOptions = $derived(() => {
    const options = ['none']

    // Add fixed-{unit} options for each unit in the pattern
    pattern.units.forEach(unit => {
      const unitLabel = getUnitLabel(unit.value, currency, currentLocale)
      options.push(`fixed-${unitLabel}`)
    })

    options.push('flexible')
    return options
  })

  // Convert internal value to combined format for display
  let selectedOption = $derived(() => {
    if (value.mode === 'none') return 'none'
    if (value.mode === 'flexible') return 'flexible'
    if (value.mode === 'fixed' && value.fixed_unit) {
      return `fixed-${value.fixed_unit}`
    }
    return 'none'
  })

  // Parse combined option and update settings
  function handleChange(option: string) {
    if (option === 'none') {
      onChange({ mode: 'none' })
    } else if (option === 'flexible') {
      onChange({ mode: 'flexible' })
    } else if (option.startsWith('fixed-')) {
      const unit = option.substring(6) // Remove "fixed-" prefix
      onChange({ mode: 'fixed', fixed_unit: unit })
    }
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
      title="Price Display"
      options={combinedOptions()}
      value={selectedOption()}
      onchange={(e) => handleChange(e.currentTarget.value)}
    />

    {#if value.mode !== 'none'}
      <div class="text-sm text-gray-600">
        Preview: $13,520 → {previewAmount()}
      </div>
    {/if}
  </div>
</div>
