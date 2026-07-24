<script lang="ts">
  import { onMount } from 'svelte'
  import FormButton from '../form/Button.svelte'
  import SvgIcon from '../SvgIcon.svelte'
  import Alert from '../Alert.svelte'
  import { translate } from '$lib/i18n'
  import { showMessage } from '$lib/utils'

  let t = $derived($translate)

  interface Props {
    onImportComplete?: () => void
  }

  let { onImportComplete }: Props = $props()

  let selectedFile = $state<File | null>(null)
  let previewResult = $state<any>(null)
  let importing = $state(false)
  let exporting = $state(false)
  let showPreview = $state(false)

  function handleFileSelect(event: Event) {
    const target = event.target as HTMLInputElement
    if (target.files && target.files.length > 0) {
      selectedFile = target.files[0]
      previewResult = null
      showPreview = false
    }
  }

  async function handlePreview() {
    if (!selectedFile) {
      showMessage(t('products.csv.noFileSelected'), 'connextError')
      return
    }

    const formData = new FormData()
    formData.append('file', selectedFile)

    try {
      importing = true
      const response = await fetch('/api/_/products/import/preview', {
        method: 'POST',
        body: formData
      })

      const data = await response.json()

      if (response.ok) {
        previewResult = data.result
        showPreview = true
      } else {
        showMessage(data.message || t('products.csv.previewFailed'), 'connextError')
      }
    } catch (error) {
      showMessage(t('products.csv.previewFailed'), 'connextError')
      console.error('Preview error:', error)
    } finally {
      importing = false
    }
  }

  async function handleImport() {
    if (!selectedFile) {
      showMessage(t('products.csv.noFileSelected'), 'connextError')
      return
    }

    const formData = new FormData()
    formData.append('file', selectedFile)

    try {
      importing = true
      const response = await fetch('/api/_/products/import', {
        method: 'POST',
        body: formData
      })

      const data = await response.json()

      if (response.ok) {
        showMessage(t('products.csv.importSuccess', { count: data.result.imported }), 'connextSuccess')
        selectedFile = null
        previewResult = null
        showPreview = false

        // Reset file input
        const fileInput = document.getElementById('csv-file-input') as HTMLInputElement
        if (fileInput) fileInput.value = ''

        onImportComplete?.()
      } else {
        showMessage(data.message || t('products.csv.importFailed'), 'connextError')
      }
    } catch (error) {
      showMessage(t('products.csv.importFailed'), 'connextError')
      console.error('Import error:', error)
    } finally {
      importing = false
    }
  }

  async function handleExport() {
    try {
      exporting = true
      const response = await fetch('/api/_/products/export', {
        method: 'GET'
      })

      if (response.ok) {
        const blob = await response.blob()
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `products-${new Date().toISOString().split('T')[0]}.csv`
        document.body.appendChild(a)
        a.click()
        window.URL.revokeObjectURL(url)
        document.body.removeChild(a)
        showMessage(t('products.csv.exportSuccess'), 'connextSuccess')
      } else {
        showMessage(t('products.csv.exportFailed'), 'connextError')
      }
    } catch (error) {
      showMessage(t('products.csv.exportFailed'), 'connextError')
      console.error('Export error:', error)
    } finally {
      exporting = false
    }
  }
</script>

