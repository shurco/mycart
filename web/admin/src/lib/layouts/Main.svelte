<script lang="ts">
  import { onMount } from 'svelte'
  import { page } from '$app/state'
  import { goto } from '$app/navigation'
  import { base } from '$app/paths'
  import { systemStore } from '$lib/stores/system'
  import { apiGet, apiPost } from '$lib/utils/api'
  import SvgIcon from '$lib/components/SvgIcon.svelte'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    children?: import('svelte').Snippet
  }

  let { children }: Props = $props()

  let version = $state<Record<string, any>>({})

  let currentRoute = $derived(page.url.pathname)

  const loadVersionInfo = async () => {
    const res = await apiGet(`/api/_/version`)
    if (res.success) {
      systemStore.update((store) => ({
        ...store,
        version: res.result
      }))
    }
  }

  onMount(() => {
    loadVersionInfo()
    const unsubscribe = systemStore.subscribe((store) => {
      version = store.version
    })
    return unsubscribe
  })

  const goToRelease = async () => {
    const releaseUrl = version.release_url || 'https://github.com/shurco/mycart'
    window.open(releaseUrl, '_blank')
  }

  const signOut = async (event: MouseEvent) => {
    event.preventDefault()
    const res = await apiPost('/api/sign/out')
    if (res?.success) {
      goto(`${base}/signin`)
    }
  }

  let mainMenu = $derived([
    { name: 'products', path: `${base}/products`, meta: { ico: 'cube', label: () => t('menu.products') } },
    { name: 'carts', path: `${base}/carts`, meta: { ico: 'cart', label: () => t('menu.carts') } },
    { name: 'pages', path: `${base}/pages`, meta: { ico: 'docs', label: () => t('menu.pages') } },
    { name: 'settings', path: `${base}/settings`, meta: { ico: 'booth', divider: true, label: () => t('menu.settings') } }
  ])

  let mainMenuSections = $derived.by(() => {
    if (currentRoute?.includes('/settings')) {
      return [
        { name: 'settingsMain', path: `${base}/settings/main`, meta: { ico: 'home', title: () => t('settings.main') } },
        { name: 'settingsAuth', path: `${base}/settings/auth`, meta: { ico: 'finger-print', title: () => t('settings.auth') } },
        { name: 'settingsPayment', path: `${base}/settings/payment`, meta: { ico: 'money', title: () => t('settings.payment') } },
        {
          name: 'settingsWebhook',
          path: `${base}/settings/webhook`,
          meta: { ico: 'webhook', title: () => t('settings.webhook') }
        },
        { name: 'settingsSocials', path: `${base}/settings/socials`, meta: { ico: 'user-group', title: () => t('settings.social') } },
        { name: 'settingsMail', path: `${base}/settings/mail`, meta: { ico: 'at-symbol', title: () => t('settings.mail') } }
      ]
    }
    return []
  })

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      goToRelease()
    }
  }
</script>

