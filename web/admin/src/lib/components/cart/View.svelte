<script lang="ts">
  import { onMount } from 'svelte'
  import FormButton from '../form/Button.svelte'
  import DetailList from '../DetailList.svelte'
  import SvgIcon from '../SvgIcon.svelte'
  import { costFormat, formatDate, STRIPE_DASHBOARD_URL } from '$lib/utils'
  import { formatCurrencyWithTruncation } from '$lib/utils/currency'
  import { loadData } from '$lib/utils/apiHelpers'
  import type { CartDetail } from '$lib/types/models'
  import { paymentSettingsStore } from '$lib/stores/payment'
  import { translate, locale } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)
  let currentLocale = $derived($locale)
  let paymentSettings = $derived($paymentSettingsStore)

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
              {formatCurrencyWithTruncation(
                cart.amount_total,
                cart.currency || 'USD',
                'admin',
                paymentSettings?.truncation,
                currentLocale
              )}
            </a>
          {:else}
            {formatCurrencyWithTruncation(
              cart.amount_total,
              cart.currency || 'USD',
              'admin',
              paymentSettings?.truncation,
              currentLocale
            )}
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
          <DetailList name={t('carts.items')} grid={false}>
            <div class="space-y-4">
              {#each cart.items as item (item.id)}
                <div class="flex items-start gap-4 border-b border-gray-200 pb-4 last:border-0">
                  {#if item.image}
                    <div class="flex-shrink-0">
                      <a
                        href="/uploads/{item.image.name}.{item.image.ext}"
                        target="_blank"
                        aria-label={t('carts.viewFullSizeImage')}
                      >
                        <img
                          class="w-20 h-20 object-cover rounded"
                          src="/uploads/{item.image.name}_sm.{item.image.ext}"
                          alt="{item.name}"
                          loading="lazy"
                        />
                      </a>
                    </div>
                  {/if}
                  <div class="flex-1 min-w-0">
                    <div class="font-medium text-gray-900">{item.name}</div>
                    <div class="text-sm text-gray-500">{t('carts.slug')} {item.slug}</div>
                    <div class="mt-1 text-sm text-gray-700">
                      <span class="font-medium">{t('carts.price')}</span>
                      {formatCurrencyWithTruncation(
                        item.amount,
                        cart.currency || 'USD',
                        'admin',
                        paymentSettings?.truncation,
                        currentLocale
                      )}
                    </div>
                    <div class="mt-1 text-sm text-gray-700">
                      <span class="font-medium">{t('carts.quantity')}</span> {item.quantity}
                    </div>
                    <div class="mt-1 text-sm text-gray-700">
                      <span class="font-medium">{t('carts.subtotal')}</span>
                      {formatCurrencyWithTruncation(
                        item.amount * item.quantity,
                        cart.currency || 'USD',
                        'admin',
                        paymentSettings?.truncation,
                        currentLocale
                      )}
                    </div>
                  </div>
                </div>
              {/each}
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
