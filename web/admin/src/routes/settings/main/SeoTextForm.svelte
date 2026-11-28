<script lang="ts">
  import { DrawerHeader, DrawerFooter, FormToggle, FormTextarea } from '$lib/components'
  import { translate } from '$lib/i18n'

  let t = $derived($translate)

  let { file, settings = $bindable(), onsubmit, onclose }: {
    file: 'llms' | 'ai' | 'robots'
    settings: any
    onsubmit: () => void
    onclose: () => void
  } = $props()
  
  const fileMap = {
    llms: { enabled: 'llms_enabled', template: 'llms_template', titleKey: 'settings.llmsTxt' },
    ai: { enabled: 'ai_enabled', template: 'ai_template', titleKey: 'settings.aiTxt' },
    robots: { enabled: 'robots_enabled', template: 'robots_template', titleKey: 'settings.robotsTxt' }
  }

  const f = $derived(fileMap[file])
  const fileUrl = $derived(`/${file}.txt`)

  async function resetDefault() {
    const res = await fetch('/api/_/seo/defaults')
    const data = await res.json()
    const defaults = data.result || data.Result || data
    settings[f.template] = defaults[file]
  }
</script>

<DrawerHeader title={t(f.titleKey)} />
<form onsubmit={(e) => { e.preventDefault(); onsubmit() }} class="space-y-4">
  <FormToggle id="{file}-enabled" title={t('settings.enableFile')} bind:value={settings[f.enabled]} />
  <p class="text-xs text-gray-600 dark:text-gray-400 -mt-2">{t('settings.viewFile')}: <a href={fileUrl} target="_blank" class="text-blue-600 dark:text-blue-400 hover:underline">{fileUrl}</a></p>
  <FormTextarea id="{file}-template" title={t('settings.template')} rows={16} bind:value={settings[f.template]} class="font-mono text-sm" />
  <p class="text-xs text-gray-600 dark:text-gray-400">{t('settings.templateVariables')}</p>
  <DrawerFooter {onclose} submitLabel={t('common.save')} ondelete={resetDefault} deleteLabel={t('settings.resetToDefault')} actionVariant="default" />
</form>
