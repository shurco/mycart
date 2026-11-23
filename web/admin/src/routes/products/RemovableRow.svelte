<script lang="ts">
  import type { Snippet } from 'svelte'
  import { SvgIcon } from '$lib/components'

  interface Props {
    onremove: () => void
    children: Snippet
  }

  let { onremove, children }: Props = $props()
</script>

<div class="flex">
  {@render children()}
  <div
    class="flex-none cursor-pointer pt-3 pl-3"
    role="button"
    tabindex="0"
    onclick={onremove}
    onkeydown={(e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        onremove()
      }
    }}
  >
    <SvgIcon name="trash" className="h-5 w-5" stroke="currentColor" />
  </div>
</div>
