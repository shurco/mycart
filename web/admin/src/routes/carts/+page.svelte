<script lang="ts">
  import { onMount } from 'svelte'
  import Main from '$lib/layouts/Main.svelte'
  import Drawer from '$lib/components/Drawer.svelte'
  import CartView from '$lib/components/cart/View.svelte'
  import SvgIcon from '$lib/components/SvgIcon.svelte'
  import Pagination from '$lib/components/Pagination.svelte'
  import { loadData, handleApiCall } from '$lib/utils/apiHelpers'
  import { apiPost } from '$lib/utils'
  import { formatCurrency, formatCurrencyWithTruncation } from '$lib/utils/currency'
  import { costFormat, formatDate } from '$lib/utils'
  import { STRIPE_DASHBOARD_URL } from '$lib/utils/constants'
  import type { Cart } from '$lib/types/models'
  import { paymentSettingsStore } from '$lib/stores/payment'
  import { translate, locale } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)
  let currentLocale = $derived($locale)
  let paymentSettings = $derived($paymentSettingsStore)

  interface DrawerCart {
    cart: Cart
  }

  interface CartsResponse {
    carts: Cart[]
    total: number
    page: number
    limit: number
  }

  import { DEFAULT_PAGE_SIZE } from '$lib/constants/pagination'
  import { DRAWER_CLOSE_DELAY_MS } from '$lib/constants/ui'

  let carts = $state<Cart[]>([])
  let loading = $state(true)
  let drawerOpen = $state(false)
  let drawerCart = $state<DrawerCart | null>(null)
  let currentPage = $state(1)
  let limit = $state(DEFAULT_PAGE_SIZE)
  let total = $state(0)

  onMount(async () => {
    await loadCarts()
  })

  async function loadCarts(page = currentPage) {
    loading = true
    currentPage = page
    const result = await loadData<CartsResponse>(
      `/api/_/carts?page=${page}&limit=${limit}`,
      t('carts.failedToLoad')
    )
    if (result) {
      carts = result.carts || []
      total = result.total || 0
    }
    loading = false
  }

  function handlePageChange(page: number) {
    loadCarts(page)
  }

  function openView(cart: Cart) {
    drawerCart = { cart }
    drawerOpen = true
  }

  function closeDrawer() {
    if (drawerOpen) {
      drawerOpen = false
      setTimeout(() => {
        drawerCart = null
      }, DRAWER_CLOSE_DELAY_MS)
    }
  }

  async function sendMail(cartId: string, event: Event) {
    event.stopPropagation()
    await handleApiCall(
      () => apiPost(`/api/_/carts/${cartId}/mail`, {}),
      t('carts.mailSentSuccessfully'),
      t('carts.failedToSendMailMessage')
    )
  }
</script>

<Main>
  <div class="mb-5 flex items-center justify-between">
    <h1>{t('carts.title')}</h1>
  </div>

  {#if loading}
    <div class="py-8 text-center">{t('common.loading')}</div>
  {:else if carts.length === 0}
    <div class="py-8 text-center text-gray-500">{t('carts.noCarts')}</div>
  {:else}
    <table>
      <thead>
        <tr>
          <th>{t('carts.email')}</th>
          <th>{t('carts.priceColumn')}</th>
          <th>{t('carts.statusColumn')}</th>
          <th>{t('carts.paymentColumn')}</th>
          <th class="w-48">{t('common.created')}</th>
          <th class="w-48">{t('common.updated')}</th>
          <th class="w-12"></th>
        </tr>
      </thead>
      <tbody>
        {#each carts as cart, index (cart.id)}
          <tr
            class:bg-green-50={cart.payment_status === 'paid'}
            class="cursor-pointer hover:bg-gray-50"
            onclick={() => openView(cart)}
          >
            <td>{cart.email || '-'}</td>
            <td>
              {formatCurrencyWithTruncation(
                cart.amount_total,
                cart.currency || 'USD',
                'admin',
                paymentSettings?.truncation,
                currentLocale
              )}
            </td>
            <td
              class={cart.payment_status === 'paid'
                ? 'text-green-600'
                : cart.payment_status === 'pending'
                  ? 'text-yellow-600'
                  : cart.payment_status === 'failed'
                    ? 'text-red-600'
                    : 'text-gray-600'}
            >
              {cart.payment_status || '-'}
            </td>
            <td>{cart.payment_system || '-'}</td>
            <td>{formatDate(cart.created)}</td>
            <td>
              {#if cart.updated}
                {formatDate(cart.updated)}
              {/if}
            </td>
            <td onclick={(e) => e.stopPropagation()}>
              {#if cart.payment_status === 'paid'}
                <SvgIcon
                  name="envelope"
                  className="h-5 w-5 cursor-pointer"
                  onclick={(e: Event) => sendMail(cart.id, e)}
                  stroke="currentColor"
                />
              {:else}
                <SvgIcon name="envelope" className="h-5 w-5 opacity-30" stroke="currentColor" />
              {/if}
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
    {#if drawerCart}
      <CartView drawer={drawerCart} onclose={closeDrawer} />
    {/if}
  </Drawer>
{/if}
