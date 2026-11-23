<script lang="ts">
  import { FormUpload, SortableImage } from '$lib/components'
  import { loadData } from '$lib/utils/apiHelpers'
  import { showMessage } from '$lib/utils'
  import type { Product } from '$lib/types/models'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    productId: string
    images: Product['images']
    /** Runs after a successful upload, so the list behind the drawer can refresh. */
    onuploaded?: () => void
  }

  let { productId, images = $bindable(), onuploaded }: Props = $props()

  function handleUpload(result: any) {
    if (result?.success) {
      showMessage(t('products.imageUploaded'), 'connextSuccess')
      if (result?.result) {
        images = [...(images || []), result.result]
      }
      onuploaded?.()
    } else {
      // Show error message from server
      const errorMsg = result?.message || result?.result || t('products.imageUploadFailed')
      showMessage(errorMsg, 'connextError')
    }
  }

  // POSTs an image action, then refetches the product so positions and the
  // representative flag come back from the server.
  async function postImageAction(path: string, body: unknown, failedKey: string, successKey: string) {
    try {
      const res = await fetch(`/api/_/products/${productId}/${path}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        ...(body === undefined ? {} : { body: JSON.stringify(body) })
      })

      if (!res.ok) {
        const data = await res.json()
        showMessage(data.message || t(failedKey), 'connextError')
        return
      }

      showMessage(t(successKey), 'connextSuccess')

      const updatedProduct = await loadData<Product>(`/api/_/products/${productId}`, 'Failed to load product')
      if (updatedProduct) {
        images = updatedProduct.images || []
      }
    } catch (error) {
      showMessage(t('common.networkError'), 'connextError')
    }
  }

  function handleImageReorder(updates: Array<{ imageId: string; position: number }>) {
    return postImageAction('images/reorder', { updates }, 'products.failedToReorderImages', 'products.imagesReordered')
  }

  function handleSetRepresentative(imageId: string) {
    return postImageAction(
      `images/${imageId}/representative`,
      undefined,
      'products.failedToSetRepresentative',
      'products.representativeSet'
    )
  }
</script>

{#if images && images.length > 0}
  <SortableImage
    images={images.map(img => ({
      id: img.id,
      url: `/uploads/${img.name}_sm.${img.ext}`,
      position: img.position || 0,
      is_representative: img.is_representative || false
    }))}
    onReorder={handleImageReorder}
    onSetRepresentative={handleSetRepresentative}
  />
{/if}
<FormUpload section="image" {productId} accept=".jpg,.jpeg,.png" onadded={handleUpload} />