<div class="relative flex h-screen overflow-hidden">
  <div class="flex h-screen w-16 flex-col justify-between border-e border-e-gray-200 bg-white">
    <div>
      <div class="inline-flex h-16 w-16 items-center justify-center">
        <a href={base || '/'} data-sveltekit-preload-data="hover" aria-label="Home">
          <svg class="h-8 text-blue-700" viewBox="0 0 28 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path
              d="M0.41 10.3847C1.14777 7.4194 2.85643 4.7861 5.2639 2.90424C7.6714 1.02234 10.6393 0 13.695 0C16.7507 0 19.7186 1.02234 22.1261 2.90424C24.5336 4.7861 26.2422 7.4194 26.98 10.3847H25.78C23.7557 10.3549 21.7729 10.9599 20.11 12.1147C20.014 12.1842 19.9138 12.2477 19.81 12.3047H19.67C19.5662 12.2477 19.466 12.1842 19.37 12.1147C17.6924 10.9866 15.7166 10.3841 13.695 10.3841C11.6734 10.3841 9.6976 10.9866 8.02 12.1147C7.924 12.1842 7.8238 12.2477 7.72 12.3047H7.58C7.4762 12.2477 7.376 12.1842 7.28 12.1147C5.6171 10.9599 3.6343 10.3549 1.61 10.3847H0.41ZM23.62 16.6547C24.236 16.175 24.9995 15.924 25.78 15.9447H27.39V12.7347H25.78C24.4052 12.7181 23.0619 13.146 21.95 13.9547C21.3243 14.416 20.5674 14.6649 19.79 14.6649C19.0126 14.6649 18.2557 14.416 17.63 13.9547C16.4899 13.1611 15.1341 12.7356 13.745 12.7356C12.3559 12.7356 11.0001 13.1611 9.86 13.9547C9.2343 14.416 8.4774 14.6649 7.7 14.6649C6.9226 14.6649 6.1657 14.416 5.54 13.9547C4.4144 13.1356 3.0518 12.7072 1.66 12.7347H0V15.9447H1.61C2.39051 15.924 3.154 16.175 3.77 16.6547C4.908 17.4489 6.2623 17.8747 7.65 17.8747C9.0377 17.8747 10.392 17.4489 11.53 16.6547C12.1468 16.1765 12.9097 15.9257 13.69 15.9447C14.4708 15.9223 15.2348 16.1735 15.85 16.6547C16.9901 17.4484 18.3459 17.8738 19.735 17.8738C21.1241 17.8738 22.4799 17.4484 23.62 16.6547ZM23.62 22.3947C24.236 21.915 24.9995 21.664 25.78 21.6847H27.39V18.4747H25.78C24.4052 18.4581 23.0619 18.886 21.95 19.6947C21.3243 20.156 20.5674 20.4049 19.79 20.4049C19.0126 20.4049 18.2557 20.156 17.63 19.6947C16.4899 18.9011 15.1341 18.4757 13.745 18.4757C12.3559 18.4757 11.0001 18.9011 9.86 19.6947C9.2343 20.156 8.4774 20.4049 7.7 20.4049C6.9226 20.4049 6.1657 20.156 5.54 19.6947C4.4144 18.8757 3.0518 18.4472 1.66 18.4747H0V21.6847H1.61C2.39051 21.664 3.154 21.915 3.77 22.3947C4.908 23.1889 6.2623 23.6147 7.65 23.6147C9.0377 23.6147 10.392 23.1889 11.53 22.3947C12.1468 21.9165 12.9097 21.6657 13.69 21.6847C14.4708 21.6623 15.2348 21.9135 15.85 22.3947C16.9901 23.1884 18.3459 23.6138 19.735 23.6138C21.1241 23.6138 22.4799 23.1884 23.62 22.3947Z"
              fill="currentColor"
            />
          </svg>
        </a>
      </div>

      <div class="border-t border-gray-100 px-2">
        <ul class="py-4">
          {#each mainMenu as item (item.name)}
            <li class="pb-2 {item.meta.divider ? 'space-y-1 border-t border-gray-100 pt-2' : ''}">
              <a
                href={item.path}
                data-sveltekit-preload-data="hover"
                class="group relative flex justify-center rounded px-2 py-1.5 {currentRoute === item.path ||
                (item.path === `${base}/settings` && currentRoute?.startsWith(`${base}/settings`))
                  ? 'bg-blue-100 text-blue-700'
                  : 'text-gray-500 hover:bg-gray-200 hover:text-gray-700'}"
              >
                <SvgIcon name={item.meta.ico} stroke="currentColor" className="h-6 w-6" />
              </a>
            </li>
          {/each}
        </ul>
      </div>
    </div>

    <div class="sticky right-0 bottom-0 left-0 border-t border-gray-100 bg-white px-2">
      <ul class="py-2">
        <li class="pb-2">
          <a href="/" target="_blank" class="bg-white hover:bg-green-50 hover:text-green-500">
            <div
              class="group relative flex cursor-pointer justify-center rounded px-2 py-1.5 text-gray-500 hover:bg-green-100 hover:text-green-700"
            >
              <SvgIcon name="glob-alt" stroke="currentColor" className="h-6 w-6" />
            </div>
          </a>
        </li>
        <li class="pb-2">
          <a href={`${base}/signin`} onclick={signOut} class="block" aria-label="Sign out">
            <div
              class="group relative flex justify-center rounded px-2 py-1.5 text-gray-500 hover:bg-red-100 hover:text-red-700"
            >
              <SvgIcon name="exit" stroke="currentColor" className="h-6 w-6" />
            </div>
          </a>
        </li>
      </ul>
    </div>
  </div>

  {#if mainMenuSections.length}
    <div class="h-screen w-52 flex-col justify-between border-e border-e-gray-200 bg-white px-2">
      <div class="px-2 py-5">
        <h1><span class="text-gray-300">{t('settings.title')}</span></h1>
      </div>
      <ul class="mt-1.5 space-y-1">
        {#each mainMenuSections as item (item.name)}
          <li>
            <a
              href={item.path}
              data-sveltekit-preload-data="hover"
              class="flex items-center gap-2 rounded px-4 py-2 {currentRoute === item.path
                ? 'bg-blue-100 text-blue-700'
                : 'text-gray-500 hover:bg-gray-200 hover:text-gray-700'}"
            >
              <SvgIcon name={item.meta.ico} stroke="currentColor" className="h-5 w-5" />
              <span class="text-sm whitespace-nowrap">{item.meta.title()}</span>
            </a>
          </li>
        {/each}
      </ul>
    </div>
  {/if}

  <div class="flex flex-grow flex-col overflow-x-hidden overflow-y-auto">
    <div class="relative block w-full grow px-5 pt-5">
      {#if children}
        {@render children()}
      {/if}
    </div>

    <div class="sticky right-0 bottom-0 left-0 flex h-12 border-t border-t-gray-200 bg-zinc-50">
      <div class="flex-none"></div>
      <div class="grow"></div>
      <div
        role="button"
        tabindex="0"
        onclick={goToRelease}
        onkeydown={handleKeydown}
        class="flex-none cursor-pointer p-4 text-xs {version.new
          ? 'border-gray-300 bg-yellow-200 text-gray-500'
          : 'border-gray-100 text-gray-300'}"
        aria-label="View release information"
      >
        {t('common.poweredBy')}
        {#if version.new}
          <span>({version.current_version}→<span class="pl-1 text-red-500">{version.new}</span>)</span>
        {:else}
          <span>({version.current_version || ''})</span>
        {/if}
      </div>
    </div>
  </div>
</div>
