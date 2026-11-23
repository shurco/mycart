<script lang="ts">
  import type { PaymentMethods } from '$lib/types/models'
  import { getAvailableProviders } from '$lib/utils/payment'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    payments: PaymentMethods
    provider: string
  }

  let { payments, provider = $bindable() }: Props = $props()
</script>

<fieldset class="space-y-4">
  {#each getAvailableProviders(payments) as name (name)}
    <div>
      <input type="radio" bind:group={provider} value={name} id={name} class="peer hidden" />
      <label
        for={name}
        class="block cursor-pointer border-4 border-black bg-white p-6 peer-checked:border-yellow-300 peer-checked:bg-yellow-300"
      >
        <p class="mb-2 text-xl font-black tracking-tight text-black uppercase">{t(`cart.${name}`)}</p>
        <p class="text-lg text-black">{t(`cart.${name}Description`)}</p>
      </label>
    </div>
  {/each}
</fieldset>
