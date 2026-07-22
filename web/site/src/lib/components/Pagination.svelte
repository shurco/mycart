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
      rangeWithDots.push(1, -1) // -1 represents leading ellipsis
    } else {
      rangeWithDots.push(1)
    }

    rangeWithDots.push(...range)

    if (currentPage + delta < totalPages - 1) {
      rangeWithDots.push(-2, totalPages) // -2 represents trailing ellipsis (different from -1)
    } else if (totalPages > 1) {
      rangeWithDots.push(totalPages)
    }

    return rangeWithDots
  }
</script>

{#if totalPages > 1}
  <div class="mt-12 flex items-center justify-center gap-2">
    <button
      class="rounded border-2 border-black bg-white px-4 py-2 font-bold uppercase transition-colors {currentPage === 1
        ? 'cursor-not-allowed opacity-50'
        : 'cursor-pointer hover:bg-black hover:text-white'}"
      disabled={currentPage === 1}
      onclick={() => goToPage(currentPage - 1)}
    >
      {t('pagination.previous')}
    </button>

    {#each getVisiblePages() as page (page)}
      {#if page === -1 || page === -2}
        <span class="px-2 font-bold">...</span>
      {:else}
        <button
          class="cursor-pointer rounded border-2 border-black px-4 py-2 font-bold uppercase transition-colors {page ===
          currentPage
            ? 'bg-black text-white'
            : 'bg-white hover:bg-black hover:text-white'}"
          onclick={() => goToPage(page)}
        >
          {page}
        </button>
      {/if}
    {/each}

    <button
      class="rounded border-2 border-black bg-white px-4 py-2 font-bold uppercase transition-colors {currentPage ===
      totalPages
        ? 'cursor-not-allowed opacity-50'
        : 'cursor-pointer hover:bg-black hover:text-white'}"
      disabled={currentPage === totalPages}
      onclick={() => goToPage(currentPage + 1)}
    >
      {t('pagination.next')}
    </button>
  </div>
{/if}
