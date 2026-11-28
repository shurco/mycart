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
    settings.cdn_rules = defaults.cdn_rules || ''
  }
</script>

<DrawerHeader title={t('settings.cdnCache')} />
<form onsubmit={(e) => { e.preventDefault(); onsubmit() }} class="space-y-4">
  <FormToggle id="cdn-enabled" title={t('settings.enableCdn')} bind:value={settings.cdn_enabled} />
  <p class="text-xs text-gray-600 dark:text-gray-400 -mt-2">{t('settings.cdnRulesDesc')}</p>
  <FormTextarea id="cdn-rules" title={t('settings.cdnRules')} rows={8} bind:value={settings.cdn_rules} placeholder="/products/*|24h&#10;/sitemap.xml|10d" />
  <p class="text-xs text-gray-600 dark:text-gray-400">{t('settings.cdnRulesFormat')}</p>

  <DrawerFooter {onclose} submitLabel={t('common.save')} ondelete={resetDefault} deleteLabel={t('settings.resetToDefault')} actionVariant="default" />
</form>
