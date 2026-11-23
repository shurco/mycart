<script lang="ts">
  import type { CartItem } from '$lib/types/models'
  import CartItemCard from '$lib/components/CartItemCard.svelte'
  import { getItemKey } from '$lib/utils/checkout'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    cart: CartItem[]
    highlightedItems: Set<string>
  }

  let { cart, highlightedItems }: Props = $props()
</script>

<!-- Cart Items -->
<div class="mb-8">
  <h2 class="mb-6 text-3xl font-black tracking-tighter text-black uppercase">
    {t('cart.itemsCount', { count: cart.length })}
  </h2>
  <div class="space-y-4">
    {#each cart as item (`${item.id}-${item.variant_id || 'no-variant'}`)}
      <CartItemCard
        {item}
        highlighted={highlightedItems.has(getItemKey(item))}
        needsDeletion={item.needsDeletion || false}
      />
    {/each}
  </div>
</div>
