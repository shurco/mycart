export function sortByDate<T extends { created?: string }>(items: T[], order: 'asc' | 'desc' = 'desc'): T[] {
  return [...items].sort((a, b) => {
    const dateA = a.created ? new Date(a.created).getTime() : 0
    const dateB = b.created ? new Date(b.created).getTime() : 0
    return order === 'desc' ? dateB - dateA : dateA - dateB
  })
}

export function sortByField<T>(items: T[], field: keyof T, order: 'asc' | 'desc' = 'asc'): T[] {
  return [...items].sort((a, b) => {
    const aVal = a[field]
    const bVal = b[field]

    if (aVal === bVal) return 0
    if (aVal == null) return 1
    if (bVal == null) return -1

    const comparison = aVal < bVal ? -1 : 1
    return order === 'desc' ? -comparison : comparison
  })
}
