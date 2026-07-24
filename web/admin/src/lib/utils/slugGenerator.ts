/**
 * Generate URL-friendly slug from product name (client-side fallback)
 * Transforms: "My Product!" → "my-product"
 *
 * @param name - Product name to convert
 * @returns URL-friendly slug string
 */
export function generateSlugFallback(name: string): string {
  return name
    .toLowerCase()                 // "My Product" → "my product"
    .replace(/\s+/g, '-')          // "my product" → "my-product"
    .replace(/[^a-z0-9-]/g, '')    // Remove special chars
    .replace(/-+/g, '-')           // Remove consecutive hyphens
    .replace(/^-|-$/g, '')         // Trim hyphens from edges
}

/**
 * Generate slug from API endpoint
 *
 * @param name - Product name
 * @returns Promise<string> - Generated slug from server
 * @throws Error if API call fails or returns invalid response
 */
export async function generateSlugFromAPI(name: string): Promise<string> {
  const response = await fetch('/api/_/products/slug/generate', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ name })
  })

  if (!response.ok) {
    throw new Error(`API returned ${response.status}`)
  }

  const data = await response.json()

  if (!data.result?.slug) {
    throw new Error('Invalid API response format')
  }

  return data.result.slug
}

/**
 * Generate slug with API-first approach and client-side fallback
 *
 * @param name - Product name
 * @returns Promise<{slug: string, error?: string}> - Generated slug and optional error message
 */
export async function generateSlug(name: string): Promise<{slug: string, error?: string}> {
  try {
    const slug = await generateSlugFromAPI(name)
    return { slug }
  } catch (error) {
    // API failed - use client-side fallback
    const slug = generateSlugFallback(name)
    return {
      slug,
      error: 'api_failed'
    }
  }
}
