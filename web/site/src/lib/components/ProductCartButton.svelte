<script lang="ts">
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    inCart: boolean
    disabled: boolean
    onToggle: () => void
  }

  let { inCart, disabled, onToggle }: Props = $props()
</script>

<button
  onclick={onToggle}
  {disabled}
  class="w-full cursor-pointer border-4 border-black px-8 py-4 text-lg font-black tracking-wider uppercase transition-all duration-200 hover:-translate-x-1 hover:-translate-y-1 hover:shadow-[12px_12px_0px_0px_rgba(0,0,0,1)] disabled:cursor-not-allowed disabled:opacity-50 {inCart
    ? 'bg-red-500 text-white'
    : 'bg-green-500 text-white'}"
>
  {#if !inCart}
    <span class="flex items-center justify-center gap-3">
      <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <use href="/assets/img/sprite.svg#plus" />
      </svg>
      <span>{t('product.addToCart')}</span>
    </span>
  {:else}
    <span class="flex items-center justify-center gap-3">
      <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <use href="/assets/img/sprite.svg#minus" />
      </svg>
      <span>{t('product.removeFromCart')}</span>
    </span>
  {/if}
</button>
