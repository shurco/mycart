<script lang="ts">
  import Overlay from '$lib/components/Overlay.svelte'
  import { formatCurrency } from '$lib/utils/currency'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    errors: any[]
    currency: string
    onClose: () => void
  }

  let { errors, currency, onClose }: Props = $props()
</script>

<Overlay show={true} {onClose}>
  <div class="validation-modal bg-white p-8 border-4 border-black max-w-lg mx-auto">
    <h2 class="text-2xl font-black tracking-tight text-red-600 uppercase mb-4">{t('cart.validation_errors_title')}</h2>
    <ul class="space-y-3 mb-6">
      {#each errors as error}
        <li class="bg-yellow-100 p-4 border-2 border-black">
          {#if error.error_type === 'quantity_unavailable'}
            <p class="font-bold">{t('cart.out_of_stock')} - {t('cart.please_remove')}</p>
          {:else if error.error_type === 'price_changed'}
            <p>{t('cart.price_updated')}:
              <span class="line-through text-gray-600">{formatCurrency(error.requested_unit_price, currency)}</span>
              → <span class="font-bold text-green-700">{formatCurrency(error.current_unit_price, currency)}</span>
            </p>
          {:else if error.error_type === 'product_inactive' || error.error_type === 'product_not_found'}
            <p class="font-bold">{t('cart.out_of_stock')} - {t('cart.please_remove')}</p>
          {/if}
        </li>
      {/each}
    </ul>
    <button
      onclick={onClose}
      class="w-full border-4 border-black bg-blue-500 px-6 py-3 text-lg font-black tracking-wider text-white uppercase hover:bg-blue-600"
    >
      {t('cart.review_cart')}
    </button>
  </div>
</Overlay>
