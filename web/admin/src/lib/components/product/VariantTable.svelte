<script lang="ts">
  import FormInput from '../form/Input.svelte'
  import FormButton from '../form/Button.svelte'
  import SvgIcon from '../SvgIcon.svelte'
  import type { ProductVariant, ProductOption } from '$lib/types/models'
  import { translate } from '$lib/i18n'
  import { formatPrice } from '$lib/utils'

  let t = $derived($translate)

  interface Props {
    variants: ProductVariant[]
    options: ProductOption[]
    basePrice: number
    currency: string
    onUpdate: (variants: ProductVariant[]) => void
    disabled?: boolean
  }

  let { variants, options, basePrice, currency, onUpdate, disabled = false }: Props = $props()

  let localVariants = $state<ProductVariant[]>([...variants])
  let optionsSignature = $state('')

  // Create signature from options (names + all values)
  function createOptionsSignature(opts: ProductOption[]): string {
    return JSON.stringify(
      opts.map(o => ({
        name: o.name,
        values: o.values.map(v => v.value)
      }))
    )
  }

  // Sync when variant count OR options change
  $effect(() => {
    const currentSignature = createOptionsSignature(options)
    const countChanged = variants.length !== localVariants.length
    const optionsChanged = currentSignature !== optionsSignature

    if (countChanged || optionsChanged) {
      optionsSignature = currentSignature
      localVariants = [...variants]
    }
  })

  function updateVariant(index: number, field: keyof ProductVariant, value: any) {
    localVariants = localVariants.map((v, i) => i === index ? { ...v, [field]: value } : v)
    onUpdate(localVariants)
  }

  function updateVariantSKU(index: number, event: Event) {
    const target = event.target as HTMLInputElement
    updateVariant(index, 'sku', target.value)
  }

  function updateVariantQuantity(index: number, event: Event) {
    const target = event.target as HTMLInputElement
    const qty = parseInt(target.value) || 0
    updateVariant(index, 'quantity', qty)
  }

  function updateVariantPrice(index: number, event: Event) {
    const target = event.target as HTMLInputElement
    const price = parseInt(target.value) || 0
    updateVariant(index, 'price_surcharge', price)
  }

  function toggleVariantActive(index: number) {
    updateVariant(index, 'active', !localVariants[index].active)
  }

  function getVariantLabel(variant: ProductVariant): string {
    return Object.entries(variant.option_values)
      .map(([key, value]) => `${key}: ${value}`)
      .join(', ')
  }

  function getTotalPrice(variant: ProductVariant): number {
    return basePrice + variant.price_surcharge
  }
</script>

<div class="overflow-x-auto">
  <table class="min-w-full divide-y divide-gray-200">
    <thead class="bg-gray-50">
      <tr>
        <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
          {t('products.variant')}
        </th>
        <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
          {t('products.sku')}
        </th>
        <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
          {t('products.priceSurcharge')}
        </th>
        <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
          {t('products.totalPrice')}
        </th>
        <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
          {t('products.quantity')}
        </th>
        <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
          {t('products.active')}
        </th>
      </tr>
    </thead>
    <tbody class="divide-y divide-gray-200 bg-white">
      {#each localVariants as variant, index}
        <tr class:bg-gray-50={!variant.active}>
          <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-900">
            {getVariantLabel(variant)}
          </td>
          <td class="whitespace-nowrap px-4 py-3 text-sm">
            <FormInput
              id="variant-{index}-sku"
              label="SKU"
              type="text"
              value={variant.sku || ''}
              oninput={(e) => updateVariantSKU(index, e)}
              placeholder="SKU"
              {disabled}
              compact
            />
          </td>
          <td class="whitespace-nowrap px-4 py-3 text-sm">
            <FormInput
              id="variant-{index}-price"
              label="Price Surcharge"
              type="number"
              value={String(variant.price_surcharge)}
              oninput={(e) => updateVariantPrice(index, e)}
              placeholder="0"
              {disabled}
              compact
            />
          </td>
          <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-700">
            {formatPrice(getTotalPrice(variant), currency)}
          </td>
          <td class="whitespace-nowrap px-4 py-3 text-sm">
            <FormInput
              id="variant-{index}-quantity"
              label="Quantity"
              type="number"
              value={String(variant.quantity)}
              oninput={(e) => updateVariantQuantity(index, e)}
              placeholder="0"
              {disabled}
              min="0"
              compact
            />
          </td>
          <td class="whitespace-nowrap px-4 py-3 text-sm">
            <button
              type="button"
              onclick={() => toggleVariantActive(index)}
              {disabled}
              class="disabled:opacity-50"
            >
              <SvgIcon
                name={variant.active ? 'eye' : 'eye-slash'}
                className="h-5 w-5 cursor-pointer {variant.active ? 'text-green-600' : 'text-gray-400'}"
              />
            </button>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>

  {#if localVariants.length === 0}
    <div class="py-8 text-center text-sm text-gray-500">
      {t('products.noVariants')}
    </div>
  {/if}

  {#if localVariants.length > 100}
    <div class="mt-2 text-sm text-red-600">
      {t('products.tooManyVariants')}
    </div>
  {/if}
</div>
