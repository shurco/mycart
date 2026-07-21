<script lang="ts">
  interface Props {
    quantity: number
    min?: number
    max?: number
    disabled?: boolean
    onIncrement: () => void
    onDecrement: () => void
    onChange: (value: number) => void
  }

  let { quantity, min = 1, max, disabled = false, onIncrement, onDecrement, onChange }: Props = $props()

  let inputValue = $state(String(quantity))

  // Sync input value when quantity prop changes
  $effect(() => {
    inputValue = String(quantity)
  })

  function handleIncrement() {
    if (disabled || (max !== undefined && quantity >= max)) return
    onIncrement()
  }

  function handleDecrement() {
    if (disabled || quantity <= min) return
    onDecrement()
  }

  function handleInputChange(e: Event) {
    const target = e.target as HTMLInputElement
    inputValue = target.value
  }

  function handleInputBlur() {
    const parsed = parseInt(inputValue)

    if (isNaN(parsed)) {
      // Invalid input, revert to current quantity
      inputValue = String(quantity)
      return
    }

    // Clamp to min/max
    let newValue = Math.max(min, parsed)
    if (max !== undefined) {
      newValue = Math.min(max, newValue)
    }

    inputValue = String(newValue)

    if (newValue !== quantity) {
      onChange(newValue)
    }
  }

  let decrementDisabled = $derived(disabled || quantity <= min)
  let incrementDisabled = $derived(disabled || (max !== undefined && quantity >= max))
</script>

<div class="flex items-center gap-2">
  <button
    type="button"
    onclick={handleDecrement}
    disabled={decrementDisabled}
    class="border-4 border-black bg-white px-3 py-2 text-xl font-black transition-all hover:-translate-x-0.5 hover:-translate-y-0.5 hover:shadow-[4px_4px_0px_0px_rgba(0,0,0,1)] disabled:opacity-50 disabled:hover:translate-x-0 disabled:hover:translate-y-0 disabled:hover:shadow-none"
    aria-label="Decrease quantity"
  >
    -
  </button>

  <input
    type="text"
    value={inputValue}
    oninput={handleInputChange}
    onblur={handleInputBlur}
    {disabled}
    class="w-16 border-4 border-black bg-white px-3 py-2 text-center text-lg font-black disabled:opacity-50"
    aria-label="Quantity"
  />

  <button
    type="button"
    onclick={handleIncrement}
    disabled={incrementDisabled}
    class="border-4 border-black bg-white px-3 py-2 text-xl font-black transition-all hover:-translate-x-0.5 hover:-translate-y-0.5 hover:shadow-[4px_4px_0px_0px_rgba(0,0,0,1)] disabled:opacity-50 disabled:hover:translate-x-0 disabled:hover:translate-y-0 disabled:hover:shadow-none"
    aria-label="Increase quantity"
  >
    +
  </button>
</div>
