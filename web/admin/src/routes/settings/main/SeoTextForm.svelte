<script lang="ts">
  import { DrawerHeader, DrawerFooter, FormToggle, FormTextarea } from '$lib/components'
  
  let { file, settings = $bindable(), onsubmit, onclose }: {
    file: 'llms' | 'ai' | 'robots'
    settings: any
    onsubmit: () => void
    onclose: () => void
  } = $props()
  
  const fileMap = {
    llms: { enabled: 'llms_enabled', template: 'llms_template', title: 'llms.txt' },
    ai: { enabled: 'ai_enabled', template: 'ai_template', title: 'ai.txt' },
    robots: { enabled: 'robots_enabled', template: 'robots_template', title: 'robots.txt' }
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

<DrawerHeader title={f.title} />
<form onsubmit={(e) => { e.preventDefault(); onsubmit() }} class="space-y-4">
  <FormToggle id="{file}-enabled" title="Enable" bind:value={settings[f.enabled]} />
  <p class="text-xs text-gray-600 dark:text-gray-400 -mt-2">View: <a href={fileUrl} target="_blank" class="text-blue-600 dark:text-blue-400 hover:underline">{fileUrl}</a></p>
  <FormTextarea id="{file}-template" title="Template" rows={16} bind:value={settings[f.template]} class="font-mono text-sm" />
  <p class="text-xs text-gray-600 dark:text-gray-400">Variables: site.name, site.url, products[], pages[]</p>
  <DrawerFooter {onclose} submitLabel="Save" ondelete={resetDefault} deleteLabel="Reset to default" actionVariant="default" />
</form>
