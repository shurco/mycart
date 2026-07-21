<script lang="ts">
  import FormButton from '../form/Button.svelte'
  import FormToggle from '../form/Toggle.svelte'
  import SvgIcon from '../SvgIcon.svelte'
  import OptionEditor from './OptionEditor.svelte'
  import VariantTable from './VariantTable.svelte'
  import type { ProductOption, ProductVariant } from '$lib/types/models'
  import { translate } from '$lib/i18n'

  let t = $derived($translate)

  interface Props {
    hasVariants: boolean
    options: ProductOption[]
    variants: ProductVariant[]
    basePrice: number
    currency: string
    onUpdate: (data: { hasVariants: boolean; options: ProductOption[]; variants: ProductVariant[] }) => void
    disabled?: boolean
  }

  let { hasVariants, options, variants, basePrice, currency, onUpdate, disabled = false }: Props = $props()

  let localHasVariants = $state(hasVariants)
  let localOptions = $state<ProductOption[]>([...options])
  let localVariants = $state<ProductVariant[]>([...variants])

  // Sync options only (variants managed independently to preserve user edits)
  $effect(() => {
    localOptions = [...options]
  })

  function emitUpdate() {
    onUpdate({
      hasVariants: localHasVariants,
      options: localOptions,
      variants: localVariants
    })
  }

  function toggleHasVariants() {
    localHasVariants = !localHasVariants
    if (localHasVariants && localOptions.length === 0) {
      // Add default option when enabling variants
      addOption()
      return // addOption will call emitUpdate
    }
    emitUpdate()
  }

  function addOption() {
    if (localOptions.length >= 3) {
      return // Max 3 options
    }
    localOptions = [
      ...localOptions,
      {
        name: '',
        values: [{ value: '', position: 0 }],
        position: localOptions.length
      }
    ]
    generateVariants()
  }

  function updateOption(index: number, option: ProductOption) {
    // Check for duplicate option names (excluding current option)
    const duplicate = localOptions.find((opt, i) =>
      i !== index && opt.name.trim().toLowerCase() === option.name.trim().toLowerCase()
    )

    if (duplicate) {
      alert(`Option name "${option.name}" already exists. Please use a unique name.`)
      return
    }

    // Immutable update for Svelte 5 reactivity
    localOptions = localOptions.map((opt, i) => i === index ? option : opt)
    generateVariants()
  }

  function deleteOption(index: number) {
    localOptions = localOptions
      .filter((_, i) => i !== index)
      .map((opt, i) => ({ ...opt, position: i }))
    generateVariants()
  }

  function generateVariants() {
    // Generate cartesian product of all option values
    const validOptions = localOptions.filter(
      (opt) => opt.name.trim() && opt.values.some((v) => v.value.trim())
    )

    if (validOptions.length === 0) {
      localVariants = []
      emitUpdate()
      return
    }

    // Create cartesian product
    const combinations = cartesianProduct(validOptions)

    // Preserve existing variant data where possible
    const existingVariantsMap = new Map(
      localVariants.map((v) => [JSON.stringify(v.option_values), v])
    )

    localVariants = combinations.map((combo) => {
      const key = JSON.stringify(combo)
      const existing = existingVariantsMap.get(key)
      return (
        existing || {
          option_values: combo,
          sku: '',
          price_surcharge: 0,
          quantity: 0,
          active: true
        }
      )
    })

    emitUpdate()
  }

  function cartesianProduct(opts: ProductOption[]): Array<Record<string, string>> {
    if (opts.length === 0) return []

    let result: Array<Record<string, string>> = [{}]

    for (const option of opts) {
      const validValues = option.values.filter((v) => v.value.trim())
      if (validValues.length === 0) continue

      const newResult: Array<Record<string, string>> = []
      for (const existing of result) {
        for (const value of validValues) {
          newResult.push({
            ...existing,
            [option.name]: value.value
          })
        }
      }
      result = newResult
    }

    return result
  }

  function updateVariants(updatedVariants: ProductVariant[]) {
    localVariants = updatedVariants
    emitUpdate()
  }
</script>

<div class="mb-6">
  <div class="mb-4 flex items-center justify-between">
    <div>
      <h3 class="text-base font-semibold text-gray-900">{t('products.productVariants')}</h3>
      <p class="text-sm text-gray-500">{t('products.variantsDescription')}</p>
    </div>
    <FormToggle
      value={localHasVariants}
      onchange={toggleHasVariants}
      {disabled}
    />
  </div>

  {#if localHasVariants}
    <div class="space-y-6">
      <!-- Options Section -->
      <div>
        <div class="mb-4 flex items-center justify-between">
          <h4 class="text-sm font-medium text-gray-700">{t('products.options')}</h4>
          {#if localOptions.length < 3}
            <FormButton
              type="button"
              variant="secondary"
              size="sm"
              onclick={addOption}
              {disabled}
            >
              <SvgIcon name="plus" className="mr-1 h-4 w-4" />
              {t('products.addOption')}
            </FormButton>
          {/if}
        </div>

        {#if localOptions.length === 0}
          <div class="rounded-lg border-2 border-dashed border-gray-300 p-8 text-center">
            <p class="mb-4 text-sm text-gray-500">{t('products.noOptions')}</p>
            <FormButton
              type="button"
              variant="secondary"
              onclick={addOption}
              {disabled}
            >
              <SvgIcon name="plus" className="mr-2 h-4 w-4" />
              {t('products.addOption')}
            </FormButton>
          </div>
        {:else}
          {#each localOptions as option, index (option)}
            <OptionEditor
              {option}
              optionIndex={index}
              onUpdate={(opt) => updateOption(index, opt)}
              onDelete={() => deleteOption(index)}
              {disabled}
            />
          {/each}
        {/if}
      </div>

      <!-- Variants Section -->
      {#if localVariants.length > 0}
        <div>
          <div class="mb-4">
            <h4 class="text-sm font-medium text-gray-700">
              {t('products.generatedVariants')} ({localVariants.length})
            </h4>
            <p class="text-xs text-gray-500">{t('products.variantsAutoGenerated')}</p>
          </div>

          <VariantTable
            variants={localVariants}
            options={localOptions}
            {basePrice}
            {currency}
            onUpdate={updateVariants}
            {disabled}
          />
        </div>
      {/if}
    </div>
  {/if}
</div>
