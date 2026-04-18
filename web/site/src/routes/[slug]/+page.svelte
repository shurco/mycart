<script lang="ts">
  import { page } from '$app/state'
  import { apiGet } from '$lib/utils/api'
  import type { Page } from '$lib/types/models'
  import { updateSEOTags } from '$lib/utils/seo'
  import { isBrowser } from '$lib/utils/browser'
  import NotFoundPage from '$lib/components/NotFoundPage.svelte'
  import { translate } from '$lib/i18n'
  import { sanitizeHTML } from '$lib/utils/sanitize'

  // Reactive translation function
  let t = $derived($translate)

  let content = $state<Page | null>(null)
  let notFound = $state(false)
  let loading = $state(true)

  $effect(() => {
    const slug = page.params.slug
    if (slug) {
      // Reset state when slug changes
      content = null
      notFound = false
      loading = true
      loadPage(slug)
    }
  })

  async function loadPage(slug: string) {
    const res = await apiGet<Page>(`/api/pages/${slug}`)
    loading = false

    if (res.success && res.result) {
      content = res.result

      if (isBrowser() && content.seo) {
        updateSEOTags(content.seo)
      }
    } else {
      // Page not found
      notFound = true
    }
  }
</script>

{#if loading}
  <div class="flex min-h-screen items-center justify-center bg-white">
    <div class="inline-block border-4 border-black bg-yellow-300 px-8 py-6">
      <p class="text-xl font-black tracking-wider text-black uppercase">{t('common.loading')}</p>
    </div>
  </div>
{:else if notFound}
  <NotFoundPage />
{:else if content}
  <section class="min-h-screen bg-white px-4 py-12 sm:px-6 lg:px-8">
    <div class="mx-auto max-w-screen-xl">
      <div>
        <h1
          class="mb-8 border-b-4 border-black pb-6 text-4xl font-black tracking-tighter text-black uppercase sm:text-5xl"
        >
          {content.name}
        </h1>
        <div class="tiptap space-y-6 text-lg leading-relaxed text-black">
          {@html sanitizeHTML(content.content)}
        </div>
      </div>
    </div>
  </section>
{/if}
