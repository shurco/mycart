<script lang="ts">
  import { onMount } from 'svelte'
  import Main from '$lib/layouts/Main.svelte'
  import Drawer from '$lib/components/Drawer.svelte'
  import ProductView from '$lib/components/product/View.svelte'
  import ProductSeo from '$lib/components/product/Seo.svelte'
  import ProductDigital from '$lib/components/product/Digital.svelte'
  import FormButton from '$lib/components/form/Button.svelte'
  import FormInput from '$lib/components/form/Input.svelte'
  import FormSelect from '$lib/components/form/Select.svelte'
  import FormTextarea from '$lib/components/form/Textarea.svelte'
  import Editor from '$lib/components/Editor.svelte'
  import Upload from '$lib/components/form/Upload.svelte'
  import SvgIcon from '$lib/components/SvgIcon.svelte'
  import Pagination from '$lib/components/Pagination.svelte'
  import { loadData, saveData, deleteData, toggleActive as toggleActiveApi } from '$lib/utils/apiHelpers'
  import { costFormat, formatPrice, formatDate, sortByDate, confirmDelete, showMessage } from '$lib/utils'
  import { formatCurrencyWithTruncation } from '$lib/utils/currency'
  import { apiDelete, apiUpdate } from '$lib/utils/api'
  import { validators, validateFields } from '$lib/utils/validation'
  import { MIN_NAME_LENGTH, MIN_SLUG_LENGTH, ERROR_MESSAGES } from '$lib/constants/validation'
  import { CENTS_PER_UNIT, DEFAULT_AMOUNT } from '$lib/constants/pricing'
  import { DEFAULT_PAGE_SIZE } from '$lib/constants/pagination'
  import { DRAWER_CLOSE_DELAY_MS } from '$lib/constants/ui'
  import type { Product } from '$lib/types/models'
  import { paymentSettingsStore } from '$lib/stores/payment'
  import { translate, locale } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)
  let currentLocale = $derived($locale)
  let paymentSettings = $derived($paymentSettingsStore)

  interface ProductsResponse {
    products: Product[]
    currency: string
    total: number
  }

  interface DrawerProduct {
    product: Product
    index: number
    currency?: string
  }

  let products = $state<Product[]>([])
  let currency = $state('')
  let loading = $state(true)
  let drawerOpen = $state(false)
  let drawerMode = $state<'view' | 'add' | 'edit' | 'seo' | 'digital'>('view')
  let drawerProduct = $state<DrawerProduct | null>(null)
  let drawerIndex = $state(-1)
  let currentPage = $state(1)
  let limit = $state(DEFAULT_PAGE_SIZE)
  let total = $state(0)

  interface ProductFormData {
    name: string
    slug: string
    brief: string
    description: string
    amount: string | number
    active: boolean
    metadata: Array<{ key: string; value: string }>
    attributes: string[]
    digital: {
      type: '' | 'file' | 'data' | 'api'
    }
  }

  let formData = $state<ProductFormData>({
    name: '',
    slug: '',
    brief: '',
    description: '',
    amount: DEFAULT_AMOUNT,
    active: true,
    metadata: [],
    attributes: [],
    digital: {
      type: ''
    }
  })
  let formErrors = $state<Record<string, string>>({})
  let productImages = $state<Product['images']>([])
  let fullProductData = $state<Product | null>(null)

  // Display value for price (in regular units, not cents)
  let amountDisplay = $state('0')

  function handleAmountInput(event: Event) {
    const target = event.target as HTMLInputElement
    let value = target.value

    // Remove invalid characters (keep only digits and dot)
    value = value.replace(/[^0-9.]/g, '')

    // Add leading zero if starts with dot
    if (value.startsWith('.')) {
      value = '0' + value
    }

    // Limit to single dot
    const parts = value.split('.')
    if (parts.length > 2) {
      value = parts[0] + '.' + parts.slice(1).join('')
    }

    amountDisplay = value
    formData.amount = value
    target.value = value
  }

  onMount(async () => {
    await loadProducts()
  })

  async function loadProducts(page = currentPage) {
    loading = true
    currentPage = page
    const result = await loadData<ProductsResponse>(
      `/api/_/products?page=${page}&limit=${limit}`,
      t('products.failedToLoad')
    )
    if (result) {
      products = sortByDate(result.products || [])
      currency = result.currency || ''
      total = result.total || 0
    }
    loading = false
  }

  function handlePageChange(page: number) {
    loadProducts(page)
  }

  function openView(product: Product, index: number) {
    drawerProduct = { product, index, currency }
    drawerMode = 'view'
    drawerOpen = true
  }

  async function openAdd() {
    formData = {
      name: '',
      slug: '',
      brief: '',
      description: '',
      amount: '0',
      active: true,
      metadata: [],
      attributes: [],
      digital: {
        type: ''
      }
    }
    amountDisplay = '0'
    productImages = []
    fullProductData = null
    formErrors = {}
    drawerMode = 'add'
    drawerOpen = true
  }

  async function openEdit(product: Product, index: number) {
    drawerProduct = { product, index }
    formErrors = {}
    drawerMode = 'edit'

    const result = await loadData<Product>(`/api/_/products/${product.id}`, 'Failed to load product')
    if (result) {
      fullProductData = result
      // Convert price from cents to regular number for form
      const amountValue = typeof result.amount === 'string' ? parseFloat(result.amount) : (result.amount || 0)
      const amountInUnits = amountValue / CENTS_PER_UNIT
      const amountStr = amountInUnits.toFixed(2)
      formData = {
        name: result.name || '',
        slug: result.slug || '',
        brief: result.brief || '',
        description: result.description || '',
        amount: amountStr,
        active: result.active !== undefined ? result.active : true,
        metadata: result.metadata || [],
        attributes: result.attributes || [],
        digital: result.digital || { type: '' }
      }
      amountDisplay = amountStr
      productImages = result.images || []
      drawerOpen = true
    }
  }

  function openSeo(product: Product, index: number) {
    drawerProduct = { product, index, currency }
    drawerMode = 'seo'
    drawerOpen = true
  }

  function openDigital(product: Product, index: number) {
    drawerProduct = { product, index, currency }
    drawerMode = 'digital'
    drawerOpen = true
  }

  async function handleDigitalContentUpdate() {
    await loadProducts()
  }

  function closeDrawer() {
    if (drawerOpen) {
      drawerOpen = false
      setTimeout(() => {
        drawerProduct = null
        drawerMode = 'view'
      }, DRAWER_CLOSE_DELAY_MS)
    }
  }

  function updateProductInList(product: Product) {
    const index = products.findIndex((p) => p.id === product.id)
    if (index !== -1) {
      // Preserve digital.filled from original product if it wasn't in response
      const originalProduct = products[index]
      if (originalProduct.digital?.filled !== undefined && (!product.digital || product.digital.filled === undefined)) {
        if (!product.digital) {
          product.digital = { type: originalProduct.digital?.type || '', filled: originalProduct.digital.filled }
        } else {
          product.digital = { ...product.digital, filled: originalProduct.digital.filled }
        }
      }
      products[index] = product
    }
    if (drawerProduct?.product.id === product.id) {
      // Preserve digital.filled from original product if it wasn't in response
      if (drawerProduct.product.digital?.filled !== undefined && (!product.digital || product.digital.filled === undefined)) {
        if (!product.digital) {
          product.digital = { type: drawerProduct.product.digital?.type || '', filled: drawerProduct.product.digital.filled }
        } else {
          product.digital = { ...product.digital, filled: drawerProduct.product.digital.filled }
        }
      }
      drawerProduct.product = product
    }
    if (fullProductData?.id === product.id) {
      fullProductData = product
      formData.active = product.active
    }
  }

  function validateProductForm(): number | null {
    formErrors = validateFields(formData, [
      { field: 'name', ...validators.minLength(MIN_NAME_LENGTH, ERROR_MESSAGES.NAME_TOO_SHORT) },
      { field: 'slug', ...validators.minLength(MIN_SLUG_LENGTH, ERROR_MESSAGES.SLUG_TOO_SHORT) }
    ])

    const amountValue = typeof formData.amount === 'string' ? parseFloat(formData.amount) : formData.amount
    if (isNaN(amountValue) || amountValue < 0) {
      formErrors.amount = ERROR_MESSAGES.AMOUNT_INVALID
      return null
    }

    if (drawerMode === 'add' && (!formData.digital?.type || formData.digital.type.trim() === '')) {
      formErrors.digital_type = ERROR_MESSAGES.DIGITAL_TYPE_REQUIRED
    }

    return Object.keys(formErrors).length > 0 ? null : amountValue
  }

  function prepareSubmitData(amountValue: number): Partial<Product> {
    const amountInCents = Math.round(amountValue * CENTS_PER_UNIT)
    return {
      ...formData,
      amount: amountInCents
    }
  }

  async function handleSaveResult(result: Product | null, isUpdate: boolean) {
    if (!result) return

    if (isUpdate) {
      updateProductInList(result)
    } else {
      products = [result, ...products]
      total++
    }
    closeDrawer()
  }

  async function handleSubmit() {
    const amountValue = validateProductForm()
    if (amountValue === null) return

    const isUpdate = drawerMode === 'edit' && drawerProduct !== null
    const url = isUpdate ? `/api/_/products/${drawerProduct!.product.id}` : '/api/_/products'
    const submitData = prepareSubmitData(amountValue)

    const result = await saveData<Product>(url, submitData, isUpdate, t('products.failedToSave'), t('products.failedToSave'))
    await handleSaveResult(result, isUpdate)
  }

  async function handleDeleteProduct() {
    if (!fullProductData || !confirmDelete('product', fullProductData.name)) {
      return
    }

    const success = await deleteData(
      `/api/_/products/${fullProductData.id}`,
      t('products.failedToDelete'),
      t('products.failedToDelete')
    )
    if (success) {
      await loadProducts()
      closeDrawer()
    }
  }

  async function toggleActiveInEdit() {
    if (!fullProductData) return

    const newActive = !fullProductData.active
    const updatedProduct = { ...fullProductData, active: newActive }
    updateProductInList(updatedProduct)

    const result = await toggleActiveApi(
      `/api/_/products/${fullProductData.id}/active`,
      t('products.productStatusUpdated'),
      t('products.failedToUpdateProduct')
    )

    if (result === null) {
      updateProductInList(fullProductData)
    } else {
      const serverProduct = await loadData<Product>(`/api/_/products/${fullProductData.id}`, 'Failed to load product')
      if (serverProduct) {
        updateProductInList(serverProduct)
      }
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

  async function deleteProductImage(index: number) {
    if (!fullProductData || !productImages || !productImages[index]) return

    try {
      const imageId = productImages[index].id
      const res = await apiDelete(`/api/_/products/${fullProductData.id}/image/${imageId}`)
      if (res.success && productImages) {
        productImages = productImages.filter((_, i) => i !== index)
        showMessage(t('products.imageDeleted'), 'connextSuccess')
      } else {
        showMessage(res.message || t('products.failedToDeleteImage'), 'connextError')
      }
    } catch (error) {
      showMessage(t('common.networkError'), 'connextError')
    }
  }

  async function handleDelete(product: Product, index: number) {
    if (!confirmDelete('product', product.name)) {
      return
    }

    const success = await deleteData(`/api/_/products/${product.id}`, t('products.failedToDelete'), t('products.failedToDelete'))
    if (success) {
      await loadProducts()
    }
  }

  async function toggleActive(product: Product, index: number) {
    const originalActive = product.active
    const newActive = !product.active
    
    // Optimistic update - update directly instead of map
    products[index] = { ...products[index], active: newActive }

    try {
      const res = await apiUpdate(`/api/_/products/${product.id}/active`, {})
      if (!res.success) {
        // Revert on failure
        products[index] = { ...products[index], active: originalActive }
        showMessage(res.message || t('products.failedToUpdateProduct'), 'connextError')
      } else {
        showMessage(t('products.productStatusUpdated') || res.message, 'connextSuccess')
      }
    } catch (error) {
      // Revert on error
      products[index] = { ...products[index], active: originalActive }
      showMessage(t('common.networkError'), 'connextError')
    }
  }

  function digitalTypeIco(type: string | undefined): string {
    if (!type) return 'cube-transparent'
    switch (type) {
      case 'file':
        return 'paper-clip'
      case 'data':
        return 'queue-list'
      default:
        return 'cube-transparent'
    }
  }

  async function updateActive(index: number) {
    await toggleActive(products[index], index)
  }

  function handleEditorUpdate(value: string) {
    formData.description = value
  }

  function handleUpload(result: any) {
    if (result?.success) {
      showMessage('File uploaded', 'connextSuccess')
      if (result?.result && fullProductData) {
        productImages = [...(productImages || []), result.result]
      }
      loadProducts()
    }
  }
</script>

<Main>
  <div class="mb-5 flex items-center justify-between">
    <h1>{t('products.title')}</h1>
    <FormButton name={t('products.addProduct')} color="green" ico="plus" onclick={openAdd} />
  </div>

  {#if loading}
    <div class="py-8 text-center">{t('common.loading')}</div>
  {:else if products.length === 0}
    <div class="py-8 text-center text-gray-500">{t('products.noProducts')}</div>
  {:else}
    <table>
      <thead>
        <tr>
          <th class="w-28"></th>
          <th>{t('products.name')}</th>
          <th class="w-32">{t('products.slug')}</th>
          <th class="w-32">{t('products.price')}</th>
          <th class="w-12 px-4 py-2">
            <SvgIcon name="cube" className="h-5 w-5" stroke="currentColor" />
          </th>
          <th class="w-24 px-4 py-2"></th>
        </tr>
      </thead>
      <tbody>
        {#each products as product, index (product.id)}
          <tr class:opacity-30={!product.active} data-testid="product-row">
            <td>
              {#if product.images && product.images.length > 0}
                <a href="/uploads/{product.images[0].name}.{product.images[0].ext}" target="_blank">
                  <img
                    style="width: 100%; max-width: 80px"
                    src="/uploads/{product.images[0].name}_sm.{product.images[0].ext}"
                    alt={product.name}
                    loading="lazy"
                  />
                </a>
              {:else}
                <img style="width: 100%; max-width: 80px" src="/assets/img/noimage.png" alt="" loading="lazy" />
              {/if}
            </td>
            <td onclick={() => openView(product, index)}>
              <div class="font-bold">{product.name}</div>
              {#if product.brief}
                <span class="hidden text-gray-400 xl:block">{product.brief}</span>
              {/if}
            </td>
            <td>
              {#if product.active}
                <a href="/products/{product.slug}" target="_blank">{product.slug}</a>
              {:else}
                <span>{product.slug}</span>
              {/if}
            </td>
            <td onclick={() => openView(product, index)}>
              {#if !product.amount || parseFloat(String(product.amount)) === 0}
                <span class="font-bold text-green-600">free</span>
              {:else}
                {formatCurrencyWithTruncation(
                  product.amount,
                  currency || 'USD',
                  'admin',
                  paymentSettings?.truncation,
                  currentLocale
                )}
              {/if}
            </td>
            <td class="px-4 py-2">
              {#if product.digital && product.digital.type}
                <SvgIcon
                  name={digitalTypeIco(product.digital.type)}
                  className="h-5 w-5 cursor-pointer {product.digital.filled === true ? 'text-black' : 'text-red-600'}"
                  onclick={() => openDigital(product, index)}
                  stroke="currentColor"
                />
              {/if}
            </td>
            <td class="px-4 py-2">
              <div class="flex">
                <div class="pr-3">
                  <SvgIcon
                    name="pencil-square"
                    className="h-5 w-5 cursor-pointer"
                    onclick={() => openEdit(product, index)}
                    stroke="currentColor"
                  />
                </div>
                <div class="pr-3">
                  <SvgIcon
                    name="rocket"
                    className="h-5 w-5 cursor-pointer"
                    onclick={() => openSeo(product, index)}
                    stroke="currentColor"
                  />
                </div>
                <div>
                  <SvgIcon
                    name={product.active ? 'eye' : 'eye-slash'}
                    className="h-5 w-5 cursor-pointer"
                    onclick={() => toggleActive(product, index)}
                    stroke="currentColor"
                  />
                </div>
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>

    {#if total > 0}
      <Pagination
        currentPage={currentPage}
        totalPages={Math.ceil(total / limit)}
        onPageChange={handlePageChange}
      />
    {/if}
    {/if}
</Main>

{#if drawerOpen}
  <Drawer isOpen={drawerOpen} onclose={closeDrawer} maxWidth="710px">
    {#if drawerMode === 'view' && drawerProduct}
      <ProductView drawer={drawerProduct} {updateActive} onclose={closeDrawer} />
    {:else if drawerMode === 'seo' && drawerProduct}
      <ProductSeo drawer={drawerProduct} onclose={closeDrawer} />
    {:else if drawerMode === 'digital' && drawerProduct}
      <ProductDigital drawer={drawerProduct} onContentUpdate={handleDigitalContentUpdate} onclose={closeDrawer} />
    {:else}
      <div>
        <div class="pb-8">
          <div class="flex items-center">
            <div class="pr-3">
              <h1>{drawerMode === 'add' ? t('products.addProduct') : `${t('products.editProduct')} ${formData.name || ''}`}</h1>
            </div>
            {#if drawerMode === 'edit' && fullProductData}
              <div>
                <SvgIcon
                  name={fullProductData.active ? 'eye' : 'eye-slash'}
                  className="h-5 w-5 cursor-pointer"
                  onclick={toggleActiveInEdit}
                  stroke="currentColor"
                />
              </div>
            {/if}
          </div>
        </div>

        <form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }}>
          <div class="flow-root">
            <dl class="mx-auto -my-3 mt-2 mb-0 space-y-4 text-sm">
              <FormInput id="name" title={t('products.name')} bind:value={formData.name} error={formErrors.name} ico="at-symbol" />
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

              {#if drawerMode === 'add'}
                <div class="flex">
                  <div class="grow pr-3">
                    <FormInput
                      id="slug"
                      title={t('products.slug')}
                      bind:value={formData.slug}
                      error={formErrors.slug}
                      ico="glob-alt"
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
                <FormInput id="slug" title={t('products.slug')} bind:value={formData.slug} error={formErrors.slug} ico="glob-alt" />
              {/if}

              <hr />
              <p class="font-semibold">{t('products.metadata')}</p>
              {#each formData.metadata || [] as metadata, index (index)}
                <div class="flex">
                  <div class="grow pr-3">
                    <FormInput id="mtd-key-{index}" type="text" title={t('products.key')} bind:value={metadata.key} />
                  </div>
                  <div class="grow">
                    <FormInput id="mtd-value-{index}" type="text" title={t('products.value')} bind:value={metadata.value} />
                  </div>
                  <div
                    class="flex-none cursor-pointer pt-3 pl-3"
                    role="button"
                    tabindex="0"
                    onclick={() => deleteMetadataRecord(index)}
                    onkeydown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault()
                        deleteMetadataRecord(index)
                      }
                    }}
                  >
                    <SvgIcon name="trash" className="h-5 w-5" stroke="currentColor" />
                  </div>
                </div>
              {/each}
              <div class="flex">
                <div class="grow"></div>
                <div class="mt-2 flex-none">
                  <button
                    type="button"
                    class="shrink-0 rounded-lg bg-gray-200 p-2 text-sm font-medium text-gray-700"
                    onclick={addMetadataRecord}
                  >
                    {t('products.addMetadataRecord')}
                  </button>
                </div>
              </div>

              <hr />
              <p class="font-semibold">{t('products.attributes')}</p>
              {#each formData.attributes || [] as attribute, index (index)}
                <div class="flex">
                  <div class="grow">
                    <FormInput id="atr-key-{index}" type="text" title="" bind:value={formData.attributes[index]} />
                  </div>
                  <div
                    class="flex-none cursor-pointer pt-3 pl-3"
                    role="button"
                    tabindex="0"
                    onclick={() => deleteAttributeRecord(index)}
                    onkeydown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault()
                        deleteAttributeRecord(index)
                      }
                    }}
                  >
                    <SvgIcon name="trash" className="h-5 w-5" stroke="currentColor" />
                  </div>
                </div>
              {/each}
              <div class="flex">
                <div class="grow"></div>
                <div class="mt-2 flex-none">
                  <button
                    type="button"
                    class="shrink-0 rounded-lg bg-gray-200 p-2 text-sm font-medium text-gray-700"
                    onclick={addAttributeRecord}
                  >
                    {t('products.addAttributeRecord')}
                  </button>
                </div>
              </div>

              {#if drawerMode === 'edit' && fullProductData}
                <hr />
                <p class="font-semibold">{t('products.images')}</p>
                {#if productImages && productImages.length > 0}
                  <div class="grid grid-cols-4 content-start gap-4">
                    {#each productImages as image, index (image.id || index)}
                      <div class="relative" style="width: 100%; max-width: 150px">
                        <a href="/uploads/{image.name}.{image.ext}" target="_blank">
                          <img src="/uploads/{image.name}_sm.{image.ext}" alt="" />
                        </a>
                        <div
                          role="button"
                          tabindex="0"
                          class="absolute end-4 top-4 cursor-pointer bg-white p-2"
                          onclick={() => deleteProductImage(index)}
                          onkeydown={(e) => {
                            if (e.key === 'Enter' || e.key === ' ') {
                              e.preventDefault()
                              deleteProductImage(index)
                            }
                          }}
                        >
                          <SvgIcon name="trash" className="h-5 w-5" stroke="currentColor" />
                        </div>
                      </div>
                    {/each}
                  </div>
                {/if}
                <Upload
                  section="image"
                  productId={fullProductData.id}
                  accept=".jpg,.jpeg,.png"
                  onadded={handleUpload}
                />
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
                <FormButton type="submit" name={drawerMode === 'add' ? t('common.add') : t('common.save')} color="green" />
                <FormButton type="button" name={t('common.close')} color="gray" onclick={closeDrawer} />
              </div>
              <div class="grow"></div>
              {#if drawerMode === 'edit' && fullProductData}
                <div class="mt-2 flex-none">
                  <span
                    onclick={handleDeleteProduct}
                    onkeydown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault()
                        handleDeleteProduct()
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
    {/if}
  </Drawer>
{/if}
