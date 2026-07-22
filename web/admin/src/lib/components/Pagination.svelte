<script lang="ts">
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    currentPage: number
    totalPages: number
    onPageChange: (page: number) => void
  }

  let { currentPage, totalPages, onPageChange }: Props = $props()

  function goToPage(page: number) {
    if (page >= 1 && page <= totalPages && page !== currentPage) {
      onPageChange(page)
    }
  }

  function getVisiblePages(): number[] {
    const delta = 2
    const range: number[] = []
    const rangeWithDots: number[] = []

    for (let i = Math.max(2, currentPage - delta); i <= Math.min(totalPages - 1, currentPage + delta); i++) {
      range.push(i)
    }

    if (currentPage - delta > 2) {
      rangeWithDots.push(1, -1) // -1 represents ellipsis
    } else {
      rangeWithDots.push(1)
    }

    rangeWithDots.push(...range)

    if (currentPage + delta < totalPages - 1) {
      rangeWithDots.push(-1, totalPages) // -1 represents ellipsis
    } else if (totalPages > 1) {
      rangeWithDots.push(totalPages)
    }

    console.log('[Pagination] getVisiblePages called:', { currentPage, totalPages, result: rangeWithDots })
    return rangeWithDots
  }
</script>

{#if totalPages > 1}
  <div class="flex items-center justify-center gap-2 py-4">
    <button
      class="rounded px-3 py-1 text-sm {currentPage === 1 ? 'cursor-not-allowed opacity-50' : 'cursor-pointer hover:bg-gray-100'}"
      disabled={currentPage === 1}
      onclick={() => goToPage(currentPage - 1)}
    >
      {t('pagination.previous')}
    </button>

    {#each getVisiblePages() as page (page)}
      {#if page === -1}
        <span class="px-2">...</span>
      {:else}
        <button
          class="rounded px-3 py-1 text-sm cursor-pointer {page === currentPage
            ? 'bg-gray-900 text-white'
            : 'hover:bg-gray-100'}"
          onclick={() => goToPage(page)}
        >
          {page}
        </button>
      {/if}
    {/each}

    <button
      class="rounded px-3 py-1 text-sm {currentPage === totalPages
        ? 'cursor-not-allowed opacity-50'
        : 'cursor-pointer hover:bg-gray-100'}"
      disabled={currentPage === totalPages}
      onclick={() => goToPage(currentPage + 1)}
    >
      {t('pagination.next')}
    </button>
  </div>
{/if}
