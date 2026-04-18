<script lang="ts">
  import { onMount } from 'svelte'
  import FormButton from '../form/Button.svelte'
  import DetailList from '../DetailList.svelte'
  import SvgIcon from '../SvgIcon.svelte'
  import { costFormat, formatDate } from '$lib/utils'
  import { loadData } from '$lib/utils/apiHelpers'
  import type { Product } from '$lib/types/models'
  import { translate } from '$lib/i18n'
  import { sanitizeHTML } from '$lib/utils/sanitize'

  // Reactive translation function
  let t = $derived($translate)

  interface DrawerProduct {
    product: Product
    index: number
    currency?: string
  }

  interface Props {
    drawer: DrawerProduct
    updateActive?: ((index: number) => void) | undefined
    onclose?: () => void
  }

  let { drawer, updateActive, onclose }: Props = $props()

  let product = $state<Product | null>(null)
  let loading = $state(true)
  let lastProductId = $state<string | null>(null)

  async function loadProduct() {
    if (!drawer?.product?.id) return

    loading = true
    const result = await loadData<Product>(`/api/_/products/${drawer.product.id}`, t('products.failedToLoadProduct'))
    if (result) {
      product = result
      lastProductId = drawer.product.id
    }
    loading = false
  }

  onMount(async () => {
    await loadProduct()
  })

  // Reload product when drawer.product.id changes
  $effect(() => {
    if (drawer?.product?.id && drawer.product.id !== lastProductId) {
      loadProduct()
    }
  })

  function close() {
    onclose?.()
  }

  async function active() {
    if (updateActive && product) {
      await updateActive(drawer.index)
      // Update local product state reactively
      if (product) {
        product = { ...product, active: !product.active }
      }
    }
  }
</script>

<div>
  <div class="pb-8">
    <div class="flex items-center">
      <div class="pr-3">
        <h1>{t('products.viewProduct')} {product?.name || t('products.title')}</h1>
      </div>
      <div>
        {#if product}
          <SvgIcon
            name={product.active ? 'eye' : 'eye-slash'}
            className="h-5 w-5 cursor-pointer"
            onclick={active}
            stroke="currentColor"
          />
        {/if}
      </div>
    </div>
  </div>

  {#if loading}
    <div class="py-8 text-center">{t('common.loading')}</div>
  {:else if product}
    <div class="flow-root">
      <dl class="-my-3 mt-2 divide-y divide-gray-100 text-sm">
        <DetailList name={t('products.id')}>{product.id}</DetailList>
        <DetailList name={t('products.name')}>{product.name}</DetailList>
        <DetailList name={t('products.price')}>
          {#if !product.amount || parseFloat(String(product.amount)) === 0}
            <span class="font-bold text-green-600">{t('carts.free')}</span>
          {:else}
            {costFormat(product.amount)} {drawer.currency || ''}
          {/if}
        </DetailList>
        <DetailList name={t('products.slug')}>{product.slug}</DetailList>
        <DetailList name={t('products.metadata')}>
          {#each product.metadata || [] as data (data.key)}
            <div>{data.key}: {data.value}</div>
          {/each}
        </DetailList>
        <DetailList name={t('products.attributes')}>
          {#each product.attributes || [] as item (item)}
            <div>{item}</div>
          {/each}
        </DetailList>
        <DetailList name={t('common.created')}>{formatDate(product.created)}</DetailList>
        {#if product.updated}
          <DetailList name={t('common.updated')}>{formatDate(product.updated)}</DetailList>
        {/if}
        {#if product.images}
          <DetailList name={t('products.images')} grid={true}>
            {#each product.images as item (item.id)}
              <div>
                <a href="/uploads/{item.name}.{item.ext}" target="_blank" aria-label={t('carts.viewFullSizeImage')}>
                  <img
                    style="width: 100%; max-width: 150px"
                    src="/uploads/{item.name}_sm.{item.ext}"
                    alt="{product.name} - {item.name}"
                    loading="lazy"
                  />
                </a>
              </div>
            {/each}
          </DetailList>
        {/if}
        <DetailList name={t('products.briefShortDescription')}>{product.brief}</DetailList>

        <div class="tiptap pt-3">{@html sanitizeHTML(product.description)}</div>
      </dl>
    </div>
  {:else}
    <div class="py-8 text-center text-gray-500">{t('products.failedToLoadProduct')}</div>
  {/if}

  <div class="pt-5">
    <FormButton type="button" name={t('common.close')} color="green" onclick={close} />
  </div>
</div>
