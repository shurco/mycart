<script lang="ts">
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    accountEnabled: boolean
  }

  let { accountEnabled }: Props = $props()
</script>

<!-- A download has no quantity to choose, so the stepper gives way
     to what the buyer needs to know instead: nothing arrives by
     post, and the link reaches them either way. The second
     sentence is about the cabinet, and is said only where the
     cabinet is switched on. -->
<div class="mb-6 border-4 border-black bg-blue-100 p-4">
  <span
    class="mb-3 inline-block border-4 border-black bg-yellow-300 px-3 py-1 text-xs font-black tracking-wider text-black uppercase"
  >
    {t('product.download')}
  </span>
  <p class="text-sm leading-relaxed font-bold text-black">
    {t('product.downloadDelivery')}
  </p>
  {#if accountEnabled}
    <p class="mt-2 text-sm leading-relaxed text-black">
      {t('product.downloadInAccount')}
    </p>
  {/if}
</div>
