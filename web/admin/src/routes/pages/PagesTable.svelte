<script lang="ts">
  import { IconButton } from '$lib/components'
  import { formatDate } from '$lib/utils'
  import type { Page } from '$lib/types/models'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    pages: Page[]
    onedit: (page: Page) => void
    onseo: (page: Page) => void
    ontoggle: (page: Page, index: number) => void
  }

  let { pages, onedit, onseo, ontoggle }: Props = $props()
</script>

<div class="table-wrap">
  <table>
    <thead>
      <tr>
        <th>{t('pages.name')}</th>
        <th class="w-32">{t('pages.position')}</th>
        <th class="w-32">{t('pages.slug')}</th>
        <th class="w-48">{t('common.created')}</th>
        <th class="w-48">{t('common.updated')}</th>
        <th class="w-24"></th>
      </tr>
    </thead>
    <tbody>
      {#each pages as page, index (page.id)}
        <tr class:opacity-30={!page.active}>
          <td>{page.name}</td>
          <td>{page.position || '-'}</td>
          <td>
            <a href="/{page.slug}" target="_blank" class="a-link">{page.slug}</a>
          </td>
          <td>
            {formatDate(page.created)}
          </td>
          <td>
            {#if page.updated}
              {formatDate(page.updated)}
            {/if}
          </td>
          <td>
            <div class="flex items-center gap-2">
              <IconButton ico="pencil-square" label={t('common.edit')} onclick={() => onedit(page)} />
              <IconButton ico="rocket" label={t('pages.seo')} onclick={() => onseo(page)} />
              <IconButton
                ico={page.active ? 'eye' : 'eye-slash'}
                label={t('pages.toggleActive')}
                onclick={() => ontoggle(page, index)}
              />
            </div>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
