<script lang="ts">
  interface Props {
    id: string
    title: string
    description?: string
    value: boolean
    error?: string
  }

  let { id, title, description, value = $bindable(false), error }: Props = $props()
</script>

<div class="mb-4">
  <label for={id} class="flex items-center cursor-pointer">
    <input
      type="checkbox"
      {id}
      bind:checked={value}
      class="mr-3 h-5 w-5 cursor-pointer"
    />
    <div>
      <span class="text-sm font-medium text-gray-700">{title}</span>
      {#if description}
        <p class="text-sm text-gray-500">{description}</p>
      {/if}
    </div>
  </label>
  {#if error}
    <p class="mt-1 text-sm text-red-600">{error}</p>
  {/if}
</div>
