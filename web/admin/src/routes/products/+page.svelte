<script lang="ts">
  import { onMount } from 'svelte'
  import Main from '$lib/layouts/Main.svelte'
  import { Drawer, FormButton, Pagination, ProductView } from '$lib/components'
  import ProductSeo from '$lib/components/product/Seo.svelte'
  import ProductDigital from '$lib/components/product/Digital.svelte'
  import CsvImportExport from '$lib/components/product/CsvImportExport.svelte'
  import { loadData, saveData, deleteData, toggleActive as toggleActiveApi } from '$lib/utils/apiHelpers'
  import { confirmDelete } from '$lib/utils'
  import { DRAWER_CLOSE_DELAY_MS } from '$lib/constants/ui'
  import { createDelayedReset } from '$lib/utils/delayedReset'
  import type { Product } from '$lib/types/models'
  import { translate } from '$lib/i18n'
  import ProductForm from './ProductForm.svelte'
  import { ProductsList } from './productsList.svelte'
  import ProductsTable from './ProductsTable.svelte'
  import { preserveDigitalFilled } from './productDigital'
  import {
    convertProductToFormData,
    emptyProductForm,
    prepareSubmitData,
    validateProductForm,
    type ProductFormData
  } from './productForm'

  // Reactive translation function
  let t = $derived($translate)

  interface DrawerProduct {
    product: Product
    index: number
    currency?: string
  }

  const list = new ProductsList()

  let drawerOpen = $state(false)
  let drawerMode = $state<'view' | 'add' | 'edit' | 'seo' | 'digital' | 'csv'>('view')
  let drawerProduct = $state<DrawerProduct | null>(null)

  let formData = $state<ProductFormData>(emptyProductForm())
  let formErrors = $state<Record<string, string>>({})
  let productImages = $state<Product['images']>([])
  let fullProductData = $state<Product | null>(null)

  onMount(async () => {
    await list.load()
  })

  function handlePageChange(page: number) {
    list.load(page)
  }

  function openView(product: Product, index: number) {
    drawerProduct = { product, index, currency: list.currency }
    drawerMode = 'view'
    drawerOpen = true
  }


  async function openAdd() {
    formData = emptyProductForm()
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
      formData = convertProductToFormData(result)
      productImages = result.images || []
      drawerOpen = true
    }
  }

  function openSeo(product: Product, index: number) {
    drawerProduct = { product, index, currency: list.currency }
    drawerMode = 'seo'
    drawerOpen = true
  }

  function openDigital(product: Product, index: number) {
    drawerProduct = { product, index, currency: list.currency }
    drawerMode = 'digital'
    drawerOpen = true
  }

  async function handleDigitalContentUpdate() {
    await list.load()
  }

  function openCsv() {
    drawerMode = 'csv'
    drawerOpen = true
  }

  async function handleCsvImportComplete() {
    await list.load()
    closeDrawer()
  }

  const resetDrawer = createDelayedReset(DRAWER_CLOSE_DELAY_MS)

  function closeDrawer() {
    if (drawerOpen) {
      drawerOpen = false
      resetDrawer(() => {
        drawerProduct = null
        drawerMode = 'view'
      })
    }
  }

  function updateProductInList(product: Product) {
    const index = list.products.findIndex((p) => p.id === product.id)
    if (index !== -1) {
      preserveDigitalFilled(product, list.products[index])
      list.products[index] = product
    }
    if (drawerProduct?.product.id === product.id) {
      preserveDigitalFilled(product, drawerProduct.product)
      drawerProduct.product = product
    }
    if (fullProductData?.id === product.id) {
      fullProductData = product
      formData.active = product.active
    }
  }

  async function handleSubmit() {
    // Validate form
    formErrors = validateProductForm(formData, drawerMode)
    if (Object.keys(formErrors).length > 0) {
      return
    }

    // Determine mode and URL
    const isUpdate = drawerMode === 'edit' && drawerProduct !== null
    const url = isUpdate ? `/api/_/products/${drawerProduct!.product.id}` : '/api/_/products'

    // Prepare data for submission
    const submitData = prepareSubmitData(formData)

    console.log('=== PRODUCT SAVE DEBUG ===')
    console.log('has_variants:', submitData.has_variants)
    console.log('options:', JSON.stringify(submitData.options, null, 2))
    console.log('variants:', JSON.stringify(submitData.variants, null, 2))
    console.log('Full submitData:', submitData)

    // Submit to API
    const result = await saveData<Product>(
      url,
      submitData,
      isUpdate,
      isUpdate ? t('products.updated') : t('products.created'),
      t('products.failedToSave')
    )

    console.log('=== SAVE RESULT ===')
    console.log('Result has_variants:', result?.has_variants)
    console.log('Result options:', result?.options)
    console.log('Result variants:', result?.variants)

    // Handle response
    if (result) {
      if (isUpdate) {
        updateProductInList(result)
      } else {
        await list.load()
      }
      closeDrawer()
    } else if (isUpdate && drawerProduct) {
      const updatedProduct = await loadData<Product>(
        `/api/_/products/${drawerProduct.product.id}`,
        'Failed to load product'
      )
      if (updatedProduct) {
        updateProductInList(updatedProduct)
      }
    }
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
      await list.load()
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

  async function updateActive(index: number) {
    await list.toggleActive(list.products[index], index)
  }
</script>

<Main>
  <div class="mb-5 flex items-center justify-between">
    <h1>{t('products.title')}</h1>
    <div class="flex gap-2">
      <FormButton name={`${t('products.csv.import')} / ${t('products.csv.export')}`} variant="secondary" ico="arrow-path" onclick={openCsv} />
      <FormButton name={t('products.addProduct')} ico="plus" onclick={openAdd} />
    </div>
  </div>

  {#if list.loading}
    <div class="py-8 text-center">{t('common.loading')} [PAGE {list.currentPage}] [FIX v2.0]</div>
  {:else if list.products.length === 0}
    <div class="py-8 text-center text-gray-500">{t('products.noProducts')}</div>
  {:else}
    <ProductsTable
      products={list.products}
      currency={list.currency}
      onview={openView}
      onedit={openEdit}
      onseo={openSeo}
      ondigital={openDigital}
      ontoggle={list.toggleActive}
    />

    {#if list.total > 0}
      <Pagination
        currentPage={list.currentPage}
        totalPages={Math.ceil(list.total / list.limit)}
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
    {:else if drawerMode === 'csv'}
      <div class="pb-8">
        <h1>{t('products.csv.importProducts')} / {t('products.csv.exportProducts')}</h1>
      </div>
      <CsvImportExport onImportComplete={handleCsvImportComplete} />
    {:else}
      <ProductForm
        mode={drawerMode}
        bind:formData
        bind:formErrors
        bind:productImages
        product={fullProductData}
        currency={list.currency}
        onsubmit={handleSubmit}
        onclose={closeDrawer}
        ondelete={handleDeleteProduct}
        ontoggleactive={toggleActiveInEdit}
        onuploaded={list.load}
      />
    {/if}
  </Drawer>
{/if}
