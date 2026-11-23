<script lang="ts">
  import { SvgIcon } from '$lib/components'
  import { formatCurrencyWithTruncation } from '$lib/utils/currency'
  import type { Product } from '$lib/types/models'
  import { paymentSettingsStore } from '$lib/stores/payment'
  import { translate, locale } from '$lib/i18n'
  import { digitalTypeIco } from './productDigital'

  // Reactive translation function
  let t = $derived($translate)
  let currentLocale = $derived($locale)
  let paymentSettings = $derived($paymentSettingsStore)

  type RowAction = (product: Product, index: number) => void

  interface Props {
    products: Product[]
    currency: string
    onview: RowAction
    onedit: RowAction
    onseo: RowAction
    ondigital: RowAction
    ontoggle: RowAction
  }

  let { products, currency, onview, onedit, onseo, ondigital, ontoggle }: Props = $props()
</script>

<table>
  <thead>
    <tr>
      <th class="w-28"></th>
      <th>{t('products.name')}</th>
      <th class="w-32">{t('products.slug')}</th>
      <th class="w-32">{t('products.price')}</th>
      <th class="w-12 px-4 py-2">
        <SvgIcon name="cube" className="h-5 w-5" stroke="currentColor" />
      </th>
      <th class="w-24 px-4 py-2"></th>
    </tr>
  </thead>
  <tbody>
    {#each products as product, index (product.id)}
      <tr class:opacity-30={!product.active} data-testid="product-row">
        <td>
          {#if product.images && product.images.length > 0}
            <a href="/uploads/{product.images[0].name}.{product.images[0].ext}" target="_blank">
              <img
                style="width: 100%; max-width: 80px"
                src="/uploads/{product.images[0].name}_sm.{product.images[0].ext}"
                alt={product.name}
                loading="lazy"
              />
            </a>
          {:else}
            <img style="width: 100%; max-width: 80px" src="/assets/img/noimage.png" alt="" loading="lazy" />
          {/if}
        </td>
        <td onclick={() => onview(product, index)}>
          <div class="font-bold">{product.name}</div>
          {#if product.brief}
            <span class="hidden text-gray-400 xl:block">{product.brief}</span>
          {/if}
        </td>
        <td>
          {#if product.active}
            <a href="/products/{product.slug}" target="_blank">{product.slug}</a>
          {:else}
            <span>{product.slug}</span>
          {/if}
        </td>
        <td onclick={() => onview(product, index)}>
          {#if !product.amount || parseFloat(String(product.amount)) === 0}
            <span class="font-bold text-green-600">free</span>
          {:else}
            {formatCurrencyWithTruncation(
              product.amount,
              currency || 'USD',
              'admin',
              paymentSettings?.truncation,
              currentLocale,
              paymentSettings?.number_format,
              paymentSettings?.symbol_display?.admin
            )}
          {/if}
        </td>
        <td class="px-4 py-2">
          {#if product.digital && product.digital.type}
            <SvgIcon
              name={digitalTypeIco(product.digital.type)}
              className="h-5 w-5 cursor-pointer {product.digital.filled === true ? 'text-black' : 'text-red-600'}"
              onclick={() => ondigital(product, index)}
              stroke="currentColor"
            />
          {/if}
        </td>
        <td class="px-4 py-2">
          <div class="flex">
            <div class="pr-3">
              <SvgIcon
                name="pencil-square"
                className="h-5 w-5 cursor-pointer"
                onclick={() => onedit(product, index)}
                stroke="currentColor"
              />
            </div>
            <div class="pr-3">
              <SvgIcon
                name="rocket"
                className="h-5 w-5 cursor-pointer"
                onclick={() => onseo(product, index)}
                stroke="currentColor"
              />
            </div>
            <div>
              <SvgIcon
                name={product.active ? 'eye' : 'eye-slash'}
                className="h-5 w-5 cursor-pointer"
                onclick={() => ontoggle(product, index)}
                stroke="currentColor"
              />
            </div>
          </div>
        </td>
      </tr>
    {/each}
  </tbody>
</table>
