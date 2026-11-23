<script lang="ts">
  import { Chip, ChipGroup, DrawerFooter, DrawerHeader } from '$lib/components'
  import type { SymbolDisplaySettings } from '$lib/types/models'
  import { translate } from '$lib/i18n'

  // Reactive translation function
  let t = $derived($translate)

  interface Props {
    symbolDisplay: SymbolDisplaySettings
    onsubmit: () => void
    onclose: () => void
  }

  let { symbolDisplay = $bindable(), onsubmit, onclose }: Props = $props()

  const sections = [
    { context: 'admin', title: 'settings.adminPanelDisplay' },
    { context: 'storefront', title: 'settings.storefrontDisplay' }
  ] as const

  const choices = [
    { value: 'currency', label: 'settings.currencySymbol', sample: '$130' },
    { value: 'language', label: 'settings.languageSymbol', sample: '130 Dollar' }
  ] as const
</script>

<DrawerHeader title={t('settings.currencyDisplay')} />

<form onsubmit={(e) => { e.preventDefault(); onsubmit() }}>
  <div class="flow-root">
    {#each sections as section (section.context)}
      <div class="mb-5">
        <h3>{t(section.title)}</h3>
        <ChipGroup class="mt-3">
          {#each choices as choice (choice.value)}
            <Chip
              class="flex-1 py-3 text-center"
              active={symbolDisplay[section.context] === choice.value}
              onclick={() => symbolDisplay[section.context] = choice.value}
            >
              {t(choice.label)}
              <span class="mt-1 block text-xs text-gray-500">{choice.sample}</span>
            </Chip>
          {/each}
        </ChipGroup>
      </div>
    {/each}
  </div>

  <DrawerFooter {onclose} submitLabel={t('common.save')} />
</form>
