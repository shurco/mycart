<script lang="ts">
  import { onMount } from 'svelte'
  import { toast } from 'svelte-french-toast'
  import SvgIcon from './SvgIcon.svelte'

  const NOTIFICATION_DURATION = 4000

  const handleNotification = (eventName: string, notificationType: string) => {
    const callback = (e: Event) => {
      const message = (e as CustomEvent<string>).detail
      const options = {
        duration: NOTIFICATION_DURATION,
        style:
          notificationType === 'error'
            ? 'background: #ef4444; color: white;'
            : notificationType === 'warning'
              ? 'background: #eab308; color: white;'
              : notificationType === 'info'
                ? 'background: #3b82f6; color: white;'
                : 'background: #22c55e; color: white;'
      }
      toast(message, options)
    }

    addEventListener(eventName, callback)
    return () => removeEventListener(eventName, callback)
  }

  onMount(() => {
    const cleanup1 = handleNotification('connextError', 'error')
    const cleanup2 = handleNotification('connextSuccess', 'success')
    const cleanup3 = handleNotification('connextWarning', 'warning')
    const cleanup4 = handleNotification('connextInfo', 'info')

    return () => {
      cleanup1()
      cleanup2()
      cleanup3()
      cleanup4()
    }
  })
</script>

<!-- Empty element to scope styles -->
<div class="alert" aria-hidden="true" style="display: none;"></div>

<style>
  @reference "tailwindcss";

  :global(.alert) {
    @apply pointer-events-none fixed right-0 bottom-0 left-0 flex items-start justify-end p-6 px-4 py-6;
    z-index: 999;
  }
</style>
