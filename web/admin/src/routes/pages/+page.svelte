<script lang="ts">
  import { onMount } from 'svelte'
  import Main from '$lib/layouts/Main.svelte'
  import Drawer from '$lib/components/Drawer.svelte'
  import PageSeo from '$lib/components/page/Seo.svelte'
  import FormButton from '$lib/components/form/Button.svelte'
  import FormInput from '$lib/components/form/Input.svelte'
  import FormSelect from '$lib/components/form/Select.svelte'
  import Editor from '$lib/components/Editor.svelte'
  import PagesTable from './PagesTable.svelte'
  import Pagination from '$lib/components/Pagination.svelte'
  import { PageHeader, PageState, DrawerHeader, DrawerFooter, FormGroup } from '$lib/components'
  import { loadData, saveData, deleteData, toggleActive as toggleActiveApi } from '$lib/utils/apiHelpers'
  import { confirmDelete } from '$lib/utils'
  import { validators, validateFields } from '$lib/utils/validation'
  import { MIN_NAME_LENGTH, MIN_SLUG_LENGTH, ERROR_MESSAGES } from '$lib/constants/validation'
  import type { Page } from '$lib/types/models'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface PagesResponse {
    pages: Page[]
    total: number
    page: number
    limit: number
  }

  import { DEFAULT_PAGE_SIZE } from '$lib/constants/pagination'
  import { DRAWER_CLOSE_DELAY_MS } from '$lib/constants/ui'
  import { createDelayedReset } from '$lib/utils/delayedReset'

  type PageForm = Omit<Page, 'id' | 'created' | 'updated' | 'seo'>

  const emptyForm = (): PageForm => ({
    name: '',
    slug: '',
    position: 'header',
    content: '',
    active: true
  })

  let pages = $state<Page[]>([])
  let loading = $state(true)
  let drawerOpen = $state(false)
  let drawerMode = $state<'add' | 'edit' | 'seo' | 'view'>('add')
  let drawerPage = $state<Page | null>(null)
  let currentPage = $state(1)
  let limit = $state(DEFAULT_PAGE_SIZE)
  let total = $state(0)

  let formData = $state<PageForm>(emptyForm())

  const positionOptions = ['header', 'footer']
  let formErrors = $state<Record<string, string>>({})

  onMount(async () => {
    await loadPages()
  })

  async function loadPages(page = currentPage) {
    loading = true
    currentPage = page
    const result = await loadData<PagesResponse>(
      `/api/_/pages?page=${page}&limit=${limit}`,
      t('pages.failedToLoad')
    )
    if (result) {
      pages = result.pages || []
      total = result.total || 0
    }
    loading = false
  }

  function handlePageChange(page: number) {
    loadPages(page)
  }

  function openAdd() {
    formData = emptyForm()
    formErrors = {}
    drawerPage = null
    drawerMode = 'add'
    drawerOpen = true
  }

  async function openEdit(page: Page) {
    // Load full page data including content
    const fullPage = await loadData<Page>(`/api/_/pages/${page.id}`, t('pages.failedToLoadPage'))
    if (!fullPage) {
      return
    }

    formData = {
      name: fullPage.name || '',
      slug: fullPage.slug || '',
      position: fullPage.position || 'header',
      content: fullPage.content ?? '',
      active: fullPage.active !== undefined ? fullPage.active : true
    }
    drawerPage = fullPage
    formErrors = {}
    drawerMode = 'edit'
    drawerOpen = true
  }

  const resetDrawer = createDelayedReset(DRAWER_CLOSE_DELAY_MS)

  function closeDrawer() {
    if (drawerOpen) {
      drawerOpen = false
      resetDrawer(() => {
        drawerPage = null
        drawerMode = 'add'
      })
    }
  }

  async function handleSubmit() {
    formErrors = validateFields(formData, [
      { field: 'name', ...validators.minLength(MIN_NAME_LENGTH, ERROR_MESSAGES.NAME_TOO_SHORT) },
      { field: 'slug', ...validators.minLength(MIN_SLUG_LENGTH, ERROR_MESSAGES.SLUG_TOO_SHORT) }
    ])

    if (Object.keys(formErrors).length > 0) {
      return
    }

    const isUpdate = drawerMode === 'edit' && drawerPage !== null
    const url = isUpdate && drawerPage ? `/api/_/pages/${drawerPage.id}` : '/api/_/pages'

    const result = await saveData<Page>(url, formData, isUpdate, t('pages.pageSaved'), t('pages.failedToSavePage'))
    if (result) {
      if (isUpdate && drawerPage) {
        // Find and update the specific page in the list reactively
        const pageId = drawerPage.id
        const pageIndex = pages.findIndex((p) => p.id === pageId)
        if (pageIndex !== -1) {
          pages = pages.map((p, i) => (i === pageIndex ? { ...result } : p))
        }
      } else {
        // For new pages, reload the list
        await loadPages()
      }
      closeDrawer()
    }
  }

  async function handleDelete(page: Page) {
    if (!confirmDelete('page', page.name)) {
      return
    }

    const success = await deleteData(`/api/_/pages/${page.id}`, t('pages.failedToDelete'), t('pages.failedToDelete'))
    if (success) {
      await loadPages()
    }
  }

  async function toggleActive(page: Page, index: number) {
    const originalActive = page.active
    const newActive = !page.active
    
    // Optimistic update - update directly instead of map
    pages[index] = { ...pages[index], active: newActive }

    // Then make API call
    const updatedPage = await toggleActiveApi<Page>(
      `/api/_/pages/${page.id}/active`,
      t('pages.pageStatusUpdated'),
      t('pages.failedToUpdatePage')
    )

    // If API call failed, revert the change
    if (!updatedPage) {
      pages[index] = { ...pages[index], active: originalActive }
    } else {
      // Update with the response from server including updated timestamp
      pages[index] = updatedPage
    }
  }

  function handleEditorUpdate(value: string) {
    formData.content = value
  }

  function deleteFromFooter() {
    if (drawerPage) {
      handleDelete(drawerPage)
    }
  }

  function openSeo(page: Page) {
    drawerPage = page
    drawerMode = 'seo'
    drawerOpen = true
  }
