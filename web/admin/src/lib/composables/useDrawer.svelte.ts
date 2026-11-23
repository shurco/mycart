import { DRAWER_CLOSE_DELAY_MS } from '$lib/constants/ui'

export interface DrawerState<T = unknown> {
  isOpen: boolean
  mode: string
  data: T | null
}

export function useDrawer<T = unknown>(initialMode = 'view') {
  let isOpen = $state(false)
  let mode = $state(initialMode)
  let data = $state<T | null>(null)

  function open(newMode: string, newData: T | null = null) {
    mode = newMode
    data = newData
    isOpen = true
  }

  function close() {
    if (isOpen) {
      isOpen = false
      setTimeout(() => {
        data = null
        mode = initialMode
      }, DRAWER_CLOSE_DELAY_MS)
    }
  }

  return {
    get isOpen() {
      return isOpen
    },
    get mode() {
      return mode
    },
    get data() {
      return data
    },
    open,
    close
  }
}
