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
    settings.openapi_include = ''
    settings.openapi_exclude = defaults.openapi_exclude || ''
  }
</script>

<DrawerHeader title="OpenAPI / Swagger" />
<form onsubmit={(e) => { e.preventDefault(); onsubmit() }} class="space-y-4">
  <FormToggle id="openapi-enabled" title={t('settings.enableOpenAPI')} bind:value={settings.openapi_enabled} />
  <p class="text-xs text-gray-600 dark:text-gray-400 -mt-2">{t('settings.viewSwaggerUI')}: <a href="/_/swagger/" target="_blank" class="text-blue-600 dark:text-blue-400 hover:underline">/_/swagger/</a></p>
  <FormTextarea id="openapi-include" title={t('settings.openAPIInclude')} rows={4} bind:value={settings.openapi_include} placeholder="" />
  <FormTextarea id="openapi-exclude" title={t('settings.openAPIExclude')} rows={6} bind:value={settings.openapi_exclude} placeholder="/api/_/*&#10;/api/install*&#10;/api/customer/*" />

  <DrawerFooter {onclose} submitLabel={t('common.save')} ondelete={resetDefault} deleteLabel={t('settings.resetToDefault')} actionVariant="default" />
</form>
