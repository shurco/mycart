<script lang="ts">
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    email: string
    isFree: boolean
  }

  let { email = $bindable(), isFree }: Props = $props()
</script>

<!-- Email Input -->
<div class="mt-16 mb-8">
  <h2 class="mb-6 text-3xl font-black tracking-tighter text-black uppercase">{t('cart.enterEmail')}</h2>
  <p class="mb-4 text-lg tracking-wide text-black">
    {#if isFree}
      {t('cart.emailFreeDescription')}
    {:else}
      {t('cart.emailPaidDescription')}
    {/if}
  </p>
  <label for="email" class="block">
    <input
      type="email"
      bind:value={email}
      id="email"
      required
      class="w-full border-4 border-black bg-white px-6 py-4 text-lg font-black tracking-wider text-black uppercase focus:ring-4 focus:ring-yellow-300 focus:outline-none"
      placeholder={t('cart.emailPlaceholder')}
    />
  </label>
</div>
