<script lang="ts">
  import SvgIcon from './SvgIcon.svelte'

  interface ImageItem {
    id: string
    url: string
    position: number
    is_representative: boolean
  }

  interface Props {
    images: ImageItem[]
    onReorder?: (updates: Array<{ imageId: string; position: number }>) => void
    onSetRepresentative?: (imageId: string) => void
  }

  let { images, onReorder, onSetRepresentative }: Props = $props()

  let draggedIndex = $state<number | null>(null)
  let dropTargetIndex = $state<number | null>(null)

  // Sort images by position for display
  let sortedImages = $derived(
    [...images].sort((a, b) => a.position - b.position)
  )

  function handleDragStart(event: DragEvent, index: number) {
    if (!event.dataTransfer) return
    draggedIndex = index
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/html', String(index))

    // Add opacity to dragged element
    if (event.target instanceof HTMLElement) {
      event.target.style.opacity = '0.4'
    }
  }

  function handleDragEnd(event: DragEvent) {
    // Reset opacity
    if (event.target instanceof HTMLElement) {
      event.target.style.opacity = '1'
    }
    draggedIndex = null
    dropTargetIndex = null
  }

  function handleDragOver(event: DragEvent, index: number) {
    event.preventDefault()
    if (!event.dataTransfer) return

    event.dataTransfer.dropEffect = 'move'
    dropTargetIndex = index
  }

  function handleDragLeave() {
    dropTargetIndex = null
  }

  function handleDrop(event: DragEvent, targetIndex: number) {
    event.preventDefault()

    if (draggedIndex === null || draggedIndex === targetIndex) {
      draggedIndex = null
      dropTargetIndex = null
      return
    }

    const reorderedImages = [...sortedImages]
    const [draggedItem] = reorderedImages.splice(draggedIndex, 1)
    reorderedImages.splice(targetIndex, 0, draggedItem)

    // Build updates array with new positions
    const updates = reorderedImages.map((img, idx) => ({
      imageId: img.id,
      position: idx
    }))

    onReorder?.(updates)

    draggedIndex = null
    dropTargetIndex = null
  }

  function handleSetRepresentative(imageId: string) {
    onSetRepresentative?.(imageId)
  }
</script>

<div class="sortable-image-container">
  {#if sortedImages.length === 0}
    <div class="empty-state">
      <SvgIcon name="photo" className="h-12 w-12 text-gray-400" />
      <p class="text-sm text-gray-500 mt-2">No images uploaded</p>
    </div>
  {:else}
    <div class="info-text">
      <p class="text-sm text-gray-600 mb-3">
        <span class="font-medium">First image is shown in product listings</span> and accessible at <code>/products/{'{urlslug}'}.png</code>
      </p>
    </div>
    <div class="image-grid">
      {#each sortedImages as image, index (image.id)}
        <div
          class="image-item {dropTargetIndex === index ? 'drop-target' : ''}"
          draggable="true"
          ondragstart={(e) => handleDragStart(e, index)}
          ondragend={handleDragEnd}
          ondragover={(e) => handleDragOver(e, index)}
          ondragleave={handleDragLeave}
          ondrop={(e) => handleDrop(e, index)}
          role="button"
          tabindex="0"
        >
          <div class="image-wrapper">
            <img src={image.url} alt="Product image {index + 1}" class="image-preview" />

            {#if index === 0}
              <div class="rep-badge">
                <span class="text-xs font-bold">REP</span>
              </div>
            {/if}

            {#if image.is_representative}
              <div class="representative-badge">
                <SvgIcon name="star" className="h-4 w-4 text-yellow-500" fill="currentColor" />
              </div>
            {/if}

            <div class="image-overlay">
              <button
                type="button"
                class="overlay-button"
                onclick={() => handleSetRepresentative(image.id)}
                aria-label="Set as representative image"
              >
                <SvgIcon
                  name="star"
                  className="h-5 w-5"
                  fill={image.is_representative ? 'currentColor' : 'none'}
                />
                <span class="text-xs">
                  {image.is_representative ? 'Main' : 'Set Main'}
                </span>
              </button>
            </div>

            <div class="position-indicator">
              {index + 1}
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  @reference "tailwindcss";

  :global(.sortable-image-container) {
    @apply w-full;
  }

  :global(.info-text) {
    @apply mb-4 p-3 bg-blue-50 border border-blue-200 rounded-lg;
  }

  :global(.info-text code) {
    @apply bg-white px-2 py-0.5 rounded text-blue-700 font-mono text-xs;
  }

  :global(.empty-state) {
    @apply flex flex-col items-center justify-center p-8 border-2 border-dashed border-gray-300 rounded-lg;
  }

  :global(.image-grid) {
    @apply grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4;
  }

  :global(.image-item) {
    @apply relative cursor-move rounded-lg border-2 border-gray-200 bg-white transition-all;
  }

  :global(.image-item:hover) {
    @apply border-blue-400 shadow-md;
  }

  :global(.image-item.drop-target) {
    @apply border-blue-500 border-dashed bg-blue-50;
  }

  :global(.image-wrapper) {
    @apply relative aspect-square overflow-hidden rounded-md;
  }

  :global(.image-preview) {
    @apply w-full h-full object-cover;
  }

  :global(.rep-badge) {
    @apply absolute top-2 left-2 bg-blue-600 text-white rounded px-2 py-1 shadow-md z-10;
  }

  :global(.representative-badge) {
    @apply absolute top-2 right-2 bg-white rounded-full p-1 shadow-md;
  }

  :global(.image-overlay) {
    @apply absolute inset-0 bg-black/0 hover:bg-black/50 transition-all duration-200 flex items-center justify-center opacity-0 hover:opacity-100;
  }

  :global(.overlay-button) {
    @apply flex flex-col items-center gap-1 text-white font-medium px-3 py-2 rounded-md bg-black/50 hover:bg-black/70 transition-all;
  }

  :global(.position-indicator) {
    @apply absolute bottom-2 left-2 bg-gray-900/75 text-white text-xs font-bold rounded-full w-6 h-6 flex items-center justify-center;
  }
</style>
