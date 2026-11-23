<script lang="ts">
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    isFree: boolean
    disabled: boolean
  }

  let { isFree, disabled }: Props = $props()
</script>

<!-- Checkout Button -->
<div class="flex justify-end">
  <button
    type="submit"
    {disabled}
    class="cursor-pointer border-4 border-black bg-green-500 px-12 py-4 text-xl font-black tracking-wider text-white uppercase transition-all duration-200 enabled:hover:-translate-x-1 enabled:hover:-translate-y-1 enabled:hover:shadow-[14px_14px_0px_0px_rgba(0,0,0,1)] disabled:cursor-not-allowed disabled:opacity-50"
  >
    {#if isFree}
      {t('cart.getForFree')}
    {:else}
      {t('cart.checkout')}
    {/if}
  </button>
</div>
