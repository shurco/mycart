<script lang="ts">
  import type { ProductOption, ProductVariant } from '$lib/types/models'
  import { translate } from '$lib/i18n'
  import { costFormat } from '$lib/utils/costFormat'

  let t = $derived($translate)

  interface Props {
    options: ProductOption[]
    variants: ProductVariant[]
    basePrice: number
    currency: string
    onVariantChange: (variant: ProductVariant | null) => void
  }

  let { options, variants, basePrice, currency, onVariantChange }: Props = $props()

  let selectedOptions = $state<Record<string, string>>({})
  let selectedVariant = $state<ProductVariant | null>(null)

  // Initialize with first available option for each
  $effect(() => {
    if (options.length > 0 && Object.keys(selectedOptions).length === 0) {
      const initial: Record<string, string> = {}
      for (const option of options) {
        if (option.values.length > 0) {
          initial[option.name] = option.values[0].value
        }
      }
      selectedOptions = initial
      updateSelectedVariant()
    }
  })

  function selectOption(optionName: string, value: string) {
    selectedOptions = { ...selectedOptions, [optionName]: value }
    updateSelectedVariant()
  }

  function updateSelectedVariant() {
    // Find matching variant
    const variant = variants.find((v) => {
      if (!v.active) return false
      return Object.entries(selectedOptions).every(
        ([key, value]) => v.option_values[key] === value
      )
    })

    selectedVariant = variant || null
    onVariantChange(selectedVariant)
  }

  function getVariantPrice(): number {
    if (!selectedVariant) return basePrice
    return basePrice + selectedVariant.price_surcharge
  }

  function isVariantAvailable(): boolean {
    return selectedVariant !== null && selectedVariant.quantity > 0
  }

  function isOptionValueAvailable(optionName: string, value: string): boolean {
    // Check if there's any active variant with this option value
    const testOptions = { ...selectedOptions, [optionName]: value }
    return variants.some((v) => {
      if (!v.active || v.quantity === 0) return false
      return Object.entries(testOptions).every(
        ([key, val]) => v.option_values[key] === val
      )
    })
  }

  function getOptionValueLabel(optionName: string, value: string): string {
    if (!isOptionValueAvailable(optionName, value)) {
      return `${value} (${t('product.outOfStock')})`
    }
    return value
  }
</script>

<div class="space-y-6">
  <!-- Option Selectors -->
  {#each options as option (option.id || option.name)}
    <div>
      <label class="mb-3 block text-sm font-black tracking-wider text-black uppercase">
        {option.name}
      </label>
      <div class="flex flex-wrap gap-2">
        {#each option.values as value (value.id || value.value)}
          {@const isSelected = selectedOptions[option.name] === value.value}
          {@const isAvailable = isOptionValueAvailable(option.name, value.value)}
          <button
            type="button"
            onclick={() => selectOption(option.name, value.value)}
            disabled={!isAvailable}
            class="brutal-btn px-6 py-3 text-base font-black tracking-wider uppercase transition-all duration-200 disabled:cursor-not-allowed disabled:opacity-50 {isSelected
              ? 'bg-yellow-300 text-black'
              : 'bg-white text-black hover:bg-gray-100'}"
          >
            {value.value}
            {#if !isAvailable}
              <span class="ml-1 text-xs">({t('product.outOfStock')})</span>
            {/if}
          </button>
        {/each}
      </div>
    </div>
  {/each}

  <!-- Selected Variant Info -->
  {#if selectedVariant}
    <div class="brutal-card bg-blue-50 p-4">
      <div class="flex items-baseline justify-between">
        <div>
          <p class="text-sm font-black tracking-wider text-gray-700 uppercase">
            {t('product.selectedVariant')}
          </p>
          <p class="mt-1 text-lg font-bold text-black">
            {Object.entries(selectedVariant.option_values)
              .map(([k, v]) => `${k}: ${v}`)
              .join(', ')}
          </p>
        </div>
        <div class="text-right">
          <p class="text-3xl font-black text-black">
            {costFormat(getVariantPrice())}
            <span class="text-lg uppercase">{currency}</span>
          </p>
          {#if selectedVariant.price_surcharge !== 0}
            <p class="text-xs text-gray-600">
              (+{costFormat(selectedVariant.price_surcharge)})
            </p>
          {/if}
        </div>
      </div>
    </div>
  {:else}
    <div class="brutal-card bg-red-50 p-4">
      <p class="font-black text-red-600 uppercase">
        {t('product.variantNotAvailable')}
      </p>
    </div>
  {/if}
</div>

<style>
  .brutal-btn {
    border: 3px solid black;
    box-shadow: 4px 4px 0px 0px rgba(0, 0, 0, 1);
  }

  .brutal-btn:hover:not(:disabled) {
    transform: translate(-2px, -2px);
    box-shadow: 6px 6px 0px 0px rgba(0, 0, 0, 1);
  }

  .brutal-card {
    border: 3px solid black;
  }
</style>
