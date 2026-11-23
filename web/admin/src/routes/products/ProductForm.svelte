<script lang="ts">
  import { Editor, FormButton, FormInput, FormSelect, FormTextarea, SvgIcon } from '$lib/components'
  import VariantManager from '$lib/components/product/VariantManager.svelte'
  import type { Product, ProductOption, ProductVariant } from '$lib/types/models'
  import { translate } from '$lib/i18n'
  import { generateSlug } from '$lib/utils/slugGenerator'
  import ProductImages from './ProductImages.svelte'
  import RecordsSection from './RecordsSection.svelte'
  import RemovableRow from './RemovableRow.svelte'
  import { sanitizeAmountInput, type ProductFormData } from './productForm'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    mode: string
    formData: ProductFormData
    formErrors: Record<string, string>
    productImages: Product['images']
    /** The full product being edited; null while adding. */
    product: Product | null
    currency: string
    onsubmit: () => void
    onclose: () => void
    ondelete: () => void
    ontoggleactive: () => void
    onuploaded: () => void
  }

  let {
    mode,
    formData = $bindable(),
    formErrors = $bindable(),
    productImages = $bindable(),
    product,
    currency,
    onsubmit,
    onclose,
    ondelete,
    ontoggleactive,
    onuploaded
  }: Props = $props()

  // Display value for price (in regular units, not cents)
  let amountDisplay = $state(String(formData.amount))

  function handleAmountInput(event: Event) {
    const target = event.target as HTMLInputElement
    const value = sanitizeAmountInput(target.value)

    amountDisplay = value
    formData.amount = value
    target.value = value
  }

  async function handleNameBlur() {
    // Only generate if slug is empty
    if (formData.slug) return

    // Don't generate for empty name
    if (!formData.name.trim()) return

    const result = await generateSlug(formData.name)
    formData.slug = result.slug

    if (result.error) {
      formErrors.slug = t('products.slugGenerationError')
    } else {
      delete formErrors.slug
    }
  }

  function addMetadataRecord() {
    formData.metadata = [...(formData.metadata || []), { key: '', value: '' }]
  }

  function deleteMetadataRecord(index: number) {
    formData.metadata = (formData.metadata || []).filter((_, i) => i !== index)
  }

  function addAttributeRecord() {
    formData.attributes = [...(formData.attributes || []), '']
  }

  function deleteAttributeRecord(index: number) {
    formData.attributes = (formData.attributes || []).filter((_, i) => i !== index)
  }

  function handleVariantUpdate(data: { hasVariants: boolean; options: ProductOption[]; variants: ProductVariant[] }) {
    formData.has_variants = data.hasVariants
    formData.options = data.options
    formData.variants = data.variants
  }

  function handleEditorUpdate(value: string) {
    formData.description = value
  }
</script>

