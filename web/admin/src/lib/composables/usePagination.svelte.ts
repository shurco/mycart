import { DEFAULT_PAGE_SIZE, DEFAULT_PAGE } from '$lib/constants/pagination'

export function usePagination(initialLimit = DEFAULT_PAGE_SIZE) {
  let currentPage = $state(DEFAULT_PAGE)
  let limit = $state(initialLimit)
  let total = $state(0)

  const totalPages = $derived(Math.ceil(total / limit))

  function setPage(page: number) {
    currentPage = page
  }

  function setTotal(newTotal: number) {
    total = newTotal
  }

  function setLimit(newLimit: number) {
    limit = newLimit
  }

  return {
    get currentPage() {
      return currentPage
    },
    get limit() {
      return limit
    },
    get total() {
      return total
    },
    get totalPages() {
      return totalPages
    },
    setPage,
    setTotal,
    setLimit
  }
}
