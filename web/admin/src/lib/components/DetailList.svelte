<script lang="ts">
  interface Props {
    name: string
    grid?: boolean
    fullWidth?: boolean
    children?: import('svelte').Snippet
  }

  let { name, grid = false, fullWidth = false, children }: Props = $props()
</script>

{#if fullWidth}
  <div class="py-3">
    <div class="mb-3 font-medium text-gray-900">{name}</div>
    <div class="text-gray-700">
      {#if children}
        {@render children()}
      {/if}
    </div>
  </div>
{:else}
  <div class="flex gap-4 py-3">
    <div class="w-1/5 flex-none font-medium text-gray-900">{name}</div>
    <div class="w-4 flex-none"></div>
    <div class="grow text-gray-700 {grid ? 'grid grid-cols-3 content-start gap-4' : ''}">
      {#if children}
        {@render children()}
      {/if}
    </div>
  </div>
{/if}
