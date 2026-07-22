<script lang="ts">
  import { onMount } from 'svelte'
  import FormButton from '../form/Button.svelte'
  import DetailList from '../DetailList.svelte'
  import SvgIcon from '../SvgIcon.svelte'
  import { costFormat, formatDate, STRIPE_DASHBOARD_URL } from '$lib/utils'
  import { loadData } from '$lib/utils/apiHelpers'
  import type { CartDetail } from '$lib/types/models'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface DrawerCart {
    cart: {
      id: string
      email: string
      amount_total: number
      currency: string
      payment_status: 'paid' | 'pending' | 'failed'
      payment_system?: string
      payment_id?: string
      created?: string
      updated?: string
    }
  }

  interface Props {
    drawer: DrawerCart
    onclose?: () => void
  }

  let { drawer, onclose }: Props = $props()

  let cart = $state<CartDetail | null>(null)
  let loading = $state(true)
  let lastCartId = $state<string | null>(null)

  async function loadCart() {
    if (!drawer?.cart?.id) return

    loading = true
    const result = await loadData<CartDetail>(`/api/_/carts/${drawer.cart.id}`, t('carts.failedToLoadCart'))
    if (result) {
      cart = result
      lastCartId = drawer.cart.id
    }
    loading = false
  }

  onMount(async () => {
    await loadCart()
  })

  // Reload cart when drawer.cart.id changes
  $effect(() => {
    if (drawer?.cart?.id && drawer.cart.id !== lastCartId) {
      loadCart()
    }
  })

  function close() {
    onclose?.()
  }

  function getPaymentStatusColor(status: string) {
    switch (status) {
      case 'paid':
        return 'text-green-600'
      case 'pending':
        return 'text-yellow-600'
      case 'failed':
        return 'text-red-600'
      default:
        return 'text-gray-600'
    }
  }
</script>

