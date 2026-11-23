<script lang="ts">
  import type { Snippet } from 'svelte'

  interface Props {
    title: string
    addLabel: string
    onadd: () => void
    children: Snippet
  }

  let { title, addLabel, onadd, children }: Props = $props()
</script>

<hr />
<p class="font-semibold">{title}</p>
{@render children()}
<div class="flex">
  <div class="grow"></div>
  <div class="mt-2 flex-none">
    <button
      type="button"
      class="shrink-0 rounded-lg bg-gray-200 p-2 text-sm font-medium text-gray-700"
      onclick={onadd}
    >
      {addLabel}
    </button>
  </div>
</div>
