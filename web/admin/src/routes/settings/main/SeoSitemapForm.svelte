<script lang="ts">
  import { DrawerHeader, DrawerFooter, FormToggle, FormTextarea } from '$lib/components'
  
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

<DrawerHeader title="Sitemap" />
<form onsubmit={(e) => { e.preventDefault(); onsubmit() }} class="space-y-4">
  <FormToggle id="sitemap-enabled" title="Enable sitemap.xml" bind:value={settings.sitemap_enabled} />
  <p class="text-xs text-gray-600 dark:text-gray-400 -mt-2">View: <a href="/sitemap.xml" target="_blank" class="text-blue-600 dark:text-blue-400 hover:underline">/sitemap.xml</a></p>
  <FormTextarea id="sitemap-include" title="Include (one rule per line)" rows={8} bind:value={settings.sitemap_include} placeholder="/&#10;/products/*" />
  <FormTextarea id="sitemap-exclude" title="Exclude" rows={4} bind:value={settings.sitemap_exclude} />
  
  <DrawerFooter {onclose} submitLabel="Save" ondelete={resetDefault} deleteLabel="Reset to default" actionVariant="default" />
</form>
