<script lang="ts">
  import type { Snippet } from 'svelte'
  import SvgIcon from '../SvgIcon.svelte'
  import { DEFAULT_BUTTON_NAME } from '$lib/constants/ui'

  interface Props {
    color?: string
    variant?: 'primary' | 'secondary' | 'danger'
    size?: 'sm' | 'md' | 'lg'
    name?: string
    ico?: string
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
    onclick?: (event: MouseEvent) => void
    class?: string
    children?: Snippet
  }

  let {
    color,
    variant = 'primary',
    size = 'md',
    name = DEFAULT_BUTTON_NAME,
    ico = undefined,
    type = 'button',
    disabled = false,
    onclick,
    class: className = '',
    children
  }: Props = $props()

  const COLOR_CLASSES: Record<string, string[]> = {
    gray: ['bg-gray-600', 'bg-gray-500'],
    gray_lite: ['bg-gray-400', 'bg-gray-300'],
    green: ['bg-green-600', 'bg-green-500'],
    yellow: ['bg-yellow-600', 'bg-yellow-500'],
    red: ['bg-red-600', 'bg-red-500'],
    cyan: ['bg-cyan-600', 'bg-cyan-500']
  }

  const VARIANT_CLASSES: Record<string, string> = {
    primary: 'bg-blue-600 hover:bg-blue-700 text-white',
    secondary: 'bg-gray-200 hover:bg-gray-300 text-gray-800 border border-gray-300',
    danger: 'bg-red-600 hover:bg-red-700 text-white'
  }

  const SIZE_CLASSES: Record<string, string> = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-8 py-2 text-sm',
    lg: 'px-10 py-3 text-base'
  }

  let colorClasses = $derived(
    color && COLOR_CLASSES[color]
      ? `${COLOR_CLASSES[color][0]} active:${COLOR_CLASSES[color][1]}`
      : VARIANT_CLASSES[variant]
  )
  let sizeClasses = $derived(SIZE_CLASSES[size])
  let icoClasses = $derived(ico ? 'focus:outline-none focus:ring' : '')
</script>

<button
  class="group relative inline-flex cursor-pointer items-center overflow-hidden rounded font-medium {colorClasses} {sizeClasses} {icoClasses} {className} disabled:opacity-50 disabled:cursor-not-allowed"
  {type}
  {disabled}
  onclick={onclick}
>
  {#if ico}
    <SvgIcon
      name={ico}
      stroke="currentColor"
      className="h-4 w-4 absolute -start-full transition-all group-hover:start-4"
    />
  {/if}
  {#if children}
    {@render children()}
  {:else}
    <span class="{ico ? 'transition-all group-hover:ms-2 group-hover:-me-2' : ''}">
      {name}
    </span>
  {/if}
</button>
