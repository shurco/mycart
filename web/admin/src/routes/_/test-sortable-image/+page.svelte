<script lang="ts">
  import SortableImage from '$lib/components/SortableImage.svelte'

  let images = $state([
    { id: 'img1', url: 'https://via.placeholder.com/300x300?text=Image+1', position: 0, is_representative: true },
    { id: 'img2', url: 'https://via.placeholder.com/300x300?text=Image+2', position: 1, is_representative: false },
    { id: 'img3', url: 'https://via.placeholder.com/300x300?text=Image+3', position: 2, is_representative: false },
    { id: 'img4', url: 'https://via.placeholder.com/300x300?text=Image+4', position: 3, is_representative: false }
  ])

  function handleReorder(updates: Array<{ imageId: string; position: number }>) {
    console.log('Reorder callback received:', updates)

    // Update local state to reflect new order
    const updatesMap = new Map(updates.map(u => [u.imageId, u.position]))
    images = images.map(img => ({
      ...img,
      position: updatesMap.get(img.id) ?? img.position
    }))
  }

  function handleSetRepresentative(imageId: string) {
    console.log('Set representative callback received:', imageId)

    // Update local state
    images = images.map(img => ({
      ...img,
      is_representative: img.id === imageId
    }))
  }
</script>

<div class="p-8">
  <div class="mb-6">
    <h1 class="text-2xl font-bold mb-2">SortableImage Component Test</h1>
    <p class="text-gray-600">Drag images to reorder. Click star button to set as representative.</p>
    <p class="text-sm text-gray-500 mt-2">Open browser console to see callback logs.</p>
  </div>

  <div class="bg-white rounded-lg shadow p-6">
    <SortableImage
      {images}
      onReorder={handleReorder}
      onSetRepresentative={handleSetRepresentative}
    />
  </div>

  <div class="mt-6">
    <h2 class="text-lg font-semibold mb-2">Current State:</h2>
    <pre class="bg-gray-100 p-4 rounded text-xs overflow-auto">{JSON.stringify(images, null, 2)}</pre>
  </div>
</div>
