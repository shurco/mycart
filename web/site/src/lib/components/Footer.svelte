<script lang="ts">
  import { settingsStore } from '$lib/stores/settings'
  import { socialUrl } from '$lib/utils/socialUrl'
  import { handleNavigation } from '$lib/utils/navigation'
  import { page } from '$app/state'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  const SOCIALS_SVG_PATH = '/assets/img/socials.svg#'

  let settings = $derived($settingsStore)
  let pages = $derived(settings?.pages || [])
  let socials = $derived(settings?.socials || {})
  let footerPages = $derived(pages.filter((p) => p.position === 'footer'))
  
  // Reactive current pathname
  let currentPathname = $derived(page.url.pathname)

  // Checks if page is active by current URL
  function isActive(slug: string): boolean {
    return currentPathname === `/${slug}`
  }
</script>

<footer class="border-t-4 border-yellow-300 bg-black text-white">
  <div class="mx-auto max-w-screen-xl px-4 py-10 sm:px-6 lg:px-8">
    <div class="mb-8 flex flex-col items-start justify-between gap-8 lg:flex-row lg:items-center">
      {#if footerPages.length > 0}
        <nav class="flex flex-wrap gap-4">
          {#each footerPages as page (page.id)}
            <a
              href="/{page.slug}"
              onclick={(e) => handleNavigation(e, `/${page.slug}`)}
              class="inline-block cursor-pointer border-2 px-4 py-2 text-sm font-black uppercase transition-all duration-200 {isActive(page.slug)
                ? 'border-yellow-300 bg-yellow-300 text-black'
                : 'border-transparent hover:border-yellow-300 hover:bg-yellow-300 hover:text-black'}"
            >
              {page.name}
            </a>
          {/each}
        </nav>
      {/if}

      {#if Object.keys(socials).length > 0}
        <div class="flex items-center gap-3">
          <span class="mr-2 text-xs font-black tracking-wider text-yellow-300 uppercase">{t('footer.follow')}</span>
          <ul class="flex items-center gap-2">
            {#each Object.entries(socials) as [key, value], i (key)}
              {#if value}
                <li>
                  <a
                    href="{socialUrl[key]}{value}"
                    rel="noreferrer"
                    target="_blank"
                    class="block cursor-pointer border-4 border-white bg-white p-1 text-black transition-all duration-200 hover:-translate-x-1 hover:-translate-y-1 hover:shadow-[6px_6px_0px_0px_rgba(255,235,59,1)]"
                    aria-label={key}
                  >
                    <svg class="h-7 w-7">
                      <use href="{SOCIALS_SVG_PATH}{key}" />
                    </svg>
                  </a>
                </li>
              {/if}
            {/each}
          </ul>
        </div>
      {/if}
    </div>

    <div class="border-t-4 border-yellow-300 pt-6">
      <div class="flex flex-col items-center justify-between gap-4 sm:flex-row">
        <p class="text-xs font-black tracking-widest text-yellow-300 uppercase">
          © {new Date().getFullYear()} {t('footer.allRightsReserved')}
        </p>
        <a
          target="_blank"
          rel="noopener noreferrer"
          href="https://github.com/shurco/mycart"
          class="cursor-pointer text-xs font-black tracking-wider text-yellow-300 uppercase transition-colors duration-200 hover:text-white"
        >
          {t('footer.poweredBy')}
        </a>
      </div>
    </div>
  </div>
</footer>
