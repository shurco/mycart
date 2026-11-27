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
    settings.cdn_rules = defaults.cdn_rules || ''
  }
</script>

<DrawerHeader title="CDN Cache" />
<form onsubmit={(e) => { e.preventDefault(); onsubmit() }} class="space-y-4">
  <FormToggle id="cdn-enabled" title="Enable CDN caching" bind:value={settings.cdn_enabled} />
  <p class="text-xs text-gray-600 dark:text-gray-400 -mt-2">Applies Cache-Control headers to all requests matching rules below</p>
  <FormTextarea id="cdn-rules" title="Rules (path|duration)" rows={8} bind:value={settings.cdn_rules} placeholder="/products/*|24h&#10;/sitemap.xml|10d" />
  <p class="text-xs text-gray-600 dark:text-gray-400">Format: /path|duration (e.g., 24h, 10d)</p>
  
  <DrawerFooter {onclose} submitLabel="Save" ondelete={resetDefault} deleteLabel="Reset to default" actionVariant="default" />
</form>