</script>

<Main>
  <PageHeader title={t('pages.title')}>
    {#snippet actions()}
      <FormButton name={t('pages.addPage')} variant="primary" ico="plus" onclick={openAdd} />
    {/snippet}
  </PageHeader>

  {#if loading}
    <PageState kind="loading" />
  {:else if pages.length === 0}
    <PageState kind="empty" message={t('pages.noPages')} />
  {:else}
    <PagesTable {pages} onedit={openEdit} onseo={openSeo} ontoggle={toggleActive} />

    {#if total > 0}
      <Pagination
        currentPage={currentPage}
        totalPages={Math.ceil(total / limit)}
        onPageChange={handlePageChange}
      />
    {/if}
  {/if}
</Main>

{#if drawerOpen}
  <Drawer isOpen={drawerOpen} onclose={closeDrawer} maxWidth="710px">
    {#if drawerMode === 'seo' && drawerPage}
      <PageSeo page={drawerPage} onclose={closeDrawer} />
    {:else}
      <DrawerHeader title={drawerMode === 'add' ? t('pages.addPage') : t('pages.editPage')} />

      <form onsubmit={(e) => { e.preventDefault(); handleSubmit() }}>
        <div class="flow-root">
          <dl class="mx-auto -my-3 mt-4 mb-0 space-y-4 text-sm">
            <FormInput id="name" title={t('pages.name')} bind:value={formData.name} error={formErrors.name} ico="at-symbol" />
            <div class="flex">
              <div class="pr-3">
                <FormSelect
                  id="position"
                  title={t('pages.position')}
                  options={positionOptions}
                  bind:value={formData.position}
                  error={formErrors.position}
                />
              </div>
              <div>
                <FormInput id="slug" title={t('pages.slug')} bind:value={formData.slug} error={formErrors.slug} ico="glob-alt" />
              </div>
            </div>

            <FormGroup label={t('pages.content')}>
              <Editor
                bind:modelValue={formData.content}
                placeholder={t('pages.typeContentHere')}
                onupdateModelValue={handleEditorUpdate}
              />
            </FormGroup>
          </dl>
        </div>

        <DrawerFooter
          onclose={closeDrawer}
          submitLabel={drawerMode === 'add' ? t('common.add') : t('common.save')}
          ondelete={drawerMode === 'edit' && drawerPage ? deleteFromFooter : undefined}
          deleteLabel={t('common.delete')}
        />
      </form>
    {/if}
  </Drawer>
{/if}
