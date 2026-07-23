<script lang="ts">
  import FormSelect from './form/Select.svelte'
  import { getCurrencyPattern, getUnitLabel } from '$lib/config/currencyUnits'
  import { formatCurrency, formatCurrencyWithTruncation } from '$lib/utils/currency'
  import { locale } from '$lib/i18n'
  import type { CurrencyTruncationSettings, TruncationSettings, NumberFormatSettings } from '$lib/types/models'

  interface Props {
    currency: string
    context: 'admin' | 'storefront'
    value: CurrencyTruncationSettings
    onChange: (settings: CurrencyTruncationSettings) => void
    numberFormat?: NumberFormatSettings
  }

  let { currency, context, value, onChange, numberFormat }: Props = $props()

  let currentLocale = $derived($locale)
  let pattern = $derived(getCurrencyPattern(currency))

  // Local state for the selected option string
  let selectedOptionValue = $state('')

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

  // Initialize selectedOptionValue from value prop
  // Only runs when selectedOption() changes (not when selectedOptionValue changes)
  $effect(() => {
    selectedOptionValue = selectedOption()
  })

  // Watch for changes to selectedOptionValue and call onChange
  $effect(() => {
    // Skip if unchanged or empty
    if (!selectedOptionValue || selectedOptionValue === selectedOption()) {
      return
    }

    handleChange(selectedOptionValue)
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
  let previewBefore = $derived(formatCurrency(13520, currency, numberFormat))
  let previewAfter = $derived(() => {
    const tempSettings: TruncationSettings = {
      admin: context === 'admin' ? { [currency]: value } : {},
      storefront: context === 'storefront' ? { [currency]: value } : {}
    }
    return formatCurrencyWithTruncation(1352000, currency, context, tempSettings, currentLocale, numberFormat)
  })
</script>

<div class="mb-4 max-w-2xl">
  <FormSelect
    id="{context}-{currency}-mode"
    title="Price Display"
    options={combinedOptions()}
    bind:value={selectedOptionValue}
  />

  {#if value.mode !== 'none'}
    <div class="mt-2 text-sm text-gray-600">
      Preview: {previewBefore} → {previewAfter()}
    </div>
  {/if}
</div>
