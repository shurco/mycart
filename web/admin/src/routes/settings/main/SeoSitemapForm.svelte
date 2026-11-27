<script lang="ts">
  import { DrawerHeader, DrawerFooter, FormToggle, FormTextarea } from '$lib/components'
  import { translate } from '$lib/i18n'

  let t = $derived($translate)

  let { settings = $bindable(), onsubmit, onclose }: {
    settings: any
    onsubmit: () => void
    onclose: () => void
  } = $props()

  async function resetDefault() {
    const res = await fetch('/api/_/seo/defaults')
    const data = await res.json()
    const defaults = data.result || data.Result || data
    settings.sitemap_include = defaults.sitemap_include || ''
    settings.sitemap_exclude = defaults.sitemap_exclude || ''
  }
</script>

<DrawerHeader title={t('settings.sitemap')} />
<form onsubmit={(e) => { e.preventDefault(); onsubmit() }} class="space-y-4">
  <FormToggle id="sitemap-enabled" title={t('settings.enableSitemap')} bind:value={settings.sitemap_enabled} />
  <p class="text-xs text-gray-600 dark:text-gray-400 -mt-2">{t('settings.viewFile')}: <a href="/sitemap.xml" target="_blank" class="text-blue-600 dark:text-blue-400 hover:underline">/sitemap.xml</a></p>
  <FormTextarea id="sitemap-include" title={t('settings.sitemapInclude')} rows={8} bind:value={settings.sitemap_include} placeholder="/&#10;/products/*" />
  <FormTextarea id="sitemap-exclude" title={t('settings.sitemapExclude')} rows={4} bind:value={settings.sitemap_exclude} />

  <DrawerFooter {onclose} submitLabel={t('common.save')} ondelete={resetDefault} deleteLabel={t('settings.resetToDefault')} actionVariant="default" />
</form>