<div>
  <div class="pb-8">
    <div class="flex items-center">
      <div class="pr-3">
        <h1>{t('carts.cartDetails')}</h1>
      </div>
    </div>
  </div>

  {#if loading}
    <div class="py-8 text-center">{t('common.loading')}</div>
  {:else if cart}
    <div class="flow-root">
      <dl class="-my-3 mt-2 divide-y divide-gray-100 text-sm">
        <DetailList name={t('carts.cartId')}>{cart.id}</DetailList>
        
        <DetailList name={t('carts.customerEmail')}>
          {#if cart.email}
            <a href="mailto:{cart.email}" class="text-blue-600 hover:underline">{cart.email}</a>
          {:else}
            <span class="text-gray-400">-</span>
          {/if}
        </DetailList>

        <DetailList name={t('carts.totalAmount')}>
          {#if !cart.amount_total || cart.amount_total === 0}
            <span class="font-bold text-green-600">{t('carts.free')}</span>
          {:else if cart.payment_id && cart.payment_system === 'stripe'}
            <a
              href="{STRIPE_DASHBOARD_URL}/{cart.payment_id}"
              target="_blank"
              class="text-blue-600 hover:underline"
            >
              {costFormat(cart.amount_total)} {cart.currency || ''}
            </a>
          {:else}
            {costFormat(cart.amount_total)} {cart.currency || ''}
          {/if}
        </DetailList>

        <DetailList name={t('carts.paymentStatus')}>
          <span class={getPaymentStatusColor(cart.payment_status || '')}>
            {cart.payment_status || '-'}
          </span>
        </DetailList>

        <DetailList name={t('carts.paymentSystem')}>{cart.payment_system || '-'}</DetailList>

        {#if cart.payment_id}
          <DetailList name={t('carts.paymentId')}>
            {#if cart.payment_system === 'stripe'}
              <a
                href="{STRIPE_DASHBOARD_URL}/{cart.payment_id}"
                target="_blank"
                class="text-blue-600 hover:underline"
              >
                {cart.payment_id}
              </a>
            {:else}
              {cart.payment_id}
            {/if}
          </DetailList>
        {/if}

        <DetailList name={t('common.created')}>{formatDate(cart.created)}</DetailList>
        
        {#if cart.updated}
          <DetailList name={t('common.updated')}>{formatDate(cart.updated)}</DetailList>
        {/if}

        {#if cart.items && cart.items.length > 0}
          <DetailList name={t('carts.items')} fullWidth={true}>
            <div class="overflow-x-auto">
              <table class="min-w-full divide-y divide-gray-300">
                <thead>
                  <tr>
                    <th scope="col" class="py-3 pl-4 pr-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">Product</th>
                    <th scope="col" class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">Variant</th>
                    <th scope="col" class="px-3 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">Qty</th>
                    <th scope="col" class="px-3 py-3 text-right text-xs font-medium uppercase tracking-wide text-gray-500">Unit Price</th>
                    <th scope="col" class="py-3 pl-3 pr-4 text-right text-xs font-medium uppercase tracking-wide text-gray-500">Subtotal</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-200">
                  {#each cart.items as item (`${item.id}-${item.variant_id || 'no-variant'}`)}
                    <tr>
                      <td class="whitespace-nowrap py-4 pl-4 pr-3">
                        <div class="flex items-center gap-3">
                          {#if item.image}
                            <div class="flex-shrink-0">
                              <a
                                href="/uploads/{item.image.name}.{item.image.ext}"
                                target="_blank"
                                aria-label={t('carts.viewFullSizeImage')}
                              >
                                <img
                                  class="h-12 w-12 rounded object-cover"
                                  src="/uploads/{item.image.name}_sm.{item.image.ext}"
                                  alt="{item.name}"
                                  loading="lazy"
                                />
                              </a>
                            </div>
                          {/if}
                          <div>
                            <div class="font-medium text-gray-900">{item.name}</div>
                            <div class="text-xs text-gray-500">{item.slug}</div>
                          </div>
                        </div>
                      </td>
                      <td class="whitespace-nowrap px-3 py-4 text-sm text-gray-700">
                        {#if item.variant_options && Object.keys(item.variant_options).length > 0}
                          <div class="space-y-1">
                            {#each Object.entries(item.variant_options) as [key, value]}
                              <div><span class="font-medium">{key}:</span> {value}</div>
                            {/each}
                          </div>
                          {#if item.variant_sku}
                            <div class="mt-1 text-xs text-gray-500">SKU: {item.variant_sku}</div>
                          {/if}
                        {:else}
                          <span class="text-gray-400">-</span>
                        {/if}
                      </td>
                      <td class="whitespace-nowrap px-3 py-4 text-sm text-gray-700">
                        {item.quantity}
                      </td>
                      <td class="whitespace-nowrap px-3 py-4 text-sm text-gray-700 text-right">
                        {costFormat(item.amount)} {cart.currency || ''}
                      </td>
                      <td class="whitespace-nowrap py-4 pl-3 pr-4 text-sm font-medium text-gray-900 text-right">
                        {costFormat(item.amount * item.quantity)} {cart.currency || ''}
                      </td>
                    </tr>
                  {/each}
                </tbody>
                <tfoot class="border-t-2 border-gray-300">
                  <tr>
                    <td colspan="4" class="py-4 pl-4 pr-3 text-right text-sm font-bold text-gray-900 uppercase">
                      {t('cart.total')}
                    </td>
                    <td class="py-4 pl-3 pr-4 text-right text-base font-bold text-gray-900">
                      {costFormat(cart.amount_total)} {cart.currency || ''}
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>
          </DetailList>
        {:else}
          <DetailList name={t('carts.items')}>
            <span class="text-gray-400">{t('carts.noItems')}</span>
          </DetailList>
        {/if}
      </dl>
    </div>
  {:else}
    <div class="py-8 text-center text-gray-500">{t('carts.failedToLoadCart')}</div>
  {/if}

  <div class="pt-5">
    <FormButton type="button" name={t('common.close')} color="green" onclick={close} />
  </div>
</div>