<div>
  <div class="pb-8">
    <div class="flex items-center">
      <div class="pr-3">
        <h1>{mode === 'add' ? t('products.addProduct') : `${t('products.editProduct')} ${formData.name || ''}`}</h1>
      </div>
      {#if mode === 'edit' && product}
        <div>
          <SvgIcon
            name={product.active ? 'eye' : 'eye-slash'}
            className="h-5 w-5 cursor-pointer"
            onclick={ontoggleactive}
            stroke="currentColor"
          />
        </div>
      {/if}
    </div>
  </div>

  <form onsubmit={(e) => {
    e.preventDefault();
    const form = e.currentTarget;
    if (!form.checkValidity()) {
      form.reportValidity();
      return;
    }
    onsubmit();
  }}>
    <div class="flow-root">
      <dl class="mx-auto -my-3 mt-2 mb-0 space-y-4 text-sm">
        <FormInput id="name" title={t('products.name')} bind:value={formData.name} error={formErrors.name} ico="at-symbol" onfocusout={handleNameBlur} />
        <div class="flex flex-row">
          <div class="pr-3">
            <FormInput
              id="amount"
              type="text"
              title={t('products.amount')}
              bind:value={amountDisplay}
              oninput={handleAmountInput}
              error={formErrors.amount}
              ico="money"
            />
          </div>
          <div class="mt-3">
            {currency}
            {#if parseFloat(amountDisplay) === 0}
              <span class="ml-2 font-bold text-green-600">free</span>
            {/if}
            <span class="ml-2 text-xs text-gray-500">{t('products.ifZeroPriceFree')}</span>
          </div>
        </div>

        {#if mode === 'add'}
          <div class="flex">
            <div class="grow pr-3">
              <FormInput
                id="slug"
                title={t('products.slug')}
                bind:value={formData.slug}
                error={formErrors.slug}
                ico="glob-alt"
                placeholder={t('products.slugPlaceholder')}
              />
            </div>
            <div class="grow">
              <FormSelect
                id="digital_type"
                title={t('products.digitalType')}
                options={['file', 'data', 'api']}
                bind:value={formData.digital.type}
                error={formErrors.digital_type}
                ico="cube"
              />
            </div>
          </div>
        {:else}
          <FormInput id="slug" title={t('products.slug')} bind:value={formData.slug} error={formErrors.slug} ico="glob-alt" placeholder={t('products.slugPlaceholder')} />
        {/if}

        <RecordsSection title={t('products.metadata')} addLabel={t('products.addMetadataRecord')} onadd={addMetadataRecord}>
          {#each formData.metadata || [] as metadata, index (index)}
            <RemovableRow onremove={() => deleteMetadataRecord(index)}>
              <div class="grow pr-3">
                <FormInput id="mtd-key-{index}" type="text" title={t('products.key')} bind:value={metadata.key} />
              </div>
              <div class="grow">
                <FormInput id="mtd-value-{index}" type="text" title={t('products.value')} bind:value={metadata.value} />
              </div>
            </RemovableRow>
          {/each}
        </RecordsSection>

        <RecordsSection title={t('products.attributes')} addLabel={t('products.addAttributeRecord')} onadd={addAttributeRecord}>
          {#each formData.attributes || [] as attribute, index (index)}
            <RemovableRow onremove={() => deleteAttributeRecord(index)}>
              <div class="grow">
                <FormInput id="atr-key-{index}" type="text" title="" bind:value={formData.attributes[index]} />
              </div>
            </RemovableRow>
          {/each}
        </RecordsSection>

        <hr />
        <VariantManager
          hasVariants={formData.has_variants || false}
          options={formData.options || []}
          variants={formData.variants || []}
          basePrice={typeof formData.amount === 'string' ? parseFloat(formData.amount) * 100 : (formData.amount || 0)}
          currency={currency}
          onUpdate={handleVariantUpdate}
        />

        {#if !formData.has_variants}
          <FormInput
            id="quantity"
            type="number"
            title={t('products.quantity')}
            bind:value={formData.quantity}
            error={formErrors.quantity}
            ico="cube"
            min="0"
            placeholder="0"
          />
        {/if}

        {#if mode === 'edit' && product}
          <hr />
          <p class="font-semibold">{t('products.images')}</p>
          <ProductImages productId={product.id} bind:images={productImages} {onuploaded} />
        {/if}

        <hr />
        <p class="font-semibold">{t('products.shortDescription')}</p>
        <FormTextarea id="brief" title={t('products.brief')} bind:value={formData.brief} />

        <hr />
        <p class="font-semibold">{t('products.description')}</p>
        <Editor
          bind:modelValue={formData.description}
          placeholder={t('products.typeDescriptionHere')}
          onupdateModelValue={handleEditorUpdate}
        />
      </dl>
    </div>

    <div class="pt-8">
      <div class="flex">
        <div class="flex-none">
          <FormButton type="submit" name={mode === 'add' ? t('common.add') : t('common.save')} />
          <FormButton type="button" name={t('common.close')} variant="secondary" onclick={onclose} />
        </div>
        <div class="grow"></div>
        {#if mode === 'edit' && product}
          <div class="mt-2 flex-none">
            <span
              onclick={ondelete}
              onkeydown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  ondelete()
                }
              }}
              role="button"
              tabindex="0"
              class="cursor-pointer text-red-700">{t('common.delete')}</span
            >
          </div>
        {/if}
      </div>
    </div>
  </form>
</div>
