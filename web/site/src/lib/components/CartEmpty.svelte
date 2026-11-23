<script lang="ts">
  import { handleNavigation } from '$lib/utils/navigation'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)
</script>

<div class="brutal-card mb-8 p-8 text-center">
  <p class="mb-8 text-lg tracking-wide text-black">
    {t('cart.emptyMessage')}
  </p>

  <div class="flex justify-center">
    <a
      href="/"
      onclick={(e) => handleNavigation(e, '/')}
      class="inline-block cursor-pointer border-4 border-black bg-yellow-300 px-8 py-4 text-lg font-black tracking-wider text-black uppercase transition-all duration-200 hover:-translate-x-1 hover:-translate-y-1 hover:shadow-[12px_12px_0px_0px_rgba(0,0,0,1)]"
    >
      {t('cart.goToHome')}
    </a>
  </div>
</div>
