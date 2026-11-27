<script lang="ts">
  import { onMount } from 'svelte'
  import Main from '$lib/layouts/Main.svelte'
  import { Chip, ChipGroup, Drawer, FormButton, FormInput, PageHeader, PageState, Section } from '$lib/components'
  import { loadSettings, saveSettings } from '$lib/utils/settingsHelpers'
  import { validators, validateFields } from '$lib/utils/validation'
  import { translate, locale, availableLocales, type Locale } from '$lib/i18n'
  import SeoTextForm from './SeoTextForm.svelte'
  import SeoSitemapForm from './SeoSitemapForm.svelte'
  import SeoCdnForm from './SeoCdnForm.svelte'

  // Reactive translation function
  let t = $derived($translate)
  let currentLocale = $derived($locale)
  let locales = $derived($availableLocales)

  type SeoDrawerMode = 'sitemap' | 'llms' | 'ai' | 'robots' | 'cdn'
  let seoDrawerOpen = $state(false)
  let seoDrawerMode = $state<SeoDrawerMode | null>(null)
  let seoSettings = $state<any>(null)

  interface MainSettings {
    site_name: string
    domain: string
    email: string
  }

  let formData = $state<MainSettings>({
    site_name: '',
    domain: '',
    email: ''
  })
  let formErrors = $state<Record<string, string>>({})
  let loading = $state(true)

  onMount(async () => {
    const loaded = await loadSettings<MainSettings>('main', formData)
    if (loaded) {
      formData = loaded
    }

    // Fetch defaults first
    const defaultsRes = await fetch('/api/_/seo/defaults')
    const defaultsData = await defaultsRes.json()
    const defaults = defaultsData.result || defaultsData.Result || defaultsData

    // Load saved settings
    const seo = await loadSettings('seo', {})

    // Merge: saved settings override defaults
    seoSettings = {
      sitemap_enabled: seo?.sitemap_enabled || false,
      sitemap_include: seo?.sitemap_include || defaults.sitemap_include || '',
      sitemap_exclude: seo?.sitemap_exclude || defaults.sitemap_exclude || '',
      llms_enabled: seo?.llms_enabled || false,
      llms_template: seo?.llms_template || defaults.llms || '',
      ai_enabled: seo?.ai_enabled || false,
      ai_template: seo?.ai_template || defaults.ai || '',
      robots_enabled: seo?.robots_enabled || false,
      robots_template: seo?.robots_template || defaults.robots || '',
      openapi_enabled: seo?.openapi_enabled || false,
      openapi_include: seo?.openapi_include || '',
      openapi_exclude: seo?.openapi_exclude || defaults.openapi_exclude || '',
      cdn_enabled: seo?.cdn_enabled || false,
      cdn_rules: seo?.cdn_rules || defaults.cdn_rules || ''
    }

    loading = false
  })

  async function handleSubmit() {
    formErrors = validateFields(formData, [
      { field: 'site_name', ...validators.minLength(6, t('settings.siteNameMinLength')) },
      { field: 'domain', ...validators.required(t('settings.domainRequired')) },
      { field: 'email', ...validators.email(t('settings.validEmailRequired')) }
    ])

    if (Object.keys(formErrors).length > 0) {
      return
    }

    await saveSettings('main', formData)
  }

  function switchLocale(newLocale: Locale) {
    locale.set(newLocale)
  }

  function openSeoDrawer(mode: SeoDrawerMode) {
    seoDrawerMode = mode
    seoDrawerOpen = true
  }

  async function saveSeoSettings() {
    await saveSettings('seo', seoSettings)
    seoDrawerOpen = false
  }
</script>

<Main>
  <PageHeader title={t('settings.mainSettings')} />

  {#if loading}
    <PageState kind="loading" />
  {:else}
    <form onsubmit={(e) => { e.preventDefault(); handleSubmit() }} class="max-w-2xl space-y-4">
      <FormInput
        id="site_name"
        title={t('settings.siteName')}
        bind:value={formData.site_name}
        error={formErrors.site_name}
        ico="home"
      />
      <FormInput
        id="domain"
        title={t('settings.domain')}
        bind:value={formData.domain}
        error={formErrors.domain}
        ico="link"
      />
      <FormInput
        id="email"
        type="email"
        title={t('settings.email')}
        bind:value={formData.email}
        error={formErrors.email}
        ico="at-symbol"
      />
      <div class="pt-4">
        <FormButton type="submit" name={t('common.save')} variant="primary" />
      </div>
    </form>

    <Section title={t('settings.language')}>
      <ChipGroup>
        {#each locales as loc}
          <Chip active={currentLocale === loc.code} onclick={() => switchLocale(loc.code)}>{loc.name}</Chip>
        {/each}
      </ChipGroup>
    </Section>

    <Section title="SEO">
      <ChipGroup>
        <Chip active={seoSettings?.sitemap_enabled} onclick={() => openSeoDrawer('sitemap')}>Sitemap</Chip>
        <Chip active={seoSettings?.llms_enabled} onclick={() => openSeoDrawer('llms')}>llms.txt</Chip>
        <Chip active={seoSettings?.ai_enabled} onclick={() => openSeoDrawer('ai')}>ai.txt</Chip>
        <Chip active={seoSettings?.robots_enabled} onclick={() => openSeoDrawer('robots')}>robots.txt</Chip>
        <Chip active={seoSettings?.cdn_enabled} onclick={() => openSeoDrawer('cdn')}>CDN</Chip>
      </ChipGroup>
    </Section>
  {/if}

  {#if seoDrawerOpen}
  <Drawer isOpen={seoDrawerOpen} onclose={() => seoDrawerOpen = false} maxWidth="710px">
    {#if seoDrawerMode === 'sitemap'}
      <SeoSitemapForm bind:settings={seoSettings} onsubmit={saveSeoSettings} onclose={() => seoDrawerOpen = false} />
    {:else if seoDrawerMode === 'llms' || seoDrawerMode === 'ai' || seoDrawerMode === 'robots'}
      <SeoTextForm file={seoDrawerMode} bind:settings={seoSettings} onsubmit={saveSeoSettings} onclose={() => seoDrawerOpen = false} />
    {:else if seoDrawerMode === 'cdn'}
      <SeoCdnForm bind:settings={seoSettings} onsubmit={saveSeoSettings} onclose={() => seoDrawerOpen = false} />
    {/if}
  </Drawer>
  {/if}
</Main>
