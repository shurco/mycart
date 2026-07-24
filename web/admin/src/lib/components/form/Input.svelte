<script lang="ts">
  import SvgIcon from '../SvgIcon.svelte'

  interface Props {
    id?: string
    type?: string
    title?: string
    label?: string
    ico?: string
    error?: string
    value?: string
    placeholder?: string
    disabled?: boolean
    required?: boolean
    min?: string | number
    max?: string | number
    compact?: boolean
    onfocusout?: (event: FocusEvent) => void
    oninput?: (event: Event) => void
  }

  let {
    id = 'name',
    type = 'text',
    title = 'Name',
    label,
    ico = undefined,
    error = undefined,
    value = $bindable(''),
    placeholder = '',
    disabled = false,
    required = false,
    min = undefined,
    max = undefined,
    compact = false,
    onfocusout,
    oninput
  }: Props = $props()

  let computedPlaceholder = $derived(placeholder || `Enter ${id}`)
  let displayLabel = $derived(label || title)
  let inputClass = $derived(compact ? 'form-input field peer !px-2 !py-1 !min-h-0 !w-auto' : 'form-input field peer')
  let labelStyle = $derived(compact ? 'padding-inline-end: 0;' : '')

</script>

<div>
  <label for={id} class={error ? 'border-red-500' : ''} style={labelStyle}>
    <input
      {type}
      {id}
      bind:value
      class={inputClass}
      placeholder={computedPlaceholder}
      autocomplete="on"
      {disabled}
      {required}
      min={min}
      max={max}
      onfocusout={onfocusout}
      oninput={oninput}
    />
    {#if displayLabel && !compact}
      <span
        class="title peer-placeholder-shown:top-1/2 peer-placeholder-shown:text-sm peer-placeholder-shown:text-gray-400 peer-focus:top-0 peer-focus:text-xs peer-focus:text-gray-700"
      >
        {displayLabel}
      </span>
    {/if}
    {#if ico}
      <span class="ico">
        <SvgIcon name={ico} stroke="currentColor" className="h-5 w-5 {error ? 'text-red-500' : 'text-gray-400'}" />
      </span>
    {/if}
  </label>
  {#if error}
    <span class="pl-4 text-sm text-red-500">{error}</span>
  {/if}
</div>
