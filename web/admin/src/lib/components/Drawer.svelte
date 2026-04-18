<script lang="ts">
  import { onDestroy } from 'svelte'

  interface Props {
    isOpen?: boolean
    maxWidth?: string
    backgroundColor?: string
    onclose?: () => void
    children?: import('svelte').Snippet
  }

  let {
    isOpen = $bindable(false),
    maxWidth = '500px',
    backgroundColor = '#fafafa',
    onclose,
    children
  }: Props = $props()

  const ANIMATION_MS = 200

  let isVisible = $state(false)
  let isTransitioning = $state(false)
  let drawerContent: HTMLElement | undefined = $state()

  // Track all outstanding timers so we can clear them on destroy or when
  // isOpen flips while an animation is still in flight. Without this the
  // component leaks timeouts that keep firing (and writing to state) after
  // the drawer is gone, which in turn can trip Svelte's effect scheduler.
  let hideTimer: ReturnType<typeof setTimeout> | null = null
  let closeTimer: ReturnType<typeof setTimeout> | null = null

  function clearTimer(ref: ReturnType<typeof setTimeout> | null) {
    if (ref !== null) clearTimeout(ref)
  }

  $effect(() => {
    if (isOpen) {
      clearTimer(hideTimer)
      hideTimer = null
      if (drawerContent) drawerContent.scrollTop = 0
      toggleBackgroundScrolling(true)
      isVisible = true
    } else {
      toggleBackgroundScrolling(false)
      clearTimer(hideTimer)
      hideTimer = setTimeout(() => {
        isVisible = false
        hideTimer = null
      }, ANIMATION_MS)
    }
  })

  function toggleBackgroundScrolling(enable: boolean) {
    if (typeof document === 'undefined') return
    document.body.style.overflow = enable ? 'hidden' : ''
  }

  function closeDrawer(event: MouseEvent) {
    if (isTransitioning || event.target !== event.currentTarget) return
    isTransitioning = true
    clearTimer(closeTimer)
    closeTimer = setTimeout(() => {
      onclose?.()
      isTransitioning = false
      closeTimer = null
    }, ANIMATION_MS)
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape' || event.key === 'Enter' || event.key === ' ') {
      event.preventDefault()
      closeDrawer(event as unknown as MouseEvent)
    }
  }

  onDestroy(() => {
    clearTimer(hideTimer)
    clearTimer(closeTimer)
    toggleBackgroundScrolling(false)
  })
</script>

{#if isVisible}
  <div class="drawer">
    <div
      class="overlay fixed inset-x-0 inset-y-0 z-50 w-full bg-black transition-opacity select-none {isOpen
        ? 'opacity-50'
        : 'opacity-0'}"
      style="transition-duration: 200ms"
      role="button"
      tabindex="0"
      aria-label="Close drawer"
      onclick={closeDrawer}
      onkeydown={handleKeydown}
    ></div>

    <div
      bind:this={drawerContent}
      id="drawer_content"
      class="content {isOpen ? 'translate-x-0' : 'translate-x-full'}"
      style="max-width: {maxWidth}; transition-duration: 200ms; background-color: {backgroundColor};"
    >
      {#if children}
        {@render children()}
      {/if}
    </div>
  </div>
{/if}

<style>
  @reference "tailwindcss";

  :global(.drawer .content) {
    @apply fixed inset-y-0 right-0 z-[999] flex h-full w-full flex-col overflow-auto bg-white p-6 shadow-2xl transition-transform;
  }
</style>
