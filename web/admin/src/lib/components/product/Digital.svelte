<script lang="ts">
  import { onMount } from 'svelte'
  import { DrawerFooter, DrawerHeader, FormInput, FormUpload, IconButton, PageState } from '$lib/components'
  import { loadData } from '$lib/utils/apiHelpers'
  import { apiPost, apiUpdate, apiDelete } from '$lib/utils/api'
  import { showMessage } from '$lib/utils'
  import type { Product } from '$lib/types/models'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Digital {
    type: string
    files: Array<{
      id: string
      name: string
      ext: string
      orig_name?: string
    }>
    data: Array<{
      id: string
      content: string
      cart_id: string | null
    }>
  }

  interface DrawerProduct {
    product: Product
    index: number
    currency?: string
  }

  interface Props {
    drawer: DrawerProduct
    onContentUpdate?: (() => void) | undefined
    onclose?: () => void
  }

  let { drawer, onContentUpdate, onclose }: Props = $props()

  let digital = $state<Digital>({
    type: '',
    files: [],
    data: []
  })
  let loading = $state(true)

  onMount(async () => {
    await loadDigital()
  })

  async function loadDigital() {
    loading = true
    const result = await loadData<Digital>(
      `/api/_/products/${drawer.product.id}/digital`,
      t('digital.failedToLoadContent')
    )
    if (result) {
      digital = {
        type: result.type || '',
        files: result.files || [],
        data: result.data || []
      }
    }
    loading = false
  }

  function close() {
    onclose?.()
  }

  async function handleUpload(event: CustomEvent) {
    if (event.detail.success && event.detail.result) {
      digital.files = [...digital.files, event.detail.result]
      showMessage(t('common.fileUploaded'), 'connextSuccess')
      if (onContentUpdate) {
        onContentUpdate()
      }
    }
  }

  function isCodeSold(cartId: string | null | undefined): boolean {
    if (!cartId || cartId === null) return false
    const trimmed = String(cartId).trim()
    return trimmed !== '' && trimmed !== 'null' && trimmed !== 'undefined'
  }

  async function addDigitalData() {
    const result = await apiPost(`/api/_/products/${drawer.product.id}/digital`)
    if (result.success && result.result) {
      digital.data = [...digital.data, result.result]
      showMessage(t('digital.dataAdded'), 'connextSuccess')
      if (onContentUpdate) {
        onContentUpdate()
      }
    } else {
      showMessage(result.message || t('digital.failedToAddData'), 'connextError')
    }
  }

  async function saveData(index: number) {
    const dataItem = digital.data[index]
    // Don't allow saving if code is sold (has cart_id)
    if (!dataItem || isCodeSold(dataItem.cart_id)) return

    const update = {
      content: dataItem.content
    }
    const result = await apiUpdate(`/api/_/products/${drawer.product.id}/digital/${dataItem.id}`, update)
    if (result.success) {
      showMessage(t('common.dataSaved'), 'connextSuccess')
    } else {
      showMessage(result.message || t('common.failedToSaveData'), 'connextError')
    }
  }

  async function deleteDigital(type: 'file' | 'data', index: number) {
    const digitalId = type === 'file' ? digital.files[index].id : digital.data[index].id
    const result = await apiDelete(`/api/_/products/${drawer.product.id}/digital/${digitalId}`)

    if (result.success) {
      if (type === 'file') {
        digital.files = digital.files.filter((_, i) => i !== index)
      } else {
        digital.data = digital.data.filter((_, i) => i !== index)
      }
      showMessage(t('common.deleted'), 'connextSuccess')
      if (onContentUpdate) {
        onContentUpdate()
      }
    } else {
      showMessage(result.message || t('common.failedToDelete'), 'connextError')
    }
  }
</script>

<div>
  <DrawerHeader title={t('digital.digitalType', { type: digital.type })} />

  {#if digital.type === 'file'}
    <p class="mb-4">
      {t('digital.fileDescription')}
    </p>
  {/if}
  {#if digital.type === 'data'}
    <p class="mb-4">
      {t('digital.dataDescription')}
    </p>
  {/if}

  {#if loading}
    <PageState kind="loading" />
  {:else if digital.type === 'file'}
    <!-- File section -->
    <div class="flow-root">
      <div class="mx-auto -my-3 mt-2 mb-0 space-y-4 text-sm">
        {#if digital.files && digital.files.length > 0}
          <div class="grid content-start">
            {#each digital.files as file, index (file.id)}
              <div class="relative mt-4 flex first:mt-0">
                <a
                  href={`/api/_/products/${drawer.product.id}/digital/${file.id}/download`}
                  download={file.orig_name || `${file.name}.${file.ext}`}
                  class="rounded-lg bg-gray-200 px-3 py-3"
                  rel="noopener noreferrer"
                >
                  {file.orig_name || file.name}.{file.ext}
                </a>
                <div class="mt-3 ml-3">
                  <IconButton
                    ico="trash"
                    label={t('common.delete')}
                    variant="danger"
                    onclick={() => deleteDigital('file', index)}
                  />
                </div>
              </div>
            {/each}
          </div>
        {/if}
        <FormUpload section="digital" productId={drawer.product.id} onadded={handleUpload} />
      </div>
    </div>
  {:else if digital.type === 'data'}
    <!-- Data section -->
    <div class="flow-root">
      <div class="mx-auto -my-3 mt-4 mb-0 space-y-4 text-sm">
        {#if digital.data && digital.data.length > 0}
          {#each digital.data as dataItem, index (dataItem.id)}
            <div class="flex">
              {#if !isCodeSold(dataItem.cart_id)}
                <!-- Not sold - editable with delete button -->
                <div class="grow">
                  <FormInput
                    id="data-{dataItem.id}"
                    type="text"
                    title=""
                    bind:value={dataItem.content}
                    onfocusout={() => saveData(index)}
                  />
                </div>
                <div class="flex-none pt-3 pl-3">
                  <IconButton
                    ico="trash"
                    label={t('common.delete')}
                    variant="danger"
                    onclick={() => deleteDigital('data', index)}
                  />
                </div>
              {:else}
                <!-- Sold - read-only with badge -->
                <div class="grow">
                  <div class="flex items-center gap-2 rounded-lg bg-gray-200 px-3 py-3">
                    <span class="flex-1">{dataItem.content}</span>
                    <span
                      class="inline-flex items-center rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-medium text-red-800"
                      title={t('digital.codeSold', { cart_id: dataItem.cart_id ?? '' })}
                    >
                      {t('digital.sold')}
                    </span>
                  </div>
                </div>
              {/if}
            </div>
          {/each}
        {/if}
        <div class="flex">
          <div class="grow"></div>
          <div class="mt-2 flex-none">
            <button
              type="button"
              class="shrink-0 rounded-lg bg-gray-200 p-2 text-sm font-medium text-gray-700"
              onclick={addDigitalData}
            >
              {t('digital.addData')}
            </button>
          </div>
        </div>
      </div>
    </div>
  {:else}
    <PageState kind="empty" message={t('products.selectDigitalType')} />
  {/if}

  <DrawerFooter onclose={close} />
</div>