<div class="space-y-6">
  <!-- Export Section -->
  <div class="rounded-lg border border-gray-200 bg-white p-6">
    <div class="mb-4 flex items-center justify-between">
      <div>
        <h3 class="text-lg font-semibold text-gray-900">{t('products.csv.exportProducts')}</h3>
        <p class="text-sm text-gray-500">{t('products.csv.exportDescription')}</p>
      </div>
      <FormButton
        type="button"
        onclick={handleExport}
        disabled={exporting}
        variant="secondary"
      >
        <SvgIcon name="arrow-path" className="mr-2 h-4 w-4" />
        {exporting ? t('common.loading') : t('products.csv.export')}
      </FormButton>
    </div>
  </div>

  <!-- Import Section -->
  <div class="rounded-lg border border-gray-200 bg-white p-6">
    <div class="mb-4">
      <h3 class="text-lg font-semibold text-gray-900">{t('products.csv.importProducts')}</h3>
      <p class="text-sm text-gray-500">{t('products.csv.importDescription')}</p>
    </div>

    <div class="mb-4">
      <label
        for="csv-file-input"
        class="mb-2 block text-sm font-medium text-gray-700"
      >
        {t('products.csv.selectFile')}
      </label>
      <input
        id="csv-file-input"
        type="file"
        accept=".csv"
        onchange={handleFileSelect}
        class="block w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 focus:border-blue-500 focus:ring-blue-500"
      />
    </div>

    {#if selectedFile}
      <div class="mb-4 flex gap-2">
        <FormButton
          type="button"
          onclick={handlePreview}
          disabled={importing}
          variant="secondary"
        >
          <SvgIcon name="eye" className="mr-2 h-4 w-4" />
          {t('products.csv.preview')}
        </FormButton>
        <FormButton
          type="button"
          onclick={handleImport}
          disabled={importing || !previewResult}
        >
          <SvgIcon name="arrow-path" className="mr-2 h-4 w-4" />
          {importing ? t('common.loading') : t('products.csv.import')}
        </FormButton>
      </div>
    {/if}

    <!-- Preview Results -->
    {#if showPreview && previewResult}
      <div class="mt-4 rounded-lg bg-gray-50 p-4">
        <h4 class="mb-3 font-semibold text-gray-900">{t('products.csv.previewResults')}</h4>

        <div class="mb-4 grid grid-cols-2 gap-4 md:grid-cols-4">
          <div class="rounded bg-white p-3">
            <div class="text-2xl font-bold text-gray-900">{previewResult.total_rows}</div>
            <div class="text-xs text-gray-500">{t('products.csv.totalRows')}</div>
          </div>
          <div class="rounded bg-white p-3">
            <div class="text-2xl font-bold text-green-600">{previewResult.to_add}</div>
            <div class="text-xs text-gray-500">{t('products.csv.toAdd')}</div>
          </div>
          <div class="rounded bg-white p-3">
            <div class="text-2xl font-bold text-blue-600">{previewResult.to_update}</div>
            <div class="text-xs text-gray-500">{t('products.csv.toUpdate')}</div>
          </div>
          <div class="rounded bg-white p-3">
            <div class="text-2xl font-bold text-red-600">{previewResult.errors?.length || 0}</div>
            <div class="text-xs text-gray-500">{t('products.csv.errors')}</div>
          </div>
        </div>

        {#if previewResult.errors && previewResult.errors.length > 0}
          <div class="mt-4">
            <h5 class="mb-2 font-semibold text-red-600">{t('products.csv.validationErrors')}</h5>
            <div class="max-h-60 space-y-2 overflow-y-auto">
              {#each previewResult.errors as error}
                <Alert type="error">
                  <strong>{t('products.csv.line')} {error.line}:</strong> {error.message}
                </Alert>
              {/each}
            </div>
          </div>
        {/if}

        {#if previewResult.errors?.length === 0 && previewResult.total_rows > 0}
          <Alert type="success">
            {t('products.csv.readyToImport')}
          </Alert>
        {/if}
      </div>
    {/if}
  </div>

  <!-- Format Documentation -->
  <div class="rounded-lg border border-gray-200 bg-blue-50 p-6">
    <h3 class="mb-3 flex items-center text-sm font-semibold text-blue-900">
      <SvgIcon name="docs" className="mr-2 h-5 w-5" />
      {t('products.csv.formatGuide')}
    </h3>
    <div class="space-y-2 text-sm text-blue-800">
      <p><strong>{t('products.csv.requiredFields')}:</strong> name, slug, amount, digital</p>
      <p><strong>{t('products.csv.optionalFields')}:</strong> brief, description, images, attributes, quantity, sku, active</p>
      <p><strong>{t('products.csv.variantFields')}:</strong> variants</p>
      <p class="text-xs">{t('products.csv.variantNote')}</p>
    </div>
  </div>
</div>
