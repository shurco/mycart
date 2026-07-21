<script lang="ts">
  interface Props {
    id?: string
    value?: boolean
    checked?: boolean
    label?: string
    disabled?: boolean
    onchange?: () => void
  }

  let {
    id = 'name',
    value = $bindable(false),
    checked,
    label,
    disabled = false,
    onchange
  }: Props = $props()

  let computedValue = $derived(checked ?? value)

  function handleChange() {
    onchange?.()
  }
</script>

<label
  for="toggle_{id}"
  class="none relative h-6 w-10 cursor-pointer [-webkit-tap-highlight-color:_transparent]"
  class:opacity-25={disabled}
>
  <input
    type="checkbox"
    id="toggle_{id}"
    bind:checked={value}
    {disabled}
    onchange={handleChange}
    class="peer sr-only [&:checked_+_span_svg[data-checked-icon]]:block [&:checked_+_span_svg[data-unchecked-icon]]:hidden"
  />

  <span
    class="absolute inset-y-0 start-0 z-10 m-1 inline-flex h-4 w-4 items-center justify-center rounded-full bg-white text-gray-400 transition-all peer-checked:start-4 peer-checked:text-green-600"
  >
    <svg data-unchecked-icon xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" viewBox="0 0 20 20" fill="currentColor">
      <path
        fill-rule="evenodd"
        d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
        clip-rule="evenodd"
      />
    </svg>

    <svg
      data-checked-icon
      xmlns="http://www.w3.org/2000/svg"
      class="hidden h-3 w-3"
      viewBox="0 0 20 20"
      fill="currentColor"
    >
      <path
        fill-rule="evenodd"
        d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
        clip-rule="evenodd"
      />
    </svg>
  </span>

  <span class="absolute inset-0 rounded-full bg-gray-300 transition peer-checked:bg-green-500"></span>
</label>
{#if label}
  <span class="ml-3 text-sm font-medium text-gray-700">{label}</span>
{/if}
